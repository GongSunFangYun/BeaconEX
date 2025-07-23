"""------------------------外部库------------------------"""
import requests
import time
import os
import sys
import subprocess
from concurrent.futures import ThreadPoolExecutor
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_error, log_warn, log_debug

# 为了方便起见，我使用了第三方提供的免费Github代理服务，以此提升更新速度
# 如果本地版本滞后于Github最新版本，则提示用户进行更新操作
# 如果本地版本与Github最新版本一致，则提示用户当前版本为最新版本
PROXY_SITES = [
    "https://gh-proxy.com/",
    "https://gh.nxnow.top/",
    "https://gh.b52m.cn/",
    "https://git.yylx.win/"
]

# 硬编码版本号，用于软件校验自身版本和Github 最新分发版本
def Version():
    return "1.0.3"

# 获取Github最新的Release信息
def GetLatestRelease():
    api_url = "https://api.github.com/repos/GongSunFangYun/BeaconEX/releases/latest" # 通过Github API获取我的仓库中的最新版本信息
    headers = {
        "User-Agent": "BeaconEX-Updater",
        "Accept": "application/vnd.github.v3+json"
    }
    try:
        response = requests.get(api_url, headers=headers, timeout=10)
        if response.status_code == 200:
            return response.json()
        else:
            log_error(f"GitHub API 请求失败: HTTP {response.status_code}")
            log_debug(f"响应内容: {response.text[:200]}")
    except Exception as e:
        log_error(f"获取最新版本信息时出错: {e}")
    return None

# 请求代理网站进行延迟测试
def TestProxy(url):
    try:
        start = time.time()
        headers = {"User-Agent": "BeaconEX-Updater"}
        response = requests.head(
            url,
            headers=headers,
            timeout=5,
            allow_redirects=False
        )
        response.close()
        latency = (time.time() - start) * 1000
        return url, latency
    except Exception as e:
        log_warn(f"代理测试失败: {str(e)}")
        return url, float('inf')

# 测试代理站点延迟并返回测试结果
def TestAndDisplayProxies(proxy_urls, test_message="正在测试代理站点..."):
    log_info(f"{TextFormat.CYAN}{test_message}{TextFormat.CLEAR}")

    with ThreadPoolExecutor() as executor:
        results = list(executor.map(TestProxy, proxy_urls))

    working_proxies = []
    for url, latency in results:
        proxy_name = url.split('/')[2]
        if latency < float('inf'):
            working_proxies.append((url, latency, proxy_name))
            status = f"{TextFormat.GREEN}✓{TextFormat.CLEAR}"
            latency_display = f"{latency:.2f}ms"
        else:
            status = f"{TextFormat.RED}✗{TextFormat.CLEAR}"
            latency_display = f"{TextFormat.RED}超时{TextFormat.CLEAR}"

        proxy_display = proxy_name.ljust(20)
        log_info(f"  {status} {proxy_display} 延迟: {TextFormat.YELLOW}{latency_display}{TextFormat.CLEAR}")

    return working_proxies

# 综合分析，选取最低代理节点
def SelectBestProxy(github_url):
    working_proxies = TestAndDisplayProxies(github_url, "正在测试代理站点延迟...")

    if not working_proxies:
        log_error("所有代理站点均不可用！")
        return None

    working_proxies.sort(key=lambda x: x[1])
    best_url, best_latency, best_proxy = working_proxies[0]
    log_info(
        f"{TextFormat.GREEN}已优选代理站点: {best_proxy} (延迟最低: {best_latency:.2f}ms){TextFormat.CLEAR}")
    return best_url

# 检查版本是否需要更新
def CheckForUpdates():
    current_version = Version()
    log_info(f"当前版本: {current_version}")

    release_info = GetLatestRelease()
    if not release_info:
        return False

    latest_version = release_info['tag_name']
    log_info(f"最新版本: {latest_version}")

    current_parts = list(map(int, current_version.split('.')))
    latest_parts = list(map(int, latest_version.split('.')))

    if latest_parts > current_parts:
        log_info(f"{TextFormat.GREEN}有新版本可用!{TextFormat.CLEAR}")
        log_info(f"发布说明: \n{release_info.get('body', '无')}")
        log_info(f"{TextFormat.YELLOW}使用 {sys.argv[0]} -update 进行自动更新{TextFormat.CLEAR}")
        return True
    else:
        log_info("当前已是最新版本")
        return False

# 使用最优代理进行更新操作
def SelectProxy(proxy_urls):
    working_proxies = TestAndDisplayProxies(proxy_urls)

    if not working_proxies:
        return None

    working_proxies.sort(key=lambda x: x[1])
    return working_proxies[0][0]

# 下载Github最新版本并进行下载
def DownloadUpdate():
    current_version = Version()
    release_info = GetLatestRelease()
    if not release_info:
        return False

    latest_version = release_info['tag_name']
    current_parts = list(map(int, current_version.split('.')))
    latest_parts = list(map(int, latest_version.split('.')))

    if latest_parts <= current_parts:
        log_info("当前已是最新版本，无需更新")
        return False

    # 获取程序安装目录
    if getattr(sys, 'frozen', False):
        base_dir = os.path.dirname(sys.executable)
    else:
        base_dir = os.path.dirname(os.path.abspath(__file__))

    asset = next((a for a in release_info.get('assets', [])
                if a['name'].lower().endswith('.exe')), None)
    if not asset:
        log_error("未找到可下载的更新文件 (bex.exe）")
        return False

    download_url = asset['browser_download_url']
    proxy_urls = [proxy + download_url for proxy in PROXY_SITES]

    working_proxies = TestAndDisplayProxies(proxy_urls, "正在测试代理站点...")

    if working_proxies:
        working_proxies.sort(key=lambda x: x[1])
        best_url, best_latency, best_proxy = working_proxies[0]
        log_info(
            f"{TextFormat.GREEN}已选择最优代理: {best_proxy} (延迟: {best_latency:.2f}ms){TextFormat.CLEAR}")
    else:
        log_error("所有代理均不可用，尝试直接下载...")
        best_url = download_url

    try:
        log_info(f"正在从 {best_url} 下载更新: {asset['name']}")

        response = requests.get(best_url, stream=True, timeout=30)
        response.raise_for_status()

        total_size = int(response.headers.get('content-length', 0))
        block_size = 8192
        progress_bar_width = 50
        downloaded = 0

        # 确保下载到程序目录而不是用户目录
        filename = os.path.join(base_dir, asset['name'] + '.tmp')
        with open(filename, 'wb') as f:
            for chunk in response.iter_content(chunk_size=block_size):
                f.write(chunk)
                downloaded += len(chunk)

                percent = min(100, downloaded * 100 / total_size) if total_size > 0 else 0
                filled_length = int(progress_bar_width * downloaded // total_size) if total_size > 0 else 0

                bar = '━' * filled_length + '-' * (progress_bar_width - filled_length)
                sys.stdout.write(f"\r|{bar}| {percent:.2f}% ({downloaded}/{total_size} bytes)")
                sys.stdout.flush()

        print() # 换行
        log_info(f"下载完成: {filename}")

        update_exe_path = os.path.join(base_dir, "Update.exe")
        log_debug(f"查找Update.exe路径: {update_exe_path}")

        if not os.path.exists(update_exe_path):
            log_error(f"未找到 Update.exe - 检查路径: {update_exe_path}")
            try:
                dir_contents = os.listdir(base_dir)
                log_debug(f"目录内容: {dir_contents}")
            except Exception as e:
                log_debug(f"无法列出目录内容: {e}")
            return False

        current_exe_path = sys.executable if getattr(sys, 'frozen', False) else __file__

        # 确保Update.exe在程序目录中运行（使用psutil启动更新程序）
        subprocess.Popen([update_exe_path, current_exe_path, filename],
                        cwd=base_dir,  # 设置工作目录
                        shell=True,
                        creationflags=subprocess.DETACHED_PROCESS | subprocess.CREATE_NEW_PROCESS_GROUP)

        log_info("更新程序已启动，主程序即将退出...")
        return True

    except Exception as e:
        log_error(f"下载更新失败: {e}")
        return False