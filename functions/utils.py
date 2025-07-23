"""------------------------外部库------------------------"""
import functools
import socket
import sys
import time

output = functools.partial(print, flush=True)
# 该类被整个程序所调用，主要是用来格式化输出日志，同时解析Minecraft特有的颜色代码（分节符）
class TextFormat: # 基本颜色代码，最后输出是转换为ANSI颜色代码
    RED = '\033[91m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    PURPLE = '\033[95m'
    CYAN = '\033[96m'
    CLEAR = '\033[0m'
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
    return time.strftime("%H:%M:%S", time.localtime())

# 初始化日志级别
def log_info(message):
    timestamp = get_timestamp()
    output(f"{TextFormat.YELLOW}{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} {message}")

def log_warn(message):
    timestamp = get_timestamp()
    output(f"{TextFormat.YELLOW}{timestamp} {TextFormat.YELLOW}[Application Thread/WARN]{TextFormat.CLEAR} {message}")

def log_error(message):
    timestamp = get_timestamp()
    output(f"{TextFormat.YELLOW}{timestamp} {TextFormat.RED}[Application Thread/ERROR]{TextFormat.CLEAR} {message}", file=sys.stderr)

def log_debug(message):
    timestamp = get_timestamp()
    output(f"{TextFormat.YELLOW}{timestamp} {TextFormat.BLUE}[Application Thread/DEBUG]{TextFormat.CLEAR} {message}")

def ResolveDomainName(host):
    try:
        socket.inet_aton(host)  # 检查是否是有效的IP地址
        return host
    except socket.error:
        try:
            ip = socket.gethostbyname(host)  # 解析域名
            log_debug(f"解析主机名：{host}")
            return ip
        except socket.gaierror as e:
            log_error(f"无法解析主机名: {host} (错误: {e})")
            sys.exit(1)