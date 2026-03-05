package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bex/utils"
	"github.com/sashabaranov/go-openai"
)

const (
	defaultAPIKey = ""
	defaultAPIURL = ""
	defaultModel  = ""
)

func LaunchBat(
	request string,
	outputDir string,
) {
	generator := &LaunchBatInstance{
		Request:   request,
		OutputDir: outputDir,
	}

	generator.Execute()
}

type LaunchBatInstance struct {
	Request   string
	OutputDir string
}

func (l *LaunchBatInstance) Execute() {
	utils.LogInfo("正在生成启动脚本...")

	if !l.validateArgs() {
		return
	}

	if l.OutputDir == "" {
		l.OutputDir, _ = os.Getwd()
	} else {
		err := os.MkdirAll(l.OutputDir, 0755)
		if err != nil {
			utils.LogError("无法创建输出目录: %s", l.OutputDir)
			return
		}
	}

	script, err := l.generateScriptWithAI(l.Request)
	if err != nil {
		utils.LogError("生成失败: %s", err)
		return
	}

	script = l.cleanScript(script)

	filename := "start.bat"
	if strings.Contains(script, "#!/bin/bash") || strings.Contains(script, "#!/bin/sh") {
		filename = "start.sh"
	}

	outputPath := filepath.Join(l.OutputDir, filename)
	err = os.WriteFile(outputPath, []byte(script), 0644)
	if err != nil {
		utils.LogError("写入文件失败: %s", err)
		return
	}

	absPath, _ := filepath.Abs(outputPath)

	utils.LogInfo("脚本路径为：%s", absPath)

	fmt.Printf("\n%s✓ 脚本已生成%s\n", utils.ColorGreen, utils.ColorClear)
	fmt.Printf("%s内容预览:%s\n", utils.ColorCyan, utils.ColorClear)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(l.highlightScript(script))
	fmt.Println(strings.Repeat("-", 60))

	utils.LogInfo("启动脚本生成完成！")
}

func (l *LaunchBatInstance) validateArgs() bool {
	if l.Request == "" {
		utils.LogError("生成要求不能为空")
		return false
	}
	return true
}

func (l *LaunchBatInstance) generateScriptWithAI(request string) (string, error) {
	config := openai.DefaultConfig(defaultAPIKey)
	config.BaseURL = defaultAPIURL
	client := openai.NewClientWithConfig(config)

	prompt := fmt.Sprintf(`你是一个专业的Minecraft服务器管理员。请根据需求生成服务器启动脚本：

需求：%s

要求：
1. 根据用户的操作系统类型（Windows/Linux/macOS）生成对应的启动脚本
2. 脚本命名规则：
   - Windows: start.bat 或 start.cmd
   - Linux/macOS: start.sh
3. 自动判断服务器类型（原版/Paper/Spigot/Forge/Fabric/Bukkit）
4. 按照用户需求进行内存分配，默认则不进行分配
5. 按照用户需求添加JVM优化参数（如G1GC、Aikar flags）
6. 必须包含必要的脚本要素：
   - Windows: @echo off 和 pause 命令
   - Linux/macOS: #!/bin/bash 或 #!/bin/sh 和 read -p 命令
7. 只返回代码，不要任何解释
8. 确保生成的脚本是纯粹可执行的，无多余内容
9. 如果用户未指定操作系统，默认生成Windows和Linux双版本并用注释标注

Windows示例：
@echo off
java -Xms1G -Xmx8G -XX:+UseG1GC -jar server.jar nogui
pause

Linux/macOS示例：
#!/bin/bash
java -Xms1G -Xmx8G -XX:+UseG1GC -jar server.jar nogui
read -p "Press any key to continue..."`, request)

	resp, err := client.CreateChatCompletion(
		context.Background(), // context
		openai.ChatCompletionRequest{
			Model: defaultModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "你专注生成高度优化的Minecraft启动脚本",
				},
				{
					Role:    "user",
					Content: prompt,
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("API 调用失败: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("API 返回空结果")
	}

	return resp.Choices[0].Message.Content, nil
}

func (l *LaunchBatInstance) cleanScript(script string) string {
	script = strings.ReplaceAll(script, "```bat", "")
	script = strings.ReplaceAll(script, "```", "")
	script = strings.TrimSpace(script)
	script = strings.ReplaceAll(script, "\r\n", "\n")

	return script
}

func (l *LaunchBatInstance) highlightScript(script string) string {
	lines := strings.Split(script, "\n")
	var highlighted []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@echo off") {
			highlighted = append(highlighted, fmt.Sprintf("%s%s%s",
				utils.ColorBrightYellow, line, utils.ColorClear))
		} else if strings.HasPrefix(line, "java") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				highlighted = append(highlighted, fmt.Sprintf("%s%s%s %s",
					utils.ColorGreen, parts[0], utils.ColorClear, parts[1]))
			} else {
				highlighted = append(highlighted, fmt.Sprintf("%s%s%s",
					utils.ColorGreen, line, utils.ColorClear))
			}
		} else if strings.HasPrefix(line, "pause") {
			highlighted = append(highlighted, fmt.Sprintf("%s%s%s",
				utils.ColorCyan, line, utils.ColorClear))
		} else {
			highlighted = append(highlighted, line)
		}
	}

	return strings.Join(highlighted, "\n")
}
