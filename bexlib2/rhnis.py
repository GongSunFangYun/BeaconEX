# rhnis.py
import socket
import sys
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from typing import List, Dict, Any, Tuple

import dns.exception
import dns.rdatatype
import dns.resolver

from .lg4pb import log_error, log_info, log_warn


class ParallelDNSResolver:
    def __init__(self, timeout: int = 2, retries: int = 1):
        # 初始化并行DNS解析器
        self.timeout = timeout
        self.retries = retries
        self.results = {}
        self.lock = threading.Lock()

        # 配置 DNS 解析器参数
        self.resolver = dns.resolver.Resolver()
        self.resolver.timeout = timeout
        self.resolver.lifetime = timeout

        # 配置快速公共DNS服务器列表
        self.resolver.nameservers = [
            '119.29.29.29',  # Public+ DNS
            '208.67.222.222',  # OpenDNS
            '223.5.5.5',  # 阿里云 DNS
            '114.114.114.114'  # 114DNS
        ]

    def ResolveAll(self, host: str) -> Dict[str, Any]:
        # 并行解析A、AAAA、SRV记录的主方法
        self.results = {
            'A': [],
            'AAAA': [],
            'SRV': [],
            'errors': []
        }

        # 如果是IP地址，直接返回结果，不进行DNS查询
        if self._IsValidAddress_PrivateFunction(host):
            self.results['A'] = [host] if '.' in host else []
            self.results['AAAA'] = [host] if ':' in host else []
            return self.results

        # 定义并行解析任务
        tasks = [
            ('A', self._Resolve_A_Parallel),
            ('AAAA', self._Resolve_AAAA_Parallel),
            ('SRV', self._Resolve_SRV_Parallel)
        ]

        # 使用线程池并行执行DNS查询
        with ThreadPoolExecutor(max_workers=3) as executor:
            future_to_type = {
                executor.submit(func, host): rtype for rtype, func in tasks
            }

            # 收集所有任务结果
            for future in as_completed(future_to_type):
                rtype = future_to_type[future]
                try:
                    result = future.result(timeout=self.timeout + 1)
                    with self.lock:
                        self.results[rtype] = result
                except Exception as e:
                    with self.lock:
                        self.results['errors'].append(f"{rtype}解析失败: {e}")

        return self.results

    def _Resolve_A_Parallel(self, host: str) -> List[str]:
        # 解析IPv4地址记录
        if self._IsValidAddress_PrivateFunction(host):
            return [host] if '.' in host else []

        # 带重试机制的A记录解析
        for attempt in range(self.retries + 1):
            try:
                answers = self.resolver.resolve(host, 'A', raise_on_no_answer=False)
                if answers:
                    ips = [answer.address for answer in answers]
                    return ips
            except (dns.resolver.NoAnswer, dns.resolver.NXDOMAIN):
                return []
            except (dns.resolver.Timeout, dns.exception.Timeout):
                if attempt == self.retries:
                    return []
                continue
            except Exception:
                return []
        return []

    def _Resolve_AAAA_Parallel(self, host: str) -> List[str]:
        # 解析IPv6地址记录
        if self._IsValidAddress_PrivateFunction(host):
            return [host] if ':' in host else []

        # 带重试机制的AAAA记录解析
        for attempt in range(self.retries + 1):
            try:
                answers = self.resolver.resolve(host, 'AAAA', raise_on_no_answer=False)
                if answers:
                    ips = [answer.address for answer in answers]
                    return ips
            except (dns.resolver.NoAnswer, dns.resolver.NXDOMAIN):
                return []
            except (dns.resolver.Timeout, dns.exception.Timeout):
                if attempt == self.retries:
                    return []
                continue
            except Exception:
                return []
        return []

    def _Resolve_SRV_Parallel(self, host: str) -> List[Dict[str, Any]]:
        # 解析SRV服务记录
        if self._IsValidAddress_PrivateFunction(host):
            return []

        # 定义可能的SRV记录格式
        srv_formats = [
            f"_minecraft._tcp.{host}",
            f"_minecraft._udp.{host}",
            f"_mc._tcp.{host}",
        ]

        # 依次尝试不同的SRV记录格式
        for srv_host in srv_formats:
            for attempt in range(self.retries + 1):
                try:
                    results = self._ResolveSingle_SRV_Parallel(srv_host)
                    if results:
                        return results
                    break
                except (dns.resolver.Timeout, dns.exception.Timeout):
                    if attempt == self.retries:
                        break
                    continue
                except Exception:
                    break

        return []

    def _ResolveSingle_SRV_Parallel(self, srv_host: str) -> List[Dict[str, Any]]:
        # 解析单个SRV记录
        try:
            answers = self.resolver.resolve(srv_host, 'SRV', raise_on_no_answer=False)
            if not answers:
                return []

            # 处理SRV记录结果
            results = []
            for answer in answers:
                results.append({
                    "port": answer.port,
                    "priority": answer.priority,
                    "weight": answer.weight,
                    "original_target": str(answer.target).rstrip('.'),
                    "srv_record": srv_host
                })

            return results

        except (dns.resolver.NoAnswer, dns.resolver.NXDOMAIN):
            return []
        except (dns.resolver.Timeout, dns.exception.Timeout):
            raise
        except Exception:
            return []

    @staticmethod
    def _IsValidAddress_PrivateFunction(host: str) -> bool:
        # 快速验证IP地址格式
        if not host or ' ' in host:
            return False

        # 检查IPv4格式
        if host.count('.') == 3 and all(part.isdigit() for part in host.split('.')):
            try:
                socket.inet_pton(socket.AF_INET, host)
                return True
            except socket.error:
                pass

        # 检查IPv6格式
        if ':' in host:
            try:
                socket.inet_pton(socket.AF_INET6, host)
                return True
            except socket.error:
                pass

        return False


def ResolveDomainName(target: str, server_type: str = "java") -> Tuple[str, int, bool]:
    # 主域名解析函数
    import time
    start_time = time.time()

    # 提取主机部分进行IP地址检查
    host_part = target.split(':')[0]
    is_ip = _IsValidAddress_PublicMethod(host_part)

    # 解析主机名和端口
    host, user_port = _TargetParser(target, server_type)

    # 检查用户是否明确指定了端口
    user_specified_port = ':' in target

    # 如果是IP地址，直接处理返回
    if is_ip:
        ip, port = _HandleIPAddress(target, server_type)
        return ip, port, False

    # 并行DNS解析
    resolver = ParallelDNSResolver(timeout=2, retries=1)
    results = resolver.ResolveAll(host)

    # 性能监控
    resolve_time = time.time() - start_time
    if resolve_time > 1.0:
        log_warn(f"DNS解析较慢: {resolve_time:.2f}秒")

    # 选择可用的IP地址（优先IPv4）
    available_ips = []
    if results['A']:
        available_ips = results['A']
    elif results['AAAA']:
        available_ips = results['AAAA']
    else:
        log_error(f"无法解析主机名: {host}")
        sys.exit(1)

    # 使用第一个可用IP
    ip = available_ips[0]
    log_info(f"成功解析 {host} 的IP与端口")

    # 端口选择逻辑：用户指定 > SRV记录 > 默认端口
    is_srv = False
    if results['SRV'] and not user_specified_port:
        srv_record = results['SRV'][0]
        port = srv_record['port']
        is_srv = True
    else:
        port = user_port

    return ip, port, is_srv


def _IsValidAddress_PublicMethod(host: str) -> bool:
    # 公开的IP地址验证方法
    return ParallelDNSResolver._IsValidAddress_PrivateFunction(host)


def _TargetParser(target: str, server_type: str) -> Tuple[str, int]:
    # 解析目标地址字符串，分离主机和端口
    if ':' in target:
        parts = target.split(':', 1)
        host = parts[0].strip()
        try:
            user_port = int(parts[1].strip())
            if not (1 <= user_port <= 65535):
                raise ValueError(log_error("端口必须在1-65535之间"))
            return host, user_port
        except ValueError:
            log_error(f"无效的端口号: {parts[1]}")
            sys.exit(1)
    else:
        host = target.strip()
        user_port = 25565 if server_type.lower() == "java" else 19132
        return host, user_port


def _HandleIPAddress(target: str, server_type: str) -> Tuple[str, int]:
    # 处理直接使用IP地址的情况
    if ':' in target:
        # IP:端口格式
        parts = target.split(':', 1)
        ip = parts[0].strip()
        try:
            port = int(parts[1].strip())
            if not (1 <= port <= 65535):
                raise ValueError(log_error("端口必须在1-65535之间"))
            return ip, port
        except ValueError:
            sys.exit(1)
    else:
        # 纯IP地址格式
        port = 25565 if server_type.lower() == "java" else 19132
        return target, port


def QuickResolve(host: str) -> str:
    # 快速解析兼容函数，只返回IP地址
    ip, port, is_srv = ResolveDomainName(host, "java")
    return ip