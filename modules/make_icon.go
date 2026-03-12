package modules

import (
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

	srcImage, err := imaging.Open(m.PicturePath)
	if err != nil {
		utils.LogError("无法打开图片: %s", err)
		return
	}
	srcBounds := srcImage.Bounds()
	absInput, _ := filepath.Abs(m.PicturePath)
	utils.LogInfo("输入文件 %s，格式 %s，尺寸 %dx%d，大小 %s",
		absInput,
		strings.ToUpper(strings.TrimPrefix(filepath.Ext(m.PicturePath), ".")),
		srcBounds.Dx(), srcBounds.Dy(),
		utils.FormatFileSize(fileSize))

	utils.LogInfo("正在处理文件")
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
	if err != nil {
		utils.LogError("获取输出文件信息失败: %s", err)
		return
	}
	outputSize := outputInfo.Size()
	absOutput, _ := filepath.Abs(outputPath)

	ratio := 0.0
	if fileSize > 0 {
		ratio = float64(outputSize) / float64(fileSize) * 100
	}
	utils.LogInfo("文件处理完成，压缩率 %.1f%% (%s 转为 %s)",
		ratio, utils.FormatFileSize(fileSize), utils.FormatFileSize(outputSize))

	utils.LogInfo("文件已保存至 %s", absOutput)
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
