//go:build windows

package modules

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"bex/utils"
	"github.com/shirou/gopsutil/v3/process"
)

var injectorConfigPath string

func init() {
	baseDir := utils.GetBaseDirectory()
	injectorConfigPath = filepath.Join(baseDir, "injector.config.json")
}

type InjectorConfig struct {
	LastInjectPath string `json:"last_inject_path"`
}

func DLLInjector(
	dllPath string,
	processName string,
	taskTime string,
	inject bool,
	resetConfig bool,
) {
	if runtime.GOOS != "windows" {
		utils.LogError("DLL注入器仅支持 Windows 系统")
		utils.LogInfo("当前系统: %s", runtime.GOOS)
		return
	}

	injector := &DLLInjectorInstance{
		DLLPath:     dllPath,
		ProcessName: processName,
		TaskTime:    taskTime,
		Inject:      inject,
		ResetConfig: resetConfig,
	}

	injector.Execute()
}

type DLLInjectorInstance struct {
	DLLPath     string
	ProcessName string
	TaskTime    string
	Inject      bool
	ResetConfig bool
}

func (d *DLLInjectorInstance) Execute() {
	if d.ResetConfig {
		d.resetConfig()
		return
	}

	if d.TaskTime != "" {
		d.handleTaskMode()
		return
	}

	var dllPath string
	if d.Inject {
		config, err := d.loadConfig()
		if err != nil {
			utils.LogError("读取配置文件失败: %s", err)
			utils.LogInfo("请先使用 -d 指定 DLL 路径进行注入")
			return
		}
		if config.LastInjectPath == "" {
			utils.LogError("配置文件中没有找到上次注入的 DLL 路径")
			utils.LogInfo("请先使用 -d 指定 DLL 路径进行注入")
			return
		}
		dllPath = config.LastInjectPath
	} else {
		dllPath = d.DLLPath
	}

	success := d.InjectDLL(d.ProcessName, dllPath)

	if success && !d.Inject && d.DLLPath != "" {
		err := d.saveConfig(dllPath)
		if err != nil {
			return
		}
	}
}

func (d *DLLInjectorInstance) InjectDLL(processName string, dllPath string) bool {
	utils.LogInfo("开始进行 DLL 注入：")
	utils.LogInfo("目标进程: %s | DLL文件: %s", processName, dllPath)

	pid, err := d.findProcessByName(processName)
	if err != nil {
		utils.LogError("%s[1/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}
	utils.LogInfo("%s[1/9]%s 找到目标进程 PID: %d",
		utils.ColorBlue, utils.ColorClear, pid)

	handle, err := d.openProcess(pid)
	if err != nil {
		utils.LogError("%s[2/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}
	defer func() {
		_ = d.closeHandle(handle)
	}()
	utils.LogInfo("%s[2/9]%s 打开进程并获取句柄",
		utils.ColorBlue, utils.ColorClear)

	kernel32, err := d.getModuleHandle("kernel32.dll")
	if err != nil {
		utils.LogError("%s[3/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}
	utils.LogInfo("%s[3/9]%s 获取 kernel32 模块句柄",
		utils.ColorBlue, utils.ColorClear)

	loadLibraryAddr, err := d.getProcAddress(kernel32, "LoadLibraryW")
	if err != nil {
		utils.LogError("%s[4/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}
	utils.LogInfo("%s[4/9]%s 获取 LoadLibraryW 函数地址: 0x%X",
		utils.ColorBlue, utils.ColorClear, loadLibraryAddr)

	dllPathUTF16, err := d.stringToUTF16Ptr(dllPath)
	if err != nil {
		utils.LogError("%s[5/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}

	allocAddr, err := d.virtualAllocEx(handle, len(dllPathUTF16)*2)
	if err != nil {
		utils.LogError("%s[5/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		return false
	}
	utils.LogInfo("%s[5/9]%s 在目标进程分配内存: 0x%X",
		utils.ColorBlue, utils.ColorClear, allocAddr)

	err = d.writeProcessMemory(handle, allocAddr, dllPathUTF16)
	if err != nil {
		utils.LogError("%s[6/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		_ = d.virtualFreeEx(handle, allocAddr)
		return false
	}
	utils.LogInfo("%s[6/9]%s 将 DLL 路径写入目标进程内存",
		utils.ColorBlue, utils.ColorClear)

	threadHandle, threadID, err := d.createRemoteThread(handle, loadLibraryAddr, allocAddr)
	if err != nil {
		utils.LogError("%s[7/9]%s %s",
			utils.ColorYellow, utils.ColorClear, err.Error())
		_ = d.virtualFreeEx(handle, allocAddr)
		return false
	}
	defer func() {
		_ = d.closeHandle(threadHandle)
	}()
	utils.LogInfo("%s[7/9]%s 创建远程线程执行 LoadLibraryW [TID: %d]",
		utils.ColorBlue, utils.ColorClear, threadID)

	err = d.waitForSingleObject(threadHandle, 5000)
	if err != nil {
		utils.LogWarn("%s[8/9]%s 线程执行超时，但注入可能仍然成功",
			utils.ColorCyan, utils.ColorClear)
	} else {
		utils.LogInfo("%s[8/9]%s 远程线程执行完成",
			utils.ColorBlue, utils.ColorClear)
	}

	exitCode, err := d.getExitCodeThread(threadHandle)
	if err != nil {
		utils.LogWarn("%s[9/9]%s 无法获取线程退出码，但注入可能成功",
			utils.ColorCyan, utils.ColorClear)
	} else if exitCode == 0 {
		utils.LogError("%s[9/9]%s DLL 注入失败，LoadLibraryW 返回 NULL",
			utils.ColorYellow, utils.ColorClear)
		_ = d.virtualFreeEx(handle, allocAddr)
		return false
	} else {
		utils.LogInfo("%s[9/9]%s DLL 加载成功，基地址: 0x%X",
			utils.ColorBlue, utils.ColorClear, exitCode)
	}

	_ = d.virtualFreeEx(handle, allocAddr)

	utils.LogInfo("DLL 注入完成！")
	return true
}

func (d *DLLInjectorInstance) findProcessByName(name string) (uint32, error) {
	processes, err := process.Processes()
	if err != nil {
		return 0, fmt.Errorf("无法获取进程列表: %v", err)
	}

	nameLower := strings.ToLower(name)
	for _, p := range processes {
		procName, err := p.Name()
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(procName), nameLower) {
			return uint32(p.Pid), nil
		}
	}

	return 0, fmt.Errorf("未找到目标进程: %s", name)
}

func (d *DLLInjectorInstance) openProcess(pid uint32) (uintptr, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess := kernel32.NewProc("OpenProcess")

	handle, _, lastErr := procOpenProcess.Call(
		0x1F0FFF,
		0,
		uintptr(pid),
	)

	if handle == 0 {
		return 0, fmt.Errorf("OpenProcess 失败: %v", lastErr)
	}

	return handle, nil
}

func (d *DLLInjectorInstance) closeHandle(handle uintptr) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procCloseHandle := kernel32.NewProc("CloseHandle")

	ret, _, _ := procCloseHandle.Call(handle)
	if ret == 0 {
		return fmt.Errorf("CloseHandle 失败")
	}
	return nil
}

func (d *DLLInjectorInstance) getModuleHandle(moduleName string) (uintptr, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandle := kernel32.NewProc("GetModuleHandleW")

	namePtr, err := syscall.UTF16PtrFromString(moduleName)
	if err != nil {
		return 0, err
	}

	handle, _, lastErr := procGetModuleHandle.Call(uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return 0, fmt.Errorf("GetModuleHandle 失败: %v", lastErr)
	}

	return handle, nil
}

func (d *DLLInjectorInstance) getProcAddress(module uintptr, procName string) (uintptr, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetProcAddress := kernel32.NewProc("GetProcAddress")

	nameBytes := []byte(procName + "\x00")
	addr, _, lastErr := procGetProcAddress.Call(
		module,
		uintptr(unsafe.Pointer(&nameBytes[0])),
	)

	if addr == 0 {
		return 0, fmt.Errorf("GetProcAddress 失败: %v", lastErr)
	}

	return addr, nil
}

func (d *DLLInjectorInstance) stringToUTF16Ptr(s string) ([]uint16, error) {
	return syscall.UTF16FromString(s)
}

func (d *DLLInjectorInstance) virtualAllocEx(handle uintptr, size int) (uintptr, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procVirtualAllocEx := kernel32.NewProc("VirtualAllocEx")

	addr, _, lastErr := procVirtualAllocEx.Call(
		handle,
		0,
		uintptr(size),
		0x3000,
		0x04,
	)

	if addr == 0 {
		return 0, fmt.Errorf("VirtualAllocEx 失败: %v", lastErr)
	}

	return addr, nil
}

func (d *DLLInjectorInstance) writeProcessMemory(handle uintptr, address uintptr, data []uint16) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procWriteProcessMemory := kernel32.NewProc("WriteProcessMemory")

	var bytesWritten uintptr
	dataBytes := make([]byte, len(data)*2)
	for i, v := range data {
		dataBytes[i*2] = byte(v)
		dataBytes[i*2+1] = byte(v >> 8)
	}

	ret, _, lastErr := procWriteProcessMemory.Call(
		handle,
		address,
		uintptr(unsafe.Pointer(&dataBytes[0])),
		uintptr(len(dataBytes)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)

	if ret == 0 {
		return fmt.Errorf("WriteProcessMemory 失败: %v", lastErr)
	}

	if int(bytesWritten) != len(dataBytes) {
		return fmt.Errorf("写入字节数不匹配: 期望 %d, 实际 %d", len(dataBytes), bytesWritten)
	}

	return nil
}

func (d *DLLInjectorInstance) createRemoteThread(handle uintptr, startAddress uintptr, paramAddress uintptr) (uintptr, uint32, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procCreateRemoteThread := kernel32.NewProc("CreateRemoteThread")

	var threadID uint32
	threadHandle, _, lastErr := procCreateRemoteThread.Call(
		handle,
		0,
		0,
		startAddress,
		paramAddress,
		0,
		uintptr(unsafe.Pointer(&threadID)),
	)

	if threadHandle == 0 {
		return 0, 0, fmt.Errorf("CreateRemoteThread 失败: %v", lastErr)
	}

	return threadHandle, threadID, nil
}

func (d *DLLInjectorInstance) waitForSingleObject(handle uintptr, timeoutMs uint32) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingleObject := kernel32.NewProc("WaitForSingleObject")

	ret, _, _ := procWaitForSingleObject.Call(handle, uintptr(timeoutMs))

	if ret != 0 {
		return fmt.Errorf("WaitForSingleObject 超时或失败: %d", ret)
	}

	return nil
}

func (d *DLLInjectorInstance) getExitCodeThread(handle uintptr) (uintptr, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetExitCodeThread := kernel32.NewProc("GetExitCodeThread")

	var exitCode uintptr
	ret, _, lastErr := procGetExitCodeThread.Call(
		handle,
		uintptr(unsafe.Pointer(&exitCode)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("GetExitCodeThread 失败: %v", lastErr)
	}

	if exitCode == 259 {
		return 0, fmt.Errorf("线程仍在运行")
	}

	return exitCode, nil
}

func (d *DLLInjectorInstance) virtualFreeEx(handle uintptr, address uintptr) error {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procVirtualFreeEx := kernel32.NewProc("VirtualFreeEx")

	ret, _, lastErr := procVirtualFreeEx.Call(
		handle,
		address,
		0,
		0x8000,
	)

	if ret == 0 {
		return fmt.Errorf("VirtualFreeEx 失败: %v", lastErr)
	}

	return nil
}

func (d *DLLInjectorInstance) handleTaskMode() {
	utils.LogInfo("使用 Ctrl+C 取消注入")

	totalSeconds := utils.ParseTimeString(d.TaskTime)
	if totalSeconds <= 0 {
		utils.LogError("无效的时间格式")
		return
	}

	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	utils.LogInfo("定时注入模式 - 等待时间: %d分%d秒", minutes, seconds)

	var dllPath string
	if d.Inject {
		config, err := d.loadConfig()
		if err != nil {
			utils.LogError("读取配置文件失败: %s", err)
			return
		}
		dllPath = config.LastInjectPath
		if dllPath == "" {
			utils.LogError("配置文件中没有上次注入的 DLL 路径")
			return
		}
	} else {
		dllPath = d.DLLPath
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	remaining := totalSeconds
	utils.LogInfo("等待 %d 秒后进行注入...", totalSeconds)

	for {
		select {
		case <-ticker.C:
			remaining--
			if remaining <= 0 {
				utils.LogInfo("\n开始执行注入...")
				success := d.InjectDLL(d.ProcessName, dllPath)
				if success && !d.Inject {
					_ = d.saveConfig(dllPath)
				}
				return
			}
			mins := remaining / 60
			secs := remaining % 60
			fmt.Printf("\r剩余时间: %02d分%02d秒", mins, secs)
		case <-sigChan:
			fmt.Println()
			utils.LogInfo("注入任务已取消")
			return
		}
	}
}

func (d *DLLInjectorInstance) loadConfig() (*InjectorConfig, error) {
	config := &InjectorConfig{}

	if !utils.FileExists(injectorConfigPath) {
		return config, fmt.Errorf("配置文件不存在")
	}

	err := utils.LoadJSON(injectorConfigPath, config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func (d *DLLInjectorInstance) saveConfig(dllPath string) error {
	config := &InjectorConfig{
		LastInjectPath: dllPath,
	}

	err := utils.SaveJSON(injectorConfigPath, config)
	if err != nil {
		utils.LogError("保存配置文件失败: %s", err)
		return err
	}

	utils.LogInfo("已保存 DLL 路径到配置文件，后续注入可以直接使用\"bex dll -i\"快速注入")
	return nil
}

func (d *DLLInjectorInstance) resetConfig() {
	utils.LogInfo("正在重置配置文件...")

	config := &InjectorConfig{
		LastInjectPath: "",
	}

	err := utils.SaveJSON(injectorConfigPath, config)
	if err != nil {
		utils.LogError("重置配置文件失败: %s", err)
		return
	}

	utils.LogInfo("配置文件重置完成")
}
