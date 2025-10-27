"""------------------------外部库------------------------"""
import asyncio
import os
import time
import argparse
import sys
import matplotlib.pyplot as plt
import nbtlib

from concurrent.futures import ThreadPoolExecutor
from PIL import Image
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import log_info, log_error, log_debug

# 设置中文字体
plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei'] if os.name == 'nt' else ['WenQuanYi Micro Hei']
plt.rcParams['axes.unicode_minus'] = False

# 生成热力图
# noinspection PyUnusedLocal
def _GenerateHeatMap(data_chunk, chunk_num, timestamp, image_path):
    players, days = zip(*data_chunk)

    fig, ax = plt.subplots(figsize=(14, 7), dpi=120)
    colors = plt.cm.viridis([x / max(days) for x in days])

    bars = ax.bar(players, days, color=colors)
    for bar in bars:
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width() / 2, height,
                f'{height:.1f}天', ha='center', va='bottom', fontsize=9)

    ax.grid(axis='y', alpha=0.4)

    try:
        log_info(f"正在合并图表标签与数据...")
        ax.set_title(f'玩家游玩时长TOP{len(data_chunk)}\n生成时间: {timestamp}', pad=20)
        ax.set_xlabel('玩家名称')
        ax.set_ylabel('游玩天数')
        time.sleep(0.2)
        log_info(f"已生成图表: {image_path}")
    except:
        log_error("设置图表标签失败！这可能是中文字符串造成的！")
        log_error(f"返回至等效英文标签...")
        ax.set_title(f'Play Time TOP{len(data_chunk)}\n{timestamp}', pad=20)
        ax.set_xlabel('Player Name')
        ax.set_ylabel('Days Played')

    plt.xticks(rotation=45, ha='right')
    fig.savefig(image_path, bbox_inches='tight', dpi=120)
    plt.close(fig)

    # 使用PIL展示图像
    try:
        img = Image.open(image_path)
        img.show()
    except Exception as e:
        log_error(f"无法显示图像: {str(e)}")

# 考虑到一些服务器/存档的玩家数量会很多，所对应的NBT文件也会很多
# 为了防止程序主线程爆炸，因此使用asyncio库进行文件异步处理
# 这样不光高效，还很带派
async def _ProcessNBTFile(nbt_path, filename, playtimes, player_names):
    if not filename.endswith('.dat'):
        return

    try:
        with nbtlib.load(os.path.join(nbt_path, filename)) as nbt:
            # 玩家名称解析
            player_name = str(nbt.get('bukkit', {}).get('lastKnownName', filename[:-4]))
            player_names[filename[:-4]] = player_name

            # 游玩时长计算
            if 'playerGameTime' in nbt:
                hours = nbt['playerGameTime'] / 72000  # 72000刻=1小时
                days = max(0.1, round(hours / 24, 1))
            # 兼容多核心
            else:
                first_played = nbt.get('FirstPlayed', nbt.get('bukkit', {}).get('firstPlayed', 0))
                last_played = nbt.get('LastPlayed', nbt.get('bukkit', {}).get('lastPlayed', 0))
                days = max(0.1, round((last_played - first_played) / (1000 * 60 * 60 * 24), 1)) if all(
                    [first_played, last_played]) else 0.1

            playtimes[filename[:-4]] = days
    except Exception as e:
        log_error(f"跳过损坏文件 {filename}: {str(e)}")

    log_debug(f"处理文件 {filename}")

# 主函数，用于调用各种方法然后综合生成热力图
def ProcessHeatMap(nbt_path, players_per_chart=15, output_dir=None):
    log_info("开始处理玩家数据...")
    time.sleep(0.5)
    if not os.path.isdir(nbt_path):
        raise ValueError(f"路径不存在: {nbt_path}")

    # 处理输出目录
    if output_dir is None:
        output_dir = os.getcwd()
    elif not os.path.isdir(output_dir):
        os.makedirs(output_dir, exist_ok=True)

    playtimes = {}
    player_names = {}
    timestamp = time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())

    # 使用线程池处理IO密集型任务
    with ThreadPoolExecutor(max_workers=min(32, os.cpu_count() * 4)):
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)

        tasks = [
            _ProcessNBTFile(nbt_path, filename, playtimes, player_names)
            for filename in os.listdir(nbt_path)
        ]
        loop.run_until_complete(asyncio.gather(*tasks))
        loop.close()

    # 数据过滤和排序
    valid_players = {
        uuid: (player_names[uuid], days)
        for uuid, days in playtimes.items() if days > 0
    }
    if not valid_players:
        raise ValueError("未找到有效玩家数据")

    sorted_players = sorted(valid_players.values(), key=lambda x: -x[1])

    log_info(f"共找到有效玩家数据: {len(valid_players)}")

    # 分块生成图表
    for i, chunk in enumerate([sorted_players[i:i + players_per_chart]
                             for i in range(0, len(sorted_players), players_per_chart)]):
        image_path = os.path.join(output_dir, f'heatmap_{i + 1}.png')
        _GenerateHeatMap(chunk, i + 1, timestamp, image_path)

def main():
    parser = argparse.ArgumentParser(
        description='生成玩家游玩时长热力图',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-dfp', '--data-folder-path', required=True,
                       help='指定playerdata文件夹路径')

    # 可选参数
    parser.add_argument('-mp', '--max-player', type=int, default=15,
                       help='每张图表显示的最大玩家数 (默认: 15)')
    parser.add_argument('-od', '--output-dir',
                       help='指定输出目录 (默认: 当前目录)')
    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 热力图生成器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 参数验证
        if not os.path.exists(args.data_folder_path):
            log_error(f"指定的路径不存在: {args.data_folder_path}")
            sys.exit(1)

        if args.max_player < 1 or args.max_player > 50:
            log_error("每页玩家数必须在1-50之间")
            sys.exit(1)

        # 执行热力图生成
        ProcessHeatMap(
            nbt_path=args.data_folder_path,
            players_per_chart=args.max_player,
            output_dir=args.output_dir
        )

        log_info("热力图生成完成！")

    except Exception as e:
        log_error(f"热力图生成失败: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()