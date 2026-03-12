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
	CurrentVersion      = "3.0.2"
)

type VersionInfo struct {
	Version       string `json:"version"`
	BuildDate     string `json:"build_date"`
	RequireUpdate bool   `json:"require_update"`
}

type Config struct {
	LastCheckUpdate              string `json:"last_check_update"`
	DisableUpdate                bool   `json:"disable_update"`
	DLLInjectorTargetFilePath    string `json:"dll_injector_target_file_path,omitempty"`
	DLLInjectorTargetProcessName string `json:"dll_injector_target_process_name,omitempty"`
}

func main() {
	CheckUpdate()

	if len(os.Args) < 2 {
		showGeneralHelp()
		waitForKey()
		return
	}

	if handleHelpRequest() {
		waitForKey()
		return
	}

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
	case "checknbt", "c":
		handleCheckNBT(args)
	case "editnbt", "e":
		handleEditNBT(args)
	case "script", "s":
		handleLaunchBat(args)
	case "heatmap", "h":
		handleHeatMap(args)
	case "world", "w":
		handleWorld(args)
	case "dll", "d":
		handleDLLInjector(args)
	case "icon", "i":
		handleMakeIcon(args)
	case "backup", "b":
		handleBackup(args)
	case "about", "!":
		showAbout()
	case "help", "?":
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

	if len(os.Args) >= 3 && os.Args[2] == "help" {
		showModuleHelp(os.Args[1])
		return true
	}

	return false
}

func waitForKey() {
	fmt.Print("\n按任意键继续...")
	var b [1]byte
	_, err := os.Stdin.Read(b[:])
	if err != nil {
		return
	}
}

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

func parseInt(s string, defaultVal int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

func parseFloat(s string, defaultVal float64) float64 {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defaultVal
	}
	return val
}

func handleQuery(args []string) {
	if len(args) == 0 {
		utils.LogError("查询模块缺少目标地址")
		showModuleHelp("query")
		waitForKey()
		return
	}

	target := args[0]

	if len(args) > 1 && args[1] == "?" {
		showModuleHelp("query")
		waitForKey()
		return
	}

	modules.QueryServer(false, false, target)
}

func handlePing(args []string) {
	if len(args) == 0 {
		utils.LogError("Ping模块缺少目标地址")
		showModuleHelp("ping")
		return
	}

	target := args[0]

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

func handleRCON(args []string) {
	if len(args) == 0 {
		utils.LogError("RCON模块缺少参数")
		showModuleHelp("rcon")
		return
	}

	if args[0] == "?" {
		showModuleHelp("rcon")
		return
	}

	loginStr := args[0]

	modules.RconExecutorEntry(loginStr)
}

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

func handleCheckNBT(args []string) {
	if len(args) == 0 {
		utils.LogError("CheckNBT模块缺少文件路径")
		showModuleHelp("checknbt")
		return
	}

	if args[0] == "?" {
		showModuleHelp("checknbt")
		return
	}

	modules.CheckNBT(args[0])
}

func handleEditNBT(args []string) {
	if len(args) == 0 {
		utils.LogError("NBT编辑器模块缺少文件路径")
		showModuleHelp("editnbt")
		return
	}

	if args[0] == "?" {
		showModuleHelp("editnbt")
		return
	}

	modules.NBTEditor(args[0])
}

func handleLaunchBat(args []string) {
	if len(args) == 0 {
		utils.LogError("启动脚本生成模块缺少要求")
		showModuleHelp("script")
		return
	}

	if args[0] == "?" {
		showModuleHelp("script")
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

func handleDLLInjector(args []string) {
	if len(args) == 0 {
		utils.LogError("DLL注入模块缺少参数")
		showModuleHelp("dll")
		waitForKey()
		return
	}

	if args[0] == "?" {
		showModuleHelp("dll")
		waitForKey()
		return
	}

	if len(args) == 1 && args[0] == "-i" {
		config := LoadConfig()
		if config.DLLInjectorTargetFilePath == "" {
			utils.LogError("配置文件中没有找到上次注入的 DLL 路径")
			utils.LogInfo("请先指定 DLL 路径进行注入，例如: bex dll C:\\path\\to\\mod.dll")
			waitForKey()
			return
		}

		processName := config.DLLInjectorTargetProcessName
		if processName == "" {
			processName = "Minecraft.Windows.exe"
			utils.LogInfo("使用默认目标进程: %s", processName)
		} else {
			utils.LogInfo("使用上次注入的目标进程: %s", processName)
		}

		modules.DLLInjector(config.DLLInjectorTargetFilePath, processName, nil)
		return
	}

	if len(args) == 1 && args[0] == "-c" {
		config := LoadConfig()
		config.DLLInjectorTargetFilePath = ""
		config.DLLInjectorTargetProcessName = ""
		if err := SaveConfig(config); err != nil {
			utils.LogError("重置配置失败: %s", err)
			waitForKey()
			return
		}
		utils.LogInfo("DLL 注入配置已重置")
		waitForKey()
		return
	}

	dllPath := ""
	processName := "Minecraft.Windows.exe"

	if !strings.HasPrefix(args[0], "-") {
		dllPath = args[0]
		args = args[1:]
	}

	params := parseKeyValue(args)
	if p, ok := params["p"]; ok {
		processName = p
	}

	if dllPath == "" {
		utils.LogError("请指定要注入的 DLL 文件路径")
		waitForKey()
		return
	}

	config := LoadConfig()
	onSuccess := func(absPath string) {
		config.DLLInjectorTargetFilePath = absPath
		config.DLLInjectorTargetProcessName = processName
		if err := SaveConfig(config); err != nil {
			utils.LogError("保存配置失败，下次将无法使用 -i 快速注入: %s", err)
		} else {
			utils.LogInfo("已保存 DLL 路径和目标进程到配置文件，后续可使用 \"bex dll -i\" 快速注入")
		}
	}

	modules.DLLInjector(dllPath, processName, onSuccess)
}

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

	targetDir := ""
	outputDir := ""
	backupTime := ""
	repeat := false
	maxBackups := 0

	if !strings.HasPrefix(args[0], "-") {
		targetDir = args[0]
		args = args[1:]
	}

	params := parseKeyValue(args)
	if o, ok := params["o"]; ok {
		outputDir = o
	}
	if v, ok := params["v"]; ok {
		backupTime = v
	}
	if _, ok := params["r"]; ok {
		repeat = true
	}
	if mx, ok := params["x"]; ok {
		maxBackups = parseInt(mx, 0)
	}

	if targetDir == "" {
		utils.LogError("请指定要备份的文件夹路径")
		showModuleHelp("backup")
		return
	}

	modules.WorldBackup(targetDir, outputDir, backupTime, repeat, maxBackups)
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
	b.WriteString("  bex <模块> ?          查看模块详细用法\n\n")

	b.WriteString(utils.ColorYellow)
	b.WriteString("  提示：本工具为命令行程序，请在终端中运行\n")
	b.WriteString(utils.ColorClear)
	b.WriteString("\n")

	b.WriteString(utils.ColorGreen)
	b.WriteString("可用模块:\n")
	b.WriteString(utils.ColorClear)
	b.WriteString("  query,    q   查询服务器状态（自动识别 Java/基岩版）\n")
	b.WriteString("  ping,     p   测试服务器网络延迟\n")
	b.WriteString("  rcon,     r   远程执行控制台命令\n")
	b.WriteString("  log,      l   分析日志文件并定位错误\n")
	b.WriteString("  checknbt, c   查看 NBT 文件内容\n")
	b.WriteString("  editnbt,  e   交互式 NBT 文件编辑器\n")
	b.WriteString("  script,   s   生成服务器启动脚本\n")
	b.WriteString("  heatmap,  h   基于玩家活跃度生成热力图\n")
	b.WriteString("  world,    w   扫描并分析世界文件\n")
	b.WriteString("  dll,      d   DLL 注入工具（仅 Windows）\n")
	b.WriteString("  icon,     i   生成服务器图标\n")
	b.WriteString("  backup,   b   自动备份世界文件\n")
	b.WriteString("  about,    !   关于 BeaconEX\n")
	b.WriteString("  help,     ?   显示本帮助\n")

	fmt.Print(b.String())
}

var moduleHelps = map[string]string{
	"query": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("查询模块  query / q\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("自动检测服务器版本类型并查询状态信息\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex query mc.hypixel.net\n")
		b.WriteString("  bex query play.cubecraft.net:19132\n")
		b.WriteString("  bex q 127.0.0.1:11451\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 同时通过 RakNet（基岩版）和 TCP（Java版）协议查询\n")
		b.WriteString("  • 以最先响应的协议为准返回结果\n")
		b.WriteString("  • 未指定端口时，Java版默认 25565，基岩版默认 19132\n")
		return b.String()
	}(),

	"ping": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("Ping 测试模块  ping / p\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("测试目标服务器的网络连通性和响应延迟\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex ping 8.8.8.8\n")
		b.WriteString("  bex ping mc.hypixel.net -f 10\n")
		b.WriteString("  bex ping 127.0.0.1 -r\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  -f <次数>   发送的 ping 包数量（默认 4）\n")
		b.WriteString("  -v <秒>     ping 包发送间隔（默认 1.0）\n")
		b.WriteString("  -r          持续 ping 模式（Ctrl+C 退出）\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 目标地址无需指定端口\n")
		b.WriteString("  • 支持域名和 IP 地址\n")
		return b.String()
	}(),

	"rcon": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("RCON 远程控制模块  rcon / r\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("通过 RCON 协议远程执行服务器命令\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex rcon server@<地址>[:端口]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex rcon server@127.0.0.1           # 默认端口 25575\n")
		b.WriteString("  bex rcon server@127.0.0.1:25575     # 指定端口\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 用户名固定为 server\n")
		b.WriteString("  • 连接成功后进入交互式命令行\n")
		b.WriteString("  • 输入 exit 或 quit 退出连接\n")
		b.WriteString("  • 使用 Ctrl+C 强制断开\n")
		return b.String()
	}(),

	"log": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("日志分析模块  log / l\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("分析服务器日志，提取错误信息并提供排查建议\n\n")
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

	"checknbt": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("NBT 文件查看模块  checknbt / c\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("查看 Minecraft NBT 格式文件的完整数据结构\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex checknbt <文件路径>\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex checknbt level.dat\n")
		b.WriteString("  bex checknbt player.dat\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 支持查看玩家数据、世界数据等 NBT 文件\n")
		b.WriteString("  • 以树形结构展示数据层级\n")
		return b.String()
	}(),

	"editnbt": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("NBT 编辑器模块  editnbt / e\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("全功能交互式 NBT 编辑器，支持增删改查操作\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex editnbt <文件路径>\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex editnbt level.dat\n")
		b.WriteString("  bex editnbt player.dat\n")
		b.WriteString("  bex e ./world/playerdata/xxx.dat\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("支持的文件格式:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  .dat  .nbt  .schematic  .litematic  .mca  .mcr\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("编辑器快捷键:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  ↑/↓/PgUp/PgDn   导航\n")
		b.WriteString("  Enter/Space     展开/折叠节点\n")
		b.WriteString("  e               编辑当前节点值\n")
		b.WriteString("  r               重命名当前节点\n")
		b.WriteString("  a               新增子节点\n")
		b.WriteString("  d/Delete        删除当前节点\n")
		b.WriteString("  Ctrl+S          保存更改\n")
		b.WriteString("  Ctrl+Q          退出编辑器\n")
		b.WriteString("  ?               显示帮助\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 文件不存在时自动创建空 NBT 文件（Region 格式除外）\n")
		b.WriteString("  • 退出时自动恢复终端状态\n")
		b.WriteString("  • 若终端显示异常，可设置环境变量：\n")
		b.WriteString("    BEX_NO_ALTSCREEN=1 bex editnbt <文件>\n")
		return b.String()
	}(),

	"script": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("启动脚本生成模块  script / s\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("根据服务器配置需求，自动生成优化的启动脚本\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex script \"<需求描述>\" [-o <输出目录>]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex script \"1.20.1 Paper，分配 4G 内存\"\n")
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
		b.WriteString("分析 playerdata 目录，生成玩家在线时长的可视化分布\n\n")
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
		b.WriteString("  -o <路径>   将结果保存为文本文件（可选）\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("时长分级:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 蓝色  < 1 天    新玩家\n")
		b.WriteString("  • 绿色  1~7 天    普通玩家\n")
		b.WriteString("  • 黄色  7~30 天   活跃玩家\n")
		b.WriteString("  • 红色  > 30 天   核心玩家\n")
		return b.String()
	}(),

	"world": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("世界分析模块  world / w\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("扫描目录下的所有 Minecraft 世界，统计文件信息并检测异常\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex world <目录路径>\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex world ./server\n")
		b.WriteString("  bex world ./worlds\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("分析内容:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 识别所有包含 level.dat 的世界目录\n")
		b.WriteString("  • 统计各维度文件数量和大小\n")
		b.WriteString("  • 检测可能损坏的世界文件\n")
		b.WriteString("  • 提供修复建议\n")
		return b.String()
	}(),

	"dll": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("DLL 注入模块  dll / d  （仅 Windows）\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("将指定 DLL 注入到目标进程（需要管理员权限）\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex dll mod.dll                    # 基本用法\n")
		b.WriteString("  bex dll mod.dll -p javaw.exe       # 指定目标进程\n")
		b.WriteString("  bex dll -i                          # 使用上次的 DLL 和进程名注入\n")
		b.WriteString("  bex dll -c                          # 重置保存的配置\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  <DLL路径>   要注入的 DLL 文件路径（第一个参数）\n")
		b.WriteString("  -p <进程名>  目标进程名（默认 Minecraft.Windows.exe）\n")
		b.WriteString("  -i           使用上次成功注入的 DLL 路径和进程名\n")
		b.WriteString("  -c           重置配置文件中的 DLL 路径和进程名\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 成功注入后会自动保存 DLL 路径和目标进程名，下次可用 -i 快速注入\n")
		b.WriteString("  • 需要以管理员身份运行\n")
		b.WriteString("  • 支持注入到 Java 版 Minecraft（javaw.exe）和基岩版（Minecraft.Windows.exe）\n")
		return b.String()
	}(),

	"icon": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("服务器图标生成模块  icon / i\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("将图片转换为 64×64 的 server-icon.png 格式\n\n")
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
		b.WriteString(utils.ColorGreen)
		b.WriteString("支持的输入格式:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  PNG、JPG、BMP、GIF\n")
		return b.String()
	}(),

	"backup": func() string {
		var b strings.Builder
		b.WriteString(utils.ColorCyan)
		b.WriteString("世界备份模块  backup / b\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("支持冷/热备份，可定时定量备份世界、模组等文件\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("用法:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex backup <备份目录> [选项...]\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("示例:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  bex backup ./server/world\n")
		b.WriteString("  bex backup ./server/world -v 1h -r -x 10\n")
		b.WriteString("  bex backup ./server/world -o D:/backups\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("参数:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  <备份目录>  要备份的文件夹路径（必需）\n")
		b.WriteString("  -o <路径>   指定备份输出路径（默认保存至目标目录上一级的 bex_backup 文件夹）\n")
		b.WriteString("  -v <时间>   备份间隔，支持格式：30m、1h30m\n")
		b.WriteString("  -r          循环执行模式（需配合 -v 使用）\n")
		b.WriteString("  -x <次数>   最大保留的备份数量\n\n")
		b.WriteString(utils.ColorGreen)
		b.WriteString("说明:\n")
		b.WriteString(utils.ColorClear)
		b.WriteString("  • 默认备份至目标目录上一级的 bex_backup 文件夹\n")
		b.WriteString("  • 指定 -o 时直接输出到该路径，不创建 bex_backup 子文件夹\n")
		b.WriteString("  • 支持热备份（服务器运行时备份）\n")
		return b.String()
	}(),
}

var aliasToModule = map[string]string{
	"q": "query", "p": "ping", "r": "rcon", "l": "log",
	"c": "checknbt", "e": "editnbt", "s": "script", "h": "heatmap", "w": "world",
	"d": "dll", "i": "icon", "b": "backup",
}

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
      ├─ 本软件（BeaconEX）已取得国家版权局《计算机软件著作权登记证书》并受法律保护
      ├─ 未经书面授权，严禁任何商业用途（销售/租赁/商业集成/付费服务等）
      ├─ 严禁违法犯罪活动（入侵服务器/破坏系统/非法获取数据等）
      ├─ 侵权必究，违法必惩，著作权人保留一切法律追究权利
      └─ 使用者自行承担因违反法律法规导致的一切后果
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
