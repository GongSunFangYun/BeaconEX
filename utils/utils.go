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
	"sync"
	"time"
)

var (
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

type TermCaps struct {
	EnterAltScreen string
	LeaveAltScreen string
	HideCursor     string
	ShowCursor     string
	EnableMouse    string
	DisableMouse   string
	AltScreenOK    bool
	MouseOK        bool
	FullSGR        bool
	BasicSGR       bool
}

var (
	TC     TermCaps
	tcOnce sync.Once
)

func InitTermCap() {
	tcOnce.Do(func() {
		TC = detectTermCaps()
		applyToGlobals(TC)
	})
}

func init() {
	InitTermCap()
}

func detectTermCaps() TermCaps {
	var c TermCaps

	term := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	colorterm := strings.ToLower(strings.TrimSpace(os.Getenv("COLORTERM")))
	noColor := os.Getenv("NO_COLOR") != ""
	termProg := strings.ToLower(strings.TrimSpace(os.Getenv("TERM_PROGRAM")))
	isTermux := os.Getenv("TERMUX_VERSION") != "" || strings.Contains(os.Getenv("PREFIX"), "termux")

	if term == "dumb" || noColor {
		return c
	}

	forceNoPrivate := os.Getenv("BEX_NO_ALTSCREEN") == "1"

	windowsVTOK := tryEnableWindowsVT()

	if !windowsVTOK {
		c.BasicSGR = true
		return c
	}

	isTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	if isTTY {
		if isTermux {
			c.FullSGR = true
			c.BasicSGR = true
		}

		if term == "" && windowsVTOK {
			c.FullSGR = true
			c.BasicSGR = true
		}

		if strings.Contains(term, "256color") || strings.Contains(term, "truecolor") ||
			colorterm == "truecolor" || colorterm == "24bit" {
			c.FullSGR = true
			c.BasicSGR = true
		}

		if !c.FullSGR {
			for _, t := range []string{
				"xterm", "rxvt", "screen", "tmux", "linux",
				"konsole", "gnome", "vte", "alacritty", "kitty",
				"wezterm", "iterm", "st", "foot",
			} {
				if strings.HasPrefix(term, t) || strings.Contains(termProg, t) {
					c.FullSGR = true
					c.BasicSGR = true
					break
				}
			}
		}

		if !c.BasicSGR {
			c.BasicSGR = true
		}
	}

	if forceNoPrivate {
		return c
	}

	privateOK := false

	if term == "" && windowsVTOK {
		privateOK = true
	}

	if termProg != "" {
		privateOK = true
	}

	if isTermux {
		privateOK = true
	}

	if colorterm == "truecolor" || colorterm == "24bit" {
		privateOK = true
	}

	if !privateOK {
		for _, t := range []string{
			"xterm-256color", "xterm-direct",
			"rxvt-unicode", "rxvt-unicode-256color",
			"screen", "screen-256color",
			"tmux", "tmux-256color",
			"alacritty", "kitty", "foot", "st-256color",
			"linux",
		} {
			if term == t || strings.HasPrefix(term, t+"-") {
				privateOK = true
				break
			}
		}
	}

	if privateOK {
		c.AltScreenOK = true
		c.EnterAltScreen = "\x1b[?1049h"
		c.LeaveAltScreen = "\x1b[?1049l"
		c.HideCursor = "\x1b[?25l"
		c.ShowCursor = "\x1b[?25h"
		c.EnableMouse = "\x1b[?1000h\x1b[?1006h"
		c.DisableMouse = "\x1b[?1000l\x1b[?1006l"
		c.MouseOK = true
	}

	return c
}

func applyToGlobals(c TermCaps) {
	if c.FullSGR {
		return
	}

	if c.BasicSGR {
		StyleDim = ""
		StyleItalic = ""
		ColorBrightBlack = ColorBlack
		ColorBrightRed = ColorRed
		ColorBrightGreen = ColorGreen
		ColorBrightYellow = ColorYellow
		ColorBrightBlue = ColorBlue
		ColorBrightPurple = ColorPurple
		ColorBrightCyan = ColorCyan
		ColorBrightWhite = ColorWhite
		return
	}

	ColorClear = ""
	StyleBold = ""
	StyleDim = ""
	StyleItalic = ""
	StyleUnderline = ""
	ColorBlack = ""
	ColorRed = ""
	ColorGreen = ""
	ColorYellow = ""
	ColorBlue = ""
	ColorPurple = ""
	ColorCyan = ""
	ColorWhite = ""
	ColorBrightBlack = ""
	ColorBrightRed = ""
	ColorBrightGreen = ""
	ColorBrightYellow = ""
	ColorBrightBlue = ""
	ColorBrightPurple = ""
	ColorBrightCyan = ""
	ColorBrightWhite = ""
}

func (tc *TermCaps) Reverse() string {
	if tc.BasicSGR || tc.FullSGR {
		return "\x1b[7m"
	}
	return StyleBold
}

func (tc *TermCaps) FillReverse() string {
	if tc.AltScreenOK && (tc.BasicSGR || tc.FullSGR) {
		return "\x1b[7m\x1b[K" + ColorClear
	}
	return ""
}

func Colorize(text string, colorCode string) string {
	if colorCode == "" {
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
