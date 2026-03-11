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
	defaultAPIURL = "https://api-inference.modelscope.cn/v1"
	defaultModel  = "deepseek-ai/DeepSeek-V3.2"
)

const APIKey = "ms-otto-mom"

func resolveAPIKey() (string, error) {
	return APIKey, nil
}

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
	utils.LogInfo(strings.Repeat("=", 40))
	utils.LogInfo("正在生成启动脚本...(这可能需要一段时间)")
	utils.LogInfo(strings.Repeat("=", 40))

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

	scripts := l.splitScripts(script)
	if len(scripts) == 0 {
		utils.LogError("未能从 AI 输出中解析出任何脚本")
		return
	}

	for _, s := range scripts {
		outputPath := filepath.Join(l.OutputDir, s.filename)
		err = os.WriteFile(outputPath, []byte(s.content), 0644)
		if err != nil {
			utils.LogError("写入文件失败 [%s]: %s", s.filename, err)
			continue
		}

		_, _ = filepath.Abs(outputPath)

		utils.LogInfo("%s已生成脚本 %s%s%s", utils.ColorGreen, utils.ColorBrightYellow, s.filename, utils.ColorClear)
		utils.LogInfo("%s内容预览:%s", utils.ColorCyan, utils.ColorClear)
		fmt.Println("")
		fmt.Println(s.content)
		fmt.Println("")
	}
}

func (l *LaunchBatInstance) validateArgs() bool {
	if l.Request == "" {
		utils.LogError("生成要求不能为空")
		return false
	}
	return true
}

func (l *LaunchBatInstance) generateScriptWithAI(request string) (string, error) {
	apiKey, err := resolveAPIKey()
	if err != nil {
		return "", err
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = defaultAPIURL
	client := openai.NewClientWithConfig(config)

	prompt := fmt.Sprintf(`你是一个专业的Minecraft服务器管理员。请根据需求生成服务器启动脚本：

需求：%s

要求：
1. 根据用户的操作系统类型（Windows/Linux/macOS）生成对应的启动脚本
2. 脚本命名规则：
   - Windows: start.bat
   - Linux/macOS: start.sh
3. 自动判断服务器类型（原版/Paper/Spigot/Forge/Fabric/Bukkit）
4. 按照用户需求进行内存分配，默认则不进行分配
5. 按照用户需求添加JVM优化参数（如G1GC、Aikar flags）
6. 【重要】每个脚本必须包含固定的首行标记和结尾命令，不得省略：
   - Windows bat脚本：首行必须是 ":: !/bin/batch"，结尾必须有 pause 命令
   - Linux/macOS sh脚本：首行必须是 "#!/bin/bash"，结尾必须有 read 命令
7. 只返回代码，不要任何解释、注释说明或 Markdown 代码块标记
8. 确保生成的脚本是纯粹可执行的，无多余内容
9. 如果用户未指定操作系统，必须同时生成 Windows 和 Linux 两份完整脚本，两份脚本直接相邻输出，中间不加任何分隔符或说明文字

Windows示例（首行标记必须完整保留）：
::!/bin/batch
@echo off
java -Xms1G -Xmx8G -XX:+UseG1GC -jar server.jar nogui
pause

Linux/macOS示例（shebang必须完整保留）：
#!/bin/bash
java -Xms1G -Xmx8G -XX:+UseG1GC -jar server.jar nogui
read -p "Press any key to continue..."`, request)

	resp, err := client.CreateChatCompletion(
		context.Background(),
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
	for _, tag := range []string{"```batch", "```bat", "```cmd", "```bash", "```sh", "```shell", "```"} {
		script = strings.ReplaceAll(script, tag, "")
	}
	script = strings.TrimSpace(script)
	script = strings.ReplaceAll(script, "\r\n", "\n")
	return script
}

type scriptResult struct {
	filename string
	content  string
}

func (l *LaunchBatInstance) splitScripts(raw string) []scriptResult {
	const (
		batMarker = "::!/bin/batch"
		shMarker1 = "#!/bin/bash"
		shMarker2 = "#!/bin/sh"
	)

	lines := strings.Split(raw, "\n")

	type boundary struct {
		start int
		isBat bool
	}
	var bounds []boundary

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == batMarker {
			bounds = append(bounds, boundary{start: i, isBat: true})
		} else if trimmed == shMarker1 || trimmed == shMarker2 {
			bounds = append(bounds, boundary{start: i, isBat: false})
		}
	}

	if len(bounds) == 0 {
		hasBat := strings.Contains(raw, "pause")
		hasSh := strings.Contains(raw, "read ")
		switch {
		case hasBat && !hasSh:
			return []scriptResult{{filename: "start.bat", content: raw}}
		case hasSh && !hasBat:
			return []scriptResult{{filename: "start.sh", content: raw}}
		default:
			return []scriptResult{{filename: "start.txt", content: raw}}
		}
	}

	var results []scriptResult
	for idx, b := range bounds {
		end := len(lines)
		if idx+1 < len(bounds) {
			end = bounds[idx+1].start
		}

		block := strings.TrimSpace(strings.Join(lines[b.start:end], "\n"))
		if block == "" {
			continue
		}

		if b.isBat {
			results = append(results, scriptResult{filename: "start.bat", content: block})
		} else {
			results = append(results, scriptResult{filename: "start.sh", content: block})
		}
	}

	return results
}
