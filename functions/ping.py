"""------------------------外部库------------------------"""
import asyncio
import sys
import time
from concurrent.futures import ThreadPoolExecutor
from icmplib import ping as icmp_ping
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_warn, log_error, ResolveDomainName

# 基于异步执行的ICMP协议Ping测试，提高了运行速度
async def AsyncPing(host: str, count: int = 4) -> None:
    # 单次Ping
    async def _SinglePing(ip: str) -> tuple[bool, float]:
        try:
            start = time.time()
            # icmplib的ping函数本身是同步的，需要在线程中运行
            result = await loop.run_in_executor(
                executor,
                lambda: icmp_ping(ip, count=1, timeout=2, privileged=False
            ))
            latency = (time.time() - start) * 1000
            return result.is_alive, latency
        except Exception as e:
            log_warn(f"单次Ping异常: {str(e)}")
            return False, 0
    # 调用utils.ResolveDomainName方法解析域名
    ip = ResolveDomainName(host) or host
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

# 主函数
def Ping(host: str, count: int = 4) -> None:
    try:
        if sys.platform == 'win32':
            asyncio.set_event_loop_policy(asyncio.WindowsSelectorEventLoopPolicy())
        asyncio.run(AsyncPing(host, count))
    except KeyboardInterrupt:
        log_info("用户中断测试")
    except Exception as e:
        log_error(f"Ping测试失败: {str(e)}")
        log_info("请检查网络连接或目标地址是否正确")