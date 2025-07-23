"""------------------------外部库------------------------"""
import codecs
import sys
import chardet
from openai import OpenAI
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_error, output
# # 使用第三方人工智能API（阿里云）进行日志分析
DEFAULT_API_CONFIG = {
    "api_key": "sk-11EU9DH9SHD2OEWDIOSANCSKOFNONEOWQPDSA3",
    "api_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model": "qwen-plus"
}

#日志分析类
# noinspection PyTypeChecker
class LogAnalyzer:
    def __init__(self, api_config=None):
        config = api_config or DEFAULT_API_CONFIG
        self.client = OpenAI(
            api_key=config["api_key"],
            base_url=config["api_url"],
        )
        self.model_name = config["model"]

    # 向人工智能提出要求，并返回分析结果
    def Analysis(self, log_content):
        try:
            completion = self.client.chat.completions.create(
                model=self.model_name,
                messages=[
                    {'role': 'user',
                     'content': f'进行日志分析，然后只给出错误原因和对应的解决方案，以及具体操作步骤，确保足够精简的同时让Java零基础者也能了解错误原因。'
                                f'我要把内容输出到powershell上，所以请你不要使用markdown格式。'
                                f'你的每一句话都不要超过40字或者你的每一个长句带句号了就另起一行。'
                                f'而且，给出错误原因和解决方案之后,只空一行。再给出具体操作步骤（列出1，2，3，4这种），这样更加清晰。'
                                f'如果有多个需要解决的问题，请一一列举出来。'
                                f'\n日志内容：\n{log_content}'}
                ]
            )
            return completion.choices[0].message.content
        except Exception as e:
            log_error(f"日志分析失败: {str(e)}")
            return None

# 加载日志文件，并提取关键词进行专项分析
def LoadLogFile(file_path):
    try:
        with open(file_path, 'rb') as f:
            rawdata = f.read()
            result = chardet.detect(rawdata)
            encoding = result['encoding']

        with codecs.open(file_path, 'r', encoding=encoding) as f:
            log_content = f.read()

        error_lines = []
        is_error_block = False
        # 为了防止日志过长，只提取异常行
        for line in log_content.split('\n'):
            if any(keyword in line for keyword in ['ERROR', 'FATAL','WARN', 'Caused by', 'Exception', '错误', '致命','警告']): #只提取错误行
                is_error_block = True
                error_lines.append(line)
            elif is_error_block and line.strip().startswith('at '):
                error_lines.append(line)
            elif is_error_block and not line.strip():
                is_error_block = False
            elif is_error_block and 'WARN' in line:
                is_error_block = False

        return '\n'.join(error_lines) if error_lines else log_content[:2000]  # 没问题就返回前2000的字符

    except Exception as e:
        log_error(f"无法加载日志文件: {str(e)}")
        return None

# 处理异常，提示并返回结果部分
def PerformLogAnalysis(args):
    if not args.log_path:
        log_error("必须指定日志文件路径")
        sys.exit(1)

    log_content = LoadLogFile(args.log_path)
    if not log_content:
        sys.exit(1)

    log_info(f"正在分析日志文件: {args.log_path}")

    analyzer = LogAnalyzer()
    result = analyzer.Analysis(log_content)

    if result:
        output("\n" + "=" * 50 + F" {TextFormat.GREEN}分析结果{TextFormat.CLEAR} " + "=" * 50)
        output(result)
        output("=" * 110 + "\n")
    else:
        log_error("日志分析失败")
