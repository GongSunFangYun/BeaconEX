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

type WorldInfo struct {
	Path        string
	Name        string
	Dimension   string
	LevelDatOK  bool
	FileCount   int64
	TotalSize   int64
	RegionFiles int64
	PlayerFiles int64
	DataPacks   int64
	LastPlayed  int64
	GameTime    int64
	RandomSeed  int64
	Version     string
	Errors      []string
}

func WorldAnalyzer(
	worldPath string,
) {
	scanner := &WorldScannerInstance{
		RootPath:   worldPath,
		MaxWorkers: 4,
		Worlds:     make(map[string]*WorldInfo),
	}

	scanner.Execute()
}

type WorldScannerInstance struct {
	RootPath   string
	MaxWorkers int
	Worlds     map[string]*WorldInfo
}

func (w *WorldScannerInstance) Execute() {
	utils.LogInfo("开始扫描目录: %s", w.RootPath)
	startTime := time.Now()

	if !w.validateArgs() {
		return
	}

	worldDirs := w.findAllWorldDirs()
	if len(worldDirs) == 0 {
		utils.LogError("未找到任何世界目录！")
		return
	}

	utils.LogInfo("共找到 %d 个世界目录", len(worldDirs))

	for i, worldPath := range worldDirs {
		utils.LogInfo("扫描进度: %d/%d - %s", i+1, len(worldDirs), filepath.Base(worldPath))
		w.scanWorld(worldPath)
	}

	scanTime := time.Since(startTime)
	utils.LogInfo("对所有维度扫描完成，耗时 %.2f 秒", scanTime.Seconds())

	w.outputStatistics()
}

func (w *WorldScannerInstance) validateArgs() bool {
	if !utils.FileExists(w.RootPath) {
		utils.LogError("路径不存在: %s", w.RootPath)
		return false
	}

	fileInfo, err := os.Stat(w.RootPath)
	if err != nil {
		utils.LogError("无法访问路径: %s", w.RootPath)
		return false
	}
	if !fileInfo.IsDir() {
		utils.LogError("指定的路径不是目录: %s", w.RootPath)
		return false
	}

	return true
}

func (w *WorldScannerInstance) findAllWorldDirs() []string {
	var worldDirs []string

	err := filepath.Walk(w.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == "level.dat" {
			worldDirs = append(worldDirs, filepath.Dir(path))
		}
		return nil
	})

	if err != nil {
		utils.LogError("扫描目录失败: %s", err)
	}

	return worldDirs
}

func (w *WorldScannerInstance) scanWorld(worldPath string) {
	info := &WorldInfo{
		Path:      worldPath,
		Name:      filepath.Base(worldPath),
		Dimension: w.identifyDimension(worldPath),
		Errors:    make([]string, 0),
	}

	levelDatPath := filepath.Join(worldPath, "level.dat")
	if utils.FileExists(levelDatPath) {
		w.parseLevelDat(levelDatPath, info)
	} else {
		info.Errors = append(info.Errors, "缺少 level.dat 文件")
	}

	w.scanDirectory(worldPath, info)

	w.Worlds[worldPath] = info
}

func (w *WorldScannerInstance) identifyDimension(worldPath string) string {
	pathLower := strings.ToLower(worldPath)
	nameLower := strings.ToLower(filepath.Base(worldPath))

	switch {
	case strings.Contains(pathLower, "nether") || strings.Contains(pathLower, "dim-1"):
		return "nether"
	case strings.Contains(pathLower, "end") || strings.Contains(pathLower, "dim1"):
		return "end"
	case nameLower == "world" || nameLower == "overworld":
		return "overworld"
	default:
		if utils.FileExists(filepath.Join(worldPath, "DIM-1")) {
			return "nether"
		}
		if utils.FileExists(filepath.Join(worldPath, "DIM1")) {
			return "end"
		}
		return "overworld"
	}
}

func (w *WorldScannerInstance) parseLevelDat(path string, info *WorldInfo) {
	file, err := os.Open(path)
	if err != nil {
		info.Errors = append(info.Errors, fmt.Sprintf("无法打开 level.dat: %s", err))
		return
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	gr, err := gzip.NewReader(file)
	if err != nil {
		info.Errors = append(info.Errors, fmt.Sprintf("解压 level.dat 失败: %s", err))
		return
	}
	defer func(gr *gzip.Reader) {
		_ = gr.Close()
	}(gr)

	var data map[string]interface{}
	decoder := nbt.NewDecoder(gr)
	_, err = decoder.Decode(&data)
	if err != nil {
		info.Errors = append(info.Errors, fmt.Sprintf("解析 level.dat 失败: %s", err))
		return
	}

	var levelData map[string]interface{}
	if d, ok := data["Data"].(map[string]interface{}); ok {
		levelData = d
	} else {
		levelData = data
	}

	info.LevelDatOK = true

	if lastPlayed, ok := levelData["LastPlayed"].(int64); ok {
		info.LastPlayed = lastPlayed
	}
	if gameTime, ok := levelData["Time"].(int64); ok {
		info.GameTime = gameTime
	}
	if seed, ok := levelData["RandomSeed"].(int64); ok {
		info.RandomSeed = seed
	}
	if version, ok := levelData["Version"].(map[string]interface{}); ok {
		if name, ok := version["Name"].(string); ok {
			info.Version = name
		}
	}
}

func (w *WorldScannerInstance) scanDirectory(path string, info *WorldInfo) {
	err := filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi == nil {
			return nil
		}

		if !fi.IsDir() {
			info.FileCount++
			info.TotalSize += fi.Size()

			ext := strings.ToLower(filepath.Ext(p))
			switch ext {
			case ".mca":
				info.RegionFiles++
			case ".dat":
				if strings.Contains(p, "playerdata") {
					info.PlayerFiles++
				}
			}

			if strings.Contains(p, "datapacks") && (ext == ".zip" || fi.IsDir()) {
				info.DataPacks++
			}
		}

		return nil
	})
	if err != nil {
		return
	}
}

func (w *WorldScannerInstance) outputStatistics() {
	utils.LogInfo(strings.Repeat("-", 50))

	var totalSize, totalFiles, totalRegions, totalPlayers, totalDataPacks int64
	var damagedWorlds []string

	for _, info := range w.Worlds {
		totalSize += info.TotalSize
		totalFiles += info.FileCount
		totalRegions += info.RegionFiles
		totalPlayers += info.PlayerFiles
		totalDataPacks += info.DataPacks

		if !info.LevelDatOK || len(info.Errors) > 0 {
			damagedWorlds = append(damagedWorlds, info.Name)
		}
	}

	utils.LogInfo("世界总数: %d", len(w.Worlds))
	utils.LogInfo("总文件数: %d", totalFiles)
	utils.LogInfo("总大小: %s", utils.FormatFileSize(totalSize))
	utils.LogInfo("Region文件: %d 个", totalRegions)
	utils.LogInfo("玩家数据: %d 个", totalPlayers)
	utils.LogInfo("数据包: %d 个", totalDataPacks)

	// 维度统计
	dimStats := make(map[string]int)
	for _, info := range w.Worlds {
		dimStats[info.Dimension]++
	}

	utils.LogInfo("世界分类统计:")
	for dim, count := range dimStats {
		utils.LogInfo("  %s: %d 个", dim, count)
	}

	utils.LogInfo(strings.Repeat("-", 50))
	utils.LogInfo("详细世界信息:")

	var worlds []*WorldInfo
	for _, info := range w.Worlds {
		worlds = append(worlds, info)
	}
	sort.Slice(worlds, func(i, j int) bool {
		return worlds[i].Name < worlds[j].Name
	})

	for _, info := range worlds {
		status := "✓ 正常"
		statusColor := utils.ColorGreen
		if !info.LevelDatOK || len(info.Errors) > 0 {
			status = "✗ 异常"
			statusColor = utils.ColorRed
		}

		utils.LogInfo("%s%s%s", utils.ColorCyan, info.Name, utils.ColorClear)
		utils.LogInfo("  状态: %s%s%s", statusColor, status, utils.ColorClear)
		utils.LogInfo("  维度: %s", info.Dimension)
		utils.LogInfo("  大小: %s", utils.FormatFileSize(info.TotalSize))
		utils.LogInfo("  文件: %d 个 (Region: %d, 玩家: %d, 数据包: %d)",
			info.FileCount, info.RegionFiles, info.PlayerFiles, info.DataPacks)

		if info.LevelDatOK {
			utils.LogInfo("  版本: %s", info.Version)
			utils.LogInfo("  游戏时间: %d 刻", info.GameTime)
			if info.LastPlayed > 0 {
				t := time.Unix(info.LastPlayed/1000, 0)
				utils.LogInfo("  最后游玩: %s", t.Format("2006-01-02 15:04:05"))
			}
		}

		for _, errMsg := range info.Errors {
			utils.LogWarn("  错误: %s", errMsg)
		}
		utils.LogInfo("")
	}

	if len(damagedWorlds) > 0 {
		utils.LogError("\n发现 %d 个可能损坏的世界:", len(damagedWorlds))
		for _, name := range damagedWorlds {
			utils.LogError("  - %s", name)
		}
	} else {
		utils.LogInfo("%s✓ 所有世界检查通过！%s", utils.ColorGreen, utils.ColorClear)
	}
}
