package utils

import (
	"encoding/json"
	"fmt"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ColorClear = "\033[0m"

	StyleBold      = "\033[1m"
	StyleDim       = "\033[2m"
	StyleItalic    = "\033[3m"
	StyleUnderline = "\033[4m"

	ColorBlack  = "\033[30m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"

	ColorBrightBlack  = "\033[90m"
	ColorBrightRed    = "\033[91m"
	ColorBrightGreen  = "\033[92m"
	ColorBrightYellow = "\033[93m"
	ColorBrightBlue   = "\033[94m"
	ColorBrightPurple = "\033[95m"
	ColorBrightCyan   = "\033[96m"
	ColorBrightWhite  = "\033[97m"
)

var (
	colorableStdout = colorable.NewColorableStdout()
	colorableStderr = colorable.NewColorableStderr()
)

var mcColorMap = map[string]string{
	"0": ColorBlack,
	"1": ColorBlue,
	"2": ColorGreen,
	"3": ColorCyan,
	"4": ColorRed,
	"5": ColorPurple,
	"6": ColorYellow,
	"7": ColorWhite,
	"8": ColorBrightBlack,
	"9": ColorBrightBlue,
	"a": ColorBrightGreen,
	"b": ColorBrightCyan,
	"c": ColorBrightRed,
	"d": ColorBrightPurple,
	"e": ColorBrightYellow,
	"f": ColorBrightWhite,
	"l": StyleBold,
	"m": "\033[9m",
	"n": StyleUnderline,
	"o": StyleItalic,
	"r": ColorClear,
}

func Colorize(text string, colorCode string) string {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return text
	}
	return colorCode + text + ColorClear
}

func ParseMinecraftFormat(mcText string) string {
	if mcText == "" {
		return mcText
	}

	var ansiText []rune
	i := 0
	runes := []rune(mcText)

	for i < len(runes) {
		if runes[i] == '§' && i+1 < len(runes) {
			code := strings.ToLower(string(runes[i+1]))
			if ansi, ok := mcColorMap[code]; ok {
				ansiText = append(ansiText, []rune(ansi)...)
			}
			i += 2
		} else {
			ansiText = append(ansiText, runes[i])
			i++
		}
	}

	return string(ansiText) + ColorClear
}

func JSONColorToSectionSign(color string) string {
	colorMap := map[string]string{
		"black":        "§0",
		"dark_blue":    "§1",
		"dark_green":   "§2",
		"dark_aqua":    "§3",
		"dark_red":     "§4",
		"dark_purple":  "§5",
		"gold":         "§6",
		"gray":         "§7",
		"dark_gray":    "§8",
		"blue":         "§9",
		"green":        "§a",
		"aqua":         "§b",
		"red":          "§c",
		"light_purple": "§d",
		"yellow":       "§e",
		"white":        "§f",
	}
	if code, ok := colorMap[color]; ok {
		return code
	}
	return ""
}

func JSONStyleToSectionSign(style string, enabled bool) string {
	if !enabled {
		return ""
	}

	styleMap := map[string]string{
		"bold":          "§l",
		"italic":        "§o",
		"underlined":    "§n",
		"strikethrough": "§m",
		"obfuscated":    "§k",
	}

	if code, ok := styleMap[style]; ok {
		return code
	}
	return ""
}

func ParseMOTDFromJSON(desc interface{}) string {
	if desc == nil {
		return ""
	}

	if str, ok := desc.(string); ok {
		return str
	}

	if m, ok := desc.(map[string]interface{}); ok {
		var result strings.Builder

		text := ""
		if t, ok := m["text"]; ok {
			text = fmt.Sprintf("%v", t)
		}

		if color, ok := m["color"]; ok {
			if colorStr, ok := color.(string); ok {
				result.WriteString(JSONColorToSectionSign(colorStr))
			}
		}

		if bold, ok := m["bold"]; ok && bold.(bool) {
			result.WriteString(JSONStyleToSectionSign("bold", true))
		}
		if italic, ok := m["italic"]; ok && italic.(bool) {
			result.WriteString(JSONStyleToSectionSign("italic", true))
		}
		if underlined, ok := m["underlined"]; ok && underlined.(bool) {
			result.WriteString(JSONStyleToSectionSign("underlined", true))
		}
		if strikethrough, ok := m["strikethrough"]; ok && strikethrough.(bool) {
			result.WriteString(JSONStyleToSectionSign("strikethrough", true))
		}
		if obfuscated, ok := m["obfuscated"]; ok && obfuscated.(bool) {
			result.WriteString(JSONStyleToSectionSign("obfuscated", true))
		}

		result.WriteString(text)

		if extra, ok := m["extra"]; ok {
			if extras, ok := extra.([]interface{}); ok {
				for _, e := range extras {
					result.WriteString(ParseMOTDFromJSON(e))
				}
			}
		}

		return result.String()
	}

	if arr, ok := desc.([]interface{}); ok {
		var result strings.Builder
		for _, item := range arr {
			result.WriteString(ParseMOTDFromJSON(item))
		}
		return result.String()
	}

	return ""
}

func formatMessage(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func GetBaseDirectory() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exePath)
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func SaveJSON(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func LoadJSON(path string, data interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, data)
}

func ParseTimeString(timeStr string) int {
	timeStr = strings.ToLower(timeStr)
	totalSeconds := 0

	re := regexp.MustCompile(`(\d+)([smh])`)
	matches := re.FindAllStringSubmatch(timeStr, -1)

	for _, match := range matches {
		if len(match) != 3 {
			continue
		}

		num, _ := strconv.Atoi(match[1])
		unit := match[2]

		switch unit {
		case "s":
			totalSeconds += num
		case "m":
			totalSeconds += num * 60
		case "h":
			totalSeconds += num * 3600
		}
	}
	return totalSeconds
}

func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func LogDebug(format string, args ...interface{}) {
	msg := formatMessage(format, args...)
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	_, err := fmt.Fprintf(colorableStdout, "%s%s%s %s[DEBUG]%s %s\n",
		ColorBrightBlue, timestamp, ColorClear,
		StyleDim, ColorClear, msg)
	if err != nil {
		return
	}
}

func LogInfo(format string, args ...interface{}) {
	msg := formatMessage(format, args...)
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	_, err := fmt.Fprintf(colorableStdout, "%s%s%s %s[INFO]%s %s\n",
		ColorBrightBlue, timestamp, ColorClear,
		ColorGreen, ColorClear, msg)
	if err != nil {
		return
	}
}

func LogWarn(format string, args ...interface{}) {
	msg := formatMessage(format, args...)
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	_, err := fmt.Fprintf(colorableStdout, "%s%s%s %s[WARN]%s %s\n",
		ColorBrightBlue, timestamp, ColorClear,
		ColorYellow, ColorClear, msg)
	if err != nil {
		return
	}
}

func LogError(format string, args ...interface{}) {
	msg := formatMessage(format, args...)
	timestamp := time.Now().Format("2006/01/02 15:04:05")

	_, err := fmt.Fprintf(colorableStderr, "%s%s%s %s[ERROR]%s %s\n",
		ColorBrightBlue, timestamp, ColorClear,
		ColorRed, ColorClear, msg)
	if err != nil {
		return
	}
}
