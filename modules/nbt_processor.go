package modules

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bex/utils"
	"github.com/Tnze/go-mc/nbt"
)

type NBTData struct {
	FilePath string
	FileName string
	FileSize int64
	Data     map[string]interface{}
}

func NBTProcessor(filePath string, editMode bool) {
	processor := &NBTProcessorInstance{
		FilePath: filePath,
		EditMode: editMode,
	}

	processor.Execute()
}

type NBTProcessorInstance struct {
	FilePath string
	EditMode bool
	Data     *NBTData
}

func (n *NBTProcessorInstance) Execute() {
	if !n.validateArgs() {
		return
	}

	data, err := n.parseNBTFile(n.FilePath)
	if err != nil {
		utils.LogError("解析NBT文件失败: %s", err)
		return
	}
	n.Data = data

	if n.EditMode {
		n.editNBT()
	} else {
		n.displayNBTInfo()
	}
}

func (n *NBTProcessorInstance) validateArgs() bool {
	if !utils.FileExists(n.FilePath) {
		utils.LogError("NBT文件不存在: %s", n.FilePath)
		return false
	}

	fileInfo, err := os.Stat(n.FilePath)
	if err != nil {
		utils.LogError("无法访问文件: %s", err)
		return false
	}
	if fileInfo.IsDir() {
		utils.LogError("指定的路径是目录，不是文件: %s", n.FilePath)
		return false
	}

	return true
}

func (n *NBTProcessorInstance) parseNBTFile(filePath string) (*NBTData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}

	var data map[string]interface{}
	decoder := nbt.NewDecoder(file)
	tagName, err := decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("解析NBT失败 (标签: %s): %w", tagName, err)
	}

	return &NBTData{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		FileSize: fileInfo.Size(),
		Data:     data,
	}, nil
}

func (n *NBTProcessorInstance) displayNBTInfo() {
	// 检测是否为玩家数据
	if strings.Contains(n.Data.FileName, ".dat") && len(n.Data.FileName) == 40 { // UUID格式
		n.displayPlayerData()
	} else {
		n.displayGenericNBT()
	}
}

func (n *NBTProcessorInstance) displayPlayerData() {
	data := n.Data.Data

	fmt.Printf("\n%s═══════════════════════════════════════════════════════════════%s\n",
		utils.ColorCyan, utils.ColorClear)
	fmt.Printf("%s 玩家数据分析 - %s %s\n",
		utils.ColorBrightYellow, n.Data.FileName, utils.ColorClear)
	fmt.Printf("%s═══════════════════════════════════════════════════════════════%s\n\n",
		utils.ColorCyan, utils.ColorClear)

	playerName := n.getPlayerName(data)
	fmt.Printf("%s基本信息:%s\n", utils.ColorGreen, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  %s文件:%s %s (%s)\n",
		utils.ColorBrightYellow, utils.ColorClear,
		n.Data.FileName, utils.FormatFileSize(n.Data.FileSize))

	if playerName != "" {
		fmt.Printf("  %s玩家名称:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			utils.Colorize(playerName, utils.ColorGreen))
	}

	if health, ok := data["Health"].(float32); ok {
		color := utils.ColorGreen
		if health < 10 {
			color = utils.ColorRed
		} else if health < 15 {
			color = utils.ColorYellow
		}
		fmt.Printf("  %s生命值:%s %s%.1f%s/20.0\n",
			utils.ColorBrightYellow, utils.ColorClear,
			color, health, utils.ColorClear)
	}

	if foodLevel, ok := data["foodLevel"].(int32); ok {
		color := utils.ColorGreen
		if foodLevel < 10 {
			color = utils.ColorRed
		} else if foodLevel < 15 {
			color = utils.ColorYellow
		}
		fmt.Printf("  %s饥饿度:%s %s%d%s/20\n",
			utils.ColorBrightYellow, utils.ColorClear,
			color, foodLevel, utils.ColorClear)
	}

	if xpLevel, ok := data["XpLevel"].(int32); ok {
		if xpTotal, ok := data["XpTotal"].(int32); ok {
			fmt.Printf("  %s经验值:%s 等级 %d (总计 %d)\n",
				utils.ColorBrightYellow, utils.ColorClear, xpLevel, xpTotal)
		}
	}

	if gameType, ok := data["playerGameType"].(int32); ok {
		fmt.Printf("  %s游戏模式:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			n.formatGameMode(gameType))
	}

	firstPlayed, lastPlayed := n.getPlayTime(data)
	if firstPlayed > 0 && lastPlayed > 0 {
		playDays := float64(lastPlayed-firstPlayed) / (1000 * 60 * 60 * 24)
		fmt.Printf("  %s游玩时长:%s %.1f 天\n",
			utils.ColorBrightYellow, utils.ColorClear, playDays)
	}

	if lastPlayed > 0 {
		t := time.Unix(lastPlayed/1000, 0)
		fmt.Printf("  %s最后在线:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			utils.Colorize(t.Format("2006-01-02 15:04:05"), utils.ColorBlue))
	}

	n.displayPosition(data)
	n.displayPotionEffects(data)
	n.displayInventory(data)
	n.displayAbilities(data)

	fmt.Printf("\n%s═══════════════════════════════════════════════════════════════%s\n",
		utils.ColorCyan, utils.ColorClear)
}

// displayGenericNBT 显示通用NBT数据
func (n *NBTProcessorInstance) displayGenericNBT() {
	fmt.Printf("\n%s═══════════════════════════════════════════════════════════════%s\n",
		utils.ColorCyan, utils.ColorClear)
	fmt.Printf("%s NBT文件分析 - %s %s\n",
		utils.ColorBrightYellow, n.Data.FileName, utils.ColorClear)
	fmt.Printf("%s═══════════════════════════════════════════════════════════════%s\n\n",
		utils.ColorCyan, utils.ColorClear)

	fmt.Printf("%s文件信息:%s\n", utils.ColorGreen, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  %s路径:%s %s\n",
		utils.ColorBrightYellow, utils.ColorClear, n.Data.FilePath)
	fmt.Printf("  %s大小:%s %s\n",
		utils.ColorBrightYellow, utils.ColorClear, utils.FormatFileSize(n.Data.FileSize))
	fmt.Printf("  %s根标签:%s Data\n\n", utils.ColorBrightYellow, utils.ColorClear)

	if strings.Contains(strings.ToLower(n.Data.FileName), "level.dat") {
		n.displayWorldInfo()
	} else {
		n.displayNBTStructure(n.Data.Data, 0)
	}
}

func (n *NBTProcessorInstance) displayWorldInfo() {
	var levelData map[string]interface{}
	if d, ok := n.Data.Data["Data"].(map[string]interface{}); ok {
		levelData = d
	} else {
		levelData = n.Data.Data
	}

	fmt.Printf("%s世界信息:%s\n", utils.ColorBlue, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	if levelName, ok := levelData["LevelName"].(string); ok {
		fmt.Printf("  %s世界名称:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear, levelName)
	}

	if version, ok := levelData["Version"].(map[string]interface{}); ok {
		if name, ok := version["Name"].(string); ok {
			fmt.Printf("  %s游戏版本:%s %s\n",
				utils.ColorBrightYellow, utils.ColorClear, name)
		}
	}

	if seed, ok := levelData["RandomSeed"].(int64); ok {
		fmt.Printf("  %s随机种子:%s %d\n",
			utils.ColorBrightYellow, utils.ColorClear, seed)
	}

	if gameTime, ok := levelData["Time"].(int64); ok {
		days := gameTime / 24000
		fmt.Printf("  %s游戏时间:%s %d 天 (%d 刻)\n",
			utils.ColorBrightYellow, utils.ColorClear, days, gameTime)
	}

	if lastPlayed, ok := levelData["LastPlayed"].(int64); ok && lastPlayed > 0 {
		t := time.Unix(lastPlayed/1000, 0)
		fmt.Printf("  %s最后游玩:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			t.Format("2006-01-02 15:04:05"))
	}

	if commandBlocks, ok := levelData["allowCommands"].(byte); ok {
		enabled := "否"
		color := utils.ColorRed
		if commandBlocks != 0 {
			enabled = "是"
			color = utils.ColorGreen
		}
		fmt.Printf("  %s命令方块:%s %s%s%s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			color, enabled, utils.ColorClear)
	}

	if gameRules, ok := levelData["GameRules"].(map[string]interface{}); ok {
		fmt.Printf("\n  %s⚙游戏规则:%s\n", utils.ColorPurple, utils.ColorClear)
		for k, v := range gameRules {
			fmt.Printf("    %s%s:%s %v\n",
				utils.ColorCyan, k, utils.ColorClear, v)
		}
	}

	if dataPacks, ok := levelData["DataPacks"].(map[string]interface{}); ok {
		if enabled, ok := dataPacks["Enabled"].([]interface{}); ok {
			fmt.Printf("\n  %s已启用的数据包:%s\n", utils.ColorGreen, utils.ColorClear)
			for i, pack := range enabled {
				if i < 10 { // 只显示前10个
					fmt.Printf("    • %v\n", pack)
				}
			}
			if len(enabled) > 10 {
				fmt.Printf("    ... 还有 %d 个\n", len(enabled)-10)
			}
		}
	}
}

func (n *NBTProcessorInstance) displayNBTStructure(data interface{}, depth int) {
	indent := strings.Repeat("  ", depth)

	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fmt.Printf("%s%s%s%s: ", indent, utils.ColorYellow, key, utils.ColorClear)
			n.printValue(val, depth+1)
		}
	case []interface{}:
		fmt.Printf("%s数组长度: %d\n", indent, len(v))
		for i, item := range v {
			if i < 5 {
				fmt.Printf("%s  [%d]: ", indent, i)
				n.printValue(item, depth+2)
			} else {
				fmt.Printf("%s  ... 还有 %d 项\n", indent, len(v)-5)
				break
			}
		}
	default:
		n.printValue(v, depth)
	}
}

func (n *NBTProcessorInstance) printValue(val interface{}, depth int) {
	switch v := val.(type) {
	case map[string]interface{}:
		fmt.Printf("%s{...}%s\n", utils.ColorCyan, utils.ColorClear)
		n.displayNBTStructure(v, depth)
	case []interface{}:
		fmt.Printf("%s[...]%s\n", utils.ColorCyan, utils.ColorClear)
		n.displayNBTStructure(v, depth)
	case string:
		fmt.Printf("%s\"%s\"%s\n", utils.ColorGreen, v, utils.ColorClear)
	case int32, int64, int:
		fmt.Printf("%s%v%s\n", utils.ColorBlue, v, utils.ColorClear)
	case float32, float64:
		fmt.Printf("%s%.2f%s\n", utils.ColorPurple, v, utils.ColorClear)
	case bool:
		color := utils.ColorGreen
		if !v {
			color = utils.ColorRed
		}
		fmt.Printf("%s%v%s\n", color, v, utils.ColorClear)
	case byte:
		fmt.Printf("%s0x%02X%s\n", utils.ColorCyan, v, utils.ColorClear)
	default:
		fmt.Printf("%v\n", v)
	}
}

func (n *NBTProcessorInstance) editNBT() {
	utils.LogWarn("NBT编辑器功能正在开发中")
	utils.LogInfo("当前仅支持查看模式")

	fmt.Printf("\n%s是否进入交互式查看模式？(Y/N)%s ",
		utils.ColorYellow, utils.ColorClear)

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToUpper(response))

	if response == "Y" {
		n.interactiveBrowse()
	} else {
		n.displayNBTInfo()
	}
}

func (n *NBTProcessorInstance) interactiveBrowse() {
	current := interface{}(n.Data.Data)
	path := []string{"root"}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("\n%s当前路径: /%s%s\n",
			utils.ColorGreen, strings.Join(path, "/"), utils.ColorClear)

		i := 0
		items := make([]string, 0)

		switch v := current.(type) {
		case map[string]interface{}:
			fmt.Printf("\n%s目录:%s\n", utils.ColorCyan, utils.ColorClear)
			for key := range v {
				items = append(items, key)
				fmt.Printf("  %s[%d]%s %s\n",
					utils.ColorYellow, i, utils.ColorClear, key)
				i++
			}
		case []interface{}:
			fmt.Printf("\n%s数组 (长度 %d):%s\n", utils.ColorCyan, len(v), utils.ColorClear)
			for idx, item := range v {
				if idx < 20 {
					items = append(items, fmt.Sprintf("%d", idx))
					fmt.Printf("  %s[%d]%s [%d]: ",
						utils.ColorYellow, idx, utils.ColorClear, idx)
					n.printValue(item, 0)
				}
			}
		default:
			fmt.Printf("未知类型: %T\n", v)
		}

		fmt.Printf("\n%s选项:%s\n", utils.ColorPurple, utils.ColorClear)
		fmt.Printf("  • 输入数字选择项目\n")
		fmt.Printf("  • 输入 '..' 返回上级\n")
		fmt.Printf("  • 输入 'q' 退出\n\n")
		fmt.Printf("%s>%s ", utils.ColorGreen, utils.ColorClear)

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "q":
			return
		case "..":
			if len(path) > 1 {
				path = path[:len(path)-1]
				navResult := n.navigateTo(path)
				if navResult != nil {
					current = navResult
				}
			}
		default:
			var idx int
			_, err := fmt.Sscanf(input, "%d", &idx)
			if err == nil && idx >= 0 && idx < len(items) {
				path = append(path, items[idx])
				navResult := n.navigateTo(path)
				if navResult != nil {
					current = navResult
				}
			}
		}
	}
}

func (n *NBTProcessorInstance) navigateTo(path []string) interface{} {
	var current interface{} = n.Data.Data

	for i := 1; i < len(path); i++ {
		switch v := current.(type) {
		case map[string]interface{}:
			if next, ok := v[path[i]]; ok {
				current = next
			} else {
				return current
			}
		case []interface{}:
			var idx int
			_, err := fmt.Sscanf(path[i], "%d", &idx)
			if err != nil {
				return nil
			}
			if idx >= 0 && idx < len(v) {
				current = v[idx]
			} else {
				return current
			}
		default:
			return current
		}
	}

	return current
}

func (n *NBTProcessorInstance) getPlayerName(data map[string]interface{}) string {
	if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if name, ok := bukkit["lastKnownName"].(string); ok {
			return name
		}
	}
	if name, ok := data["Name"].(string); ok {
		return name
	}
	return ""
}

func (n *NBTProcessorInstance) getPlayTime(data map[string]interface{}) (int64, int64) {
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

	return firstPlayed, lastPlayed
}

func (n *NBTProcessorInstance) displayPosition(data map[string]interface{}) {
	fmt.Printf("\n%s位置信息:%s\n", utils.ColorBlue, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	if pos, ok := data["Pos"].([]interface{}); ok && len(pos) >= 3 {
		x := n.toFloat64(pos[0])
		y := n.toFloat64(pos[1])
		z := n.toFloat64(pos[2])
		fmt.Printf("  %s当前位置:%s X=%.1f Y=%.1f Z=%.1f\n",
			utils.ColorBrightYellow, utils.ColorClear, x, y, z)
	}

	spawnX, hasX := data["SpawnX"].(int32)
	spawnY, hasY := data["SpawnY"].(int32)
	spawnZ, hasZ := data["SpawnZ"].(int32)
	if hasX && hasY && hasZ {
		fmt.Printf("  %s重生点:%s X=%d Y=%d Z=%d\n",
			utils.ColorBrightYellow, utils.ColorClear, spawnX, spawnY, spawnZ)
	}

	if dimension, ok := data["Dimension"].(string); ok {
		fmt.Printf("  %s当前维度:%s %s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			n.formatDimension(dimension))
	}
}

func (n *NBTProcessorInstance) displayPotionEffects(data map[string]interface{}) {
	effects, ok := data["ActiveEffects"].([]interface{})
	if !ok || len(effects) == 0 {
		return
	}

	fmt.Printf("\n%s激活的药水效果:%s\n", utils.ColorPurple, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	for _, e := range effects {
		if effect, ok := e.(map[string]interface{}); ok {
			effectID := n.toInt32(effect["Id"])
			amplifier := n.toInt32(effect["Amplifier"])
			duration := n.toInt32(effect["Duration"])

			effectName := n.getEffectName(effectID)
			fmt.Printf("  • %s%s%s (等级 %d) - %.1f秒\n",
				utils.ColorBrightYellow, effectName, utils.ColorClear,
				amplifier+1, float64(duration)/20)
		}
	}
}

func (n *NBTProcessorInstance) displayInventory(data map[string]interface{}) {
	inventory, ok := data["Inventory"].([]interface{})
	if !ok || len(inventory) == 0 {
		return
	}

	fmt.Printf("\n%s物品栏统计:%s\n", utils.ColorYellow, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	itemCount := 0
	for _, item := range inventory {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if count, ok := itemMap["Count"].(byte); ok && count > 0 {
				itemCount++
			}
		}
	}

	fmt.Printf("  共有 %d 个物品栏位，%d 个物品\n", len(inventory), itemCount)

	// 显示快捷栏物品
	fmt.Printf("\n  %s快捷栏物品:%s\n", utils.ColorGreen, utils.ColorClear)
	shown := 0
	for _, item := range inventory {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if slot, ok := itemMap["Slot"].(int32); ok && slot < 9 {
				if shown < 5 {
					n.displayItemSummary(itemMap)
					shown++
				}
			}
		}
	}
	if shown == 0 {
		fmt.Printf("    空\n")
	}
}

func (n *NBTProcessorInstance) displayItemSummary(item map[string]interface{}) {
	id, _ := item["id"].(string)
	count, _ := item["Count"].(byte)

	// 移除 minecraft: 前缀
	id = strings.TrimPrefix(id, "minecraft:")

	fmt.Printf("    • %s x%d\n", id, count)
}

func (n *NBTProcessorInstance) displayAbilities(data map[string]interface{}) {
	abilities, ok := data["abilities"].(map[string]interface{})
	if !ok {
		return
	}

	fmt.Printf("\n%s⚡ 能力数据:%s\n", utils.ColorPurple, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	if walkSpeed, ok := abilities["walkSpeed"].(float32); ok {
		fmt.Printf("  %s行走速度:%s %.0f%%\n",
			utils.ColorBrightYellow, utils.ColorClear, walkSpeed*100)
	}

	if mayFly, ok := abilities["mayfly"].(byte); ok {
		status := "否"
		color := utils.ColorRed
		if mayFly != 0 {
			status = "是"
			color = utils.ColorGreen
		}
		fmt.Printf("  %s允许飞行:%s %s%s%s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			color, status, utils.ColorClear)
	}

	if flying, ok := abilities["flying"].(byte); ok {
		status := "否"
		color := utils.ColorRed
		if flying != 0 {
			status = "是"
			color = utils.ColorGreen
		}
		fmt.Printf("  %s正在飞行:%s %s%s%s\n",
			utils.ColorBrightYellow, utils.ColorClear,
			color, status, utils.ColorClear)
	}
}

func (n *NBTProcessorInstance) formatGameMode(mode int32) string {
	switch mode {
	case 0:
		return utils.Colorize("生存模式", utils.ColorBrightYellow)
	case 1:
		return utils.Colorize("创造模式", utils.ColorGreen)
	case 2:
		return utils.Colorize("冒险模式", utils.ColorCyan)
	case 3:
		return utils.Colorize("旁观模式", utils.ColorBlue)
	default:
		return utils.Colorize("未知模式", utils.ColorRed)
	}
}

func (n *NBTProcessorInstance) formatDimension(dim string) string {
	switch dim {
	case "minecraft:overworld", "overworld":
		return utils.Colorize("主世界", utils.ColorGreen)
	case "minecraft:the_nether", "the_nether", "nether":
		return utils.Colorize("下界", utils.ColorRed)
	case "minecraft:the_end", "the_end", "end":
		return utils.Colorize("末地", utils.ColorPurple)
	default:
		return utils.Colorize(dim, utils.ColorYellow)
	}
}

func (n *NBTProcessorInstance) getEffectName(id int32) string {
	effects := map[int32]string{
		1: "迅捷", 2: "缓慢", 3: "急迫", 4: "挖掘疲劳",
		5: "力量", 6: "瞬间治疗", 7: "瞬间伤害", 8: "跳跃提升",
		9: "反胃", 10: "生命恢复", 11: "抗性提升", 12: "抗火",
		13: "水下呼吸", 14: "隐身", 15: "失明", 16: "夜视",
		17: "饥饿", 18: "虚弱", 19: "中毒", 20: "凋零",
		21: "生命提升", 22: "伤害吸收", 23: "饱和", 24: "飘浮",
	}
	if name, ok := effects[id]; ok {
		return name
	}
	return fmt.Sprintf("未知效果(%d)", id)
}

func (n *NBTProcessorInstance) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float32:
		return float64(val)
	case float64:
		return val
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func (n *NBTProcessorInstance) toInt32(v interface{}) int32 {
	switch val := v.(type) {
	case int32:
		return val
	case int64:
		return int32(val)
	case byte:
		return int32(val)
	case float32:
		return int32(val)
	case float64:
		return int32(val)
	default:
		return 0
	}
}
