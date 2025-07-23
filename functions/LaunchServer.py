"""------------------------外部库------------------------"""
import openai
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_error, output, log_info
# 使用第三方人工智能API（阿里云）进行智能脚本生成
DEFAULT_API_CONFIG = {
    "api_key": "sk-11EU9DH9SHD2OEWDIOSANCSKOFNONEOWQPDSA3",
    "api_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model": "qwen-plus"
}
# 该函数用于语法高亮
def HighLight(script: str) -> str:
    lines = []
    for line in script.split('\n'):
        if line.startswith('@echo'):
            line = f"{TextFormat.YELLOW}{line}{TextFormat.CLEAR}"
        elif line.startswith('java'):
            parts = line.split(' ')
            highlighted = [f"{TextFormat.YELLOW}java{TextFormat.CLEAR}"]
            for part in parts[1:]:
                if part.startswith('-'):
                    highlighted.append(f"{TextFormat.BLUE}{part}{TextFormat.CLEAR}")
                else:
                    highlighted.append(part)
            line = ' '.join(highlighted)
        elif line.strip() == 'pause':
            line = f"{TextFormat.CLEAR}{line}"
        lines.append(line)
    return '\n'.join(lines)


# noinspection PyTypeChecker
# 通过请求人工智能API生成Minecraft 服务器启动脚本
def GenerateLaunchBat(request: str):
    log_info("正在生成启动脚本...")
    try:
        # 配置OpenAI客户端
        client = openai.OpenAI(
            api_key=DEFAULT_API_CONFIG["api_key"],
            base_url=DEFAULT_API_CONFIG["api_url"]
        )
        prompt = f"""你是一个专业的Minecraft服务器管理员。请根据需求生成Windows批处理脚本(start.bat)：

需求：{request}

要求：
1. 不要添加任何其他乱七八糟的批处理语法，将要求硬编码到java指令中
2. 自动判断服务器类型（原版/Paper/Spigot/Forge/Fabric）
3. 按照用户需求进行内存分配，默认则不进行分配
4. 按照用户需求添加JVM优化参数（如G1GC、Aikar flags）
5. 必须包含`@echo off`和`pause`命令
6. 只返回代码，不要任何解释
7. 纯粹的java指令，无其他任何内容
8. 每次请求都使用最新的请求生成脚本，而不是继承之前的记忆

示例格式：
@echo off
java -Xms1G -Xmx8G -jar server.jar nogui
pause
```"""

        # 调用API
        response = client.chat.completions.create(
            model=DEFAULT_API_CONFIG["model"],
            messages=[
                {"role": "system", "content": "你专注生成高度优化的Minecraft启动脚本"},
                {"role": "user", "content": prompt}
            ],
        )

        # 提取并清理脚本
        script = response.choices[0].message.content
        script = script.replace("```bat", "").replace("```", "").strip()

        # 写入文件（原始内容）
        with open("start.bat", "w", encoding="utf-8") as f:
            f.write(script)
        print(f"脚本路径为：{f}")

        # 语法高亮输出结果
        highlighted_script = HighLight(script)
        output(f"\n{TextFormat.GREEN}✓ 脚本已生成 {TextFormat.CLEAR}")
        output(f"{TextFormat.CYAN}内容预览:{TextFormat.CLEAR}")
        output("-" * 50)
        output(highlighted_script)
        output("-" * 50)

    except Exception as e:
        log_error(f"生成失败: {str(e)}")
