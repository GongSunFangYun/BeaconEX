package modules

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bex/utils"
	"github.com/Tnze/go-mc/nbt"
)

type PlayerData struct {
	UUID     string
	Name     string
	PlayTime float64
}

func HeatMap(
	dataFolderPath string,
	outputDir string,
) {
	generator := &HeatMapInstance{
		DataFolderPath: dataFolderPath,
		OutputDir:      outputDir,
	}

	generator.Execute()
}

type HeatMapInstance struct {
	DataFolderPath string
	OutputDir      string
}

func (h *HeatMapInstance) Execute() {
	utils.LogInfo("开始处理玩家数据...")
	time.Sleep(500 * time.Millisecond)

	if !h.validateArgs() {
		return
	}

	files, err := os.ReadDir(h.DataFolderPath)
	if err != nil {
		utils.LogError("读取目录失败: %s", err)
		return
	}

	var players []PlayerData
	totalFiles := 0
	validFiles := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".dat") {
			continue
		}

		totalFiles++
		utils.LogDebug("处理文件: %s", file.Name())

		playerData := h.parsePlayerNBT(filepath.Join(h.DataFolderPath, file.Name()))
		if playerData != nil {
			players = append(players, *playerData)
			validFiles++
		}
	}

	if len(players) == 0 {
		utils.LogError("未找到有效的玩家数据")
		return
	}

	utils.LogInfo("共找到 %d 个 NBT 文件，其中 %d 个有效玩家数据", totalFiles, validFiles)

	sort.Slice(players, func(i, j int) bool {
		if players[i].PlayTime != players[j].PlayTime {
			return players[i].PlayTime > players[j].PlayTime
		}
		return strings.ToLower(players[i].Name) < strings.ToLower(players[j].Name)
	})

	globalMax := players[0].PlayTime
	if globalMax <= 0 {
		globalMax = 1
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	h.displayASCIIChart(players, timestamp, globalMax)

	if h.OutputDir != "" && h.OutputDir != "none" {
		h.saveToFile(players, timestamp)
	}

	utils.LogInfo("热力图生成完成！")
}

func (h *HeatMapInstance) validateArgs() bool {
	if !utils.FileExists(h.DataFolderPath) {
		utils.LogError("路径不存在: %s", h.DataFolderPath)
		return false
	}

	fileInfo, err := os.Stat(h.DataFolderPath)
	if err != nil {
		utils.LogError("无法访问路径: %s", h.DataFolderPath)
		return false
	}
	if !fileInfo.IsDir() {
		utils.LogError("指定的路径不是目录: %s", h.DataFolderPath)
		return false
	}

	if h.OutputDir != "" && h.OutputDir != "none" {
		err := os.MkdirAll(h.OutputDir, 0755)
		if err != nil {
			utils.LogError("无法创建输出目录: %s", h.OutputDir)
			return false
		}
	}

	return true
}

func (h *HeatMapInstance) parsePlayerNBT(filePath string) *PlayerData {
	file, err := os.Open(filePath)
	if err != nil {
		utils.LogDebug("无法打开文件: %s", filePath)
		return nil
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		utils.LogDebug("解压 GZip 失败: %s: %v", filePath, err)
		return nil
	}
	defer func(gzReader *gzip.Reader) {
		_ = gzReader.Close()
	}(gzReader)

	var nbtData map[string]interface{}
	decoder := nbt.NewDecoder(gzReader)
	tagName, err := decoder.Decode(&nbtData)
	if err != nil {
		utils.LogDebug("解析 NBT 失败: %s, tagName: %s, err: %v", filePath, tagName, err)
		return nil
	}

	player := &PlayerData{
		UUID: strings.TrimSuffix(filepath.Base(filePath), ".dat"),
	}

	if bukkit, ok := nbtData["bukkit"].(map[string]interface{}); ok {
		if name, ok := bukkit["lastKnownName"].(string); ok {
			player.Name = name
		}
	}

	if player.Name == "" {
		if name, ok := nbtData["Name"].(string); ok {
			player.Name = name
		}
	}

	if player.Name == "" {
		if len(player.UUID) >= 8 {
			player.Name = "Player-" + player.UUID[:8]
		} else {
			player.Name = "Unknown-" + player.UUID
		}
	}

	player.PlayTime = h.calculatePlayTime(nbtData)

	if player.PlayTime <= 0 {
		return nil
	}

	return player
}

func (h *HeatMapInstance) calculatePlayTime(data map[string]interface{}) float64 {
	if gameTime, ok := data["playerGameTime"].(int64); ok && gameTime > 0 {
		hours := float64(gameTime) / 72000
		return maxFloat(0.1, hours/24)
	}

	var firstPlayed int64
	var lastPlayed int64

	if fp, ok := data["FirstPlayed"].(int64); ok {
		firstPlayed = fp
	} else if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if fp, ok := bukkit["firstPlayed"].(int64); ok {
			firstPlayed = fp
		}
	}

	if lp, ok := data["LastPlayed"].(int64); ok {
		lastPlayed = lp
	} else if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if lp, ok := bukkit["lastPlayed"].(int64); ok {
			lastPlayed = lp
		}
	}

	if firstPlayed > 0 && lastPlayed > 0 && lastPlayed > firstPlayed {
		days := float64(lastPlayed-firstPlayed) / (1000 * 60 * 60 * 24)
		return maxFloat(0.1, days)
	}

	return 0
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if (r >= 0x1100 && r <= 0x115F) ||
			r == 0x2329 || r == 0x232A ||
			(r >= 0x2E80 && r <= 0x303E) ||
			(r >= 0x3040 && r <= 0x33FF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0xA000 && r <= 0xA4CF) ||
			(r >= 0xAC00 && r <= 0xD7AF) ||
			(r >= 0xF900 && r <= 0xFAFF) ||
			(r >= 0xFE10 && r <= 0xFE1F) ||
			(r >= 0xFE30 && r <= 0xFE6F) ||
			(r >= 0xFF00 && r <= 0xFF60) ||
			(r >= 0xFFE0 && r <= 0xFFE6) ||
			(r >= 0x20000 && r <= 0x2A6DF) {
			width += 2
		} else {
			width++
		}
	}
	return width
}

func Pr(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateToWidth(s string, maxWidth int) string {
	if displayWidth(s) <= maxWidth {
		return s
	}
	var result []rune
	used := 0
	for _, r := range s {
		rw := displayWidth(string(r))
		if used+rw+3 > maxWidth {
			break
		}
		result = append(result, r)
		used += rw
	}
	return string(result) + "..."
}

func (h *HeatMapInstance) displayASCIIChart(players []PlayerData, timestamp string, globalMax float64) {
	const chartWidth = 20
	const nameColWidth = 22
	const rankWidth = 4

	const totalWidth = rankWidth + 1 + nameColWidth + 3 + 9 + 4 + chartWidth

	fmt.Printf("\n%s%s%s\n", utils.ColorCyan, strings.Repeat("═", totalWidth), utils.ColorClear)
	fmt.Printf("%s 玩家游玩热力图 - 生成时间: %s %s\n", utils.ColorBrightYellow, timestamp, utils.ColorClear)
	fmt.Printf("%s%s%s\n", utils.ColorCyan, strings.Repeat("═", totalWidth), utils.ColorClear)

	fmt.Printf("\n%s%s%s │  游玩天数  │ 进度条%s\n",
		utils.ColorGreen,
		Pr(strings.Repeat(" ", rankWidth+1)+"玩家名称", rankWidth+1+nameColWidth),
		utils.ColorClear,
		utils.ColorClear,
	)
	fmt.Println(strings.Repeat("─", totalWidth))

	for i, player := range players {
		barLength := int((player.PlayTime / globalMax) * float64(chartWidth))
		if barLength < 1 {
			barLength = 1
		}

		var barColor string
		switch {
		case player.PlayTime >= 30:
			barColor = utils.ColorRed
		case player.PlayTime >= 7:
			barColor = utils.ColorYellow
		case player.PlayTime >= 1:
			barColor = utils.ColorGreen
		default:
			barColor = utils.ColorBlue
		}

		name := truncateToWidth(player.Name, nameColWidth)
		paddedName := Pr(name, nameColWidth)

		fmt.Printf("%s%*d.%s ", utils.ColorBrightYellow, rankWidth-1, i+1, utils.ColorClear)
		fmt.Print(paddedName)
		fmt.Printf(" │ %s%7.1f天%s  │ ", utils.ColorBrightYellow, player.PlayTime, utils.ColorClear)
		fmt.Printf("%s%s%s%s\n",
			barColor,
			strings.Repeat("█", barLength),
			utils.ColorClear,
			strings.Repeat("░", chartWidth-barLength),
		)
	}

	fmt.Println(strings.Repeat("─", totalWidth))
	fmt.Printf("%s总计: %d 名玩家 │ 最长: %.1f 天 │ 最短: %.1f 天%s\n\n",
		utils.ColorCyan, len(players), globalMax, players[len(players)-1].PlayTime, utils.ColorClear)
}

func (h *HeatMapInstance) saveToFile(players []PlayerData, timestamp string) {
	if h.OutputDir == "" || h.OutputDir == "none" {
		return
	}

	filename := filepath.Join(h.OutputDir, "heatmap.txt")
	file, err := os.Create(filename)
	if err != nil {
		utils.LogError("无法创建文件: %s", filename)
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = fmt.Fprintf(file, "玩家游玩热力图 - 生成时间: %s\n", timestamp)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(file, "========================================\n\n")
	if err != nil {
		return
	}

	for i, player := range players {
		_, err := fmt.Fprintf(file, "%2d. %-20s : %.1f 天\n", i+1, player.Name, player.PlayTime)
		if err != nil {
			return
		}
	}

	utils.LogInfo("已保存数据到: %s", filename)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
