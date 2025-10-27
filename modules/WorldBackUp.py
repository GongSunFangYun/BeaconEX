"""------------------------外部库------------------------"""
import argparse
import datetime
import shutil
import sys
import tempfile
import time
import zipfile

from pathlib import Path
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat
from bexlib2.lg4pb import log_debug, log_info, log_warn, log_error

class MinecraftBackup:
    def __init__(self):
        # 备份工作目录路径
        self.backup_path = None
        # 要备份的目录列表
        self.selected_dirs = []
        # 备份时间间隔（秒）
        self.backup_time = None
        # 是否循环执行备份
        self.loop_execution = False
        # 备份文件保存目录
        self.backup_dir = None
        # 最大备份次数限制
        self.max_backups = None
        # 备份任务运行状态标志
        self.is_running = False
        # 已完成的备份次数计数
        self.backup_count = 0

    def ArgParser(self): # 处理参数
        parser = argparse.ArgumentParser(
            description='Minecraft服务器自动备份工具',
            formatter_class=argparse.RawDescriptionHelpFormatter,
        )

        # 必需参数
        parser.add_argument('-bp', '--backup-path',
                            required=True,
                            help='备份工作目录路径')
        parser.add_argument('-sd', '--select-dir',
                            required=True,
                            nargs='+',
                            help='要备份的目录，支持通配符和子目录，多个目录用空格分隔')

        # 可选参数
        parser.add_argument('-bt', '--backup-time',
                            help='备份时间间隔，格式如: 1h30m15s, 2h, 30m, 45s')
        parser.add_argument('-le', '--loop-execution',
                            action='store_true',
                            help='循环执行备份')
        parser.add_argument('-mx', '--max',
                            type=int,
                            help='最大备份次数（仅在定时备份时有效）')
        parser.add_argument('-about', '--about', action='version',
                            version='BeaconEX 启动脚本生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                            help='显示关于信息')

        args = parser.parse_args()

        # 验证参数
        self.backup_path = Path(args.backup_path)
        if not self.backup_path.exists():
            log_error(f"备份工作目录不存在: {self.backup_path}")
            sys.exit(1)

        # 处理选择的目录
        self.selected_dirs = []
        for pattern in args.select_dir:
            # 移除通配符，获取目录路径
            dir_path_str = pattern.rstrip('/*')
            dir_path = self.backup_path / dir_path_str

            if not dir_path.exists():
                log_warn(f"目录不存在: {dir_path}")
                continue

            self.selected_dirs.append(dir_path_str)

        if not self.selected_dirs:
            log_error("没有找到有效的备份目录")
            sys.exit(1)

        # 处理备份时间
        if args.backup_time:
            self.backup_time = self.TimeStringParser(args.backup_time)
            if self.backup_time is None:
                log_error("无效的时间格式，请使用如 1h30m15s 的格式")
                sys.exit(1)

        self.loop_execution = args.loop_execution
        self.max_backups = args.max

        # 验证最大备份次数参数
        if self.max_backups is not None and self.max_backups <= 0:
            log_error("最大备份次数必须大于0")
            sys.exit(1)

        # 设置备份输出目录为工作目录下的BEX_BackUps文件夹
        self.backup_dir = self.backup_path / "BEX_BackUps"
        self.backup_dir.mkdir(parents=True, exist_ok=True)

    @staticmethod
    def TimeStringParser(time_str): # 处理备份间隔时间
        total_seconds = 0
        current_num = ''

        for char in time_str.lower():
            if char.isdigit():
                current_num += char
            elif char in ['h', 'm', 's']:
                if not current_num:
                    return None
                if char == 'h':
                    total_seconds += int(current_num) * 3600
                elif char == 'm':
                    total_seconds += int(current_num) * 60
                elif char == 's':
                    total_seconds += int(current_num)
                current_num = ''
            else:
                return None

        # 处理最后没有单位的情况
        if current_num:
            total_seconds += int(current_num)

        return total_seconds if total_seconds > 0 else None

    @staticmethod # 把备份文件打成zip
    def ZIPCreator(zipf, directory, base_name=""):
        directory = Path(directory)
        file_count = 0
        with tempfile.TemporaryDirectory() as temp_dir:
            temp_path = Path(temp_dir)
            temp_copy_dir = temp_path / "temp_copy"

            try:
                # 使用shutil.copytree直接复制整个目录树
                shutil.copytree(directory, temp_copy_dir, dirs_exist_ok=True)

                # 从临时目录添加到zip
                for item in temp_copy_dir.rglob('*'):
                    if item.is_file():
                        try:
                            # 计算相对于临时目录的路径
                            rel_path = item.relative_to(temp_copy_dir)

                            # 构建在zip中的路径
                            if base_name:
                                arcname = str(Path(base_name) / rel_path)
                            else:
                                arcname = str(rel_path)

                            zipf.write(item, arcname)
                            file_count += 1

                        except Exception as e:
                            log_error(f"添加文件到zip失败 {item}: {e}")

            except Exception as e:
                log_error(f"目录复制失败: {e}")
        return file_count  # 确保返回文件数量

    def BackUpCreator(self): # 创建备份
        timestamp = datetime.datetime.now().strftime("%Y%m%d_%H-%M-%S")

        if len(self.selected_dirs) == 1:
            # 单个文件夹备份使用最终目录名
            dir_path = Path(self.selected_dirs[0])
            # 获取路径的最后一部分作为目录名
            dir_name = dir_path.name
            backup_filename = f"{dir_name}_backup_{timestamp}.zip"
        else:
            # 多个文件夹备份
            backup_filename = f"backup_{timestamp}.zip"

        backup_file = self.backup_dir / backup_filename

        try:
            file_count = 0
            with zipfile.ZipFile(backup_file, 'w', zipfile.ZIP_DEFLATED) as zipf:
                for dir_path_str in self.selected_dirs:
                    dir_path = self.backup_path / dir_path_str
                    if dir_path.exists():
                        # 使用原始路径作为zip中的基础路径
                        count = self.ZIPCreator(zipf, dir_path, dir_path_str)
                        file_count += count
            return backup_file

        except Exception as e:
            print()
            log_error(f"备份失败: {e}")
            # 如果备份失败，删除可能不完整的文件
            if backup_file.exists():
                try:
                    backup_file.unlink()
                    log_debug("已删除不完整的备份文件")
                except Exception as delete_error:
                    log_error(f"删除不完整备份文件失败: {delete_error}")
            return None

    @staticmethod
    def FormatFileSize(size_bytes): # 格式化大小
        for unit in ['B', 'KB', 'MB', 'GB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} TB"

    def BackUpLoop(self): # 备份循环模式
        self.is_running = True
        self.backup_count = 0

        try:
            while self.is_running:
                # 检查是否达到最大备份次数
                if self.max_backups and self.backup_count >= self.max_backups:
                    break

                if self.backup_time:
                    # 显示倒计时
                    self.ShowCountdown()

                    if not self.is_running:
                        break

                # 执行备份
                self.backup_count += 1

                log_info(f"开始执行第 {self.backup_count} 次备份...")
                backup_file = self.BackUpCreator()

                if backup_file:
                    # 获取备份文件信息
                    file_size = backup_file.stat().st_size
                    file_size_str = self.FormatFileSize(file_size)

                    # 统计文件数量
                    file_count = 0
                    with zipfile.ZipFile(backup_file, 'r') as zipf:
                        file_count = len(zipf.namelist())

                    timestamp = time.strftime(
                        f"{TextFormat.BRIGHT_BLUE}%Y-%m-%d{TextFormat.CLEAR} {TextFormat.BRIGHT_YELLOW}%H:%M:%S{TextFormat.CLEAR}",
                        time.localtime())
                    backup_success_text = f"{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} 已将存档备份至 {backup_file.name} ({file_size_str}, 共 {file_count} 个文件)"
                    print(backup_success_text)
                else:
                    log_error(f"第 {self.backup_count} 次备份失败")

                # 检查是否继续循环
                if not self.loop_execution or not self.backup_time:
                    break

                # 检查是否达到最大备份次数（备份完成后检查）
                if self.max_backups and self.backup_count >= self.max_backups:
                    break

        except KeyboardInterrupt:
            print()
            log_info("已手动取消备份！")
        finally:
            self.is_running = False
            log_info(f"备份结束，共完成 {self.backup_count} 次备份")

    def ShowCountdown(self): # 在同一行刷新时间
        total_seconds = self.backup_time
        remaining = total_seconds

        timestamp = time.strftime(
            f"{TextFormat.BRIGHT_BLUE}%Y-%m-%d{TextFormat.CLEAR} {TextFormat.BRIGHT_YELLOW}%H:%M:%S{TextFormat.CLEAR}",
            time.localtime())

        try:
            # 模拟log_info
            if self.max_backups:
                base_text = f"{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} 正在进行第 {self.backup_count + 1}/{self.max_backups} 次备份，下次备份将在 "
            else:
                base_text = f"{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} 正在进行第 {self.backup_count + 1} 次备份，下次备份将在 "

            while remaining > 0 and self.is_running:
                # 格式化剩余时间
                time_str = self.FormatDuration(remaining)

                # 构建显示文本
                display_text = base_text + f"{time_str}后执行"

                # 在同一行刷新显示，确保清除旧内容
                sys.stdout.write(f'\r{display_text}')
                sys.stdout.flush()

                # 等待1秒
                time.sleep(1)
                remaining -= 1

            # 倒计时结束，清除倒计时行
            if self.is_running:
                sys.stdout.write(f'\r{" " * 150}\r')
                sys.stdout.flush()

        except KeyboardInterrupt:
            raise

    @staticmethod
    def FormatDuration(seconds): # 格式化时间间隔
        if seconds < 60:
            return f"{seconds} 秒"  # 添加空格
        elif seconds < 3600:
            minutes = seconds // 60
            secs = seconds % 60
            if secs == 0:
                return f"{minutes} 分钟"
            else:
                return f"{minutes} 分 {secs} 秒"
        else:
            hours = seconds // 3600
            minutes = (seconds % 3600) // 60
            secs = seconds % 60
            if minutes == 0 and secs == 0:
                return f"{hours} 小时"
            elif secs == 0:
                return f"{hours} 小时 {minutes} 分"
            else:
                return f"{hours} 小时 {minutes} 分 {secs} 秒"

    def RunBackUp(self): # 运行备份
        self.ArgParser()
        log_info(f"备份工作目录: {self.backup_path}")
        log_info(f"计划备份目录: {', '.join(self.selected_dirs)}")
        log_info(f"备份保存目录: {self.backup_dir}")

        if self.backup_time:
            time_str = self.FormatDuration(self.backup_time)
            log_info("模式：循环执行备份")
            log_info(f"周期备份间隔: {time_str}")
            if self.max_backups:
                log_info(f"备份循环次数: {self.max_backups} 次")
        else:
            log_info("模式：立即执行备份")

        if self.backup_time and self.loop_execution:
            # 启动备份循环
            self.BackUpLoop()
        else:
            # 立即执行备份
            if self.backup_time:
                log_info(f"等待 {self.FormatDuration(self.backup_time)} 后执行备份...")
                time.sleep(self.backup_time)

            log_info("开始执行备份...")
            backup_file = self.BackUpCreator()
            if backup_file:
                # 获取备份文件信息
                file_size = backup_file.stat().st_size
                file_size_str = self.FormatFileSize(file_size)

                # 统计文件数量
                file_count = 0
                with zipfile.ZipFile(backup_file, 'r') as zipf:
                    file_count = len(zipf.namelist())

                timestamp = time.strftime(
                    f"{TextFormat.BRIGHT_BLUE}%Y-%m-%d{TextFormat.CLEAR} {TextFormat.BRIGHT_YELLOW}%H:%M:%S{TextFormat.CLEAR}",
                    time.localtime())
                backup_success_text = f"{timestamp} {TextFormat.GREEN}[Application Thread/INFO]{TextFormat.CLEAR} 已将存档备份至 {backup_file.name} ({file_size_str}, 共 {file_count} 个文件)"
                print(backup_success_text)
            else:
                log_error("备份任务失败")
                sys.exit(1)


def main():
    try:
        backup_tool = MinecraftBackup()
        backup_tool.RunBackUp()
    except KeyboardInterrupt:
        log_info("程序被用户中断")
    except Exception as e:
        log_error(f"程序执行出错: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()