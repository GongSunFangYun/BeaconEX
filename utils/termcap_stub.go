//go:build !windows

package utils

func tryEnableWindowsVT() bool {
	return true
}
