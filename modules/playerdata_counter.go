package modules

import (
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type MojangProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func CheckNBT(filePath string) {
	p := &CheckNBTInstance{FilePath: filePath}
	p.Execute()
}

type CheckNBTInstance struct {
	FilePath string
	Data     *NBTData
}

func (n *CheckNBTInstance) Execute() {
	if !n.validateArgs() {
		return
	}
	data, err := n.parseNBTFile(n.FilePath)
	if err != nil {
		utils.LogError("解析NBT文件失败: %s", err)
		return
	}
	n.Data = data
	n.displayNBTInfo()
}

func (n *CheckNBTInstance) validateArgs() bool {
	if !utils.FileExists(n.FilePath) {
		utils.LogError("NBT文件不存在: %s", n.FilePath)
		return false
	}
	fi, err := os.Stat(n.FilePath)
	if err != nil {
		utils.LogError("无法访问文件: %s", err)
		return false
	}
	if fi.IsDir() {
		utils.LogError("指定的路径是目录，不是文件: %s", n.FilePath)
		return false
	}
	return true
}

func (n *CheckNBTInstance) parseNBTFile(filePath string) (*NBTData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开文件: %w", err)
	}
	defer func() { _ = file.Close() }()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}

	magic := make([]byte, 2)
	if _, err := io.ReadFull(file, magic); err != nil {
		return nil, fmt.Errorf("读取文件头失败: %w", err)
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("文件 Seek 失败: %w", err)
	}

	var reader io.Reader
	switch {
	case magic[0] == 0x1f && magic[1] == 0x8b:
		gr, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("gzip 解压失败: %w", err)
		}
		defer func() { _ = gr.Close() }()
		reader = gr
	case magic[0] == 0x78:
		zr, err := zlib.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("zlib 解压失败: %w", err)
		}
		defer func() { _ = zr.Close() }()
		reader = zr
	default:
		reader = file
	}

	var data map[string]interface{}
	decoder := nbt.NewDecoder(reader)
	tagName, err := decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("解析NBT失败 (标签: %s): %w", tagName, err)
	}

	return &NBTData{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		FileSize: fi.Size(),
		Data:     data,
	}, nil
}

func (n *CheckNBTInstance) displayNBTInfo() {
	if strings.HasSuffix(n.Data.FileName, ".dat") && len(n.Data.FileName) == 40 {
		n.displayPlayerData()
	} else {
		n.displayGenericNBT()
	}
}

func tee(prefix, branch, label, value string) {
	dim := utils.ColorBrightBlack
	clr := utils.ColorClear
	lbl := utils.ColorBrightYellow + label + clr
	if value == "" {
		utils.LogInfo("%s%s%s%s%s", dim, prefix, branch, clr, lbl)
	} else {
		utils.LogInfo("%s%s%s%s%s  %s", dim, prefix, branch, clr, lbl, value)
	}
}

func stripAnsi(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func visualWidth(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x1100 &&
			(r <= 0x115F ||
				r == 0x2329 || r == 0x232A ||
				(r >= 0x2E80 && r <= 0x3247 && r != 0x303F) ||
				(r >= 0x3250 && r <= 0x4DBF) ||
				(r >= 0x4E00 && r <= 0xA4C6) ||
				(r >= 0xA960 && r <= 0xA97C) ||
				(r >= 0xAC00 && r <= 0xD7A3) ||
				(r >= 0xF900 && r <= 0xFAFF) ||
				(r >= 0xFE10 && r <= 0xFE19) ||
				(r >= 0xFE30 && r <= 0xFE6B) ||
				(r >= 0xFF01 && r <= 0xFF60) ||
				(r >= 0xFFE0 && r <= 0xFFE6) ||
				(r >= 0x1B000 && r <= 0x1B001) ||
				(r >= 0x1F200 && r <= 0x1F251) ||
				(r >= 0x20000 && r <= 0x3FFFD)) {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

func (n *CheckNBTInstance) teeBar(prefix, branch, label, numStr, bar string) {
	const numWidth = 8
	pad := numWidth - visualWidth(stripAnsi(numStr))
	if pad < 0 {
		pad = 0
	}
	dim := utils.ColorBrightBlack
	clr := utils.ColorClear
	lbl := utils.ColorBrightYellow + label + clr
	utils.LogInfo("%s%s%s%s%s  %s%s  %s",
		dim, prefix, branch, clr,
		lbl,
		numStr, strings.Repeat(" ", pad),
		bar)
}

func (n *CheckNBTInstance) displayPlayerData() {
	data := n.Data.Data
	uuid := strings.TrimSuffix(n.Data.FileName, ".dat")

	utils.LogInfo("%s%s%s  %s",
		utils.ColorBrightCyan, n.Data.FileName, utils.ColorClear,
		utils.Colorize("("+utils.FormatFileSize(n.Data.FileSize)+")", utils.ColorBrightBlack))

	cleanUUID := strings.ReplaceAll(uuid, "-", "")
	isOnline := false
	if len(cleanUUID) == 32 {
		vn, vr := cleanUUID[12], cleanUUID[16]
		isOnline = vn == '4' &&
			(vr == '8' || vr == '9' || vr == 'a' || vr == 'b' || vr == 'A' || vr == 'B')
	}

	playerName := n.resolvePlayerName(data, uuid)

	tee("", "├── ", "玩家信息", "")
	tee("│   ", "├── ", "UUID", utils.Colorize(uuid, utils.ColorCyan))
	if isOnline {
		tee("│   ", "├── ", "账户", utils.Colorize("正版登录", utils.ColorGreen))
	} else {
		tee("│   ", "├── ", "账户", utils.Colorize("离线登录", utils.ColorYellow))
	}
	if playerName != "" {
		tee("│   ", "└── ", "名称", utils.Colorize(playerName, utils.ColorBrightGreen))
	} else {
		tee("│   ", "└── ", "名称", utils.Colorize("(未知)", utils.ColorRed))
	}

	tee("", "├── ", "状态", "")
	n.treeStatus(data)
	tee("", "├── ", "位置", "")
	n.treePosition(data)
	tee("", "├── ", "能力", "")
	n.treeAbilities(data)

	effects, _ := data["ActiveEffects"].([]interface{})
	tee("", "├── ", "药水效果", fmt.Sprintf("%s%d 个%s",
		utils.ColorBrightBlack, len(effects), utils.ColorClear))
	n.treePotionEffects(effects)

	tee("", "└── ", "库存", "")
	n.treeInventory(data)
}

func (n *CheckNBTInstance) treeStatus(data map[string]interface{}) {
	if health, ok := data["Health"].(float32); ok {
		hColor := utils.ColorGreen
		if health < 10 {
			hColor = utils.ColorRed
		} else if health < 15 {
			hColor = utils.ColorYellow
		}
		bar := n.makeBar(int(health), 20, 20, hColor)
		numStr := fmt.Sprintf("%s%.1f/20%s", hColor, health, utils.ColorClear)
		n.teeBar("│   ", "├── ", "生命值", numStr, bar)
	}

	if food, ok := data["foodLevel"].(int32); ok {
		fColor := utils.ColorGreen
		if food < 10 {
			fColor = utils.ColorRed
		} else if food < 15 {
			fColor = utils.ColorYellow
		}
		bar := n.makeBar(int(food), 20, 20, fColor)
		numStr := fmt.Sprintf("%s%d/20%s", fColor, food, utils.ColorClear)
		n.teeBar("│   ", "├── ", "饥饿度", numStr, bar)
	}

	if xpLevel, ok := data["XpLevel"].(int32); ok {
		xpTotal, _ := data["XpTotal"].(int32)
		xpP, _ := data["XpP"].(float32)
		bar := n.makeBar(int(xpP*100), 100, 20, utils.ColorBrightGreen)
		numStr := fmt.Sprintf("%s%d级%s", utils.ColorBrightGreen, xpLevel, utils.ColorClear)
		suffix := fmt.Sprintf("  %s总计 %d  %s(%.0f%%)%s",
			utils.ColorGreen, xpTotal, utils.ColorBrightBlack, xpP*100, utils.ColorClear)
		n.teeBar("│   ", "├── ", "经验值", numStr, bar+suffix)
	}

	if gameType, ok := data["playerGameType"].(int32); ok {
		tee("│   ", "├── ", "游戏模式", n.formatGameMode(gameType))
	}

	if score, ok := data["Score"].(int32); ok {
		tee("│   ", "├── ", "死亡得分", utils.Colorize(fmt.Sprintf("%d", score), utils.ColorCyan))
	}

	firstPlayed, lastPlayed := n.getPlayTime(data)
	if firstPlayed > 0 && lastPlayed > 0 {
		playDays := float64(lastPlayed-firstPlayed) / float64(1000*60*60*24)
		tee("│   ", "├── ", "在线总计", utils.Colorize(fmt.Sprintf("%.1f 天", playDays), utils.ColorPurple))
	}
	if firstPlayed > 0 {
		t := time.Unix(firstPlayed/1000, 0)
		tee("│   ", "├── ", "首次游玩", utils.Colorize(t.Format("2006-01-02 15:04:05"), utils.ColorCyan))
	}
	if lastPlayed > 0 {
		t := time.Unix(lastPlayed/1000, 0)
		tee("│   ", "└── ", "最后游玩", utils.Colorize(t.Format("2006-01-02 15:04:05"), utils.ColorBlue))
	}
}

func (n *CheckNBTInstance) treePosition(data map[string]interface{}) {
	if pos, ok := data["Pos"].([]interface{}); ok && len(pos) >= 3 {
		x, y, z := n.toFloat64(pos[0]), n.toFloat64(pos[1]), n.toFloat64(pos[2])
		tee("│   ", "├── ", "坐标", fmt.Sprintf("%sX=%.2f%s  %sY=%.2f%s  %sZ=%.2f%s",
			utils.ColorGreen, x, utils.ColorClear,
			utils.ColorYellow, y, utils.ColorClear,
			utils.ColorGreen, z, utils.ColorClear))
	} else {
		tee("│   ", "├── ", "坐标", utils.Colorize("无", utils.ColorBrightBlack))
	}

	if rot, ok := data["Rotation"].([]interface{}); ok && len(rot) >= 2 {
		tee("│   ", "├── ", "朝向", fmt.Sprintf("%sYaw=%.1f%s  %sPitch=%.1f%s",
			utils.ColorCyan, n.toFloat64(rot[0]), utils.ColorClear,
			utils.ColorCyan, n.toFloat64(rot[1]), utils.ColorClear))
	} else {
		tee("│   ", "├── ", "朝向", utils.Colorize("无", utils.ColorBrightBlack))
	}

	spawnX, hasX := data["SpawnX"].(int32)
	spawnY, hasY := data["SpawnY"].(int32)
	spawnZ, hasZ := data["SpawnZ"].(int32)
	if hasX && hasY && hasZ {
		tee("│   ", "├── ", "重生点", fmt.Sprintf("%sX=%d%s  %sY=%d%s  %sZ=%d%s",
			utils.ColorGreen, spawnX, utils.ColorClear,
			utils.ColorYellow, spawnY, utils.ColorClear,
			utils.ColorGreen, spawnZ, utils.ColorClear))
	} else {
		tee("│   ", "├── ", "重生点", utils.Colorize("默认", utils.ColorBrightBlack))
	}

	if dimension, ok := data["Dimension"].(string); ok {
		tee("│   ", "└── ", "维度", n.formatDimension(dimension))
	} else {
		tee("│   ", "└── ", "维度", utils.Colorize("无", utils.ColorBrightBlack))
	}
}

func (n *CheckNBTInstance) treeAbilities(data map[string]interface{}) {
	abilities, _ := data["abilities"].(map[string]interface{})

	getFloat := func(key string, fallback float32) float32 {
		if abilities == nil {
			return fallback
		}
		if v, ok := abilities[key].(float32); ok {
			return v
		}
		return fallback
	}
	getBool := func(key string) bool {
		if abilities == nil {
			return false
		}
		if v, ok := abilities[key]; ok {
			return n.nbtByte(v) != 0
		}
		return false
	}
	yn := func(v bool) string {
		if v {
			return utils.Colorize("是", utils.ColorGreen)
		}
		return utils.Colorize("否", utils.ColorRed)
	}

	ws := getFloat("walkSpeed", 0.1)
	tee("│   ", "├── ", "移动速度", utils.Colorize(fmt.Sprintf("%.0f%%", ws/0.1*100), utils.ColorBrightGreen))
	fs := getFloat("flySpeed", 0.05)
	tee("│   ", "├── ", "飞行速度", utils.Colorize(fmt.Sprintf("%.0f%%", fs/0.05*100), utils.ColorBrightGreen))
	tee("│   ", "├── ", "允许飞行", yn(getBool("mayfly")))
	tee("│   ", "├── ", "正在飞行", yn(getBool("flying")))
	tee("│   ", "├── ", "无敌模式", yn(getBool("invulnerable")))
	tee("│   ", "└── ", "瞬间建造", yn(getBool("instabuild")))
}

func (n *CheckNBTInstance) treePotionEffects(effects []interface{}) {
	if len(effects) == 0 {
		tee("│   ", "└── ", "(无)", "")
		return
	}
	for i, e := range effects {
		effect, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		effectID := n.toInt32(effect["Id"])
		amplifier := n.toInt32(effect["Amplifier"])
		duration := n.toInt32(effect["Duration"])
		ambient := n.toInt32(effect["Ambient"])

		branch := "├── "
		cont := "│   "
		if i == len(effects)-1 {
			branch = "└── "
			cont = "    "
		}

		ambientTag := ""
		if ambient != 0 {
			ambientTag = "  " + utils.Colorize("[信标]", utils.ColorCyan)
		}
		val := fmt.Sprintf("%sLv.%d%s  %s%.1fs%s%s",
			utils.ColorPurple, amplifier+1, utils.ColorClear,
			utils.ColorBrightBlack, float64(duration)/20, utils.ColorClear,
			ambientTag)
		tee("│   ", branch, n.getEffectName(effectID), val)
		_ = cont
	}
}

func (n *CheckNBTInstance) treeInventory(data map[string]interface{}) {
	type item struct {
		slot  int8
		id    string
		count int8
		enchs int
	}

	inventory, _ := data["Inventory"].([]interface{})
	var hotbar, backpack []item
	armorOrder := []int8{103, 102, 101, 100}
	armorSlotName := map[int8]string{103: "头盔", 102: "胸甲", 101: "护腿", 100: "靴子"}
	armorItems := map[int8]item{}
	var offhand *item

	for _, raw := range inventory {
		m, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		slot := n.nbtByte(m["Slot"])
		id, _ := m["id"].(string)
		id = strings.TrimPrefix(id, "minecraft:")
		count := n.nbtByte(m["Count"])
		enchCount := n.enchantCount(m)
		it := item{slot: slot, id: id, count: count, enchs: enchCount}
		switch {
		case slot >= 0 && slot <= 8:
			hotbar = append(hotbar, it)
		case slot >= 9 && slot <= 35:
			backpack = append(backpack, it)
		case slot == 100 || slot == 101 || slot == 102 || slot == 103:
			armorItems[slot] = it
		case slot == -106:
			cp := it
			offhand = &cp
		}
	}

	var armorList []item
	for _, slot := range armorOrder {
		if it, ok := armorItems[slot]; ok {
			armorList = append(armorList, it)
		}
	}

	ec, _ := data["EnderItems"].([]interface{})

	printItems := func(items []item, prefix string) {
		for i, it := range items {
			br := "├── "
			if i == len(items)-1 {
				br = "└── "
			}
			n.teeItem(prefix, br, fmt.Sprintf("%2d", it.slot), it.id, int(it.count), it.enchs)
		}
	}

	tee("    ", "├── ", fmt.Sprintf("快捷栏  %s%d 物品%s", utils.ColorBrightBlack, len(hotbar), utils.ColorClear), "")
	if len(hotbar) == 0 {
		tee("    │   ", "└── ", "(空)", "")
	} else {
		printItems(hotbar, "    │   ")
	}

	tee("    ", "├── ", fmt.Sprintf("背包    %s%d 物品%s", utils.ColorBrightBlack, len(backpack), utils.ColorClear), "")
	if len(backpack) == 0 {
		tee("    │   ", "└── ", "(空)", "")
	} else {
		printItems(backpack, "    │   ")
	}

	tee("    ", "├── ", "副手", "")
	if offhand == nil {
		tee("    │   ", "└── ", "(空)", "")
	} else {
		n.teeItem("    │   ", "└── ", "副手", offhand.id, int(offhand.count), offhand.enchs)
	}

	tee("    ", "├── ", fmt.Sprintf("护甲    %s%d 件%s", utils.ColorBrightBlack, len(armorList), utils.ColorClear), "")
	if len(armorList) == 0 {
		tee("    │   ", "└── ", "(空)", "")
	} else {
		for i, slot := range armorOrder {
			it, ok := armorItems[slot]
			if !ok {
				continue
			}

			lastIdx := -1
			for j := len(armorOrder) - 1; j >= 0; j-- {
				if _, ok2 := armorItems[armorOrder[j]]; ok2 {
					lastIdx = j
					break
				}
			}
			br := "├── "
			if i == lastIdx {
				br = "└── "
			}
			n.teeItem("    │   ", br, armorSlotName[slot], it.id, int(it.count), it.enchs)
		}
	}

	tee("    ", "└── ", fmt.Sprintf("末影箱  %s%d 物品%s", utils.ColorBrightBlack, len(ec), utils.ColorClear), "")
	if len(ec) == 0 {
		tee("        ", "└── ", "(空)", "")
	} else {
		for i, raw := range ec {
			m, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			br := "├── "
			if i == len(ec)-1 {
				br = "└── "
			}
			slot := n.nbtByte(m["Slot"])
			id, _ := m["id"].(string)
			id = strings.TrimPrefix(id, "minecraft:")
			count := n.nbtByte(m["Count"])
			enchCount := n.enchantCount(m)
			n.teeItem("        ", br, fmt.Sprintf("%2d", slot), id, int(count), enchCount)
		}
	}
}

func (n *CheckNBTInstance) teeItem(prefix, branch, slotLabel, id string, count, enchCount int) {
	enchStr := ""
	if enchCount > 0 {
		enchStr = fmt.Sprintf("  %s[附魔 x%d]%s", utils.ColorPurple, enchCount, utils.ColorClear)
	}
	val := fmt.Sprintf("%s  %sx%d%s%s",
		utils.Colorize(id, utils.ColorBrightYellow),
		utils.ColorWhite, count, utils.ColorClear,
		enchStr)
	tee(prefix, branch,
		fmt.Sprintf("%s[%s]%s", utils.ColorBrightBlack, slotLabel, utils.ColorClear),
		val)
}

func (n *CheckNBTInstance) enchantCount(item map[string]interface{}) int {
	tag, ok := item["tag"].(map[string]interface{})
	if !ok {
		return 0
	}
	enchs, ok := tag["Enchantments"].([]interface{})
	if !ok {
		return 0
	}
	return len(enchs)
}

func (n *CheckNBTInstance) resolvePlayerName(data map[string]interface{}, uuid string) string {
	if name := n.getPlayerNameFromNBT(data); name != "" {
		return name
	}
	cleanUUID := strings.ReplaceAll(uuid, "-", "")
	if len(cleanUUID) != 32 {
		return ""
	}
	vn, vr := cleanUUID[12], cleanUUID[16]
	isOnline := vn == '4' && (vr == '8' || vr == '9' || vr == 'a' || vr == 'b' || vr == 'A' || vr == 'B')
	if isOnline {
		utils.LogInfo("检测到正版UUID (v=%c, variant=%c)，查询 Mojang API...", vn, vr)
		if name := n.queryMojangAPI(uuid); name != "" {
			return name
		}
	} else {
		utils.LogInfo("检测到离线UUID (v=%c, variant=%c)，跳过 Mojang API", vn, vr)
	}
	return ""
}

func (n *CheckNBTInstance) getPlayerNameFromNBT(data map[string]interface{}) string {
	if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if name, ok := bukkit["lastKnownName"].(string); ok && name != "" {
			return name
		}
	}
	if name, ok := data["Name"].(string); ok && name != "" {
		return name
	}
	if paper, ok := data["Paper"].(map[string]interface{}); ok {
		if name, ok := paper["LastKnownName"].(string); ok && name != "" {
			return name
		}
	}
	return ""
}

func (n *CheckNBTInstance) queryMojangAPI(uuid string) string {
	cleanUUID := strings.ReplaceAll(uuid, "-", "")
	url := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/profile/%s", cleanUUID)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		utils.LogWarn("Mojang API 请求失败: %s", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 204 || resp.StatusCode == 404 {
		utils.LogWarn("Mojang API 未找到该UUID的玩家")
		return ""
	}
	if resp.StatusCode != 200 {
		utils.LogWarn("Mojang API 返回异常状态码: %d", resp.StatusCode)
		return ""
	}
	var profile MojangProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		utils.LogWarn("解析 Mojang API 响应失败: %s", err)
		return ""
	}
	return profile.Name
}

func (n *CheckNBTInstance) displayGenericNBT() {
	utils.LogInfo("%sNBT 文件%s", utils.ColorBrightCyan, utils.ColorClear)
	tee("", "├── ", "路径", utils.Colorize(n.Data.FilePath, utils.ColorWhite))
	tee("", "└── ", "大小", utils.Colorize(utils.FormatFileSize(n.Data.FileSize), utils.ColorCyan))
	if strings.Contains(strings.ToLower(n.Data.FileName), "level.dat") {
		n.displayWorldInfo()
	} else {
		n.displayNBTStructure(n.Data.Data, 0)
	}
}

func (n *CheckNBTInstance) displayWorldInfo() {
	var levelData map[string]interface{}
	if d, ok := n.Data.Data["Data"].(map[string]interface{}); ok {
		levelData = d
	} else {
		levelData = n.Data.Data
	}
	utils.LogInfo("%s世界信息%s", utils.ColorBrightCyan, utils.ColorClear)
	if v, ok := levelData["LevelName"].(string); ok {
		tee("", "├── ", "名称", utils.Colorize(v, utils.ColorBrightWhite))
	}
	if version, ok := levelData["Version"].(map[string]interface{}); ok {
		if name, ok := version["Name"].(string); ok {
			tee("", "├── ", "版本", utils.Colorize(name, utils.ColorGreen))
		}
	}
	if seed, ok := levelData["RandomSeed"].(int64); ok {
		tee("", "├── ", "随机种子", utils.Colorize(fmt.Sprintf("%d", seed), utils.ColorCyan))
	}
	if gameTime, ok := levelData["Time"].(int64); ok {
		days := gameTime / 24000
		hour := (gameTime%24000*24/24000 + 6) % 24
		tee("", "├── ", "游戏时间", fmt.Sprintf("%s第 %d 天 %02d:xx%s  %s(%d ticks)%s",
			utils.ColorWhite, days, hour, utils.ColorClear,
			utils.ColorBrightBlack, gameTime, utils.ColorClear))
	}
	if lastPlayed, ok := levelData["LastPlayed"].(int64); ok && lastPlayed > 0 {
		t := time.Unix(lastPlayed/1000, 0)
		tee("", "├── ", "最后游玩", utils.Colorize(t.Format("2006-01-02 15:04:05"), utils.ColorBlue))
	}
	if cb, ok := levelData["allowCommands"]; ok {
		s := utils.Colorize("否", utils.ColorRed)
		if n.nbtByte(cb) != 0 {
			s = utils.Colorize("是", utils.ColorGreen)
		}
		tee("", "└── ", "命令方块", s)
	}
}

func (n *CheckNBTInstance) displayNBTStructure(data interface{}, depth int) {
	indent := strings.Repeat("    ", depth)
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			fmt.Printf("%s%s%s%s: ", indent, utils.ColorYellow, key, utils.ColorClear)
			n.printValue(val, depth+1)
		}
	case []interface{}:
		fmt.Printf("%s%s[%d 项]%s\n", indent, utils.ColorBrightBlack, len(v), utils.ColorClear)
		for i, item := range v {
			fmt.Printf("%s  %s[%d]%s  ", indent, utils.ColorBrightBlack, i, utils.ColorClear)
			n.printValue(item, depth+2)
		}
	default:
		n.printValue(v, depth)
	}
}

func (n *CheckNBTInstance) printValue(val interface{}, depth int) {
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
		fmt.Printf("%s%.4f%s\n", utils.ColorPurple, v, utils.ColorClear)
	case bool:
		color := utils.ColorGreen
		if !v {
			color = utils.ColorRed
		}
		fmt.Printf("%s%v%s\n", color, v, utils.ColorClear)
	case int8:
		fmt.Printf("%s%d%s\n", utils.ColorCyan, v, utils.ColorClear)
	case byte:
		fmt.Printf("%s0x%02X%s\n", utils.ColorCyan, v, utils.ColorClear)
	default:
		fmt.Printf("%v\n", v)
	}
}

func (n *CheckNBTInstance) makeBar(current, max, width int, color string) string {
	if max == 0 {
		max = 1
	}
	filled := current * width / max
	if filled > width {
		filled = width
	}
	return color + strings.Repeat("█", filled) + utils.ColorBrightBlack + strings.Repeat("░", width-filled) + utils.ColorClear
}

func (n *CheckNBTInstance) formatGameMode(mode int32) string {
	switch mode {
	case 0:
		return utils.Colorize("生存", utils.ColorBrightYellow)
	case 1:
		return utils.Colorize("创造", utils.ColorGreen)
	case 2:
		return utils.Colorize("冒险", utils.ColorCyan)
	case 3:
		return utils.Colorize("旁观", utils.ColorBlue)
	default:
		return utils.Colorize("未知", utils.ColorRed)
	}
}

func (n *CheckNBTInstance) formatDimension(dim string) string {
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

func (n *CheckNBTInstance) getPlayTime(data map[string]interface{}) (int64, int64) {
	var first, last int64

	if v, ok := data["FirstPlayed"].(int64); ok {
		first = v
	} else if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if v, ok := bukkit["firstPlayed"].(int64); ok {
			first = v
		}
	}

	if v, ok := data["LastPlayed"].(int64); ok {
		last = v
	} else if bukkit, ok := data["bukkit"].(map[string]interface{}); ok {
		if v, ok := bukkit["lastPlayed"].(int64); ok {
			last = v
		}
	}

	return first, last
}

func (n *CheckNBTInstance) getEffectName(id int32) string {
	effects := map[int32]string{
		1: "迅捷", 2: "缓慢", 3: "急迫", 4: "挖掘疲劳",
		5: "力量", 6: "瞬间治疗", 7: "瞬间伤害", 8: "跳跃提升",
		9: "反胃", 10: "生命恢复", 11: "抗性提升", 12: "抗火",
		13: "水下呼吸", 14: "隐身", 15: "失明", 16: "夜视",
		17: "饥饿", 18: "虚弱", 19: "中毒", 20: "凋零",
		21: "生命提升", 22: "伤害吸收", 23: "饱和", 24: "飘浮",
		25: "幸运", 26: "霉运", 27: "缓降", 28: "潮涌能量",
		29: "海豚的恩惠", 30: "不祥之兆", 31: "英雄村民",
	}
	if name, ok := effects[id]; ok {
		return name
	}
	return fmt.Sprintf("未知效果(%d)", id)
}

func (n *CheckNBTInstance) nbtByte(v interface{}) int8 {
	switch val := v.(type) {
	case int8:
		return val
	case byte:
		return int8(val)
	case int32:
		return int8(val)
	case int64:
		return int8(val)
	default:
		return 0
	}
}

func (n *CheckNBTInstance) toFloat64(v interface{}) float64 {
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

func (n *CheckNBTInstance) toInt32(v interface{}) int32 {
	switch val := v.(type) {
	case int32:
		return val
	case int64:
		return int32(val)
	case int8:
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

func (n *CheckNBTInstance) toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int32:
		return int64(val)
	case int64:
		return val
	case int8:
		return int64(val)
	case byte:
		return int64(val)
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}
