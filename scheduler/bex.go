//go:generate goversioninfo
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// flag库对传递给模块的参数处理不当
// 我直接不用了👍

const (
	ColorGreen        = "\033[32m"
	ColorBrightYellow = "\033[93m"
	ColorRed          = "\033[31m"
	ColorBlue         = "\033[94m"
	ColorPurple       = "\033[95m"
	ColorCyan         = "\033[96m"
	ColorClear        = "\033[0m"
)

const (
	UpdateCheckInterval = 24 * time.Hour // 更新检查间隔改为24小时
	ReleaseURL          = "https://github.com/GongSunFangYun/BeaconEX/releases/latest"
	VersionFileURL      = "https://github.com/GongSunFangYun/BeaconEX/releases/latest/download/version.json"
	ProxyURL            = "https://gh-proxy.com/"
	CurrentVersion      = "2.0.1" // 硬编码版本号
)

// VersionInfo 版本信息结构体
type VersionInfo struct {
	Version       string `json:"version"`
	BuildDate     string `json:"build_date"`
	RequireUpdate bool   `json:"require_update"`
}

// Config 配置结构体
type Config struct {
	LastCheckUpdate string `json:"last_check_update"` // 格式: YY-MM-DD HH:MM
	DisableUpdate   bool   `json:"disable_update"`    // 是否禁用更新检查
}

// 模块映射表
var moduleMap = map[string]string{
	"query":    "bex_query",         // -query/--query-server
	"ping":     "bex_ping",          // -ping/--ping-host
	"rcon":     "bex_rcon",          // -rcon/--rcon-remotecontrol
	"log":      "bex_loganalyzer",   // -log/--log-analyzer
	"nbt":      "bex_nbtanalyzer",   // -nbt/--nbt-analyzer
	"serbat":   "bex_batmaker",      // -serbat/--generate-serverbat
	"heatmap":  "bex_heatmap",       // -heatmap/--generate-heatmap
	"world":    "bex_worldanalyzer", // -world/--world-analyzer
	"editnbt":  "bex_nbteditor",     // -editnbt/--nbt-editor
	"injector": "bex_injector",      // -injector/--dll-injector
	"p2p":      "bex_p2p",           // -p2p/--p2p
	"icon":     "bex_iconmaker",     // -icon/--icon-maker
	"backup":   "bex_backup",        // -backup/--world-backup
}

// 调度器参数列表
var schedulerParams = map[string]bool{
	// 模块选择参数
	"-query":               true,
	"--query-server":       true,
	"-ping":                true,
	"--ping-host":          true,
	"-rcon":                true,
	"--rcon-remotecontrol": true,
	"-log":                 true,
	"--log-analyzer":       true,
	"-nbt":                 true,
	"--nbt-analyzer":       true,
	"-serbat":              true,
	"--generate-serverbat": true,
	"-heatmap":             true,
	"--generate-heatmap":   true,
	"-world":               true,
	"--world-analyzer":     true,
	"-editnbt":             true,
	"--nbt-editor":         true,
	"-injector":            true,
	"--dll-injector":       true,
	"-p2p":                 true,
	"-icon":                true,
	"--icon-maker":         true,
	"-backup":              true,
	"--world-backup":       true,

	// 调度器专用参数
	"-about":  true,
	"--about": true,
	"-h":      true,
	"--help":  true,
}

// Args 参数结构体
type Args struct {
	// 调度器调用参数
	query    bool // -query/--query-server
	ping     bool // -ping/--ping-host
	rcon     bool // -rcon/--rcon-remotecontrol
	log      bool // -log/--log-analyzer
	nbt      bool // -nbt/--nbt-analyzer
	serbat   bool // -serbat/--generate-serverbat
	heatmap  bool // -heatmap/--generate-heatmap
	world    bool // -world/--world-analyzer
	editnbt  bool // -editnbt/--nbt-editor
	injector bool // -injector/--dll-injector
	p2p      bool // -p2p/--p2p
	icon     bool // -icon/--icon-maker
	backup   bool // -backup/--world-backup

	// 调度器专用参数
	about bool // -about/--about
	help  bool // -h/--help
}

// 获取日期和时间
func getDate() string {
	return time.Now().Format("2006-01-02")
}

func getTime() string {
	return time.Now().Format("15:04:05")
}

// LogInfo 定义日志等级（如同bexlib2.lg4pb）
func LogInfo(message string) {
	date := getDate()
	t := getTime()
	fmt.Println(
		ColorBlue + date + " " +
			ColorBrightYellow + t + " " +
			ColorGreen + "[Application Thread/INFO]" + ColorClear + " " +
			message,
	)

}

// LogError 定义日志等级（如同bexlib2.lg4pb）
func LogError(message string) {
	date := getDate()
	t := getTime()
	fmt.Println(
		ColorBlue + date + " " +
			ColorBrightYellow + t + " " +
			ColorRed + "[Application Thread/ERROR]" + ColorClear + " " +
			message,
	)
}

// GetBaseDirectory 获取基础目录
func GetBaseDirectory() string {
	if filepath.Base(os.Args[0]) == "bex.exe" {
		return filepath.Dir(os.Args[0])
	}
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	baseDir := GetBaseDirectory()
	return filepath.Join(baseDir, "config.json")
}

// LoadConfig 加载配置文件
func LoadConfig() *Config {
	configPath := GetConfigPath()

	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			LastCheckUpdate: "00-01-01 00:00", // 默认值，确保会检查更新
			DisableUpdate:   false,            // 默认不禁用更新
		}
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		LogError(fmt.Sprintf("读取配置文件失败: %v", err))
		return &Config{
			LastCheckUpdate: "00-01-01 00:00",
			DisableUpdate:   false,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		LogError(fmt.Sprintf("解析配置文件失败: %v", err))
		return &Config{
			LastCheckUpdate: "00-01-01 00:00",
			DisableUpdate:   false,
		}
	}

	return &config
}

// SaveConfig 保存配置文件
func SaveConfig(config *Config) error {
	configPath := GetConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, data, 0644)
}

// ShouldCheckUpdate 判断是否需要检查更新
func ShouldCheckUpdate(config *Config) bool {
	// 如果禁用了更新检查，直接返回false
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

// compareVersions 比较版本号，返回:
// -1: remoteVersion < currentVersion
//
//	0: remoteVersion == currentVersion
//	1: remoteVersion > currentVersion
func compareVersions(currentVersion, remoteVersion string) int {
	// 简单的字符串比较，适用于语义化版本号
	if remoteVersion == currentVersion {
		return 0
	}

	// 分割版本号为数字部分
	currentParts := strings.Split(currentVersion, ".")
	remoteParts := strings.Split(remoteVersion, ".")

	// 比较每个部分
	for i := 0; i < len(currentParts) && i < len(remoteParts); i++ {
		// 这里简化处理，实际可能需要将字符串转换为数字
		if remoteParts[i] > currentParts[i] {
			return 1
		} else if remoteParts[i] < currentParts[i] {
			return -1
		}
	}

	// 如果前面部分都相等，长度更长的版本号更大
	if len(remoteParts) > len(currentParts) {
		return 1
	} else if len(remoteParts) < len(currentParts) {
		return -1
	}

	return 0
}

// CheckUpdate 检查更新
func CheckUpdate() {
	config := LoadConfig()

	// 检查是否需要检查更新
	if !ShouldCheckUpdate(config) {
		return
	}

	// 更新最后检查时间
	config.LastCheckUpdate = time.Now().Format("06-01-02 15:04")
	if err := SaveConfig(config); err != nil {
		return // 静默失败
	}

	// 通过代理下载远程 version.json
	versionInfo, tempFilePath, err := downloadVersionInfo()
	if err != nil {
		// 静默失败，不打印错误信息
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
		return
	}

	// 检查完成后删除临时文件
	defer func() {
		if tempFilePath != "" {
			_ = os.Remove(tempFilePath)
		}
	}()

	// 比较版本号
	versionComparison := compareVersions(CurrentVersion, versionInfo.Version)

	if versionComparison == 1 {
		// 远程版本更高，需要更新
		LogInfo(fmt.Sprintf("%s发现新版本 %sv%s%s | %s当前版本 %sv%s%s",
			ColorBrightYellow, ColorBlue, versionInfo.Version, ColorClear, ColorBrightYellow, ColorBlue, CurrentVersion, ColorClear))
		LogInfo(fmt.Sprintf("%s请前往 %s%s%s 下载更新！%s",
			ColorBrightYellow, ColorPurple, ReleaseURL, ColorBrightYellow, ColorClear))
	} else if versionComparison == 0 {
	}
}

// downloadVersionInfo 通过代理下载远程 version.json 到临时文件
func downloadVersionInfo() (*VersionInfo, string, error) {
	// 构建代理URL
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
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 创建临时文件
	tempFile, err := ioutil.TempFile("", "bex_version_*.json")
	if err != nil {
		return nil, "", err
	}
	defer func(tempFile *os.File) {
		_ = tempFile.Close()
	}(tempFile)

	// 将响应内容写入临时文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		_ = os.Remove(tempFile.Name()) // 写入失败时删除临时文件
		return nil, "", err
	}

	// 读取临时文件内容
	tempFilePath := tempFile.Name()
	data, err := ioutil.ReadFile(tempFilePath)
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

func main() {
	// 检查更新
	CheckUpdate()

	// 先检查是否有帮助查询请求
	if len(os.Args) >= 2 {
		// 检查是否有 ? 参数
		for i, arg := range os.Args[1:] {
			if arg == "?" {
				// 查找前一个参数作为要查询的参数名
				if i > 0 {
					param := os.Args[i]
					showModuleHelp(param)
					return
				} else {
					// 只有 ? 没有前导参数，显示完整帮助
					showHelp()
					return
				}
			}
		}

		// 检查是否只有帮助请求
		if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "?" {
			showHelp()
			return
		}
	}

	args, moduleName := parseArgs()

	// 处理帮助信息
	if args.help {
		showHelp()
		return
	}

	// 处理关于信息
	if args.about {
		showAbout()
		return
	}

	// 参数完整性检查
	if !validateArgs(args) {
		LogError("参数组不正确，请使用 -h 查看参数帮助信息或使用 '-模块名称 ?' 查看该模块帮助信息")
		os.Exit(1)
	}

	// 确定要启动的模块
	module := determineModule(args)
	if module == "" {
		LogError("错误：未指定操作模式，请使用 -h 查看帮助信息")
		os.Exit(1)
	}

	// 构建模块路径
	modulePath := filepath.Join(GetBaseDirectory(), "modules", module+".exe")
	if _, err := os.Stat(modulePath); os.IsNotExist(err) {
		LogError(fmt.Sprintf("模块文件不存在: %s", modulePath))
		os.Exit(1)
	}

	// 构建命令行参数 - 过滤掉调度器参数，只传递模块参数
	cmdArgs := filterSchedulerArgs(os.Args[1:], moduleName)

	// 执行模块
	executeModule(modulePath, cmdArgs)
}

// filterSchedulerArgs 过滤掉调度器参数，只保留模块参数
//
//goland:noinspection GoDfaConstantCondition,GoUnusedParameter
func filterSchedulerArgs(args []string, moduleName string) []string {
	var filteredArgs []string
	skipNext := false

	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		// 如果是调度器参数，跳过
		if schedulerParams[arg] {
			// 如果是模块选择参数本身，完全跳过
			continue
		}

		// 保留所有其他参数
		filteredArgs = append(filteredArgs, arg)
	}

	return filteredArgs
}

// parseArgs 手动解析参数，只识别调度器参数，返回参数结构体和模块名称
func parseArgs() (*Args, string) {
	args := &Args{}
	var moduleName string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		switch arg {
		// 模块选择参数
		case "-query", "--query-server":
			args.query = true
			moduleName = "query"
		case "-ping", "--ping-host":
			args.ping = true
			moduleName = "ping"
		case "-rcon", "--rcon-remotecontrol":
			args.rcon = true
			moduleName = "rcon"
		case "-log", "--log-analyzer":
			args.log = true
			moduleName = "log"
		case "-nbt", "--nbt-analyzer":
			args.nbt = true
			moduleName = "nbt"
		case "-serbat", "--generate-serverbat":
			args.serbat = true
			moduleName = "serbat"
		case "-heatmap", "--generate-heatmap":
			args.heatmap = true
			moduleName = "heatmap"
		case "-world", "--world-analyzer":
			args.world = true
			moduleName = "world"
		case "-editnbt", "--nbt-editor":
			args.editnbt = true
			moduleName = "editnbt"
		case "-injector", "--dll-injector":
			args.injector = true
			moduleName = "injector"
		case "-p2p":
			args.p2p = true
			moduleName = "p2p"
		case "-icon", "--icon-maker":
			args.icon = true
			moduleName = "icon"
		case "-backup", "--world-backup":
			args.backup = true
			moduleName = "backup"

		// 调度器专用参数
		case "-about", "--about":
			args.about = true
		case "-h", "--help":
			args.help = true
		}
	}

	return args, moduleName
}

func validateArgs(args *Args) bool {
	// 只检查调度器调用参数
	return args.query || args.ping || args.rcon || args.log || args.nbt ||
		args.serbat || args.heatmap || args.world || args.editnbt ||
		args.injector || args.p2p || args.icon || args.backup
}

func determineModule(args *Args) string {
	// 根据调度器调用参数确定模块
	if args.query {
		return moduleMap["query"]
	}
	if args.ping {
		return moduleMap["ping"]
	}
	if args.rcon {
		return moduleMap["rcon"]
	}
	if args.log {
		return moduleMap["log"]
	}
	if args.nbt {
		return moduleMap["nbt"]
	}
	if args.serbat {
		return moduleMap["serbat"]
	}
	if args.heatmap {
		return moduleMap["heatmap"]
	}
	if args.world {
		return moduleMap["world"]
	}
	if args.editnbt {
		return moduleMap["editnbt"]
	}
	if args.injector {
		return moduleMap["injector"]
	}
	if args.p2p {
		return moduleMap["p2p"]
	}
	if args.icon {
		return moduleMap["icon"]
	}
	if args.backup {
		return moduleMap["backup"]
	}
	return ""
}

func executeModule(modulePath string, cmdArgs []string) {
	cmd := exec.Command(modulePath, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		LogError(fmt.Sprintf("模块执行失败: %v", err))
		os.Exit(1)
	}
}

func showAbout() { // 显示帮助信息[已进行美化(看着顺眼)]
	aboutText := fmt.Sprintf(`%s
%s  ┌─────────────────────────────────────────┐
  │    %sBeaconEX%s || %s强大的Minecraft工具箱%s    %s│
%s  └─────────────────────────────────────────┘
%s  • 软件名称: BeaconEX
  • 版本: v%s
  • 开发者: GongSunFangYun [https://github.com/GongSunFangYun]
  • 项目地址: https://github.com/GongSunFangYun/BeaconEX
  • 反馈邮箱: misakifeedback@outlook.com
  • 开源协议: GNU Lesser General Public License v3.0 (LGPL-3.0)
  • 授权范围: 允许用户使用，修改和分发本程序，若修改了源代码则必须开源修改部分
  • 计算机软件著作权登记: 2025SR203**** (软著登字第1669****号)
  %s• 重要声明: 
      ├─ 本软件已取得《中华人民共和国计算机软件著作权证书》
      ├─ 本软件未经授权不得用于任何商业用途
      └─ 本软件禁止用于任何违法犯罪活动
%s`,
		ColorCyan,
		ColorCyan,
		ColorGreen, ColorCyan, ColorBlue, ColorClear, ColorCyan,
		ColorCyan,
		ColorBrightYellow, CurrentVersion,
		ColorRed,
		ColorClear)

	// 添加空行并在两侧添加空格实现居中悬空效果
	fmt.Println()

	// 将文本分割成行，每行都添加相同的缩进
	lines := strings.Split(aboutText, "\n")
	for _, line := range lines {
		fmt.Printf("   %s\n", line)
	}
	fmt.Println()
}

func showHelp() {
	// 检查是否有特定参数的帮助请求
	if len(os.Args) >= 3 && os.Args[len(os.Args)-1] == "?" {
		// 查找最后一个参数前的参数名
		for i := len(os.Args) - 2; i >= 1; i-- {
			if os.Args[i] != "?" {
				param := os.Args[i]
				showModuleHelp(param)
				return
			}
		}
	}

	// 使用字符串构建器来创建帮助文本
	var helpBuilder strings.Builder
	// 调度器参数部分

	helpBuilder.WriteString(ColorBlue)
	helpBuilder.WriteString(fmt.Sprintf("*当前版本: v%s\n", CurrentVersion))
	helpBuilder.WriteString("*调度器参数用于指定模块，模块参数必须在调度器参数之后使用\n")
	helpBuilder.WriteString("*每个模块均有对应的处理参数，因此模块与模块间参数大多不可混用\n")
	helpBuilder.WriteString(ColorCyan + "[调度器参数]" + ColorPurple + " ? " + ColorBlue + "可以显示该模块帮助信息\n")
	helpBuilder.WriteString("组合顺序：" + ColorBrightYellow + "bex.exe" + ColorCyan + " [调度器参数] " + ColorPurple + "[模块参数]\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("\n")

	// 模块选择参数
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 模块选择参数：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -query, --query-server     \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 查询Minecraft服务器状态\n")
	helpBuilder.WriteString("  -ping, --ping-host         \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 执行Ping测试\n")
	helpBuilder.WriteString("  -rcon, --rcon-remotecontrol\t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" RCON远程控制\n")
	helpBuilder.WriteString("  -log, --log-analyzer       \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 分析服务器日志文件\n")
	helpBuilder.WriteString("  -nbt, --nbt-analyzer       \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 解析NBT数据文件\n")
	helpBuilder.WriteString("  -serbat, --generate-serverbat\t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 生成服务器启动脚本\n")
	helpBuilder.WriteString("  -heatmap, --generate-heatmap\t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 生成玩家热力图\n")
	helpBuilder.WriteString("  -world, --world-analyzer   \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 检查世界完整性\n")
	helpBuilder.WriteString("  -editnbt, --nbt-editor     \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 使用NBT编辑器\n")
	helpBuilder.WriteString("  -injector, --dll-injector  \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" DLL注入工具\n")
	helpBuilder.WriteString("  -p2p                       \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" P2P虚拟网络工具\n")
	helpBuilder.WriteString("  -icon, --icon-maker        \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 生成服务器图标\n")
	helpBuilder.WriteString("  -backup, --world-backup    \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 世界备份工具\n")
	helpBuilder.WriteString("  -update, --update-bex      \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 检查并更新程序\n")
	helpBuilder.WriteString("  -about, --about            \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 显示程序信息\n")
	helpBuilder.WriteString("  -h, --help                 \t\t")
	helpBuilder.WriteString(ColorCyan)
	helpBuilder.WriteString("[调度器]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 显示此帮助信息\n")
	helpBuilder.WriteString("\n")

	// 服务器查询模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 服务器查询模块 (-query)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -java, --java              \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 查询Java服务器\n")
	helpBuilder.WriteString("  -bedrock, --bedrock        \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 查询基岩版服务器\n")
	helpBuilder.WriteString("  -t, --target               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定查询目标 (hostname:port)\n")
	helpBuilder.WriteString("\n")

	// 网络测试模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 网络测试模块 (-ping)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -t, --target               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定Ping目标\n")
	helpBuilder.WriteString("  -r, --repeat               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 持续Ping模式\n")
	helpBuilder.WriteString("  -pf, --ping-frequency      \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 普通模式下的Ping次数 (默认：4)\n")
	helpBuilder.WriteString("  -pi, --ping-interval       \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 持续Ping的间隔时间 (默认：1.0秒)\n")
	helpBuilder.WriteString("\n")

	// RCON远程控制模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• RCON远程控制模块 (-rcon)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -t, --target               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定查询目标 (hostname:port)\n")
	helpBuilder.WriteString("  -rpw, --rcon-password      \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" RCON连接密码\n")
	helpBuilder.WriteString("  -rp, --rcon-port           \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" RCON端口 (默认25575)\n")
	helpBuilder.WriteString("  -cmd, --command            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 执行单个命令\n")
	helpBuilder.WriteString("  -cg, --command-group       \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 进入命令组模式\n")
	helpBuilder.WriteString("  -s, --script               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定脚本路径并执行\n")
	helpBuilder.WriteString("\n")

	// 日志分析模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 日志分析模块 (-log)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -lp, --log-path            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定日志文件路径\n")
	helpBuilder.WriteString("\n")

	// NBT解析模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• NBT解析模块 (-nbt)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -np, --nbt-path            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定NBT文件路径\n")
	helpBuilder.WriteString("\n")

	// 启动脚本生成模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 启动脚本生成模块 (-serbat)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -rq, --request             \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 生成要求\n")
	helpBuilder.WriteString("  -od, --output-dir          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定输出目录 (默认: 当前目录)\n")
	helpBuilder.WriteString("\n")

	// 热力图生成模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 热力图生成模块 (-heatmap)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -dfp, --data-folder-path   \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定playerdata文件夹路径\n")
	helpBuilder.WriteString("  -mp, --max-player          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 每张图表显示的最大玩家数 (默认: 15)\n")
	helpBuilder.WriteString("  -od, --output-dir          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定输出目录 (默认: 当前目录)\n")
	helpBuilder.WriteString("\n")

	// 世界分析模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 世界分析模块 (-world)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -wp, --world-path          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定世界文件夹路径\n")
	helpBuilder.WriteString("\n")

	// NBT编辑器模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• NBT编辑器模块 (-editnbt)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -np, --nbt-path            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定NBT文件路径\n")
	helpBuilder.WriteString("\n")

	// DLL注入工具模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• DLL注入工具模块 (-injector)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -dp, --dll-path            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定DLL路径\n")
	helpBuilder.WriteString("  -ct, --custom-target       \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 自定义注入目标\n")
	helpBuilder.WriteString("  -i, --inject               \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 读取上次注入路径并立即注入\n")
	helpBuilder.WriteString("  -tm, --task-mode           \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 计划模式\n")
	helpBuilder.WriteString("  -rc, --reset-config        \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 重置配置文件\n")
	helpBuilder.WriteString("\n")

	// P2P联机工具模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• P2P联机工具模块 (-p2p)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -cn, --create-network      \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 创建虚拟网络\n")
	helpBuilder.WriteString("  -jn, --join-network        \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 加入虚拟网络\n")
	helpBuilder.WriteString("  -l, --list                 \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 列出当前网络中的用户\n")
	helpBuilder.WriteString("  -n, --name                 \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定网络名称\n")
	helpBuilder.WriteString("  -pw, --password            \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定网络密码\n")
	helpBuilder.WriteString("\n")

	// 图标生成模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 图标生成模块 (-icon)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -pp, --picture-path        \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定图片路径\n")
	helpBuilder.WriteString("  -od, --output-dir          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定输出目录\n")
	helpBuilder.WriteString("  -pn, --picture-name        \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定输出图片名称 (默认server-icon.png)\n")
	helpBuilder.WriteString("\n")

	// 世界备份模块
	helpBuilder.WriteString(ColorGreen)
	helpBuilder.WriteString("• 世界备份模块 (-backup)：\n")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString("  -bp, --backup-path         \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定工作目录\n")
	helpBuilder.WriteString("  -sd, --select-dir          \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定备份目标文件夹\n")
	helpBuilder.WriteString("  -bt, --backup-time         \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 指定循环备份间隔\n")
	helpBuilder.WriteString("  -le, --loop-execution      \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 循环执行备份模式\n")
	helpBuilder.WriteString("  -mx, --max                 \t\t")
	helpBuilder.WriteString(ColorPurple)
	helpBuilder.WriteString("[模块]")
	helpBuilder.WriteString(ColorClear)
	helpBuilder.WriteString(" 最大备份次数")

	// 输出帮助文本
	fmt.Println(helpBuilder.String())
}

// 模块帮助内容定义（映射）
var moduleHelps = map[string]string{
	// 服务器查询模块
	"query": `服务器查询模块 (-query/--query-server)
功能：查询 Minecraft 服务器状态信息

使用示例：
  bex.exe -query -java -t mc.example.com:11451
  bex.exe -query -java -t 127.0.0.1
  bex.exe -query -java -t 127.0.0.1:11451
  bex.exe -query -bedrock -t mc.example.com:19198
  bex.exe -query -bedrock -t 127.0.0.1
  bex.exe -query -bedrock -t 127.0.0.1:19198

模块参数：
  -java, --java       查询 Java 版服务器
  -bedrock, --bedrock 查询基岩版服务器  
  -t, --target        查询目标 (格式：主机名:端口 | IP 地址:端口 | 主机名[若服务器使用默认端口可省略端口])

参数说明：
  • java 与 bedrock 是互斥项，在查询时只能选取查询 Java 版服务器或查询基岩版服务器
  • -t 用于指定查询目标，该参数是必需的
  • 默认端口：Java 服务器为 25565，基岩服务器为 19132`,

	// 网络测试模块
	"ping": `网络测试模块 (-ping/--ping-host)
功能：执行网络连通性测试

使用示例：
  bex.exe -ping -t mc.example.com
  bex.exe -ping -t mc.example.com -pf 10
  bex.exe -ping -t mc.example.com -r
  bex.exe -ping -t mc.example.com -r -pi 0.5
  bex.exe -ping -t 127.0.0.1
  bex.exe -ping -t 127.0.0.1 -pf 10
  bex.exe -ping -t 127.0.0.1 -r
  bex.exe -ping -t 127.0.0.1 -r -pi 0.5

模块参数：
  -t, --target          Ping 测试目标（只需要输入主机名，不需要端口）
  -r, --repeat          持续 Ping 模式（使用 Ctrl+C 停止）
  -pf, --ping-frequency 普通模式下的 Ping 次数（默认：4）
  -pi, --ping-interval  持续 Ping 的间隔时间（单位秒，默认：1.0）

参数说明：
  • -t 用于指定查询目标，该参数是必需的
  • -r 和 -pf 互斥，前者用于执行持续 Ping 模式，后者用于在普通 Ping 模式指定次数
  • 在指定 -r 之后，可以使用 -pi 控制持续 Ping 的间隔`,

	// RCON 远程控制模块
	"rcon": `RCON 远程控制模块 (-rcon/--rcon-remotecontrol)
功能：通过 RCON 协议远程控制 Minecraft 服务器

使用示例：
  bex.exe -rcon -t example.com -rp 12345 -rpw 123456 -cmd "say hello world！"
  bex.exe -rcon -t example.com -rp 12345 -rpw 123456 -cg
  bex.exe -rcon -t example.com -rpw 123456 -cmd "say hello world"
  bex.exe -rcon -t example.com -rpw 123456 -cg
  bex.exe -rcon -t example.com -s "C:/Server/RCONScript/script.txt"

模块参数：
  -t, --target          服务器地址（格式：主机名:端口 | IP 地址:端口 | 主机名[若服务器使用默认端口可省略端口]）
  -rpw, --rcon-password RCON 连接密码（server.properties -> rcon.password=你设定的密码）
  -rp, --rcon-port      RCON 端口（server.properties -> rcon.port=你设定的端口[默认 25575]）
  -cmd, --command       执行单个命令
  -cg, --command-group  进入交互式命令行
  -s, --script          执行脚本文件

参数说明：
  • -rp 是可选项，如果不指定则使用默认端口
  • -rpw 必须指定，若想建立 RCON 连接则必须指定密码
  • -cmd 和 -cg 是互斥的，前者用于执行单个命令，后者用于进入交互式命令行持续执行命令
  • -s 为独立参数，用于解释执行 BEXScript 编写的 RCON 远程控制脚本文件`,

	// 日志分析模块
	"log": `日志分析模块 (-log/--log-analyzer)
功能：分析 Minecraft 服务器日志文件

使用示例：
  bex.exe -log -lp "C:/Server/logs/latest.log"

模块参数：
  -lp, --log-path 日志文件路径

参数说明：
  • 必须使用 -lp 指定日志文件路径`,

	// NBT 解析模块
	"nbt": `NBT 解析模块 (-nbt/--nbt-analyzer)
功能：解析 Minecraft NBT 数据文件

使用示例：
  bex.exe -nbt -np "level.dat"

模块参数：
  -np, --nbt-path NBT 文件路径

参数说明：
  • 必须使用 -np 指定 NBT 文件路径`,

	// 启动脚本生成模块
	"serbat": `启动脚本生成模块 (-serbat/--generate-serverbat)
功能：生成服务器启动脚本

使用示例：
  bex.exe -serbat -rq "1.20.1 Paper 服务器，分配 2~4G 内存，启用 GC1"
  bex.exe -serbat -rq "1.20.1 Paper 服务器，分配 2~4G 内存，启用 GC1" -od "C:/Server"

模块参数：
  -rq, --request   生成脚本内容要求
  -od, --output-dir 输出目录（默认当前目录）

参数说明：
  • -rq 后必须提出要求，-od 是可选项，不指定则输出到当前目录（比如 cmd 工作目录在 C:/Server，则 start.bat 会输出到 C:/Server）`,

	// 热力图生成模块
	"heatmap": `热力图生成模块 (-heatmap/--generate-heatmap)
功能：生成玩家活动热力图

使用示例：
  bex.exe -heatmap -dfp "C:/Server/worlds/overworld/playerdata"
  bex.exe -heatmap -dfp "C:/Server/worlds/overworld/playerdata" -mp 20
  bex.exe -heatmap -dfp "C:/Server/worlds/overworld/playerdata" -od "C:/Server/heatmap"

模块参数：
  -dfp, --data-folder-path playerdata 文件夹路径
  -mp, --max-player        每张图表显示的最大玩家数（默认：15）
  -od, --output-dir        输出目录（默认：当前目录）

参数说明：
  • 必须使用 -dfp 指定 playerdata 文件夹路径，-mp 与 -od 是可选项
  • -od 是可选项，不指定则输出到当前目录（比如 cmd 工作目录在 C:/Server，则热力图会输出到 C:/Server）`,

	// 世界分析模块
	"world": `世界分析模块 (-world/--world-analyzer)
功能：检查 Minecraft 世界完整性

使用示例：
  bex.exe -world -wp "C:/Server/worlds"

模块参数：
  -wp, --world-path 世界文件夹路径

参数说明：
  • 必须使用 -wp 指定世界文件夹路径（如果世界文件夹是分散的，则只指定服务器根目录便可，模块会自动扫描 level.dat 文件位置）`,

	// NBT 编辑器模块
	"editnbt": `NBT 编辑器模块 (-editnbt/--nbt-editor)
功能：编辑 Minecraft NBT 数据文件

使用示例：
  bex.exe -editnbt -np "C:/Server/worlds/overworld/level.dat"

模块参数：
  -np, --nbt-path NBT 文件路径

参数说明：
  • 必须使用 -np 指定 NBT 文件路径`,

	// DLL 注入工具模块
	"injector": `DLL 注入工具模块 (-injector/--dll-injector)
功能：DLL 注入工具

使用示例：
  bex.exe -injector -dp "C:/BedrockClient/latite.dll"
  bex.exe -injector -dp "C:/BedrockClient/latite.dll" -ct "Minecraft.Windows.exe"
  bex.exe -injector -dp "C:/BedrockClient/latite.dll" -ct "Minecraft.Windows.exe" -tm 1m30s
  bex.exe -injector -i
  bex.exe -injector -rc

模块参数：
  -dp, --dll-path       指定 DLL 路径
  -ct, --custom-target  自定义注入目标
  -i, --inject          读取上次注入路径并立即执行注入
  -tm, --task-mode      计划模式
  -rc, --reset-config   重置配置文件

参数说明：
  • -dp, -ct, -tm 可以选择性使用，最终都会执行注入操作
  • -rc 与 -i 独立使用`,

	// P2P 联机工具模块
	"p2p": `P2P 联机工具模块 (-p2p)
功能：P2P 虚拟网络工具

使用示例：
  bex.exe -p2p -cn -n "MyNetwork" -pw "MyPassword"
  bex.exe -p2p -jn -n "MyNetwork" -pw "MyPassword"
  bex.exe -p2p -l

模块参数：
  -cn, --create-network 创建虚拟网络
  -jn, --join-network   加入虚拟网络
  -l, --list            列出当前网络节点中的用户
  -n, --name            指定网络名称
  -pw, --password       指定网络密码

参数说明：
  • -cn 与 -jn 是互斥的，要么加入网络，要么创建网络
  • -l 用于列出当前网络节点中的用户
  • -n 和 -pw 二者缺一不可，前者指定网络名称，后者指定网络密码`,

	// 图标生成模块
	"icon": `图标生成模块 (-icon/--icon-maker)
功能：生成服务器图标

使用示例：
  bex.exe -icon -pp "C:/Picture/vanilla-icon.png"
  bex.exe -icon -pp "C:/Picture/vanilla-icon.png" -od "C:/Server"
  bex.exe -icon -pp "C:/Picture/vanilla-icon.png" -od "C:/Server" -pn "custom-name.png"

模块参数：
  -pp, --picture-path 源图片路径
  -od, --output-dir   输出目录
  -pn, --picture-name 输出图片名称（默认 server-icon.png）

参数说明：
  • -pp 是必须指定的，-pn 是可选项，不指定则输出为 server-icon.png
  • -od 是可选项，不指定则输出到当前目录（比如 cmd 工作目录在 C:/Server，则图标会输出到 C:/Server）`,

	// 世界备份模块
	"backup": `世界备份模块 (-backup/--world-backup)
功能：世界备份工具

使用示例：
  bex.exe -backup -bp "C:/Server" -sd "worlds/*" -bt 1h30m -le -mx 10
  bex.exe -backup -bp "C:/Server" -sd "worlds/*" -bt 1h30m -le
  bex.exe -backup -bp "C:/Server" -sd "worlds/nether" -bt 1h30m
  bex.exe -backup -bp "C:/Server" -sd "worlds/nether"

模块参数：
  -bp, --backup-path    备份工作目录
  -sd, --select-dir     备份目标文件夹
  -bt, --backup-time    循环备份间隔
  -le, --loop-execution 循环执行备份模式
  -mx, --max            最大备份次数

参数说明：
  • 必须用 -bp 指定工作目录，再用 -sd 指定备份目标文件夹
  • -sd 支持通配符 [worlds/*] 和相对路径 [worlds/nether]，不使用通配符指定目录也可以，但是可能会有小问题
  • -bt, -le, -mx 均为可选项，-bt 的格式为 X 时 X 分 X 秒/轮，如 1h30m 表示 1 小时 30 分钟/备份一轮
  • -le 只能在 -bt 后指定，-mx 只能在 -le 后指定
  • 在备份开始之时，该模块会在工作目录下创建一个名为 BEX_Backup 的子目录，用于存放 *.zip 格式的备份文件`,

	// 调度器专用参数
	"help": `帮助信息 (-h/--help)
功能：显示完整的帮助信息

使用示例：
  bex.exe -h
  bex.exe ?

参数说明：
  • -h 或者 ? 均可显示完整的帮助信息`,

	"about": `关于信息 (-about/--about)
功能：显示程序信息

使用示例：
  bex.exe -about

参数说明：
  • 显示程序关于信息，字面意思`,
}

// 参数到模块的映射
var paramToModule = map[string]string{
	// 模块选择参数
	"-query":               "query",
	"--query-server":       "query",
	"-ping":                "ping",
	"--ping-host":          "ping",
	"-rcon":                "rcon",
	"--rcon-remotecontrol": "rcon",
	"-log":                 "log",
	"--log-analyzer":       "log",
	"-nbt":                 "nbt",
	"--nbt-analyzer":       "nbt",
	"-serbat":              "serbat",
	"--generate-serverbat": "serbat",
	"-heatmap":             "heatmap",
	"--generate-heatmap":   "heatmap",
	"-world":               "world",
	"--world-analyzer":     "world",
	"-editnbt":             "editnbt",
	"--nbt-editor":         "editnbt",
	"-injector":            "injector",
	"--dll-injector":       "injector",
	"-p2p":                 "p2p",
	"-icon":                "icon",
	"--icon-maker":         "icon",
	"-backup":              "backup",
	"--world-backup":       "backup",

	// 调度器专用参数
	"-about":  "about",
	"--about": "about",
	"-h":      "help",
	"--help":  "help",
}

// showModuleHelp 显示特定模块的详细帮助
func showModuleHelp(param string) {
	// 确保参数以 - 开头
	if !strings.HasPrefix(param, "-") {
		param = "-" + param
	}

	// 查找对应的模块名
	moduleName, exists := paramToModule[param]
	if !exists {
		LogError(fmt.Sprintf("参数 %s 的帮助内容不存在！", param))
		LogInfo("可用的模块帮助：-query, -ping, -rcon, -log, -nbt, -serbat, -heatmap, -world, -editnbt, -injector, -p2p, -icon, -backup, -update")
		return
	}

	// 查找对应的帮助内容
	helpContent, exists := moduleHelps[moduleName]
	if !exists {
		LogError(fmt.Sprintf("模块 %s 的帮助内容不存在！", moduleName))
		return
	}

	// 显示帮助内容
	fmt.Printf("%s========== %s%s %s模块详细帮助 %s==========%s\n",
		ColorCyan, ColorBlue, param, ColorBrightYellow, ColorCyan, ColorClear)
	fmt.Println(helpContent)
}

// 屎山，我为什么要把帮助信息写到这里
// 逻辑代码行数估计还没帮助信息多
