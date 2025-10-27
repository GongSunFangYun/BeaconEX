"""------------------------外部库------------------------"""
import argparse
import re
import socket
import subprocess
import threading
import time
import ctypes
import sys
import psutil

from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat
from bexlib2.lg4pb import log_info, log_warn, log_error, et_log_info, et_log_error, et_log_debug, et_log_warn, log_debug


# noinspection PyUnusedLocal,PyBroadException
class EasyTierManager:
    def __init__(self):
        # EasyTier 核心文件目录路径
        self.base_dir = Path(__file__).parent / "EasyTier"
        # EasyTier 核心服务可执行文件路径
        self.core_exe = self.base_dir / "easytier-core.exe"
        # EasyTier 命令行工具可执行文件路径
        self.cli_exe = self.base_dir / "easytier-cli.exe"
        # EasyTier 服务进程对象
        self.process = None
        # 输出捕获线程
        self.output_thread = None
        # 服务运行状态标志
        self.running = False
        # 分配的虚拟网络IP地址
        self.assigned_ip = None
        # 当前网络名称
        self.network_name = None
        # 当前网络密码
        self.network_password = None

        # 公共节点集合（按照优选顺序排列）
        # 注意：请勿修改此列表，否则绝对会导致虚拟网络创建失败
        # TODO: 多整点公共节点
        self.public_nodes = [
            "tcp://public.easytier.top:11010",
            "tcp://8.138.6.53:11010",
            "tcp://8.148.29.206:11010",
            "tcp://turn.js.629957.xyz:11012",
            "tcp://turn.bj.629957.xyz:11010",
            "tcp://et.sh.suhoan.cn:11010",
        ]
        self.available_nodes = []
        self.current_public_node_index = 0

        # 检查easytier-core.exe和easytier-cli.exe是否存在
        if not self.core_exe.exists():
            log_error(f"缺失 easytier-core.exe，请检查安装目录文件完整性！")
            sys.exit(1)
        if not self.cli_exe.exists():
            log_error(f"缺失 easytier-cli.exe，请检查安装目录文件完整性！")
            sys.exit(1)

    @staticmethod
    def _CheckPermission(): # 检查是否有管理员权限
        # noinspection PyUnresolvedReferences
        def is_admin():
            try:
                return ctypes.windll.shell32.IsUserAnAdmin()
            except:
                return False

        if is_admin():
            return True

        # 如果没有管理员权限，则输出错误日志后退出
        # 因为Windows对CLI的提权太费事了，所以让用户自己努力丰衣足食罢
        log_error("EasyTier Network 服务需要管理员权限才能运行！")
        log_info("请按照以下步骤操作：")
        log_info("1. 打开开始菜单，搜索 '命令提示符/PowerShell/终端'")
        log_info("2. 右键点击 '命令提示符/PowerShell/终端'，选择 '以管理员身份运行'")
        log_info("3. 在打开的窗口中，运行你刚才所运行的命令")
        log_info("*如果你的计算机启用了'sudo'命令，则可以直接使用sudo提权运行 EasyTier Network 服务")
        sys.exit(1)

    @staticmethod
    def _CheckNodeHealth(node_url): # 检查节点健康状态
        try:
              # 解析节点URL
            if node_url.startswith('tcp://'):
                parts = node_url[6:].split(':')
                if len(parts) == 2:
                    host, port = parts
                    port = int(port)

                    # 使用socket进行TCP连接测试
                    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
                        sock.settimeout(3)  # 3秒超时
                        result = sock.connect_ex((host, port))
                        return result == 0  # 返回连接是否成功
        except Exception:
            pass
        return False

    def _CheckAllNodeHealthWithMultiThreading(self): # 超超超超超超长函数，实际上就是去测试节点健康状态然后收集可用的节点
        available_nodes = []

        with ThreadPoolExecutor(max_workers=len(self.public_nodes)) as executor:
            # 提交所有健康检查任务
            future_to_node = {
                executor.submit(self._CheckNodeHealth, node): node
                for node in self.public_nodes
            }

            # 收集可用的节点（按照原始顺序）
            for future in as_completed(future_to_node):
                node = future_to_node[future]
                try:
                    if future.result():
                        available_nodes.append(node)
                except Exception:
                    pass  # 静默处理异常

        # 按照原始public_nodes的顺序排序可用节点
        available_nodes.sort(key=lambda x: self.public_nodes.index(x))
        return available_nodes

    def _GetBestAvailableNode(self): # 获取最吊的可用公共节点
        if not self.available_nodes:
            # 如果没有预先检查过的可用节点，立即进行健康检查
            self.available_nodes = self._CheckAllNodeHealthWithMultiThreading()

        if self.available_nodes:
            # 返回第一个可用的节点（按照优选顺序）
            return self.available_nodes[0]
        else:
            # 如果没有可用节点，回退到原始列表的第一个
            log_warn("所有公共节点均不可用，使用默认节点")
            return self.public_nodes[0]

    def _GetNextAvailableNode(self): # 辅助节点检查用的函数，移除当前失败节点并获取下一个可用节点
        if not self.available_nodes:
            self.available_nodes = self._CheckAllNodeHealthWithMultiThreading()

        if len(self.available_nodes) > 1:
            # 移除当前失败的节点，选择下一个
            current_node = self.available_nodes[0]
            self.available_nodes.pop(0)
            if self.available_nodes:
                return self.available_nodes[0]

        # 如果没有更多可用节点，重新进行健康检查
        # 这可能会导致检查时间延长（但是不会发生这种情况，Easytier的官方节点和社区节点还是很耐造的）
        self.available_nodes = self._CheckAllNodeHealthWithMultiThreading()
        return self.available_nodes[0] if self.available_nodes else self.public_nodes[0]

    def _GetAssignedIP(self, timeout=30): # 获得分配的IP地址
        log_debug("等待 EasyTier 分配虚拟网络IP...")
        log_debug("若 EasyTier 反复断开节点和连接节点，请检查你的网络是否稳定，这可能是你和公共节点的连接不稳定导致的")
        time.sleep(0.5)
        start_time = time.time()

        while time.time() - start_time < timeout:
            try:
                result = subprocess.run(
                    [str(self.cli_exe), "peer"],
                    capture_output=True,
                    text=True,
                    encoding='utf-8',
                    timeout=5
                )

                if result.returncode == 0 and result.stdout.strip():
                    # 解析peer列表，找到本机的IP地址（Local行）
                    peers = self._ParsePeerTable(result.stdout)
                    for peer in peers:
                        if peer['cost'] == 'Local' and peer['ipv4'] != "未知":
                            assigned_ip = self._ExtractPureIP(peer['ipv4'])
                            return assigned_ip

            except (subprocess.TimeoutExpired, subprocess.SubprocessError) as e:
                log_warn(f"获取IP地址时出现错误: {e}")

        log_warn("获取IP地址超时，请稍后手动使用 -l 参数以查看你所分配的IP地址！")
        return None

    @staticmethod
    def _ExtractPureIP(ip_with_cidr):
        return ip_with_cidr.split('/')[0]

    @staticmethod
    def _CheckEasyTierCoreRunning(): # 检查EasyTier Core是否正在运行
        for proc in psutil.process_iter(['name']):
            if proc.info['name'] == 'easytier-core.exe':
                return True
        return False

    def _CaptureOutput(self): # 捕获EasyTier Core的输出并解析，原理是管道重定向
        while self.running and self.process and self.process.stdout:
            line = self.process.stdout.readline()
            if not line:
                break
            line = line.strip()
            if line:
                self._ParseEasyTierOutput(line)

    def _ParseEasyTierOutput(self, line): # 解析EasyTier Core的输出
        # 因为EasyTier Core的日志格式不符合我的日志库规范，所以我选择截去了一部分日志
        cleaned_message = self._FormatEasyTierLog(line)

        # 根据日志级别分类
        if any(keyword in cleaned_message.lower() for keyword in ['error', 'failed', 'failure']):
            et_log_error(cleaned_message)
        elif any(keyword in cleaned_message.lower() for keyword in ['warning', 'warn']):
            et_log_warn(cleaned_message)
        elif any(keyword in cleaned_message.lower() for keyword in ['debug', 'trace']):
            et_log_debug(cleaned_message)
        else:
            et_log_info(cleaned_message)

    @staticmethod
    def _FormatEasyTierLog(line):
        # 第一步：移除外层时间戳和线程标识
        # 格式: XXXX-XX-XX XX:XX:XX [EasyTier Thread/LOG_LEVEL]
        outer_pattern = r'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \[[^]]+/[A-Z]+\] '
        cleaned = re.sub(outer_pattern, '', line)

        # 第二步：移除内层时间戳（消息内容中的时间戳）
        # 格式: XXXX-XX-XX XX:XX:XX:
        inner_pattern = r'\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}: '
        cleaned = re.sub(inner_pattern, '', cleaned)

        return cleaned.strip() # 最后就搞到完美的日志了

    def SetUpEasyTierNetwork(self, network_name, password): # 建立EasyTier Network服务
        self.network_name = network_name
        self.network_password = password
        log_info("建立 EasyTier Network 服务...")

        # 预先进行节点健康检查（静默进行不输出的那种）
        self.available_nodes = self._CheckAllNodeHealthWithMultiThreading()

        return self._StartEasyTierNetwork(network_name, password)

    def _StartEasyTierNetwork(self, network_name, password): # 启动EasyTier Network服务（节点检查完后进行）
        log_info(f"网络名称：{TextFormat.YELLOW}{network_name}{TextFormat.CLEAR} | 密码：{TextFormat.BLUE}{password}{TextFormat.CLEAR}")

        # 获取最优的可用公共节点
        current_node = self._GetBestAvailableNode()
        log_info(f"使用公共节点: {TextFormat.CYAN}{current_node}{TextFormat.CLEAR}")

        # 构建启动参数
        cmd = [
            str(self.core_exe),
            "-d",  # 守护进程模式，在后台运行
            "--network-name", network_name,  # 指定虚拟网络名称
            "--network-secret", password,  # 设置网络连接密码
            "--multi-thread",  # 启用多线程处理，提升性能
            "--enable-kcp-proxy",  # 启用KCP协议代理，优化网络传输
            "--external-node", current_node,  # 指定外部公共节点服务器
            "--use-smoltcp"  # 使用smoltcp网络协议栈
        ]

        # 开始将EasyTier Core的输出使用管道重定向输出到当前进程
        self.process = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            encoding='utf-8',
            bufsize=1,
            universal_newlines=True
        )

        self.running = True

        self.output_thread = threading.Thread(target=self._CaptureOutput)
        self.output_thread.daemon = True
        self.output_thread.start()

        # 启动监控线程，检测节点是否失效
        threading.Thread(target=self._MonitorPublicNodes, daemon=True).start()

        # 等待一段时间后获取实际分配的IP
        threading.Thread(target=self._WaitAndDisplayIP, daemon=True).start()

        log_info(f"{TextFormat.GREEN}-== EasyTier Network 服务启动成功 ==-{TextFormat.CLEAR}")
        return True

    def _MonitorPublicNodes(self): # 监控公共节点健康状态
        time.sleep(30)  # 等待一段时间后再开始监控

        while self.running:
            time.sleep(15)  # 每15秒检查一次

            # 检查进程是否还在运行
            if self.process.poll() is not None:
                log_warn("EasyTier 进程异常退出，尝试重新启动并切换公共节点...")
                self._TryNextPublicNode()
                break

    def _TryNextPublicNode(self):
        if not self.network_name or not self.network_password:
            log_error("无法重新启动：缺少网络名称或密码")
            return

        log_info("尝试切换到下一个可用公共节点...")
        self.StopService()  # 停止当前进程

        # 短暂等待，确保进程完全停止
        time.sleep(2)

        # 重新启动服务
        self._StartEasyTierNetwork(self.network_name, self.network_password)

    def _WaitAndDisplayIP(self):
        time.sleep(3)  # 等待服务稳定
        self.assigned_ip = self._GetAssignedIP()
        if self.assigned_ip:
            self._DisplayConnectionInfo(self.assigned_ip)
        else:
            log_warn("无法自动获取IP地址，请稍后手动运行 'easytier-cli.exe peer' 查看")

    @staticmethod
    def _DisplayConnectionInfo(ip): # 显示连接信息
        log_info(f"你的虚拟网络IP: {TextFormat.CYAN}{ip}{TextFormat.CLEAR}")
        log_info("你已加入该虚拟网络，因此你可以使用该IP配合任意端口连接到任意TCP[Java版]/UDP[基岩版]服务")
        log_info(f"例如 Minecraft 联机，如果局域网开放端口于 {TextFormat.CYAN}25565{TextFormat.CLEAR} 则直接使用 {TextFormat.CYAN}{ip}:25565{TextFormat.CLEAR} 进行连接")
        log_info(f"若他人想要加入该虚拟网络，请提供网络名称与密码于对方，并使用 {TextFormat.CYAN}-jn/--join-network -n/--name {TextFormat.YELLOW}NETWORK_NAME {TextFormat.CYAN}-pw/--password {TextFormat.YELLOW}NETWORK_PASSWORD{TextFormat.CLEAR} 以加入该虚拟网络")
        log_info(f"若想再次查看自己的虚拟网络IP亦或者他人的虚拟网络IP，请使用 {TextFormat.CYAN}-l{TextFormat.CLEAR} 参数列出连接者数据")

    @staticmethod
    def _ParsePeerTable(table_text): # 解析easytier-cli.exe peer命令输出的列表
        peers = []
        lines = table_text.strip().split('\n')

        # 跳过表头分隔线
        for line in lines[2:]:  # 跳过前两行（表头和分隔线）
            if not line.strip() or '|-' in line:
                continue

            # 解析表格行
            columns = [col.strip() for col in line.split('|')[1:-1]]  # 去掉首尾的空列

            if len(columns) >= 9:
                # 对每个字段检查是否为空，为空则填充为"未知"
                peer = {
                    'ipv4': columns[0] if columns[0] else "未知",
                    'hostname': columns[1] if columns[1] else "未知",
                    'cost': columns[2] if columns[2] else "未知",
                    'latency': columns[3] if columns[3] and columns[3] != '-' else "未知",
                    'loss': columns[4] if columns[4] and columns[4] != '-' else "未知",
                    'rx': columns[5] if columns[5] and columns[5] != '-' else "未知",
                    'tx': columns[6] if columns[6] and columns[6] != '-' else "未知",
                    'tunnel': columns[7] if columns[7] else "未知",
                    'nat': columns[8] if columns[8] else "未知",
                    'version': columns[9] if len(columns) > 9 and columns[9] else "未知"
                }
                peers.append(peer)

        return peers

    def _FormatPeerDisplay(self, peers): # 在解析完成后，格式化输出取得的信息
        if not peers:
            return "暂无节点信息"

        output = [f"{TextFormat.CYAN}当前网络节点信息:{TextFormat.CLEAR}", ""]

        for i, peer in enumerate(peers, 1):
            # 判断节点类型
            if peer['cost'] == 'Local':
                node_type = f"{TextFormat.GREEN}[本机]{TextFormat.CLEAR}"
            elif 'PublicServer' in peer['hostname']:
                node_type = f"{TextFormat.BLUE}[公共服务器]{TextFormat.CLEAR}"
            else:
                node_type = f"{TextFormat.YELLOW}[用户节点]{TextFormat.CLEAR}"

            # 构建节点信息
            output.append(f"{TextFormat.BOLD}{i}. {node_type} {peer['hostname']}{TextFormat.CLEAR}")

            if peer['ipv4'] and peer['ipv4'] != "未知":
                pure_ip = self._ExtractPureIP(peer['ipv4'])
                output.append(f"   IP地址: {TextFormat.CYAN}{pure_ip}{TextFormat.CLEAR}")
            else:
                output.append(f"   IP地址: 未知")

            if peer['cost'] != 'Local':
                output.append(f"   连接: {peer['cost']} | 延迟: {peer['latency']} | 丢包: {peer['loss']}")
                output.append(f"   隧道: {peer['tunnel']} | NAT类型: {peer['nat']}")
                output.append(f"   流量: 接收 {peer['rx']} | 发送 {peer['tx']}")

            output.append("")  # 空行分隔

        return '\n'.join(output)

    def ListAllConnectionPeers(self): # 运行easytier-cli.exe peer命令
        log_info("获取网络节点信息...")

        result = subprocess.run(
            [str(self.cli_exe), "peer"],
            capture_output=True,
            text=True,
            encoding='utf-8'
        )

        if result.returncode == 0 and result.stdout.strip():
            peers = self._ParsePeerTable(result.stdout)
            formatted_output = self._FormatPeerDisplay(peers)
            print(formatted_output)

            # 额外显示本机IP
            for peer in peers:
                if peer['cost'] == 'Local' and peer['ipv4'] != "未知":
                    pure_ip = self._ExtractPureIP(peer['ipv4'])
                    break
        else:
            log_warn("无法获取节点信息！")
            log_warn("请检查 Easytier Network 是否已启动！")

    def StopService(self): # 停止EasyTier Network服务
        if self.process and self.process.poll() is None:
            log_info("正在停止 EasyTier Network 服务...")
            self.running = False
            self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                log_info("正在强行停止 EasyTier Network 服务...")
                self.process.kill()
                self.process.wait()

    def wait(self):
        if self.process:
            self.process.wait()

def main():
    EasyTierManager._CheckPermission() # 在程序刚运行时检查权限

    parser = argparse.ArgumentParser(description="BEX EasyTier P2P 联机工具")
    # 构建参数
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("-cn", "--create-network", action="store_true",
                       help="创建虚拟网络")
    group.add_argument("-jn", "--join-network", action="store_true",
                       help="加入虚拟网络")
    group.add_argument("-l", "--list", action="store_true",
                       help="列出当前网络中的用户")

    parser.add_argument("-n", "--name", type=str,
                        help="网络名称")
    parser.add_argument("-pw", "--password", type=str,
                        help="网络密码")
    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')


    args = parser.parse_args()

    manager = EasyTierManager()

    # 处理参数
    if args.create_network:
        if not args.name or not args.password:
            log_error("创建网络需要指定 --name 和 --password")
            return
        manager.SetUpEasyTierNetwork(args.name, args.password)

    elif args.join_network:
        if not args.name or not args.password:
            log_error("加入网络需要指定 --name 和 --password")
            return
        manager.SetUpEasyTierNetwork(args.name, args.password)

    elif args.list:
        manager.ListAllConnectionPeers()
        return

    if args.create_network or args.join_network:
        log_info("使用 Ctrl+C 以退出网络/关闭服务")
        try:
            manager.wait()
        except KeyboardInterrupt:
            manager.StopService()
            log_info("已停止 EasyTier Network 服务")


if __name__ == "__main__":
    main()