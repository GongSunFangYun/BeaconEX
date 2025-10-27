"""------------------------外部库------------------------"""
import argparse
import os
import sys
import threading
import time
import nbtlib

from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import log_info, log_warn, log_error, log_debug

class WorldScanner:
    def __init__(self, max_workers=4):
        # 最大工作线程数
        self.max_workers = max_workers
        # 世界统计信息字典，存储扫描结果
        self.world_stats = {}
        # 线程锁，用于保护共享数据的线程安全
        self.lock = threading.Lock()

    def FindAllWorldDirs(self, root_path): # 查找含有level.dat的存档文件夹
        world_dirs = {}
        root_path = Path(root_path)

        log_info(f"开始扫描目录: {root_path}")

        for level_file in root_path.rglob("level.dat"):
            world_dir = level_file.parent
            dim = self._IdentifyDimension(world_dir, root_path)

            # 计算相对路径用于显示
            try:
                rel_path = world_dir.relative_to(root_path)
            except ValueError:
                rel_path = world_dir

            world_dirs[str(world_dir)] = {
                'dimension': dim,
                'relative_path': str(rel_path)
            }

            log_debug(f"发现世界: {rel_path}")

        return world_dirs

    # noinspection PyUnusedLocal
    @staticmethod
    def _IdentifyDimension(world_dir, root_path): # 检查维度
        world_dir_str = str(world_dir).lower()
        dir_name = world_dir.name.lower()

        # 检查标准维度特征
        if any(x in world_dir_str for x in ['nether', 'dim-1']):
            return "nether"
        elif any(x in world_dir_str for x in ['end', 'dim1']):
            return "end"
        elif any(x in dir_name for x in ['world', 'overworld']):
            return "overworld"
        else:
            # 尝试从目录结构识别自定义维度
            try:
                # 检查是否有维度特定目录
                if (world_dir / "DIM-1").exists():
                    return "nether"
                elif (world_dir / "DIM1").exists():
                    return "end"

                # 检查是否是Multiverse等多世界插件创建的世界
                parent_name = world_dir.parent.name.lower()
                if parent_name in ['worlds', 'worldcontainer', 'mvworlds']:
                    # 使用目录名作为维度标识
                    return f"custom:{world_dir.name}"

            except:
                pass

            # 默认使用目录名
            return f"world:{world_dir.name}"

    def ScanWorld(self, world_path, world_info):
        stats = {
            'path': world_path,
            'dimension': world_info['dimension'],
            'relative_path': world_info['relative_path'],
            'level_dat_ok': False,
            'file_count': 0,
            'total_size': 0,
            'region_files': 0,
            'player_data_files': 0,
            'data_packs': 0,
            'last_played': 0,
            'game_time': 0,
            'random_seed': 0,
            'version': '未知',
            'errors': []
        }

        try:
            # 检查level.dat
            level_dat_path = os.path.join(world_path, "level.dat")
            nbt_data = nbtlib.load(level_dat_path)

            if 'Data' not in nbt_data:
                raise ValueError("缺少Data标签")

            stats['level_dat_ok'] = True

            # 读取世界信息
            data = nbt_data['Data']
            stats['last_played'] = data.get('LastPlayed', 0)
            stats['game_time'] = data.get('Time', 0)
            stats['random_seed'] = data.get('RandomSeed', 0)
            stats['version'] = data.get('Version', {}).get('Name', '未知')

            # 扫描目录结构
            self._ScanWorldDirectory(world_path, stats)

        except Exception as e:
            stats['errors'].append(f"level.dat读取失败: {str(e)}")

        with self.lock:
            self.world_stats[world_path] = stats

        return stats

    @staticmethod
    def _ScanWorldDirectory(world_path, stats): #扫描世界文件结构
        world_path = Path(world_path)

        for file_path in world_path.rglob('*'):
            if file_path.is_file():
                try:
                    stats['file_count'] += 1
                    file_size = file_path.stat().st_size
                    stats['total_size'] += file_size

                    # 分类文件类型
                    if file_path.suffix == '.mca':
                        stats['region_files'] += 1
                    elif file_path.parent.name == 'playerdata' and file_path.suffix == '.dat':
                        stats['player_data_files'] += 1
                    elif file_path.parent.name == 'datapacks' and file_path.suffix == '.zip':
                        stats['data_packs'] += 1
                except (OSError, PermissionError):
                    # 跳过无法访问的文件
                    continue

    def DeepScan(self, root_path):# 深度————————————扫描————————————
        log_info("开始扫描所有世界...")
        start_time = time.time()

        # 查找所有世界目录
        world_dirs = self.FindAllWorldDirs(root_path)

        if not world_dirs:
            log_error("未找到任何世界目录！")
            return {}

        log_info(f"共找到 {len(world_dirs)} 个世界目录")

        # 使用线程池并行扫描
        with ThreadPoolExecutor(max_workers=self.max_workers) as executor:
            future_to_world = {
                executor.submit(self.ScanWorld, path, info): path
                for path, info in world_dirs.items()
            }

            completed = 0
            for future in as_completed(future_to_world):
                world_path = future_to_world[future]
                try:
                    future.result()
                    completed += 1
                    world_name = world_dirs[world_path]['relative_path']
                    log_info(f"扫描进度: {completed}/{len(world_dirs)} - {world_name}")
                except Exception as e:
                    log_error(f"扫描失败 {world_path}: {str(e)}")

        scan_time = time.time() - start_time
        log_info(f"对所有维度扫描完成，耗时 {scan_time:.2f} 秒")

        return self.world_stats


def FormatFileSize(size_bytes): # 格式化世界大小
    if size_bytes == 0:
        return "0 B"

    for unit in ['B', 'KB', 'MB', 'GB']:
        if size_bytes < 1024.0:
            return f"{size_bytes:.2f} {unit}"
        size_bytes /= 1024.0
    return f"{size_bytes:.2f} TB"


def FormatGameTime(tick_time): # 格式化游戏时间
    if hasattr(tick_time, 'value'):
        tick_time = tick_time.value

    hours = (tick_time // 1000 + 6) % 24
    minutes = (tick_time % 1000) * 60 // 1000
    return f"{int(hours):02d}:{int(minutes):02d}"


def OutputStatistics (world_stats): # 输出统计信息
    log_info("-" * 50)

    total_worlds = len(world_stats)
    total_size = sum(stats.get('total_size', 0) for stats in world_stats.values())
    total_files = sum(stats.get('file_count', 0) for stats in world_stats.values())
    total_regions = sum(stats.get('region_files', 0) for stats in world_stats.values())
    total_players = sum(stats.get('player_data_files', 0) for stats in world_stats.values())

    log_info(f"世界总数: {total_worlds}")
    log_info(f"总文件数: {total_files}")
    log_info(f"总大小: {FormatFileSize(total_size)}")
    log_info(f"Region文件: {total_regions} 个")
    log_info(f"玩家数据: {total_players} 个")

    # 按维度类型分组统计
    dim_groups = {}
    for stats in world_stats.values():
        dim = stats['dimension']
        group = "标准世界"
        if dim.startswith('custom:'):
            group = "自定义世界"
        elif dim.startswith('world:'):
            group = "独立世界"

        if group not in dim_groups:
            dim_groups[group] = {'count': 0, 'size': 0, 'files': 0}
        dim_groups[group]['count'] += 1
        dim_groups[group]['size'] += stats.get('total_size', 0)
        dim_groups[group]['files'] += stats.get('file_count', 0)

    log_info("世界分类统计:")
    for group, data in dim_groups.items():
        log_info(f"  {group}: {data['count']} 个, {FormatFileSize(data['size'])}, {data['files']} 文件")

    # 详细世界信息
    log_info("-" * 50)
    log_info("详细世界信息:")

    for world_path, stats in sorted(world_stats.items(), key=lambda x: x[1]['relative_path']):
        world_name = stats['relative_path']
        status = "✓ 正常" if stats['level_dat_ok'] and not stats['errors'] else "✗ 异常"

        log_info(f"{world_name}")
        log_info(f"  状态: {status}")
        log_info(f"  大小: {FormatFileSize(stats['total_size'])}")
        log_info(
            f"  文件: {stats['file_count']} 个 (Region: {stats['region_files']}, 玩家: {stats['player_data_files']}, 数据包: {stats['data_packs']})")

        if stats['level_dat_ok']:
            log_info(f"  版本: {stats['version']}")
            log_info(f"  游戏内时间: {FormatGameTime(stats['game_time'])}")
            log_info(f"  最后游玩: {time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(stats['last_played'] / 1000))}")

        if stats['errors']:
            for error in stats['errors']:
                log_warn(f"  错误: {error}")

        log_info("")  # 空行分隔

    # 完整性总结
    damaged_worlds = [stats for stats in world_stats.values() if not stats['level_dat_ok'] or stats['errors']]
    if damaged_worlds:
        log_error(f"\n发现 {len(damaged_worlds)} 个可能损坏的世界:")
        for stats in damaged_worlds:
            log_error(f"  - {stats['relative_path']}")
        return False
    else:
        log_info(f"✓ 所有世界检查通过！")
        return True


def main():
    parser = argparse.ArgumentParser(
        description='Minecraft世界数据完整性检查与统计分析',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-wp', '--world-path', required=True,
                        help='指定服务器根目录路径')

    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 世界数据检查工具\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 参数验证
        if not os.path.exists(args.world_path):
            log_error(f"指定的路径不存在: {args.world_path}")
            sys.exit(1)

        if not os.path.isdir(args.world_path):
            log_error(f"指定的路径不是目录: {args.world_path}")
            sys.exit(1)

        # 直接执行深度扫描
        scanner = WorldScanner(max_workers=4)
        world_stats = scanner.DeepScan(args.world_path)

        if world_stats:
            success = OutputStatistics(world_stats)
            sys.exit(0 if success else 1)
        else:
            sys.exit(1)

    except Exception as e:
        log_error(f"世界检查失败: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()