"""------------------------外部库------------------------"""
import argparse
import sys
import time

from mcstatus import JavaServer, BedrockServer
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat, log_info, log_error
from bexlib2.rhnis import ResolveDomainName, _IsValidAddress_PublicMethod

def FormattedAddress(ip: str, port: int, original_target: str, is_srv: bool = False) -> str: # 格式化主机名和端口
    # 检查原始目标是否是纯IP地址
    if _IsValidAddress_PublicMethod(original_target):
        return f"{ip}:{port}"

    # 检查是否是SRV模式
    if is_srv:
        # SRV模式下只显示域名，不显示端口
        return f"{ip}:{port}（{original_target}）"

    # 检查原始目标是否包含端口
    if ':' in original_target:
        # 域:端口格式，直接显示原始目标
        return f"{ip}:{port}（{original_target}）"
    else:
        # 纯域名格式，显示域名和默认端口
        return f"{ip}:{port}（{original_target}:{port}）"

# mcstatus 软件包获取服务器数据部分
# 该函数用于检查Minecraft 基岩版服务器
def CheckBedrockServer(target: str):
    ip, port, is_srv = ResolveDomainName(target, "bedrock")

    # 实例化 BedrockServer 对象，并获取服务器状态
    try:
        server = BedrockServer(ip, port)  # 使用解析得到的 ip 和 port
        status = server.status()
        # 返回结果
        log_info("查询请求已发送，等待服务器响应...")
        log_info(f"服务器状态：{TextFormat.BRIGHT_GREEN}在线{TextFormat.CLEAR}")
        time.sleep(0.2)
        log_info(f"地址: {FormattedAddress(ip, port, target, is_srv)}")  # 使用格式化显示
        time.sleep(0.1)
        log_info(f"服务器名称: {TextFormat.ParseMotd(status.motd)}")
        time.sleep(0.1)
        log_info(f"版本: {status.version.name}")
        log_info(f"在线玩家/最大玩家: {status.players.online}/{status.players.max}")
        time.sleep(0.1)
        log_info(f"延迟: {TextFormat.YELLOW}{status.latency:.2f}{TextFormat.CLEAR} ms")

    except Exception as e:
        log_error(f"无法连接基岩版服务器： {FormattedAddress(ip, port, target, is_srv)}")  # 使用格式化显示
        log_error(f"原因：{e}")
        sys.exit(1)

# 该函数用于检查Minecraft Java版服务器
def CheckJavaServer(target: str):
    # 解析域名
    ip, port, is_srv = ResolveDomainName(target, "java")

    # 实例化 JavaServer 对象，并获取服务器状态
    try:
        server = JavaServer(ip, port)  # 使用解析得到的 ip 和 port
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
        log_info("查询请求已发送，等待服务器响应...")
        log_info(f"服务器状态：{TextFormat.BRIGHT_GREEN}在线{TextFormat.CLEAR}")
        log_info(f"地址: {FormattedAddress(ip, port, target, is_srv)}")  # 使用格式化显示
        log_info(f"服务器名称:\n{TextFormat._MCColourToAnsi(str(server.status().description))}")
        log_info(f"版本: {server.status().version.name}")
        log_info(f"在线玩家/最大玩家: {server.status().players.online}/{server.status().players.max}")
        log_info(f"玩家列表: {player_list}")
        log_info(f"延迟: {TextFormat.YELLOW}{server.status().latency:.2f}{TextFormat.CLEAR} ms")

    except Exception as e:
        log_error(f"无法连接Java版服务器： {FormattedAddress(ip, port, target, is_srv)}")  # 使用格式化显示
        log_error(f"原因：{e}")
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(
        description='查询Minecraft服务器状态',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 服务器类型参数
    server_group = parser.add_mutually_exclusive_group(required=True)
    server_group.add_argument('-java', '--java', action='store_true',
                            help='查询Java版服务器')
    server_group.add_argument('-bedrock', '--bedrock', action='store_true',
                            help='查询基岩版服务器')

    # 必需参数
    parser.add_argument('-t', '--target', required=True,
                       help='服务器地址 (格式: hostname[:port])\n'
                            '示例: example.com:25565 或 192.168.1.1\n'
                            'Java版默认端口: 25565\n'
                            '基岩版默认端口: 19132')
    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')


    args = parser.parse_args()

    try:
        # 执行服务器查询
        if args.java:
            CheckJavaServer(args.target)
        else:
            CheckBedrockServer(args.target)

        log_info("服务器查询完成！")

    except Exception as e:
        log_error(f"服务器查询失败: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()