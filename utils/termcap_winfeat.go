//go:build windows

package utils

import (
	"golang.org/x/sys/windows"
)

const enableVirtualTerminalProcessing = 0x0004

func tryEnableWindowsVT() (vtEnabled bool) {
	stdout := windows.Stdout

	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err != nil {
		return false
	}

	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}

	newMode := mode | enableVirtualTerminalProcessing
	if err := windows.SetConsoleMode(stdout, newMode); err != nil {
		return false
	}
	return true
}
