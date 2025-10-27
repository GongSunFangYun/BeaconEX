"""------------------------外部库------------------------"""
import argparse
import ctypes
import json
import os
import re
import sys
import time
import psutil

from pymem.ressources import kernel32
from pymem.ressources.structure import MEMORY_STATE, MEMORY_PROTECTION
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import log_info, log_error, log_warn, TextFormat

# 修改配置文件存储位置到 _internal 文件夹
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
INTERNAL_DIR = os.path.join(SCRIPT_DIR, "_internal")
CONFIG_FILE_PATH = os.path.join(INTERNAL_DIR, "injector.config.json")

# 确保 _internal 目录存在
if not os.path.exists(INTERNAL_DIR):
    os.makedirs(INTERNAL_DIR, exist_ok=True)

class MBDLInjector:
    def __init__(self):
        self.Parser = self.CreateParser()

    @staticmethod
    def ReadConfig(): # 读取配置文件
        try:
            if not os.path.exists(CONFIG_FILE_PATH):
                log_error("配置文件不存在，请先设置配置")
                return None

            with open(CONFIG_FILE_PATH, 'r', encoding='utf-8') as f:
                config = json.load(f)
            return config
        except Exception as e:
            log_error(f"读取配置文件失败: {str(e)}")
            return None

    def CreateParser(self): # 创建命令行解析器
        parser = argparse.ArgumentParser(
            description='DLL注入工具'
        )
        parser.add_argument('-about', '--about', action='version',
                            version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                            help='显示关于信息')
        # 注入参数组
        inject_group = parser.add_argument_group('DLL注入参数')
        inject_group.add_argument(
            '-dp', '--dll-path',
            dest='dll_path',
            help='DLL文件路径 (与-i互斥)'
        )
        inject_group.add_argument(
            '-ct', '--custom-target',
            dest='process_name',
            default='Minecraft.Windows.exe',
            help='目标进程名称 (默认: Minecraft.Windows.exe)'
        )
        inject_group.add_argument(
            '-i', '--inject',
            dest='inject',
            action='store_true',
            help='使用上次注入的DLL路径自动注入 (与-dp互斥)'
        )

        # 任务模式组
        task_group = parser.add_argument_group('定时任务参数')
        task_group.add_argument(
            '-tm', '--task-mode',
            dest='task_time',
            type=self.ValidateTimeFormat,
            nargs='?',
            const='0s',  # 默认值，当只使用 -tm 不带参数时
            help='定时注入模式，后接时间 (格式: 数字m/数字s/数字m数字s，例如: 1m30s)'
        )

        # 配置组
        config_group = parser.add_argument_group('配置参数')
        config_group.add_argument(
            '-rc', '--reset-config',
            dest='reset_config',
            action='store_true',
            help='重置配置文件'
        )

        return parser

    @staticmethod
    def ValidateTimeFormat(time_str: str): # 验证时间格式（用在-tm参数的）
        if not re.fullmatch(r'^(\d+m)?(\d+s)?$', time_str.lower()):
            raise log_error("时间格式必须为 1m, 30s, 或 1m30s")

        minutes = 0
        seconds = 0

        if 'm' in time_str:
            minutes = int(time_str.split('m')[0])
            time_str = time_str.split('m')[-1]

        if 's' in time_str:
            seconds = int(time_str.split('s')[0])

        if minutes < 0 or seconds < 0:
            raise log_error("时间值不能为负数")
        if minutes > 60:
            raise log_error("分钟数不能超过60")
        if seconds > 59:
            raise log_error("秒数不能超过59")
        if minutes == 0 and seconds == 0:
            raise log_error("时间必须大于0")

        return minutes, seconds
    @staticmethod
    def ValidateArgs(args): # 验证参数（配合CreateParser()使用）
        # 检查互斥参数
        if args.inject and args.dll_path:
            log_error("不能同时使用 -i/--inject 和 -dp/--dll-path")

        # 检查必须提供-dp或-i或-tm或-rc
        if (not args.inject and not args.dll_path and
                not args.task_time and not args.reset_config):
            log_error("必须提供 -dp/--dll-path 或 -i/--inject 或 -tm/--task-mode 或 -rc/--reset-config")

        if args.inject and not args.task_time:
            # 检查配置文件是否存在
            if not os.path.exists(CONFIG_FILE_PATH):
                log_error("配置文件不存在，请先设置配置")

            # 检查目标DLL文件是否存在
        if args.dll_path and not os.path.isfile(args.dll_path):
            log_error(f"DLL文件未找到: {args.dll_path}")

    @staticmethod
    def InjectDll(process_name: str, dll_path: str): # 此方法用于注入DLL
        # DLL注入原理说明:
        # 1. 进程查找:
        #    - 通过psutil遍历系统进程，根据进程名找到目标进程的PID
        # 2. 进程访问:
        #    - 使用OpenProcess打开目标进程，获取进程句柄
        #    - 需要PROCESS_ALL_ACCESS权限来进行内存操作
        # 3. 函数定位:
        #    - 获取kernel32.dll的模块句柄
        #    - 找到LoadLibraryW函数的地址
        #    - LoadLibraryW是Windows API，用于动态加载DLL
        # 4. 内存分配:
        #    - 在目标进程空间中分配内存(VirtualAllocEx)
        #    - 设置内存保护为PAGE_READWRITE可读写
        # 5. 路径写入:
        #    - 将DLL路径字符串编码为UTF-16LE格式
        #    - 使用WriteProcessMemory写入到分配的内存中
        # 6. 远程执行:
        #    - 创建远程线程(CreateRemoteThread)
        #    - 线程入口点为LoadLibraryW函数
        #    - 参数为DLL路径在目标进程中的内存地址
        # 7. DLL加载:
        #    - 远程线程执行LoadLibraryW(分配的DLL路径地址)
        #    - 目标进程加载指定的DLL文件
        #    - DLL的DllMain函数会被调用
        # 8. 资源清理:
        #    - 等待线程执行完成
        #    - 释放分配的内存
        #    - 关闭进程和线程句柄
        try:
            process_id = None
            for proc in psutil.process_iter(['pid', 'name']):
                if proc.info['name'].lower() == process_name.lower():
                    process_id = proc.info['pid']
                    break

            if not process_id:
                log_error(f"未找到目标进程: {process_name}")
                return False

            log_info(f"开始进行DLL注入：")
            log_info(f"目标进程: {process_name} [PID：{process_id}] | DLL文件: {dll_path}")

            # 打开进程
            process_handle = kernel32.OpenProcess(
                0x1F0FFF,  # 使用PROCESS_ALL_ACCESS权限操作
                False,
                process_id
            )

            # 打开进程相关处理
            if not process_handle:
                log_error(f"{TextFormat.YELLOW}[1/9]{TextFormat.CLEAR} 无法打开目标进程，可能权限不足")
                return False
            log_info(f"{TextFormat.BLUE}[1/9]{TextFormat.CLEAR} 打开目标线程并获取进程句柄")

            # 获取kernel32模块句柄
            kernel32_handle = kernel32.GetModuleHandleA(b"kernel32.dll")
            if not kernel32_handle:
                log_error(f"{TextFormat.YELLOW}[2/9]{TextFormat.CLEAR} 无法获取kernel32模块句柄")
                kernel32.CloseHandle(process_handle)
                return False
            log_info(f"{TextFormat.BLUE}[2/9]{TextFormat.CLEAR} 获取kernel32模块句柄")

            # 获取LoadLibraryW函数地址
            load_library_addr = kernel32.GetProcAddress(kernel32_handle, b"LoadLibraryW")
            if not load_library_addr:
                log_error(f"{TextFormat.YELLOW}[3/9]{TextFormat.CLEAR} 无法获取LoadLibraryW函数地址")
                kernel32.CloseHandle(process_handle)
                return False
            log_info(f"{TextFormat.BLUE}[3/9]{TextFormat.CLEAR} 获取LoadLibraryW函数地址: 0x{load_library_addr:X}")

            # 准备DLL路径（UTF-16编码）
            dll_path_utf16 = dll_path.encode('utf-16le') + b'\x00\x00'
            log_info(
                f"{TextFormat.BLUE}[4/9]{TextFormat.CLEAR} 编码DLL路径为长度 {len(dll_path_utf16)} 字节的UTF-16LE字符串")

            # 分配内存
            buffer_size = 260
            arg_address = kernel32.VirtualAllocEx(
                process_handle,
                None,
                buffer_size,
                MEMORY_STATE.MEM_COMMIT.value | MEMORY_STATE.MEM_RESERVE.value,
                MEMORY_PROTECTION.PAGE_READWRITE.value
            )

            if not arg_address:
                log_error(f"{TextFormat.YELLOW}[5/9]{TextFormat.CLEAR} 无法在目标进程中分配内存")
                kernel32.CloseHandle(process_handle)
                return False
            log_info(f"{TextFormat.BLUE}[5/9]{TextFormat.CLEAR} 分配内存到地址 0x{arg_address:X}")

            # 写入DLL路径到目标进程内存
            written = ctypes.c_size_t(0)
            kernel32.WriteProcessMemory(
                process_handle,
                arg_address,
                dll_path_utf16,
                len(dll_path_utf16),
                ctypes.byref(written)
            )

            if written.value == 0:
                log_error(f"{TextFormat.YELLOW}[6/9]{TextFormat.CLEAR} 无法将DLL路径写入目标进程内存")
                kernel32.VirtualFreeEx(process_handle, arg_address, 0, 0x8000)
                kernel32.CloseHandle(process_handle)
                return False
            log_info(f"{TextFormat.BLUE}[6/9]{TextFormat.CLEAR} 将DLL路径写入到地址 0x{arg_address:X}")

            # 创建远程线程
            thread_id = ctypes.c_ulong(0)
            thread_handle = kernel32.CreateRemoteThread(
                process_handle,
                None,
                0,
                load_library_addr,
                arg_address,
                0,
                ctypes.byref(thread_id)
            )

            if not thread_handle:
                log_error(f"{TextFormat.YELLOW}[7/9]{TextFormat.CLEAR} 无法创建远程线程")
                kernel32.VirtualFreeEx(process_handle, arg_address, 0, 0x8000)
                kernel32.CloseHandle(process_handle)
                return False

            # 等待线程完成
            wait_result = kernel32.WaitForSingleObject(thread_handle, 5000)

            if wait_result == 0:
                log_info(
                    f"{TextFormat.BLUE}[7/9]{TextFormat.CLEAR} 创建远程线程执行LoadLibraryW [TID: {thread_id.value}]")
            else:
                log_warn(f"{TextFormat.CYAN}[7/9]{TextFormat.CLEAR} 远程线程执行超时，但注入可能仍然成功")

            # 获取线程退出码
            exit_code = ctypes.c_ulong(0)
            if kernel32.GetExitCodeThread(thread_handle, ctypes.byref(exit_code)):
                if exit_code.value == 0:
                    log_error(f"{TextFormat.YELLOW}[8/9]{TextFormat.CLEAR} DLL注入失败，LoadLibraryW返回NULL")
                    kernel32.CloseHandle(thread_handle)
                    kernel32.VirtualFreeEx(process_handle, arg_address, 0, 0x8000)
                    kernel32.CloseHandle(process_handle)
                    return False
                else:
                    log_info(f"{TextFormat.BLUE}[8/9]{TextFormat.CLEAR} 在基地址 0x{exit_code.value:X} 成功加载DLL")
            else:
                log_warn(f"{TextFormat.CYAN}[8/9]{TextFormat.CLEAR} 无法获取线程退出码，但注入可能成功")

            # 清理资源
            kernel32.CloseHandle(thread_handle)
            kernel32.VirtualFreeEx(process_handle, arg_address, 0, 0x8000)
            kernel32.CloseHandle(process_handle)
            log_info(f"{TextFormat.BLUE}[9/9]{TextFormat.CLEAR} 关闭线程/进程句柄并释放分配的内存")

            log_info("DLL注入完毕！")
            return True

        except Exception as e:
            log_error(f"注入过程中发生错误: {str(e)}")
            return False

    def HandleTaskMode(self, args): # 定时任务模式(Task Mode)
        # 功能: 延迟指定时间后自动注入DLL
        # 时间格式:
        #   "30s"    = 30秒
        #   "1m"     = 1分钟
        #   "1m30s"  = 1分30秒
        # 使用方式:
        #   -tm 30s           # 30秒后用上次DLL注入
        #   -tm 1m30s -dp test.dll  # 1分30秒后注入test.dll
        minutes, seconds = args.task_time
        total_seconds = minutes * 60 + seconds

        # 获取DLL路径
        if args.inject:
            config = self.ReadConfig()
            if not config:
                return
            dll_path = config.get('last_inject_path', '')
            if not dll_path:
                log_error("在配置文件中找不到上次注入的DLL路径")
                return
        else:
            dll_path = args.dll_path

        log_info("使用 Ctrl+C 取消注入")
        log_info(f"定时注入模式 - 等待时间: {minutes}分{seconds}秒")

        try:
            log_info(f"等待 {total_seconds} 秒后进行注入...")

            # 单条信息刷新倒计时
            for remaining in range(total_seconds, -1, -1):  # 改为-1，包含0秒
                mins, secs = divmod(remaining, 60)
                timestamp = time.strftime(f"{TextFormat.BRIGHT_BLUE}%Y-%m-%d{TextFormat.CLEAR} {TextFormat.BRIGHT_YELLOW}%H:%M:%S{TextFormat.CLEAR}", time.localtime())
                sys.stdout.write(
                    f"\r{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} 剩余时间: {mins:02d}分{secs:02d}秒")
                sys.stdout.flush()
                if remaining > 0:  # 只有大于0秒时才sleep
                    time.sleep(1)

            print()
            success = self.InjectDll(args.process_name, dll_path)

            if not success:
                log_error("定时注入失败！")
        # 通过捕获KeyboardInterrupt来取消注入任务
        except KeyboardInterrupt:
            print()
            log_info("注入任务已取消")
            return

    def main(self): # 主函数
        args = self.Parser.parse_args()
        self.ValidateArgs(args)

        if args.reset_config:
            log_info("正在重置配置文件...")
            self.ResetConfig()
            log_info("配置文件重置完成")
            return

        success = False

        if args.task_time:
            # 定时任务模式
            self.HandleTaskMode(args)
            return
        elif args.inject:
            config = self.ReadConfig()
            if not config:
                sys.exit(1)
            # 读取配置文件中的上次注入的DLL路径（injector.config.json）
            dll_path = config.get('last_inject_path', '')
            if not dll_path:
                log_error("在配置文件中找不到上次注入的DLL路径")
                log_error("请尝试重新运行注入或者删除配置文件后重试")
                sys.exit(1)
            success = self.InjectDll(args.process_name, dll_path)
        else:
            success = self.InjectDll(args.process_name, args.dll_path)

        if success and not args.inject and not args.task_time:
            self.SaveLastDllPath(args.dll_path)

    @staticmethod
    def SaveLastDllPath(dll_path): # 保存上次注入的DLL路径
        try:
            config = {}
            if os.path.exists(CONFIG_FILE_PATH):
                with open(CONFIG_FILE_PATH, 'r', encoding='utf-8') as f:
                    config = json.load(f)

            config['last_inject_path'] = dll_path

            with open(CONFIG_FILE_PATH, 'w', encoding='utf-8') as f:
                json.dump(config, f, ensure_ascii=False, indent=4)

        except Exception as e:
            log_error(f"保存DLL路径失败: {str(e)}")

    @staticmethod
    def ResetConfig(): # 重置配置文件
        try:
            config = {
                "last_inject_path": ""
            }

            with open(CONFIG_FILE_PATH, 'w', encoding='utf-8') as f:
                json.dump(config, f, ensure_ascii=False, indent=4)
        except Exception as e:
            log_error(f"重置配置文件时出错: {str(e)}")
            sys.exit(1)

if __name__ == "__main__":
    injector = MBDLInjector()
    injector.main()