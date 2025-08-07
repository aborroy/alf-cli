package util

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo holds information about system resources
type SystemInfo struct {
	CPUCount int64 `json:"cpu_count"`
	RAMBytes int64 `json:"ram_bytes"`
	RAMGB    int64 `json:"ram_gb"`
}

// DockerResourceDetector detects CPU & memory limits inside a container (Linux) or
// what Docker Desktop is configured to give containers (macOS/Windows).
type DockerResourceDetector struct{}

func NewDockerResourceDetector() *DockerResourceDetector { return &DockerResourceDetector{} }

// GetSystemInfo ALWAYS returns a non‑nil *SystemInfo. If any probe fails the
// field is left zero and the error is returned alongside: preventing nil‑ptr
// panics in callers that ignore the error.
func (d *DockerResourceDetector) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}
	var errs []string

	if cpu, err := d.GetCPUCount(); err == nil {
		info.CPUCount = cpu
	} else {
		errs = append(errs, err.Error())
	}

	if mem, err := d.GetRAMBytes(); err == nil {
		info.RAMBytes = mem
		info.RAMGB = (int64(mem) + (1<<30 - 1)) >> 30 // round up to GB
	} else {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return info, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return info, nil
}

// GetCPUCount returns the number of CPUs available to containers.
// Linux = real cgroup limits; macOS/Windows : docker info (Desktop limits);
// else = host count.
func (d *DockerResourceDetector) GetCPUCount() (int64, error) {
	switch runtime.GOOS {
	case "linux":
		if quota, period, err := d.readCgroupV2CPUQuota(); err == nil && quota > 0 && period > 0 {
			return quota / period, nil
		}
		if quota, period, err := d.readCgroupV1CPUQuota(); err == nil && quota > 0 && period > 0 {
			return quota / period, nil
		}
		if cpus, err := d.readCgroupCPUSet(); err == nil && cpus > 0 {
			return cpus, nil
		}
		return int64(runtime.NumCPU()), nil

	case "darwin", "windows":
		if n, err := d.dockerInfoInt("{{.NCPU}}"); err == nil && n > 0 {
			return n, nil
		}
		return int64(runtime.NumCPU()), fmt.Errorf("docker info unavailable – falling back to host CPUs")

	default:
		return int64(runtime.NumCPU()), nil
	}
}

// GetRAMBytes returns memory limit in bytes for containers.
func (d *DockerResourceDetector) GetRAMBytes() (int64, error) {
	switch runtime.GOOS {
	case "linux":
		if lim, err := d.readCgroupV2MemoryLimit(); err == nil && lim > 0 {
			return lim, nil
		}
		if lim, err := d.readCgroupV1MemoryLimit(); err == nil && lim > 0 {
			return lim, nil
		}
		return d.readProcMemInfo()

	case "darwin", "windows":
		if b, err := d.dockerInfoInt("{{.MemTotal}}"); err == nil && b > 0 {
			return b, nil
		}
		if runtime.GOOS == "darwin" {
			return d.ramDarwinSysctl()
		}
		return 0, fmt.Errorf("docker info unavailable and host RAM method not implemented for %s", runtime.GOOS)

	default:
		return 0, fmt.Errorf("unsupported OS %s", runtime.GOOS)
	}
}

// dockerInfoInt runs "docker info --format <fmt>"" and parses the result as int64.
func (d *DockerResourceDetector) dockerInfoInt(goTemplate string) (int64, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return 0, fmt.Errorf("docker not installed")
	}
	out, err := exec.Command("docker", "info", "--format", goTemplate).Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	return strconv.ParseInt(s, 10, 64)
}

// readCgroupPaths parses /proc/self/cgroup and returns controller : relativePath (v1).
func (d *DockerResourceDetector) readCgroupPaths() (map[string]string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	paths := make(map[string]string)
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		parts := strings.SplitN(scan.Text(), ":", 3)
		if len(parts) != 3 {
			continue
		}
		ctrls := strings.Split(parts[1], ",")
		for _, c := range ctrls {
			paths[c] = parts[2]
		}
	}
	return paths, scan.Err()
}

// getUnifiedCgroupPath returns the relative path in the unified (v2) hierarchy.
func (d *DockerResourceDetector) getUnifiedCgroupPath() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scan := bufio.NewScanner(f)
	for scan.Scan() {
		parts := strings.SplitN(scan.Text(), ":", 3)
		if len(parts) == 3 && parts[1] == "" {
			return parts[2], nil
		}
	}
	return "", fmt.Errorf("unified cgroup entry not found")
}

// locate constructs the absolute path to a cgroup file for this process.
// For v2 set controller="" and v2=true. For v1, provide controller name.
func (d *DockerResourceDetector) locate(controller, filename string, v2 bool) (string, error) {
	if v2 {
		rel, err := d.getUnifiedCgroupPath()
		if err != nil {
			return "", err
		}
		p := filepath.Join("/sys/fs/cgroup", rel, filename)
		if _, err := os.Stat(p); err != nil {
			return "", err
		}
		return p, nil
	}
	paths, err := d.readCgroupPaths()
	if err != nil {
		return "", err
	}
	rel, ok := paths[controller]
	if !ok {
		return "", fmt.Errorf("controller %s not in /proc/self/cgroup", controller)
	}
	p := filepath.Join("/sys/fs/cgroup", controller, rel, filename)
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

func (d *DockerResourceDetector) readCgroupV2CPUQuota() (quota, period int64, err error) {
	path, err := d.locate("", "cpu.max", true)
	if err != nil {
		return 0, 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 || fields[0] == "max" {
		return 0, 0, fmt.Errorf("no v2 CPU quota set")
	}
	q, err1 := strconv.ParseInt(fields[0], 10, 64)
	p, err2 := strconv.ParseInt(fields[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, fmt.Errorf("parse error in cpu.max")
	}
	return q, p, nil
}

func (d *DockerResourceDetector) readCgroupV1CPUQuota() (quota, period int64, err error) {
	quotaPath, err := d.locate("cpu", "cpu.cfs_quota_us", false)
	if err != nil {
		return 0, 0, err
	}
	periodPath, err := d.locate("cpu", "cpu.cfs_period_us", false)
	if err != nil {
		return 0, 0, err
	}
	q, err1 := d.readIntFromFile(quotaPath)
	p, err2 := d.readIntFromFile(periodPath)
	if err1 != nil || err2 != nil || q <= 0 || p <= 0 {
		return 0, 0, fmt.Errorf("no v1 CPU quota set")
	}
	return q, p, nil
}

func (d *DockerResourceDetector) readCgroupCPUSet() (int64, error) {
	path, err := d.locate("cpuset", "cpuset.cpus", false)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return parseCPUSet(strings.TrimSpace(string(data)))
}

func (d *DockerResourceDetector) readCgroupV2MemoryLimit() (int64, error) {
	path, err := d.locate("", "memory.max", true)
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(data))
	if s == "max" {
		return 0, fmt.Errorf("no v2 memory limit")
	}
	return strconv.ParseInt(s, 10, 64)
}

func (d *DockerResourceDetector) readCgroupV1MemoryLimit() (int64, error) {
	path, err := d.locate("memory", "memory.limit_in_bytes", false)
	if err != nil {
		return 0, err
	}
	return d.readIntFromFile(path)
}

func (d *DockerResourceDetector) readIntFromFile(p string) (int64, error) {
	data, err := os.ReadFile(p)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
}

// parseCPUSet counts CPUs in strings like "0-3,6,8-9".
func parseCPUSet(set string) (int64, error) {
	if set == "" {
		return 0, fmt.Errorf("empty cpuset")
	}
	var count int64
	segments := strings.Split(set, ",")
	for _, seg := range segments {
		if strings.Contains(seg, "-") {
			parts := strings.SplitN(seg, "-", 2)
			start, err1 := strconv.Atoi(parts[0])
			end, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || end < start {
				return 0, fmt.Errorf("invalid cpuset segment %s", seg)
			}
			count += int64(end - start + 1)
		} else {
			count++
		}
	}
	return count, nil
}

// readProcMemInfo returns MemTotal in bytes from /proc/meminfo.
func (d *DockerResourceDetector) readProcMemInfo() (int64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scan := bufio.NewScanner(file)
	for scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseInt(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return kb * 1024, nil
			}
		}
	}
	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func (d *DockerResourceDetector) ramDarwinSysctl() (int64, error) {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}
