"""------------------------外部库------------------------"""
import sys
import time
from mcstatus import JavaServer, BedrockServer
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_error, ResolveDomainName

# mcstatus 软件包获取服务器数据部分
# 该函数用于检查Minecraft 基岩版服务器
def CheckBedrockServer(host, port):
    # 解析域名
    ip = ResolveDomainName(host)
    if ip is None:
        return
    # 实例化 BedrockServer 对象，并获取服务器状态
    try:
        server = BedrockServer(ip, port)
        status = server.status()
        # 返回结果
        log_info("目标基岩版服务器在线！")
        time.sleep(0.2)
        log_info(f"地址: {host}:{port}({ip}:{port})")
        time.sleep(0.3)
        log_info(f"服务器名称: {TextFormat.ParseMotd(status.motd)}")
        time.sleep(0.1)
        log_info(f"版本: {status.version.name}")
        log_info(f"在线玩家/最大玩家: {status.players.online}/{status.players.max}")
        time.sleep(0.1)
        log_info(f"延迟: {TextFormat.YELLOW}{status.latency:.2f}{TextFormat.CLEAR} ms")

    except Exception as e:
        log_error(f"无法连接基岩版服务器： {host}:{port}")
        log_error(f"原因：{e}")
        time.sleep(0.2)
        log_error(f"临时会话已销毁")
        sys.exit(1)

# 该函数用于检查Minecraft Java版服务器
def CheckJavaServer(host, port):
    ip = ResolveDomainName(host)
    if ip is None:
        return
    # 实例化 JavaServer 对象，并获取服务器状态
    try:
        server = JavaServer(ip, port)
        player_list = "无"

        try:
            status = server.status()
            players = [p.name for p in status.players.sample] if status.players.sample else []
            player_list = ", ".join(players) if players else "无"

            if not players:
                try:
                    query = server.query()
                    player_list = ", ".join(query.players.names) if query.players.names else "无"
                except Exception:
                    pass

        except Exception:
            pass
            raise
        # 返回结果
        log_info("目标Java版服务器在线！")
        log_info(f"地址: {host}:{port}({ip}:{port})")
        log_info(f"服务器名称:\n{TextFormat._MCColourToAnsi(str(server.status().description))}")
        log_info(f"版本: {server.status().version.name}")
        log_info(f"在线玩家/最大玩家: {server.status().players.online}/{server.status().players.max}")
        log_info(f"玩家列表: {player_list}")
        log_info(f"延迟: {TextFormat.YELLOW}{server.status().latency:.2f}{TextFormat.CLEAR} ms")

    except Exception as e:
        log_error(f"无法连接Java版服务器： {host}:{port}")
        log_error(f"原因：{e}")
        time.sleep(0.2)
        log_error(f"临时会话已销毁")
        sys.exit(1)