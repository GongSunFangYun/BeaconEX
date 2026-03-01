package modules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bex/utils"
	"github.com/disintegration/imaging"
)

func MakeIcon(
	picturePath string,
	outputDir string,
	pictureName string,
) {
	generator := &MakeIconInstance{
		PicturePath: picturePath,
		OutputDir:   outputDir,
		PictureName: pictureName,
	}

	generator.Execute()
}

type MakeIconInstance struct {
	PicturePath string
	OutputDir   string
	PictureName string
}

func (m *MakeIconInstance) Execute() {
	utils.LogInfo("正在处理图片: %s", m.PicturePath)

	if !m.validateArgs() {
		return
	}

	if m.PictureName == "" {
		m.PictureName = "server-icon.png"
	}

	if !strings.HasSuffix(strings.ToLower(m.PictureName), ".png") {
		m.PictureName += ".png"
	}

	if m.OutputDir != "" {
		err := os.MkdirAll(m.OutputDir, 0755)
		if err != nil {
			utils.LogError("无法创建输出目录: %s", m.OutputDir)
			return
		}
	}

	fileInfo, err := os.Stat(m.PicturePath)
	if err != nil {
		utils.LogError("无法读取文件信息: %s", err)
		return
	}
	fileSize := fileInfo.Size()
	utils.LogInfo("原始图片尺寸: 读取中... | 格式: %s | 大小: %s",
		strings.ToUpper(strings.TrimPrefix(filepath.Ext(m.PicturePath), ".")),
		utils.FormatFileSize(fileSize))

	srcImage, err := imaging.Open(m.PicturePath)
	if err != nil {
		utils.LogError("无法打开图片: %s", err)
		return
	}

	srcBounds := srcImage.Bounds()
	originalWidth := srcBounds.Dx()
	originalHeight := srcBounds.Dy()
	utils.LogInfo("原始图片尺寸: %dx%d 像素", originalWidth, originalHeight)

	utils.LogInfo("正在调整图片尺寸为 64x64...")
	dstImage := imaging.Fit(srcImage, 64, 64, imaging.Lanczos)

	var outputPath string
	if m.OutputDir != "" {
		outputPath = filepath.Join(m.OutputDir, m.PictureName)
	} else {
		outputPath = filepath.Join(".", m.PictureName)
	}

	err = imaging.Save(dstImage, outputPath)
	if err != nil {
		utils.LogError("保存图片失败: %s", err)
		return
	}

	outputInfo, err := os.Stat(outputPath)
	if err == nil {
		outputSize := outputInfo.Size()
		absPath, _ := filepath.Abs(outputPath)

		utils.LogInfo("%s✓ 图片已保存: %s%s",
			utils.ColorGreen, utils.ColorClear, absPath)
		utils.LogInfo("文件大小: %s", utils.FormatFileSize(outputSize))

		if fileSize > 0 {
			ratio := float64(outputSize) / float64(fileSize) * 100
			utils.LogInfo("压缩率: %.1f%% (%.2f KB → %.2f KB)",
				ratio, float64(fileSize)/1024, float64(outputSize)/1024)
		}
	}

	m.showPreview(dstImage.Bounds().Dx(), dstImage.Bounds().Dy())

	utils.LogInfo("服务器图标处理完成！")
}

func (m *MakeIconInstance) validateArgs() bool {
	if !utils.FileExists(m.PicturePath) {
		utils.LogError("图片文件不存在: %s", m.PicturePath)
		return false
	}

	ext := strings.ToLower(filepath.Ext(m.PicturePath))
	supportedExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".bmp": true, ".gif": true,
	}
	if !supportedExts[ext] {
		utils.LogError("不支持的文件格式，请使用 PNG、JPG、BMP 或 GIF 格式")
		return false
	}

	return true
}

func (m *MakeIconInstance) showPreview(width, height int) {
	fmt.Printf("\n%s图片预览信息:%s\n", utils.ColorCyan, utils.ColorClear)
	fmt.Println(strings.Repeat("─", 40))

	fmt.Println("生成的图标 (64x64):")

	for y := 0; y < 8; y++ {
		fmt.Print("  ")
		for x := 0; x < 8; x++ {
			if (x+y)%2 == 0 {
				fmt.Printf("%s██%s", utils.ColorBrightYellow, utils.ColorClear)
			} else {
				fmt.Printf("%s░░%s", utils.ColorBrightBlack, utils.ColorClear)
			}
		}
		fmt.Println()
	}

	fmt.Printf("\n%s规格信息:%s\n", utils.ColorGreen, utils.ColorClear)
	fmt.Printf("  • 尺寸: %dx%d 像素 (Minecraft服务器标准)\n", width, height)
	fmt.Printf("  • 格式: PNG (支持透明通道)\n")
	fmt.Printf("  • 用途: 作为服务器图标放置于服务器根目录\n")

	if m.OutputDir == "" {
		fmt.Printf("  • 提示: 未指定输出目录，图片保存在当前工作目录\n")
	} else {
		fmt.Printf("  • 输出目录: %s\n", m.OutputDir)
	}

	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("%s使用方法:%s 将生成的 %s 文件复制到服务器根目录\n",
		utils.ColorBrightYellow, utils.ColorClear, m.PictureName)
}
