"""------------------------外部库------------------------"""
import argparse
import os
import re
import socket
import sys
import time

from typing import Any
from mcrcon import MCRcon
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat, log_info, log_error, output,log_debug, log_warn
from bexlib2.rhnis import ResolveDomainName
from bexlib2 import lg4pb

def TargetParser(target: str, default_port: int) -> tuple: # 处理要连接的目标
    if ':' in target:
        parts = target.split(':', 1)
        host = parts[0]
        try:
            port = int(parts[1])
            if not (1 <= port <= 65535):
                raise ValueError("端口必须在1-65535之间")
            return host, port
        except ValueError:
            log_error(f"无效的端口号: {parts[1]}")
            sys.exit(1)
    else:
        return target, default_port

# RCON专用的§代码解析
def ParseRCONResponse(response: str) -> str:
    if not response:
        return response

    # Minecraft颜色代码到ANSI的映射
    MC_COLOR_MAP = {
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
        'l': '\033[1m',   # 粗体
        'm': '\033[9m',   # 删除线
        'n': '\033[4m',   # 下划线
        'o': '\033[3m',   # 斜体
        'r': '\033[0m',   # 重置
    }

    ansi_text = []
    i = 0
    while i < len(response):
        if response[i] == '§' and i + 1 < len(response):
            code = response[i + 1].lower()
            ansi_text.append(MC_COLOR_MAP.get(code, ''))
            i += 2
        else:
            ansi_text.append(response[i])
            i += 1

    return ''.join(ansi_text)

# 解析脚本文件（自己整的BEXScript indev 0.0.1）
# noinspection RegExpRedundantEscape
def ParseScriptFile(file_path: str) -> dict[str, dict[Any, Any] | list[Any]] | None:
    if not os.path.exists(file_path):
        log_error(f"脚本文件不存在: {file_path}")
        return None

    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        log_error(f"读取脚本文件失败: {e}")
        return None

    script_data = {
        'login': {},
        'functions': []
    }

    # 解析登录信息
    login_match = re.search(r'LOGIN\s+([^:\s]+):(\d+)', content)
    rcon_match = re.search(r'RCONCFG\s*\[PORT=(\d+),PWD=([^\]]+)\]', content)

    if login_match and rcon_match:
        script_data['login'] = {
            'ip': login_match.group(1),
            'port': int(login_match.group(2)),
            'rcon_port': int(rcon_match.group(1)),
            'password': rcon_match.group(2)
        }
    else:
        log_error("脚本文件中缺少有效的登录信息 (LOGIN 和 RCONCFG 配置)")
        if not login_match:
            log_error("未找到有效的 LOGIN 配置")
        if not rcon_match:
            log_error("未找到有效的 RCONCFG 配置")
        return None

    # 解析功能模块
    def extract_balanced_braces(text, start_pos):
        if start_pos >= len(text) or text[start_pos] != '{':
            return None, start_pos

        brace_count = 0
        i = start_pos
        start_index = -1

        while i < len(text):
            if text[i] == '{':
                brace_count += 1
                if brace_count == 1:
                    start_index = i
            elif text[i] == '}':
                brace_count -= 1
                if brace_count == 0 and start_index != -1:
                    return text[start_index + 1:i], i + 1
            i += 1

        return None, start_pos

    # 查找所有LOOP定义
    loop_pattern = r'LOOP\s+(\w+)\s*\(freq=(\d+)\)\s*\{'
    loop_matches = list(re.finditer(loop_pattern, content))

    for loop_match in loop_matches:
        func_name = loop_match.group(1)
        freq = int(loop_match.group(2))
        start_pos = loop_match.end()

        # 提取循环体内容
        loop_content, end_pos = extract_balanced_braces(content, start_pos - 1)
        if loop_content is None:
            log_warn(f"无法找到循环 {func_name} 的结束花括号")
            continue

        function = {
            'name': func_name,
            'frequency': freq,
            'modules': []
        }

        # 在循环内容中查找EXEC模块
        exec_pattern = r'EXEC\s+(\w+)\s*\{'
        exec_matches = list(re.finditer(exec_pattern, loop_content))

        for exec_match in exec_matches:
            module_name = exec_match.group(1)
            exec_start_pos = exec_match.end()

            # 提取EXEC内容
            exec_content, exec_end_pos = extract_balanced_braces(loop_content, exec_start_pos - 1)
            if exec_content is not None:
                # 清理命令文本
                commands = []
                for line in exec_content.strip().split('\n'):
                    line = line.strip()
                    if line and not line.startswith('#'):  # 跳过空行和注释
                        # 移除可能的引号
                        cmd = line.strip('"\'')
                        commands.append(cmd)

                if commands:
                    function['modules'].append({
                        'name': module_name,
                        'commands': commands,
                        'position': exec_match.start()
                    })

        # 在循环内容中查找WAIT指令
        wait_pattern = r'WAIT\s+(\d+)'
        wait_matches = list(re.finditer(wait_pattern, loop_content))

        for wait_match in wait_matches:
            wait_time = float(wait_match.group(1))
            function['modules'].append({
                'name': f"WAIT_{wait_time}s",
                'type': 'wait',
                'time': wait_time,
                'position': wait_match.start()
            })

        # 按在循环内容中的位置排序
        function['modules'].sort(key=lambda x: x['position'])

        # 移除position字段
        for module in function['modules']:
            if 'position' in module:
                del module['position']

        script_data['functions'].append(function)

    if not script_data['functions']:
        log_warn("脚本文件中未找到任何功能模块")
        # 调试信息
        log_debug(f"找到的LOOP匹配: {len(loop_matches)}")

    return script_data

def DisplayScriptPlan(script_data: dict): # 显示脚本执行计划
    login = script_data['login']
    log_info(f"使用密码 {TextFormat.YELLOW}{login['password']}{TextFormat.CLEAR} 登录至服务器 {TextFormat.CYAN}{login['ip']}:{login['port']}{TextFormat.CLEAR} 的 {TextFormat.CYAN}{login['rcon_port']}{TextFormat.CLEAR} RCON端口...")

    if not script_data['functions']:
        log_info("脚本中没有可执行的功能模块")
        return

    for function in script_data['functions']:
        log_info(
            f"执行循环 {TextFormat.GREEN}{function['name']}{TextFormat.CLEAR}，持续 {TextFormat.YELLOW}{function['frequency']}{TextFormat.CLEAR} 次")

        module_count = len([m for m in function['modules'] if 'type' not in m])
        wait_count = len([m for m in function['modules'] if m.get('type') == 'wait'])

        if module_count == 0 and wait_count == 0:
            log_warn(f"循环 {TextFormat.CYAN}{function['name']}{TextFormat.CLEAR} 中没有找到任何模块或迟滞指令")
            continue

        module_flow = []
        for module in function['modules']:
            if module.get('type') == 'wait':
                module_flow.append(f"迟滞 {module['time']} 秒")
            else:
                module_flow.append(module['name'])

        log_info(
            f"{function['name']}包含 {module_count} 个模块和 {wait_count} 次迟滞：{TextFormat.PURPLE}{' -> '.join(module_flow)}{TextFormat.CLEAR}")

        for module in function['modules']:
            if module.get('type') != 'wait':
                log_info(f"{module['name']}将会执行 {len(module['commands'])} 个命令：")
                for cmd in module['commands']:
                    log_info(f'  "{cmd}"')


def GetTreePrefix(module_idx: int, total_modules: int,cmd_idx: int, total_commands_in_module: int,loop_idx: int, total_loops: int) -> tuple:
    # 计算树形显示的前缀
    # 参数:
    #   module_idx: 当前模块索引
    #   total_modules: 总模块数
    #   cmd_idx: 当前命令索引
    #   total_commands_in_module: 当前模块中的总命令数
    #   loop_idx: 当前循环索引
    #   total_loops: 总循环次数
    # 返回:
    #   tuple: (command_prefix, prefix, result_prefix)
    # 该函数用于格式化输出脚本执行结果
    is_last_module = (module_idx == total_modules - 1)
    is_last_loop = (loop_idx == total_loops - 1)
    is_last_command_in_module = (cmd_idx == total_commands_in_module - 1)
    is_last_module_and_last_loop = is_last_module and is_last_loop

    if total_commands_in_module > 1:
        # 多命令模块
        if is_last_module_and_last_loop:
            # 最后一个模块的所有命令都使用空格前缀
            command_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
            if is_last_command_in_module:
                prefix = "└─"
                result_prefix = "  "
            else:
                prefix = "├─"
                result_prefix = "│ "
        else:
            # 非最后一个模块使用竖线前缀
            command_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
            if is_last_command_in_module and is_last_module:
                prefix = "└─"
                result_prefix = "  "
            else:
                prefix = "├─"
                result_prefix = "│ "
    else:
        # 单命令模块
        command_prefix = f"{TextFormat.CYAN}{TextFormat.CLEAR}"
        if is_last_command_in_module and is_last_module_and_last_loop:
            prefix = "└─"
            result_prefix = "  "
        else:
            prefix = "├─"
            result_prefix = "│ "

    return command_prefix, prefix, result_prefix


def GetWaitPrefix(module_idx: int, total_modules: int,
                    loop_idx: int, total_loops: int) -> tuple:
    # 计算迟滞模块的显示前缀
    # 返回:
    #   tuple: (start_prefix, progress_prefix, completion_prefix)
    #
    is_last_module = (module_idx == total_modules - 1)
    is_last_loop = (loop_idx == total_loops - 1)
    is_last_module_and_last_loop = is_last_module and is_last_loop

    if is_last_module_and_last_loop:
        start_prefix = f"{TextFormat.CYAN}└─{TextFormat.CLEAR}"
        progress_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
        completion_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
    else:
        start_prefix = f"{TextFormat.CYAN}├─{TextFormat.CLEAR}"
        progress_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
        completion_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"

    return start_prefix, progress_prefix, completion_prefix

def ExecuteScript(script_data: dict) -> bool: # 执行所解析的脚本
    login = script_data['login']

    # 解析目标地址
    host, game_port = TargetParser(f"{login['ip']}:{login['port']}", 25565)
    ip, _, _ = ResolveDomainName(host, "java")
    if ip is None:
        return False

    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(3)
            s.connect((ip, game_port))

        with MCRcon(ip, login['password'], port=login['rcon_port']) as mcr:
            log_info(f"{TextFormat.GREEN}>> RCON连接成功，开始执行脚本...{TextFormat.CLEAR}")
            log_info("")

            if not script_data['functions']:
                log_info("没有功能模块需要执行")
                return True

            # 统计总命令数
            total_commands = 0
            for function in script_data['functions']:
                for module in function['modules']:
                    if module.get('type') != 'wait':
                        total_commands += len(module['commands'])

            log_info(
                f"{TextFormat.CYAN}>> 脚本统计: {len(script_data['functions'])} 个循环, {total_commands} 个命令{TextFormat.CLEAR}")
            log_info("")

            for function in script_data['functions']:
                # 循环开始
                log_info(f"{TextFormat.CYAN}┌── 循环: {function['name']} ({function['frequency']}次){TextFormat.CLEAR}")

                for loop_idx in range(function['frequency']):
                    # 循环进度
                    if function['frequency'] > 1:
                        log_info(f"{TextFormat.CYAN}│{TextFormat.CLEAR}")
                        log_info(
                            f"{TextFormat.CYAN}├─ 第 {loop_idx + 1}/{function['frequency']} 次循环{TextFormat.CLEAR}")

                    command_count = 0
                    total_modules = len(function['modules'])

                    for module_idx, module in enumerate(function['modules']):
                        is_last_module = (module_idx == total_modules - 1)
                        is_last_loop = (loop_idx == function['frequency'] - 1)
                        is_last_module_and_last_loop = is_last_module and is_last_loop

                        if module.get('type') == 'wait':
                            # 迟滞模块
                            wait_time = module['time']

                            if is_last_module_and_last_loop:
                                start_prefix = f"{TextFormat.CYAN}└─{TextFormat.CLEAR}"
                                progress_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
                                completion_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
                                completion_indicator = "└─"  # 添加完成指示符
                            else:
                                start_prefix = f"{TextFormat.CYAN}├─{TextFormat.CLEAR}"
                                progress_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
                                completion_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
                                completion_indicator = "└─"  # 添加完成指示符

                            log_info(f"{start_prefix} {TextFormat.BLUE}迟滞 {wait_time} 秒{TextFormat.CLEAR}")

                            # 动态倒计时显示
                            start_time = time.time()
                            remaining = wait_time

                            while remaining > 0:
                                elapsed = time.time() - start_time
                                remaining = max(0, wait_time - elapsed)

                                progress_bar = "█" * int((wait_time - remaining) / wait_time * 20)
                                empty_bar = "░" * (20 - len(progress_bar))
                                percent = int((wait_time - remaining) / wait_time * 100)

                                timestamp = lg4pb.get_timestamp()
                                sys.stdout.write(
                                    f"\r{TextFormat.YELLOW}{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} {progress_prefix}   {TextFormat.BLUE}[{progress_bar}{empty_bar}] {percent}% ({remaining:.1f}s){TextFormat.CLEAR}")
                                sys.stdout.flush()
                                time.sleep(0.1)

                            # 迟滞完毕提示
                            timestamp = lg4pb.get_timestamp()
                            sys.stdout.write(
                                f"\r{TextFormat.YELLOW}{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} {completion_prefix}   {TextFormat.BLUE}{completion_indicator}{TextFormat.CLEAR} {TextFormat.YELLOW}迟滞完毕{' ' * 40}{TextFormat.CLEAR}\n")

                        else:
                            # 命令模块
                            total_commands_in_module = len(module['commands'])

                            # 显示模块标题（仅多命令模块）
                            if total_commands_in_module > 1:
                                if is_last_module_and_last_loop:
                                    log_info(
                                        f"{TextFormat.CYAN}└─ {TextFormat.PURPLE}模块: {module['name']}{TextFormat.CLEAR}")
                                else:
                                    log_info(
                                        f"{TextFormat.CYAN}├─ {TextFormat.PURPLE}模块: {module['name']}{TextFormat.CLEAR}")

                            for cmd_idx, cmd in enumerate(module['commands']):
                                command_count += 1
                                is_last_command_in_module = (cmd_idx == total_commands_in_module - 1)
                                is_last_command_in_loop = is_last_command_in_module and is_last_module

                                try:
                                    # 修复：统一计算命令和结果的连接线
                                    if total_commands_in_module > 1:
                                        # 多命令模块
                                        if is_last_module_and_last_loop:
                                            # 最后一个模块的所有命令
                                            if is_last_command_in_module:
                                                # 模块内最后一个命令
                                                command_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
                                                prefix = "└─"
                                                result_prefix = "  "
                                            else:
                                                # 模块内非最后一个命令
                                                command_prefix = f"{TextFormat.CYAN}  {TextFormat.CLEAR}"
                                                prefix = "├─"
                                                result_prefix = "│ "
                                        else:
                                            # 非最后一个模块
                                            if is_last_command_in_module:
                                                # 模块内最后一个命令
                                                command_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
                                                prefix = "└─"
                                                result_prefix = "  "
                                            else:
                                                # 模块内非最后一个命令
                                                command_prefix = f"{TextFormat.CYAN}│{TextFormat.CLEAR}"
                                                prefix = "├─"
                                                result_prefix = "│ "

                                        log_info(
                                            f"{command_prefix}   {TextFormat.PURPLE}{prefix}{TextFormat.CLEAR} {TextFormat.YELLOW}{cmd}{TextFormat.CLEAR}")
                                    else:
                                        # 单命令模块
                                        if is_last_module_and_last_loop:
                                            log_info(f"{TextFormat.CYAN}└─ {TextFormat.YELLOW}{cmd}{TextFormat.CLEAR}")
                                        else:
                                            log_info(f"{TextFormat.CYAN}├─ {TextFormat.YELLOW}{cmd}{TextFormat.CLEAR}")

                                    response = mcr.command(cmd)

                                    # 处理响应显示
                                    if cmd.lower().startswith('say '):
                                        said_content = cmd[4:].strip()
                                        display_response = said_content if not response.strip() else response.strip()
                                        result_type = "输出"
                                    else:
                                        display_response = response.strip()
                                        result_type = "结果"

                                    # 显示结果 - 使用与命令相同的连接线逻辑
                                    if total_commands_in_module > 1:
                                        # 多命令模块的结果
                                        if is_last_command_in_module:
                                            # 模块内最后一个命令的结果
                                            log_info(
                                                f"{command_prefix}   {TextFormat.PURPLE}{result_prefix}{TextFormat.CLEAR}   {TextFormat.GREEN}{result_type}: {display_response}{TextFormat.CLEAR}")
                                        else:
                                            # 模块内非最后一个命令的结果
                                            log_info(
                                                f"{command_prefix}   {TextFormat.PURPLE}{result_prefix}{TextFormat.CLEAR}   {TextFormat.GREEN}{result_type}: {display_response}{TextFormat.CLEAR}")
                                    else:
                                        # 单命令模块的结果
                                        if is_last_module_and_last_loop:
                                            log_info(
                                                f"{TextFormat.CYAN}  {TextFormat.CLEAR}   {TextFormat.GREEN}{result_type}: {display_response}{TextFormat.CLEAR}")
                                        else:
                                            log_info(
                                                f"{TextFormat.CYAN}│{TextFormat.CLEAR}   {TextFormat.GREEN}{result_type}: {display_response}{TextFormat.CLEAR}")

                                except Exception as e:
                                    # 错误处理
                                    if total_commands_in_module > 1:
                                        log_info(
                                            f"{command_prefix}   {TextFormat.PURPLE}{prefix}{TextFormat.CLEAR} {TextFormat.RED}{cmd}{TextFormat.CLEAR}")
                                        log_error(
                                            f"{command_prefix}   {TextFormat.PURPLE}{result_prefix}{TextFormat.CLEAR}   {TextFormat.RED}错误: {str(e)}{TextFormat.CLEAR}")
                                    else:
                                        # 单命令模块错误处理
                                        if is_last_module_and_last_loop:
                                            log_info(f"{TextFormat.CYAN}└─ {TextFormat.RED}{cmd}{TextFormat.CLEAR}")
                                            log_error(
                                                f"{TextFormat.CYAN}  {TextFormat.CLEAR}   {TextFormat.RED}错误: {str(e)}{TextFormat.CLEAR}")
                                        else:
                                            log_info(f"{TextFormat.CYAN}├─ {TextFormat.RED}{cmd}{TextFormat.CLEAR}")
                                            log_error(
                                                f"{TextFormat.CYAN}│{TextFormat.CLEAR}   {TextFormat.RED}错误: {str(e)}{TextFormat.CLEAR}")

                # 循环结束后添加空行分隔
                log_info("")

            # 脚本完成
            log_info(f"{TextFormat.GREEN}>> 脚本执行完成！{TextFormat.CLEAR}")
            return True

    except Exception as e:
        log_error(f">> 执行脚本时发生错误: {e}")
        return False

# 该函数实现了下列功能：
# 1. 调用方法解析域名
# 2. 建立RCON连接
# 3. 交互式命令组录入
# 4. 执行命令组
# 5. 退出命令组模式
# 6. 处理异常情况
def InteractiveRCON(target: str, rcon_port: int, password: str):
    host, game_port = TargetParser(target, 25565)
    ip, _, _ = ResolveDomainName(host, "java")  # 接收三个返回值
    if ip is None:
        return False
    # 通过time.sleep()函数来控制输出间隔
    log_debug(f"解析主机名：{host}")
    time.sleep(0.3)
    log_info(f"初始化RCON配置: 目标 {TextFormat.YELLOW}{host}:{game_port}{TextFormat.CLEAR} | RCON端口 {TextFormat.YELLOW}{rcon_port}{TextFormat.CLEAR} | 密码 {TextFormat.YELLOW}{password}{TextFormat.CLEAR}")
    log_warn("注意！请勿将RCON端口/密码泄露给他人！")
    time.sleep(0.2)
    log_info("正在建立RCON连接...")
    time.sleep(0.4)
    # 尝试建立RCON连接
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(3)
            s.connect((ip, game_port))

        with MCRcon(ip, password, port=rcon_port) as mcr:
            log_info(f"{TextFormat.GREEN}✓{TextFormat.CLEAR} RCON连接成功")
            time.sleep(0.3)

            # 连接成功后显示提示信息
            log_info("交互式命令组使用方法:")
            time.sleep(0.1)
            log_info("1. 直接输入命令 - 添加到正在录入的命令组模块")
            log_info("2. WAIT 秒数 - 插入执行延迟")
            log_info("3. LOOP 次数 - 设置单模块循环次数")
            log_info("4. BREAK - 执行当前命令组模块并退出")
            log_info("5. 空回车 - 换行录入新命令组成员")
            log_info("6. 双击空回车 - 强制执行当前命令组模块")

            current_group = []
            loop_count = 1
            # 下为检查输入
            # 检查到LOOP，则对录入命令组的成员整体循环
            # 检查到WAIT，则迟滞指定时间后执行下一条命令
            # 检查到BREAK，则执行当前命令组模块并退出命令组模式
            while True:
                try:
                    prompt = f"{TextFormat.CYAN}->{TextFormat.CLEAR} "
                    user_input = input(prompt).strip()

                    if not user_input:
                        if current_group:
                            ExecuteCommandGroup(mcr, current_group, loop_count)
                            current_group = []
                            loop_count = 1
                        continue

                    if user_input.upper() == "BREAK":
                        if current_group:
                            ExecuteCommandGroup(mcr, current_group, loop_count)
                        log_info("已执行当前命令组模块并退出命令组模式")
                        return True

                    if user_input.upper().startswith("WAIT "):
                        try:
                            wait_time = float(user_input.split()[1])
                            current_group.append(("WAIT", wait_time))
                            print(f"{TextFormat.YELLOW}[WAIT]{TextFormat.CLEAR} 已添加延时: {wait_time}秒")
                        except (IndexError, ValueError):
                            print(f"{TextFormat.RED}[ERROR]{TextFormat.CLEAR} 无效的迟滞时间，请使用 'WAIT 秒数'")
                        continue

                    if user_input.upper().startswith("LOOP "):
                        try:
                            new_loop = int(user_input.split()[1])
                            if new_loop < 1:
                                raise ValueError
                            loop_count = new_loop
                            print(f"{TextFormat.CYAN}[LOOP]{TextFormat.CLEAR} 已设置循环模块并设置次数: {loop_count}")
                            if current_group:
                                ExecuteCommandGroup(mcr, current_group, loop_count)
                                current_group = []
                                loop_count = 1
                        except (IndexError, ValueError):
                            print(f"{TextFormat.RED}[ERROR]{TextFormat.CLEAR} 无效的循环次数，请使用 'LOOP 次数'")
                        continue

                    current_group.append(("CMD", user_input))
                    print(f"{TextFormat.PURPLE}[JOIN]{TextFormat.CLEAR} 添加命令组成员: {user_input}")

                except KeyboardInterrupt:
                    if current_group:
                        ExecuteCommandGroup(mcr, current_group, loop_count)
                    log_info("\n已执行当前命令组模块并退出命令组模式")
                    break
    except Exception as e:
        log_error(f"在运行RCON时发生了意外的错误: {e}")
        return False
    return True

# 执行命令组，通过定义的WAIT/LOOP/BREAK指令来执行命令组
def ExecuteCommandGroup(mcr: MCRcon, commands: list, loop_count: int = 1):
    if not commands:
        return

    try:
        for _ in range(loop_count):
            for idx, (cmd_type, content) in enumerate(commands):
                if cmd_type == "WAIT":
                    remaining = content
                    start_time = time.time()

                    while remaining > 0:
                        next_cmd = next((c[1] for c in commands[idx + 1:] if c[0] == "CMD"), "")

                        elapsed = time.time() - start_time
                        remaining = max(0, content - elapsed)

                        sys.stdout.write(f"\r{TextFormat.YELLOW}[WAIT]{TextFormat.CLEAR} 迟滞 {remaining:.2f} 秒后执行: {next_cmd}")
                        sys.stdout.flush()
                        time.sleep(0.01)

                    sys.stdout.write("\n")
                    continue

                try:
                    response = mcr.command(content)
                    # 使用新的§代码解析功能
                    parsed_response = ParseRCONResponse(response)
                    output(f"{TextFormat.BLUE}[EXEC]{TextFormat.CLEAR} {content}\n"
                           f"{TextFormat.GREEN}[INFO]{TextFormat.CLEAR} 服务器响应结果：\n"
                           f"\033[1m{parsed_response}\033[0m")
                except Exception as e:
                    log_error(f"命令执行失败: {content}")
                    log_error(f"错误详情: {str(e)}")
    finally:
        commands.clear()

# RCON信息录入
def RCONExecute(target: str, rcon_port: int, password: str, command: str) -> bool:
    host, game_port = TargetParser(target, 25565)
    ip, _, _ = ResolveDomainName(host, "java")  # 修复：接收三个返回值
    if ip is None:
        return False

    # 确保端口是整数类型
    try:
        game_port = int(game_port)
        rcon_port = int(rcon_port)

        if not (1 <= game_port <= 65535):
            log_error("游戏端口必须在1到65535之间")
            return False

        if not (1 <= rcon_port <= 65535):
            log_error("RCON端口必须在1到65535之间")
            return False

    except (TypeError, ValueError):
        log_error("错误: 端口号必须是整数")
        return False
    # 尝试建立RCON连接（保险）
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(3)
            s.connect((ip, game_port))

        with MCRcon(ip, password, port=rcon_port) as mcr:
            log_info(f"{TextFormat.GREEN}✓{TextFormat.CLEAR} RCON连接成功")
            response = mcr.command(command)
            # 使用新的§代码解析功能
            parsed_response = ParseRCONResponse(response)
            output(f"{TextFormat.GREEN}[INFO]{TextFormat.CLEAR} 执行结果: \n{parsed_response}", end="")
            return True
    # 显示错误信息
    except ConnectionRefusedError:
        log_error("目标主机拒绝连接！请检查：")
        log_error("1. server.properties中的enable-rcon=是否为true？")
        log_error(f"2. 端口 {rcon_port} 是否开放？")
        log_error("3. 服务器是否在线？")
        log_error("4. RCON端口/密码是否正确？")
    except socket.timeout:
        log_error("连接超时！请检查：")
        log_error("1. 如果你使用了反向代理软件，请为RCON端口也开设一条TCP连接")
    except Exception as e:
        log_error(f"在通过RCON执行命令时发生了意外的错误: {e}")

    return False

def main():
    """独立运行时的主函数"""
    parser = argparse.ArgumentParser(
        description='Minecraft RCON远程控制',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 服务器连接参数
    parser.add_argument('-t', '--target',
                       help='服务器地址 (格式: hostname[:port])\n'
                            '示例: example.com:25565 或 192.168.1.1\n'
                            '默认端口: 25565')
    parser.add_argument('-rpw', '--rcon-password',
                       help='RCON密码')
    parser.add_argument('-rp', '--rcon-port', type=int, default=25575,
                       help='RCON端口 (默认: 25575)')

    # 运行模式（互斥）
    mode_group = parser.add_mutually_exclusive_group(required=True)
    mode_group.add_argument('-cmd', '--command',
                           help='执行单个RCON命令')
    mode_group.add_argument('-cg', '--command-group', action='store_true',
                           help='进入交互式命令组模式')
    mode_group.add_argument('-s', '--script',
                           help='执行脚本文件')

    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX RCON远程控制\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 脚本模式
        if args.script:
            script_data = ParseScriptFile(args.script)
            if script_data is None:
                sys.exit(1)

            DisplayScriptPlan(script_data)

            confirm = input(f"{TextFormat.YELLOW}确定要执行该脚本吗？（Y/N）{TextFormat.CLEAR} ").strip().upper()
            if confirm != 'Y':
                log_info("已取消脚本执行")
                return

            success = ExecuteScript(script_data)
            if not success:
                sys.exit(1)
            return

        # 手动连接模式参数验证
        if not args.target:
            log_error("必须提供目标地址 (-t/--target)")
            sys.exit(1)
        if not args.rcon_password:  # 这里已经改好了
            log_error("必须提供RCON密码 (-rpw/--rcon-password)")
            sys.exit(1)

        if not (1 <= args.rcon_port <= 65535):
            log_error("RCON端口必须在1-65535之间")
            sys.exit(1)

        # 执行RCON操作 - 这里需要修改
        if args.command_group:
            success = InteractiveRCON(args.target, args.rcon_port, args.rcon_password)  # 改为 args.rcon_password
        else:
            if not args.command:
                log_error("必须提供要执行的命令 (-cmd/--command)")
                sys.exit(1)
            success = RCONExecute(args.target, args.rcon_port, args.rcon_password, args.command)  # 改为 args.rcon_password

        if not success:
            sys.exit(1)

        log_info("RCON操作完成！")

    except Exception as e:
        log_error(f"RCON操作失败: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()