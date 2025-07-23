#!/usr/bin/env python3
import os
import sys
import time
import traceback

# 日志级别颜色定义
class LogColors:
    INFO = '\033[92m'
    WARN = '\033[93m'
    ERROR = '\033[91m'
    DEBUG = '\033[94m'
    CLEAR = '\033[0m'


def log_info(message):
    print(f"{LogColors.INFO}[INFO]{LogColors.CLEAR} {message}")

def log_warn(message):
    print(f"{LogColors.WARN}[WARN]{LogColors.CLEAR} {message}")

def log_error(message):
    print(f"{LogColors.ERROR}[ERROR]{LogColors.CLEAR} {message}", file=sys.stderr)

def log_debug(message):
    print(f"{LogColors.DEBUG}[DEBUG]{LogColors.CLEAR} {message}")

# 主函数
# 流程：
# 1. 等待1秒，防止程序卡在启动界面
# 2. 获取程序目录
# 3. 读取命令行参数，获取临时文件路径
# 4. 检查临时文件（bex.exe.tmp，通过Github Release下载）是否存在
# 5. 检查目标程序是否存在，如果存在，删除
# 6. 尝试重命名临时文件为目标程序
# 7. 输出完成信息，返回退出码
def main():
    time.sleep(1)
    return_code = 1
    try:
        # 获取程序目录
        if getattr(sys, 'frozen', False):
            base_dir = os.path.dirname(sys.executable)
        else:
            base_dir = os.path.dirname(os.path.abspath(__file__))

        target_exe = os.path.join(base_dir, "bex.exe")
        tmp_file = sys.argv[2] if len(sys.argv) > 2 else os.path.join(base_dir, "bex.exe.tmp")

        print(f"-={LogColors.INFO}BeaconEX {LogColors.DEBUG}Updater {LogColors.WARN}[v1.1.0]{LogColors.CLEAR}=-")
        log_info("Starting update process...")
        log_info(f"Working directory: {base_dir}")

        if not os.path.exists(tmp_file):
            log_error(f"Cannot find update file: {tmp_file}")
            return 1

        if os.path.exists(target_exe):
            log_info("Removing old version...")
            for retry in range(5):
                try:
                    os.remove(target_exe)
                    log_info("Old version removed successfully!")
                    break
                except PermissionError:
                    log_warn(f"Failed to remove old version, retrying ({retry + 1}/5)...")
                    time.sleep(1)
            else:
                log_error("Failed to remove old version. Please ensure the program is closed and try again.")
                return 1

        try:
            os.rename(tmp_file, target_exe)
            log_info("Update completed successfully!")
            return_code = 0
        except Exception as e:
            log_error(f"Failed to rename file: {e}")
            log_debug(traceback.format_exc())
            return 1

    except Exception as e:
        log_error(f"Update failed: {e}")
        log_debug(traceback.format_exc())
        return_code = 1

    if return_code != 0:
        input("Press Enter to exit...")
    return return_code


if __name__ == "__main__":
    sys.exit(main())