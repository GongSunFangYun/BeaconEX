"""------------------------外部库------------------------"""
import argparse
import asyncio
import sys
import time

from concurrent.futures import ThreadPoolExecutor
from icmplib import ping as icmp_ping
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import log_info, log_warn, log_error, TextFormat
from bexlib2.rhnis import ResolveDomainName

def TargetParser(target: str) -> str: # 处理目标，将端口忽略掉
    if ':' in target:
        return target.split(':', 1)[0]
    return target

async def LoopPing(target: str, interval: float = 1.0) -> None:
    host = TargetParser(target)
    ip, _, _ = ResolveDomainName(host, "java")

    log_info(f"开始以 {TextFormat.YELLOW}{interval}{TextFormat.CLEAR} 秒间隔持续 Ping {host} [{ip}]")
    log_info("按 Ctrl+C 停止测试")

    success_count = 0
    total_count = 0
    total_latency = 0
    min_latency = float('inf')
    max_latency = 0
    consecutive_timeouts = 0
    last_latency = None

    try:
        while True:
            total_count += 1
            sequence = total_count

            try:
                # 单次 Ping
                start_time = time.time()
                result = await asyncio.get_event_loop().run_in_executor(
                    None,
                    lambda: icmp_ping(ip, count=1, timeout=2, privileged=False)
                )
                latency = (time.time() - start_time) * 1000

                if result.is_alive:
                    success_count += 1
                    total_latency += latency
                    min_latency = min(min_latency, latency)
                    max_latency = max(max_latency, latency)
                    consecutive_timeouts = 0

                    # 计算抖动值
                    jitter = 0
                    if last_latency is not None:
                        jitter = latency - last_latency

                    last_latency = latency

                    # 实时统计
                    current_avg = total_latency / success_count if success_count > 0 else 0
                    loss_rate = (total_count - success_count) / total_count * 100

                    # 格式化抖动值显示
                    if jitter > 0:
                        jitter_display = f"{TextFormat.RED}+{jitter:.2f}ms{TextFormat.CLEAR}"
                    elif jitter < 0:
                        jitter_display = f"{TextFormat.GREEN}{jitter:.2f}ms{TextFormat.CLEAR}"
                    else:
                        jitter_display = f"{TextFormat.PURPLE}±0ms{TextFormat.CLEAR}"

                    log_info(
                        f"[{sequence}] 延迟: {TextFormat.GREEN}{latency:.2f}ms{TextFormat.CLEAR} | "
                        f"抖动: {jitter_display} | "
                        f"平均: {TextFormat.BLUE}{current_avg:.2f}ms{TextFormat.CLEAR} | "
                        f"丢包: {TextFormat.RED}{loss_rate:.1f}%{TextFormat.CLEAR}"
                    )
                else:
                    consecutive_timeouts += 1
                    loss_rate = (total_count - success_count) / total_count * 100
                    last_latency = None

                    log_warn(
                        f"[{sequence}] {TextFormat.RED}超时{TextFormat.CLEAR} | "
                        f"连续超时: {TextFormat.YELLOW}{consecutive_timeouts}次{TextFormat.CLEAR} | "
                        f"丢包: {TextFormat.RED}{loss_rate:.1f}%{TextFormat.CLEAR}"
                    )

                await asyncio.sleep(interval)

            except Exception as e:
                consecutive_timeouts += 1
                last_latency = None
                log_warn(f"[{sequence}] 错误: {str(e)}")
                await asyncio.sleep(interval)

    except KeyboardInterrupt:
        raise  # 重新抛出异常，让外层处理
    finally:
        # 无论如何统计结果都将会显示
        if total_count > 0:
            log_info("已结束持续Ping测试，结果统计如下：")
            _ShowStatResult(success_count, total_count, total_latency, min_latency, max_latency)

def _ShowStatResult(success_count: int, total_count: int, total_latency: float,
                        min_latency: float, max_latency: float):
    loss_rate = (total_count - success_count) / total_count * 100
    avg_latency = total_latency / success_count if success_count > 0 else 0

    log_info(f"总测试次数: {TextFormat.YELLOW}{total_count}{TextFormat.CLEAR}")
    log_info(f"成功响应: {TextFormat.GREEN}{success_count}{TextFormat.CLEAR} 次")
    log_info(f"丢包率: {TextFormat.RED}{loss_rate:.2f}%{TextFormat.CLEAR}")

    if success_count > 0:
        log_info(f"最小延迟: {TextFormat.GREEN}{min_latency:.2f}ms{TextFormat.CLEAR}")
        log_info(f"最大延迟: {TextFormat.YELLOW}{max_latency:.2f}ms{TextFormat.CLEAR}")
        log_info(f"平均延迟: {TextFormat.BLUE}{avg_latency:.2f}ms{TextFormat.CLEAR}")

# 基于异步执行的ICMP协议Ping测试，提高了运行速度
async def AsyncPing(target: str, count: int = 4) -> None:
    host = TargetParser(target)

    # 单次Ping
    # noinspection PyShadowingNames
    async def _SinglePing(ip: str) -> tuple[bool, float]:
        try:
            start = time.time()
            # icmplib的ping函数本身是同步的，需要在线程中运行
            result = await loop.run_in_executor(
                executor,
                lambda: icmp_ping(ip, count=1, timeout=2, privileged=False)
            )
            latency = (time.time() - start) * 1000
            return result.is_alive, latency
        except Exception as e:
            log_warn(f"单次Ping异常: {str(e)}")
            return False, 0

    ip, _, _ = ResolveDomainName(host, "java")
    log_info(f"正在Ping {host} [{ip}]")

    # 定义相关数据
    success_count = 0
    total_latency = 0
    min_latency = float('inf')
    max_latency = 0

    # 定义线程池
    with ThreadPoolExecutor(max_workers=10) as executor:
        loop = asyncio.get_event_loop()
        tasks = [_SinglePing(ip) for _ in range(count)]
        # 异步执行Ping
        for i, task in enumerate(asyncio.as_completed(tasks), 1):
            try:
                time.sleep(0.2)
                success, latency = await task
                if success:
                    success_count += 1
                    total_latency += latency
                    min_latency = min(min_latency, latency)
                    max_latency = max(max_latency, latency)
                    log_info(f"[{i}/{count}] 正在Ping {ip}: 延迟 {TextFormat.PURPLE}{latency:.2f}{TextFormat.CLEAR}ms")
                else:
                    log_warn(f"[{i}/{count}] 正在Ping {ip}: 超时")
            except Exception as e:
                log_warn(f"[{i}/{count}] 正在Ping {ip}: 错误({str(e)})")

    # 计算统计数据
    loss_rate = (count - success_count) / count * 100
    avg_latency = total_latency / success_count if success_count > 0 else 0

    # 返回结果
    log_info("Ping结果:")
    log_info(f"已对 {ip} 进行 {TextFormat.YELLOW}{count}{TextFormat.CLEAR} 次Ping")
    log_info(f"成功接收: {TextFormat.GREEN}{success_count}{TextFormat.CLEAR} 次")
    log_info(f"丢包率: {TextFormat.RED}{loss_rate:.2f}{TextFormat.CLEAR}% ({success_count}/{count})")
    if success_count > 0:
        log_info(
            f"延迟统计: 平均 {TextFormat.BLUE}{avg_latency:.2f}{TextFormat.CLEAR}ms | 最小 {TextFormat.GREEN}{min_latency:.2f}{TextFormat.CLEAR}ms | 最大 {TextFormat.YELLOW}{max_latency:.2f}{TextFormat.CLEAR}ms")


def Ping(target: str, count: int = 4, continuous: bool = False, interval: float = 1.0) -> None:
    try:
        if sys.platform == 'win32':
            asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())

        if continuous:
            # 对于持续模式，我直接运行并处理中断
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            try:
                loop.run_until_complete(LoopPing(target, interval))
            except KeyboardInterrupt:
                # 这里不需要额外处理，LoopPing 中已经处理了
                pass
            finally:
                loop.close()
        else:
            asyncio.run(AsyncPing(target, count))

    except KeyboardInterrupt:
        # 只在普通模式下显示这个信息
        if not continuous:
            log_info("用户中断测试")
    except Exception as e:
        log_error(f"Ping测试失败: {str(e)}")
        log_info("请检查网络连接或目标地址是否正确")

def main():
    parser = argparse.ArgumentParser(
        description='执行Ping网络测试',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-t', '--target', required=True,
                       help='要Ping的目标地址 (格式: hostname[:port] 或 IP[:port])\n'
                            '示例: example.com 或 192.168.1.1:25565\n'
                            '端口部分会被忽略')

    # Ping 模式选择
    parser.add_argument('-r', '--repeat', action='store_true',
                       help='持续 Ping，直到手动停止 (Ctrl+C)')

    # Ping 参数
    parser.add_argument('-pf', '--ping-frequency', type=int, default=4,
                       help='普通模式下的 Ping 执行次数 (默认: 4)')
    parser.add_argument('-pi', '--ping-interval', type=float, default=1.0,
                       help='持续 Ping 的间隔时间，单位秒 (默认: 1.0)')
    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')


    args = parser.parse_args()

    try:
        # 参数验证
        if args.ping_frequency < 1 or args.ping_frequency > 1000:
            log_error("Ping次数必须在1-1000之间")
            sys.exit(1)

        if args.ping_interval < 0.1 or args.ping_interval > 60:
            log_error("间隔时间必须在0.1-60秒之间")
            sys.exit(1)

        # 执行Ping测试
        if args.repeat:
            Ping(args.target, continuous=True, interval=args.ping_interval)
        else:
            Ping(args.target, count=args.ping_frequency)

    except Exception as e:
        log_error(f"Ping测试失败: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()