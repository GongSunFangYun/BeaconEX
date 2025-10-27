"""------------------------外部库------------------------"""
import functools
import sys
import time

output = functools.partial(print, flush=True)
# 该类被整个程序所调用，主要是用来格式化输出日志，同时解析Minecraft特有的颜色代码（分节符）
class TextFormat: # 基本颜色代码，最后输出是转换为ANSI颜色代码
    CLEAR = '\033[0m'

    # 文本样式
    BOLD = '\033[1m'  # 粗体（加亮）
    DIM = '\033[2m'  # 暗淡（降低亮度）
    ITALIC = '\033[3m'  # 斜体（并非所有终端支持）
    UNDERLINE = '\033[4m'  # 下划线
    REVERSE = '\033[7m'  # 反色（前景色和背景色互换）
    HIDDEN = '\033[8m'  # 隐藏（隐形文字）

    # 普通颜色（深色/标准色）- 前景色
    BLACK = '\033[30m'  # 黑色
    RED = '\033[31m'  # 深红色
    GREEN = '\033[32m'  # 深绿色
    YELLOW = '\033[33m'  # 深黄色（棕色）
    BLUE = '\033[34m'  # 深蓝色
    PURPLE = '\033[35m'  # 深紫色
    CYAN = '\033[36m'  # 深青色
    WHITE = '\033[37m'  # 深白色（浅灰色）

    # 亮色（浅色/高亮色）- 前景色（更鲜艳明亮）
    BRIGHT_BLACK = '\033[90m'  # 亮黑色（深灰色）
    BRIGHT_RED = '\033[91m'  # 亮红色（鲜红色）
    BRIGHT_GREEN = '\033[92m'  # 亮绿色（鲜绿色）
    BRIGHT_YELLOW = '\033[93m'  # 亮黄色（鲜黄色）
    BRIGHT_BLUE = '\033[94m'  # 亮蓝色（鲜蓝色）
    BRIGHT_PURPLE = '\033[95m'  # 亮紫色（鲜紫色）
    BRIGHT_CYAN = '\033[96m'  # 亮青色（鲜青色）
    BRIGHT_WHITE = '\033[97m'  # 亮白色（纯白色）

    # 解析Minecraft特有的分节符颜色代码，并转换为ANSI颜色代码
    _MC_COLOR_MAP = {
        '0': '\033[30m',  # 黑色
        '1': '\033[34m',  # 深蓝
        '2': '\033[32m',  # 深绿
        '3': '\033[36m',  # 青色
        '4': '\033[31m',  # 深红
        '5': '\033[35m',  # 紫色
        '6': '\033[33m',  # 金色
        '7': '\033[37m',  # 灰色
        '8': '\033[90m',  # 深灰
        '9': '\033[94m',  # 蓝色
        'a': '\033[92m',  # 绿色
        'b': '\033[96m',  # 天蓝
        'c': '\033[91m',  # 红色
        'd': '\033[95m',  # 粉红
        'e': '\033[93m',  # 黄色
        'f': '\033[97m',  # 白色
        'l': '\033[1m',  # 粗体
        'm': '\033[9m',  # 删除线
        'n': '\033[4m',  # 下划线
        'o': '\033[3m',  # 斜体
        'r': '\033[0m',  # 重置
    }
    # 规范输出格式
    @staticmethod
    def Colorize(text: str, color_code: str) -> str:
        return f"{color_code}{text}{TextFormat.CLEAR}"

    # 处理服务器Motd信息
    @staticmethod
    def ParseMotd(motd_obj) -> str:
        if not hasattr(motd_obj, 'raw'):
            return str(motd_obj)
        motd_text = motd_obj.raw if motd_obj.raw else ' '.join(motd_obj.parsed)
        return TextFormat._MCColourToAnsi(motd_text)

    # 解析Minecraft特有的颜色代码
    @staticmethod
    def _MCColourToAnsi(mc_text: str) -> str:
        if hasattr(mc_text, '__str__'):
            mc_text = str(mc_text)
        ansi_text = []
        i = 0
        while i < len(mc_text):
            if mc_text[i] == '§' and i + 1 < len(mc_text):
                code = mc_text[i + 1].lower()
                ansi_text.append(TextFormat._MC_COLOR_MAP.get(code, ''))
                i += 2
            else:
                ansi_text.append(mc_text[i])
                i += 1
        return ''.join(ansi_text)

# 获取当前时间的格式化字符串
def get_timestamp():
    return time.strftime(f"{TextFormat.BRIGHT_BLUE}%Y-%m-%d{TextFormat.CLEAR} {TextFormat.BRIGHT_YELLOW}%H:%M:%S{TextFormat.CLEAR}", time.localtime())

# 初始化日志级别
def log_debug(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.DIM}[Application Thread/DEBUG]{TextFormat.CLEAR} {message}")

def log_info(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} {message}")

def log_warn(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.YELLOW}[Application Thread/WARN]{TextFormat.CLEAR} {message}")

def log_error(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.RED}[Application Thread/ERROR]{TextFormat.CLEAR} {message}", file=sys.stderr)

def et_log_debug(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.DIM}[EasyTier Thread/DEBUG]{TextFormat.CLEAR} {message}")

def et_log_info(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.GREEN}[EasyTier Thread/INFO]{TextFormat.CLEAR} {message}")

def et_log_warn(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.YELLOW}[EasyTier Thread/WARN]{TextFormat.CLEAR} {message}")

def et_log_error(message):
    timestamp = get_timestamp()
    output(f"{timestamp} {TextFormat.RED}[EasyTier Thread/ERROR]{TextFormat.CLEAR} {message}", file=sys.stderr)