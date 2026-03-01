//go:build !windows

package modules

func DLLInjector(
	dllPath string,
	processName string,
	taskTime string,
	inject bool,
	resetConfig bool,
) {
// 无实现，该源文件用于通过编译，实际注入器功能不会在其他平台启用
}
