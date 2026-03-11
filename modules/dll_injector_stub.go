//go:build !windows

package modules

func DLLInjector(
	dllPath string,
	processName string,
	taskTime string,
	inject bool,
	resetConfig bool,
) {
}
