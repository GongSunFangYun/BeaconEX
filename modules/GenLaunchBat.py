"""------------------------外部库------------------------"""
import openai
import os
import argparse
import sys
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat, log_error, output, log_info

# 使用第三方人工智能API（阿里云）进行智能脚本生成
# 孩子们别学我，硬编码API-KEY真的很危险（）
# 上次还推送到仓库了吓死我了
# 真正搞建议用pyenv传递环境变量，不要硬编码API-KEY！！！！！！！！！1
DEFAULT_API_CONFIG = {
    "api_key": "API-KEY",
    "api_url": "API-URL",
    "model": "qwen-plus"
}

# noinspection PyTypeChecker
# 通过请求人工智能API生成Minecraft 服务器启动脚本
def GenerateLaunchBat(request: str, output_dir=None):
    log_info("正在生成启动脚本...")
    try:
        # 处理输出目录
        if output_dir is None:
            output_dir = os.getcwd()
        elif not os.path.isdir(output_dir):
            os.makedirs(output_dir, exist_ok=True)

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
        output_path = os.path.join(output_dir, "start.bat")
        with open(output_path, "w", encoding="utf-8") as f:
            f.write(script)
            abs_path = os.path.abspath(f.name)  # 获取绝对路径
            log_info(f"脚本路径为：{abs_path}")

        # 语法高亮输出结果
        output(f"\n{TextFormat.GREEN}✓ 脚本已生成 {TextFormat.CLEAR}")
        output(f"{TextFormat.CYAN}内容预览:{TextFormat.CLEAR}")
        output("-" * 60)
        output(script)
        output("-" * 60)

    except Exception as e:
        log_error(f"生成失败: {str(e)}")

def main():
    parser = argparse.ArgumentParser(
        description='生成Minecraft服务器启动脚本',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-rq', '--request', required=True,
                       help='生成要求，例如："1.20.1原版服务器，4G内存"')

    # 可选参数
    parser.add_argument('-od', '--output-dir',
                       help='指定输出目录 (默认: 当前目录)')
    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 执行脚本生成
        GenerateLaunchBat(
            request=args.request,
            output_dir=args.output_dir
        )

        log_info("启动脚本生成完成！")

    except Exception as e:
        log_error(f"启动脚本生成失败: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()