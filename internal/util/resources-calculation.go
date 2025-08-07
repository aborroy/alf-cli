package util

import (
	"fmt"
	"math"
	"strings"
)

type (
	// CPU and memory for either "limits" or "reservations".
	CPUMem struct {
		CPU float64
		MiB int64
	}
	// Both limits and reservations for a service.
	Resource struct {
		Limits       CPUMem
		Reservations CPUMem
	}
)

// Scale returns a new map with every limit / reservation
// multiplied so that the **totals** equal targetMiB / targetCPU.
// Any service not listed in `defaults` is ignored.
func Scale(targetMiB int64, targetCPU float64) (map[string]Resource, error) {
	limitMiB, limitCPU := 0, 0.0
	for _, r := range defaults {
		limitMiB += int(r.Limits.MiB)
		limitCPU += r.Limits.CPU
	}
	memFactor := float64(targetMiB) / float64(limitMiB)
	cpuFactor := targetCPU / limitCPU

	out := make(map[string]Resource, len(defaults))
	for name, r := range defaults {
		out[name] = Resource{
			Limits: CPUMem{
				CPU: round(r.Limits.CPU * cpuFactor),
				MiB: int64(float64(r.Limits.MiB) * memFactor),
			},
			Reservations: CPUMem{
				CPU: round(r.Reservations.CPU * cpuFactor),
				MiB: int64(float64(r.Reservations.MiB) * memFactor),
			},
		}
	}
	return out, nil
}

func round(f float64) float64 { return math.Round(f*100) / 100 }

func FormatMem(miB int64) string {
	if miB%1024 == 0 {
		return fmt.Sprintf("%dg", miB/1024)
	}
	return fmt.Sprintf("%dm", miB)
}

var defaults = map[string]Resource{
	"database": {
		Limits:       CPUMem{CPU: 1, MiB: 1024},
		Reservations: CPUMem{CPU: .5, MiB: 512},
	},
	"activemq": {
		Limits:       CPUMem{CPU: 1, MiB: 1024},
		Reservations: CPUMem{CPU: .5, MiB: 512},
	},
	"transform-core-aio": {
		Limits:       CPUMem{CPU: 2, MiB: 2048},
		Reservations: CPUMem{CPU: 1, MiB: 1024},
	},
	"alfresco": {
		Limits:       CPUMem{CPU: 2, MiB: 3072},
		Reservations: CPUMem{CPU: 1, MiB: 2048},
	},
	"solr6": {
		Limits:       CPUMem{CPU: 2, MiB: 1536},
		Reservations: CPUMem{CPU: 1, MiB: 768},
	},
	"share": {
		Limits:       CPUMem{CPU: 1, MiB: 1024},
		Reservations: CPUMem{CPU: .5, MiB: 512},
	},
	"content-app": {
		Limits:       CPUMem{CPU: .5, MiB: 512},
		Reservations: CPUMem{CPU: .25, MiB: 256},
	},
	"control-center": {
		Limits:       CPUMem{CPU: .5, MiB: 512},
		Reservations: CPUMem{CPU: .25, MiB: 256},
	},
	"proxy": {
		Limits:       CPUMem{CPU: .5, MiB: 512},
		Reservations: CPUMem{CPU: .25, MiB: 256},
	},
}

// FromHuman: 20g to 20480 MiB
func FromHuman(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	scale := int64(1)
	switch {
	case strings.HasSuffix(s, "g"), strings.HasSuffix(s, "gb"):
		scale = 1024
		s = strings.TrimSuffix(strings.TrimSuffix(s, "g"), "b")
	case strings.HasSuffix(s, "m"), strings.HasSuffix(s, "mb"):
		s = strings.TrimSuffix(strings.TrimSuffix(s, "m"), "b")
	default:
		return 0, fmt.Errorf("use m/mb or g/gb")
	}
	val, err := fmt.Sscanf(s, "%d", &scale)
	if err != nil || val != 1 {
		return 0, fmt.Errorf("cannot parse %q", s)
	}
	return scale * scale, nil // val * MiB
}
