package util

import "runtime"

type OSType string

const (
	Linux   OSType = "linux"
	Mac     OSType = "mac"
	Windows OSType = "windows"
	Other   OSType = "other"
)

func DetectOS() OSType {
	switch runtime.GOOS {
	case "linux":
		return Linux
	case "darwin":
		return Mac
	case "windows":
		return Windows
	default:
		return Other
	}
}

func IsLinux() bool   { return DetectOS() == Linux }
func IsMac() bool     { return DetectOS() == Mac }
func IsWindows() bool { return DetectOS() == Windows }
func IsOther() bool   { return DetectOS() == Other }
