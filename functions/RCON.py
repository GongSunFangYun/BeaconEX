"""------------------------外部库------------------------"""
import time
import sys
from mcrcon import MCRcon
import socket
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_error, output, ResolveDomainName, log_debug, log_warn

# 该函数实现了下列功能：
# 1. 调用方法解析域名
# 2. 建立RCON连接
# 3. 交互式命令组录入
# 4. 执行命令组
# 5. 退出命令组模式
# 6. 处理异常情况
def InteractiveRCON(host: str, game_port: int, rcon_port: int, password: str):
    ip = ResolveDomainName(host)
    if ip is None:
        return False
    # 通过time.sleep()函数来控制输出间隔
    log_debug(f"解析主机名：{host}")
    time.sleep(0.3)
    log_info(f"初始化RCON配置: IP&端口 {TextFormat.YELLOW}{ip}:{game_port}{TextFormat.CLEAR} | RCON端口 {TextFormat.YELLOW}{rcon_port}{TextFormat.CLEAR} | 密码 {TextFormat.YELLOW}{password}{TextFormat.CLEAR}")
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
            # 检查到WAIT，则等待指定时间后执行下一条命令
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
                            print(f"{TextFormat.RED}[ERROR]{TextFormat.CLEAR} 无效的等待时间，请使用 'WAIT 秒数'")
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

                        sys.stdout.write(f"\r{TextFormat.YELLOW}[WAIT]{TextFormat.CLEAR} 等待执行: {next_cmd} [{remaining:.2f}s]")
                        sys.stdout.flush()
                        time.sleep(0.01)

                    sys.stdout.write("\n")
                    continue

                try:
                    response = mcr.command(content)
                    output(f"{TextFormat.BLUE}[EXEC]{TextFormat.CLEAR} {content}\n"
                           f"{TextFormat.GREEN}[INFO]{TextFormat.CLEAR} 服务器响应结果：\n"
                           f"\033[1m{response}\033[0m")
                except Exception as e:
                    log_error(f"命令执行失败: {content}")
                    log_error(f"错误详情: {str(e)}")
    finally:
        commands.clear()

# RCON信息录入
def RCONExecute(host: str, game_port: int, rcon_port: int, password: str, command: str) -> bool:
    ip = ResolveDomainName(host)
    if ip is None:
        return False

    # 确保端口是整数类型
    try:
        game_port = int(game_port)
        rcon_port = int(rcon_port)

        if not (1 <= game_port <= 65535):
            log_error("RCON端口必须在1到65535之间")
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
            output(f"{TextFormat.GREEN}[INFO]{TextFormat.CLEAR} 执行结果: \n{TextFormat.ParseMotd(response)}", end="")
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
        log_error("2. 如果你的服务器在非大陆地区，那超时也就不奇怪了（笑）")
    except Exception as e:
        log_error(f"在通过RCON执行命令时发生了意外的错误: {e}")

    return False
