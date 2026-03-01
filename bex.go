// 新版本bex.go，版本3.0.0

//go:generate goversioninfo
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bex/modules"
	"bex/utils"
)

const (
	UpdateCheckInterval = 24 * time.Hour
	ReleaseURL          = "https://github.com/GongSunFangYun/BeaconEX/releases/latest"
	VersionFileURL      = "https://github.com/GongSunFangYun/BeaconEX/releases/latest/download/version.json"
	ProxyURL            = "https://gh-proxy.com/"
	CurrentVersion      = "3.0.0"
)

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version       string `json:"version"`
	BuildDate     string `json:"build_date"`
	RequireUpdate bool   `json:"require_update"`
}

// Config 配置结构体
type Config struct {
	LastCheckUpdate string `json:"last_check_update"`
	DisableUpdate   bool   `json:"disable_update"`
}

func main() {
	CheckUpdate()

	// 如果没有参数，显示帮助并等待按键
	if len(os.Args) < 2 {
		showGeneralHelp()
		waitForKey()
		return
	}

	// 处理帮助请求
	if handleHelpRequest() {
		waitForKey()
		return
	}

	// 解析主命令
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "query", "q":
		handleQuery(args)
	case "ping", "p":
		handlePing(args)
	case "rcon", "r":
		handleRCON(args)
	case "log", "l":
		handleLog(args)
	case "nbt", "n":
		handleNBT(args)
	case "script", "s": // 修改：serbat/sb -> script/s
		handleLaunchBat(args)
	case "heatmap", "h": // 修改：heatmap/hm -> heatmap/h
		handleHeatMap(args)
	case "world", "w":
		handleWorld(args)
	case "injectdll", "i":
		handleDLLInjector(args)
	case "icon", "ic":
		handleMakeIcon(args)
	case "backup", "b":
		handleBackup(args)
	case "about", "!": // 修改：about/a -> about/!
		showAbout()
	case "help", "?": // 修改：help/h -> help/?
		showGeneralHelp()
	default:
		utils.LogError("未知命令: %s", cmd)
		utils.LogInfo("使用 bex help 查看帮助信息")
		waitForKey()
		os.Exit(1)
	}
}

func handleHelpRequest() bool {
	for i, arg := range os.Args[1:] {
		if arg == "?" {
			if i > 0 {
				showModuleHelp(os.Args[i])
				return true
			}
			showGeneralHelp()
			return true
		}
	}

	// 检查是否有 help 子命令
	if len(os.Args) >= 3 && os.Args[2] == "help" {
		showModuleHelp(os.Args[1])
		return true
	}

	return false
}

// waitForKey 等待用户按键
func waitForKey() {
	fmt.Print("\n按任意键继续...")
	var b [1]byte
	_, err := os.Stdin.Read(b[:])
	if err != nil {
		return
	}
}

// ==================== 参数解析辅助函数 ====================

// parseKeyValue 解析键值对参数 (-key value)
func parseKeyValue(args []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			key := strings.TrimPrefix(arg, "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				result[key] = args[i+1]
				i++
			} else {
				result[key] = ""
			}
		}
	}
	return result
}

// parseInt 解析整数，失败返回默认值
func parseInt(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// parseFloat 解析浮点数，失败返回默认值
func parseFloat(s string, defaultVal float64) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

// 服务器查询模块 - 自动识别版本
func handleQuery(args []string) {
	if len(args) == 0 {
		utils.LogError("查询模块缺少目标地址")
		showModuleHelp("query")
		waitForKey()
		return
	}

	// 第一个参数是目标地址
	target := args[0]

	// 检查是否有帮助请求
	if len(args) > 1 && args[1] == "?" {
		showModuleHelp("query")
		waitForKey()
		return
	}

	// 调用自动识别版本的查询函数
	// 传入 false, false 触发自动识别模式
	modules.QueryServer(false, false, target)
}

// Ping测试模块
func handlePing(args []string) {
	if len(args) == 0 {
		utils.LogError("Ping模块缺少目标地址")
		showModuleHelp("ping")
		return
	}

	// 第一个参数是目标地址
	target := args[0]

	// 解析可选参数
	count := 4
	interval := 1.0
	repeat := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-f", "--ping-frequency":
			if i+1 < len(args) {
				count = parseInt(args[i+1], 4)
				i++
			}
		case "-v", "--ping-interval":
			if i+1 < len(args) {
				interval = parseFloat(args[i+1], 1.0)
				i++
			}
		case "-r", "--repeat":
			repeat = true
		case "?":
			showModuleHelp("ping")
			return
		}
	}

	modules.PingServer(target, count, interval, repeat)
}

// RCON远程控制模块
func handleRCON(args []string) {
	if len(args) == 0 {
		utils.LogError("RCON模块缺少参数")
		showModuleHelp("rcon")
		return
	}

	// 检查是否是帮助请求
	if args[0] == "?" {
		showModuleHelp("rcon")
		return
	}

	// 第一个参数是登录字符串
	loginStr := args[0]

	modules.RconExecutorEntry(loginStr)
}

// 日志分析模块
func handleLog(args []string) {
	if len(args) == 0 {
		utils.LogError("日志分析模块缺少文件路径")
		showModuleHelp("log")
		return
	}

	if args[0] == "?" {
		showModuleHelp("log")
		return
	}

	modules.LogAnalyzer(args[0])
}

// NBT处理模块（整合分析器和编辑器）
func handleNBT(args []string) {
	if len(args) == 0 {
		utils.LogError("NBT模块缺少文件路径")
		showModuleHelp("nbt")
		return
	}

	if args[0] == "?" {
		showModuleHelp("nbt")
		return
	}

	filePath := args[0]
	editMode := false

	// 检查是否有编辑模式标志
	for i := 1; i < len(args); i++ {
		if args[i] == "-e" || args[i] == "--edit" {
			editMode = true
		}
	}

	modules.NBTProcessor(filePath, editMode)
}

// 启动脚本生成模块
func handleLaunchBat(args []string) {
	if len(args) == 0 {
		utils.LogError("启动脚本生成模块缺少要求")
		showModuleHelp("script") // 修改：serbat -> script
		return
	}

	if args[0] == "?" {
		showModuleHelp("script") // 修改：serbat -> script
		return
	}

	request := args[0]
	outputDir := ""

	params := parseKeyValue(args[1:])
	if dir, ok := params["o"]; ok {
		outputDir = dir
	}

	modules.LaunchBat(request, outputDir)
}

// 热力图生成模块
func handleHeatMap(args []string) {
	if len(args) == 0 {
		utils.LogError("热力图模块缺少playerdata路径")
		showModuleHelp("heatmap")
		return
	}

	if args[0] == "?" {
		showModuleHelp("heatmap")
		return
	}

	dataFolder := args[0]
	outputDir := "none"

	params := parseKeyValue(args[1:])
	if od, ok := params["o"]; ok {
		outputDir = od
	}

	modules.HeatMap(dataFolder, outputDir)
}

// 世界分析模块
func handleWorld(args []string) {
	if len(args) == 0 {
		utils.LogError("世界分析模块缺少世界路径")
		showModuleHelp("world")
		return
	}

	if args[0] == "?" {
		showModuleHelp("world")
		return
	}

	modules.WorldAnalyzer(args[0])
}

// DLL注入模块
func handleDLLInjector(args []string) {
	if len(args) == 0 {
		utils.LogError("DLL注入模块缺少参数")
		showModuleHelp("injectdll")
		return
	}

	if args[0] == "?" {
		showModuleHelp("injectdll")
		return
	}

	params := parseKeyValue(args)

	dllPath := params["d"]
	processName := params["p"]
	if processName == "" {
		processName = "Minecraft.Windows.exe"
	}
	taskTime := params["t"]
	inject := params["i"] != ""
	resetConfig := params["c"] != ""

	if resetConfig {
		modules.DLLInjector("", processName, "", false, true)
	} else if inject {
		modules.DLLInjector("", processName, taskTime, true, false)
	} else if dllPath != "" {
		modules.DLLInjector(dllPath, processName, taskTime, false, false)
	} else {
		utils.LogError("DLL注入参数不完整/配置文件不存在")
		showModuleHelp("injectdll")
	}
}

// 图标生成模块
func handleMakeIcon(args []string) {
	if len(args) == 0 {
		utils.LogError("图标生成模块缺少图片路径")
		showModuleHelp("icon")
		return
	}

	if args[0] == "?" {
		showModuleHelp("icon")
		return
	}

	picturePath := args[0]
	outputDir := ""
	pictureName := "server-icon.png"

	params := parseKeyValue(args[1:])
	if od, ok := params["o"]; ok {
		outputDir = od
	}
	if pn, ok := params["n"]; ok {
		pictureName = pn
	}

	modules.MakeIcon(picturePath, outputDir, pictureName)
}

// 世界备份模块
func handleBackup(args []string) {
	if len(args) == 0 {
		utils.LogError("世界备份模块缺少参数")
		showModuleHelp("backup")
		return
	}

	if args[0] == "?" {
		showModuleHelp("backup")
		return
	}

	backupPath := ""
	selectDir := ""
	backupTime := ""
	loopExecution := false
	maxBackups := 0

	params := parseKeyValue(args)
	if b, ok := params["b"]; ok {
		backupPath = b
	}
	if sd, ok := params["t"]; ok {
		selectDir = sd
	}
	if bt, ok := params["v"]; ok {
		backupTime = bt
	}
	if _, ok := params["l"]; ok {
		loopExecution = true
	}
	if mx, ok := params["x"]; ok {
		maxBackups = parseInt(mx, 0)
	}

	if backupPath == "" {
		utils.LogError("缺少基础工作目录，请使用 -b 指定")
		showModuleHelp("backup")
		return
	}

	if selectDir == "" {
		utils.LogError("缺少备份目标目录，请使用 -t 指定")
		showModuleHelp("backup")
		return
	}

	modules.WorldBackup(backupPath, selectDir, backupTime, loopExecution, maxBackups)
}

func showGeneralHelp() {
	var b strings.Builder

	b.WriteString(utils.ColorCyan)
	b.WriteString("╔═══════════════════════════════════════════════════════════════╗\n")
	b.WriteString("║              BeaconEX v" + CurrentVersion + " - Minecraft 服务器工具箱         ║\n")
	b.WriteString("╚═══════════════════════════════════════════════════════════════╝\n")
	b.WriteString(utils.ColorClear)

	b.WriteString("\n")
	b.WriteString(utils.ColorGreen)
	b.WriteString("用法:\n")
	b.WriteString(utils.ColorClear)
	b.WriteString("  bex <模块> [参数...]\n")
	b.WriteString("  bex <模块> ?          查看模块详细帮助\n\n")

	b.WriteString(utils.ColorYellow)
	b.WriteString("  本程序为命令行工具，请勿双击运行！\n")
	b.WriteString("  请在 PowerShell / CMD / Terminal / Shell 中运行本程序！\n\n")
	b.WriteString(utils.ColorClear)

	b.WriteString(utils.ColorGreen)
	b.WriteString("可用模块:\n")
	b.WriteString(utils.ColorClear)
	b.WriteString("  query,    q   查询服务器状态（自动识别 Java / 基岩版）\n")
	b.WriteString("  ping,     p   网络连通性测试\n")
	b.WriteString("  rcon,     r   RCON 远程控制\n")
	b.WriteString("  log,      l   服务器日志分析\n")
	b.WriteString("  nbt,      n   NBT 文件查看 / 编辑\n")
	b.WriteString("  script,   s   生成服务器启动脚本\n")
	b.WriteString("  heatmap,  h   玩家游玩时长热力图\n")
	b.WriteString("  world,    w   世界文件分析\n")
	b.WriteString("  injectdll, i   DLL 注入（仅 Windows）\n")
	b.WriteString("  icon,     ic  生成服务器图标\n")
	b.WriteString("  backup,   b   世界文件备份\n")
	b.WriteString("  about,    !   关于 BeaconEX\n")
	b.WriteString("  help,     ?   显示此帮助\n")

	fmt.Print(b.String())
}

// 模块帮助内容（使用helpBuilder风格）
var moduleHelps = map[string]string{
	"query": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("查询模块  query / q\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("自动识别 Java / 基岩版并查询服务器状态\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex query mc.hypixel.net\n")
		b.WriteString("  bex query play.cubecraft.net:19132\n")
		b.WriteString("  bex q 127.0.0.1:11451\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  查询操作会同时尝试RakNet和TCP协议，哪个先响应就返回哪个结果\n")
		b.WriteString("  不指定端口时，Java 默认 25565，基岩版默认 19132\n")
		return b.String()
	}(),

	"ping": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("Ping 测试模块  ping / p\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("对目标主机执行网络连通性测试。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex ping 8.8.8.8\n")
		b.WriteString("  bex ping mc.hypixel.net -f 10\n")
		b.WriteString("  bex ping 127.0.0.1 -r\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -f <次数>   Ping 次数（默认 4）\n")
		b.WriteString("  -v <秒>     Ping 间隔，单位秒（默认 1.0）\n")
		b.WriteString("  -r           持续 Ping 模式\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  目标只需主机名，无需端口\n")
		return b.String()
	}(),

	"rcon": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("RCON 远程控制模块  rcon / r\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("通过 RCON 协议远程执行服务器指令\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex rcon server@<地址>[:端口]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex rcon server@127.0.0.1           # 默认端口 25575\n")
		b.WriteString("  bex rcon server@127.0.0.1:25575     # 指定端口\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  用户名固定为 server，连接后输入密码进入 RCON Shell\n")
		b.WriteString("  输入 exit 或 quit 断开连接，Ctrl+C 强制退出\n")
		return b.String()
	}(),

	"log": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("日志分析模块  log / l\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("解析服务器日志，提取错误并给出 AI 分析建议。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex log <日志文件路径>\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex log server.log\n")
		b.WriteString("  bex log ./logs/latest.log\n")
		return b.String()
	}(),

	"nbt": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("NBT 文件处理模块  nbt / n\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("查看或编辑 Minecraft NBT 格式数据文件。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex nbt <文件路径> [-e]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex nbt level.dat           查看 NBT 文件\n")
		b.WriteString("  bex nbt player.dat -e        进入编辑模式\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -e   编辑模式（开发中）\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  支持玩家存档、level.dat 等各类 NBT 文件\n")
		b.WriteString("  由于在 CLI 中实现编辑功能过于复杂，因此暂时不进行开发...\n")
		return b.String()
	}(),

	"script": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("启动脚本生成模块  script / s\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("通过 AI 根据需求生成优化的服务器启动脚本。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex script \"<需求描述>\" [-o <输出目录>]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex script \"1.20.1 Paper，4G 内存\"\n")
		b.WriteString("  bex script \"原版 1.21\" -o ./server\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -o <路径>   脚本输出目录（默认当前目录）\n")
		return b.String()
	}(),

	"heatmap": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("热力图生成模块  heatmap / h\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("读取 playerdata 目录，在终端显示所有玩家的游玩时长热力图\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex heatmap <playerdata目录> [-o <输出目录>]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex heatmap ./world/playerdata\n")
		b.WriteString("  bex heatmap ./playerdata -o ./output\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -o <路径>   同时将结果保存为文本文件（默认不保存）\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("颜色说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  蓝色  < 1 天   萌新\n")
		b.WriteString("  绿色  1~7 天   轻度\n")
		b.WriteString("  黄色  7~30 天  中度\n")
		b.WriteString("  红色  > 30 天  重度\n")
		return b.String()
	}(),

	"world": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("世界分析模块  world / w\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("扫描指定目录下的所有 Minecraft 世界文件并统计信息。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex world <目录路径>\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex world ./server\n")
		b.WriteString("  bex world ./worlds\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  扫描所有含 level.dat 的子目录，统计文件数量、大小、维度\n")
		b.WriteString("  并检测可能损坏的世界文件且给出相应的见解\n")
		return b.String()
	}(),

	"injectdll": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("DLL 注入模块  injectdll / i  （仅 Windows）\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("将指定 DLL 注入到目标进程，需要管理员权限。\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex injectdll -d ./mod.dll\n")
		b.WriteString("  bex injectdll -d ./mod.dll -p Minecraft.Windows.exe\n")
		b.WriteString("  bex injectdll -i  \n")
		b.WriteString("  bex injectdll -c\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -d <路径>    DLL 文件路径\n")
		b.WriteString("  -p <进程>   注入目标进程名（默认 Minecraft.Windows.exe）\n")
		b.WriteString("  -t <时间>   定时注入，如 -t 1m30s\n")
		b.WriteString("  -i           使用上次保存的 DLL 路径直接注入\n")
		b.WriteString("  -c           重置配置文件\n")
		return b.String()
	}(),

	"icon": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("服务器图标生成模块  icon / ic\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("将图片转换为 64×64 的 server-icon.png\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex icon <图片路径> [-o <输出目录>] [-n <文件名>]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex icon ./logo.png\n")
		b.WriteString("  bex icon ./logo.jpg -o ./server -n icon.png\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -o <路径>   输出目录（默认当前目录）\n")
		b.WriteString("  -n <名称>   输出文件名（默认 server-icon.png）\n\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  支持 PNG、JPG、BMP、GIF 格式输入\n")
		return b.String()
	}(),

	"backup": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("世界备份模块  backup / b\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("支持冷/热备份，定时定量备份世界，模组，插件等文件夹到备份文件夹\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex backup -b <工作目录> -t <备份目标> [选项...]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex backup -b ./server -t worlds\n")
		b.WriteString("  bex backup -b ./server -t worlds -v 1h -l -x 10\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -b <目录>   基础工作目录（必填）\n")
		b.WriteString("  -t <目录>   要备份的子目录名（必填）\n")
		b.WriteString("  -v <时间>   备份间隔，如 30m、1h30m\n")
		b.WriteString("  -x <次数>   最大备份次数\n\n")
		b.WriteString("  -l          循环执行模式\n")
		b.WriteString(utils.ColorYellow)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  备份文件存放在工作目录下的 BEX_BackUps 文件夹\n")
		return b.String()
	}(),
}

// aliasToModule 将命令简写映射到模块名
var aliasToModule = map[string]string{
	"q": "query", "p": "ping", "r": "rcon", "l": "log",
	"n": "nbt", "s": "script", "h": "heatmap", "w": "world",
	"i": "injectdll", "ic": "icon", "b": "backup",
}

// showModuleHelp 显示模块帮助
func showModuleHelp(module string) {
	if alias, ok := aliasToModule[module]; ok {
		module = alias
	}
	if help, ok := moduleHelps[module]; ok {
		fmt.Println(help)
	} else {
		utils.LogError("未知模块: %s", module)
		showGeneralHelp()
	}
}

func showAbout() {
	aboutText := fmt.Sprintf(`%s
%s  ┌─────────────────────────────────────────┐
  │    %sBeaconEX%s || %s强大的Minecraft工具箱%s    %s│
%s  └─────────────────────────────────────────┘
%s  • 软件名称: BeaconEX
  • 版本: v%s
  • 开发者: GongSunFangYun [https://github.com/GongSunFangYun]
  • 项目地址: https://github.com/GongSunFangYun/BeaconEX
  • 反馈邮箱: misakifeedback@outlook.com
  • 开源协议: GNU Lesser General Public License v3.0
  • 计算机软件著作权登记: 2025SR203****
  %s• 重要声明: 
      ├─ 本软件已取得《中华人民共和国计算机软件著作权证书》
      ├─ 本软件未经授权不得用于任何商业用途
      └─ 本软件禁止用于任何违法犯罪活动
%s`,
		utils.ColorCyan, utils.ColorCyan, utils.ColorGreen, utils.ColorCyan,
		utils.ColorBlue, utils.ColorClear, utils.ColorCyan,
		utils.ColorCyan, utils.ColorBrightYellow, CurrentVersion,
		utils.ColorRed, utils.ColorClear)

	fmt.Println()
	lines := strings.Split(aboutText, "\n")
	for _, line := range lines {
		fmt.Printf("   %s\n", line)
	}
	fmt.Println()
}

func GetConfigPath() string {
	return filepath.Join(utils.GetBaseDirectory(), "config.json")
}

func LoadConfig() *Config {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			LastCheckUpdate: "00-01-01 00:00",
			DisableUpdate:   false,
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		utils.LogError("读取配置文件失败: %v", err)
		return &Config{
			LastCheckUpdate: "00-01-01 00:00",
			DisableUpdate:   false,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		utils.LogError("解析配置文件失败: %v", err)
		return &Config{
			LastCheckUpdate: "00-01-01 00:00",
			DisableUpdate:   false,
		}
	}

	return &config
}

func SaveConfig(config *Config) error {
	configPath := GetConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func ShouldCheckUpdate(config *Config) bool {
	if config.DisableUpdate {
		return false
	}

	if config.LastCheckUpdate == "00-01-01 00:00" {
		return true
	}

	lastCheck, err := time.Parse("06-01-02 15:04", config.LastCheckUpdate)
	if err != nil {
		return true
	}

	return time.Since(lastCheck) >= UpdateCheckInterval
}

func compareVersions(currentVersion, remoteVersion string) int {
	if remoteVersion == currentVersion {
		return 0
	}

	currentParts := strings.Split(currentVersion, ".")
	remoteParts := strings.Split(remoteVersion, ".")

	for i := 0; i < len(currentParts) && i < len(remoteParts); i++ {
		currentNum, _ := strconv.Atoi(currentParts[i])
		remoteNum, _ := strconv.Atoi(remoteParts[i])
		if remoteNum > currentNum {
			return 1
		} else if remoteNum < currentNum {
			return -1
		}
	}

	if len(remoteParts) > len(currentParts) {
		return 1
	} else if len(remoteParts) < len(currentParts) {
		return -1
	}

	return 0
}

func CheckUpdate() {
	config := LoadConfig()

	if !ShouldCheckUpdate(config) {
		return
	}

	config.LastCheckUpdate = time.Now().Format("06-01-02 15:04")
	if err := SaveConfig(config); err != nil {
		return
	}

	versionInfo, tempFilePath, err := downloadVersionInfo()
	if err != nil {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return
	}

	defer func() {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
	}()

	versionComparison := compareVersions(CurrentVersion, versionInfo.Version)

	if versionComparison == 1 {
		utils.LogInfo("%s发现新版本 %sv%s%s | %s当前版本 %sv%s%s",
			utils.ColorBrightYellow, utils.ColorBlue, versionInfo.Version, utils.ColorClear,
			utils.ColorBrightYellow, utils.ColorBlue, CurrentVersion, utils.ColorClear)
		utils.LogInfo("%s请前往 %s%s%s 下载更新！%s",
			utils.ColorBrightYellow, utils.ColorPurple, ReleaseURL, utils.ColorBrightYellow, utils.ColorClear)
	}
}

func downloadVersionInfo() (*VersionInfo, string, error) {
	proxyURL := ProxyURL + VersionFileURL

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", proxyURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "BeaconEX-Updater")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "bex_version_*.json")
	if err != nil {
		return nil, "", err
	}
	defer func(tempFile *os.File) {
		err := tempFile.Close()
		if err != nil {

		}
	}(tempFile)

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		_ = os.Remove(tempFile.Name())
		return nil, "", err
	}

	tempFilePath := tempFile.Name()
	data, err := os.ReadFile(tempFilePath)
	if err != nil {
		_ = os.Remove(tempFilePath)
		return nil, "", err
	}

	var versionInfo VersionInfo
	if err := json.Unmarshal(data, &versionInfo); err != nil {
		_ = os.Remove(tempFilePath)
		return nil, "", err
	}

	return &versionInfo, tempFilePath, nil
}
