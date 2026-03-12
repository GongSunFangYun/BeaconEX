//go:build !windows

package modules

func DLLInjector(
	dllPath string,
	processName string,
	onSuccess func(absPath string),
) {
}
