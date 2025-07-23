"""------------------------外部库------------------------"""
import os
import time
import matplotlib.pyplot as plt
import nbtlib
from PIL import Image
'''------------------------本地库------------------------'''
from functions.utils import log_info, log_error, log_warn

# 设置中文字体
try:
    plt.rcParams['font.sans-serif'] = ['Microsoft YaHei', 'SimHei']
    plt.rcParams['axes.unicode_minus'] = False
except:
    pass

# 此函数用于解析NBT文件和生成热力图
def ProcessHeatMap(nbt_path, players_per_chart=15):
    log_info("正在创建热力图...")
    if not os.path.isdir(nbt_path):
        raise ValueError(f"路径不存在: {nbt_path}")

    timestamp = time.strftime("%Y-%m-%d %H:%M:%S", time.localtime())

    playtimes = {}
    player_names = {}

    # 收集玩家数据（原逻辑不变）
    for filename in os.listdir(nbt_path):
        if not filename.endswith('.dat'):
            continue

        try:
            with nbtlib.load(os.path.join(nbt_path, filename)) as nbt:
                # 玩家名称解析
                player_name = str(nbt.get('bukkit', {}).get('lastKnownName', filename[:-4]))
                player_names[filename[:-4]] = player_name

                # 游玩时长计算
                if 'playerGameTime' in nbt:
                    hours = nbt['playerGameTime'] / 72000  # 72000刻=1小时
                    days = max(0.1, round(hours / 24, 1))
                else:
                    first_played = nbt.get('FirstPlayed', nbt.get('bukkit', {}).get('firstPlayed', 0))
                    last_played = nbt.get('LastPlayed', nbt.get('bukkit', {}).get('lastPlayed', 0))
                    days = max(0.1, round((last_played - first_played) / (1000 * 60 * 60 * 24), 1)) if all(
                        [first_played, last_played]) else 0.1

                playtimes[filename[:-4]] = days

        except Exception as e:
            log_error(f"跳过损坏文件 {filename}: {str(e)}")
            continue

    # 过滤有效玩家
    valid_players = {
        uuid: (player_names[uuid], days)
        for uuid, days in playtimes.items()
        if days > 0
    }

    if not valid_players:
        raise ValueError(f"{log_warn} 未找到有游玩时长的玩家数据")

    # 按游玩时长排序
    sorted_players = sorted(valid_players.values(), key=lambda x: -x[1])

    # 分块处理（每块players_per_chart个玩家）
    for i, chunk in enumerate([sorted_players[i:i + players_per_chart]
                               for i in range(0, len(sorted_players), players_per_chart)]):
        image_path = f'playtime_chunk_{i + 1}.png'
        _GeneratePNG(chunk, i + 1, timestamp, image_path)

        try:
            Image.open(image_path).show()
        except Exception as e:
            log_error(f"无法打开图片: {str(e)}")

# 使用matplotlib绘制热力图并保存到安装根目录，然后使用PIL展示
def _GeneratePNG(data_chunk, chunk_num, timestamp, image_path):
    players, days = zip(*data_chunk)

    fig, ax = plt.subplots(figsize=(14, 7), dpi=120)
    colors = plt.cm.viridis([x / max(days) for x in days])

    # 绘制柱状图
    bars = ax.bar(players, days, color=colors)
    for bar in bars:
        height = bar.get_height()
        ax.text(bar.get_x() + bar.get_width() / 2, height,
                f'{height:.1f}天', ha='center', va='bottom', fontsize=9)

    # 图表装饰
    ax.grid(axis='y', alpha=0.4)
    try:
        ax.set_title(f'玩家游玩时长TOP{len(data_chunk)}\n生成时间: {timestamp}', pad=20)
        ax.set_xlabel('玩家名称')
        ax.set_ylabel('游玩天数')
    except:
        ax.set_title(f'Play Time TOP{len(data_chunk)}\n{timestamp}', pad=20)
        ax.set_xlabel('Player Name')
        ax.set_ylabel('Days Played')

    plt.xticks(rotation=45, ha='right')
    fig.savefig(image_path, bbox_inches='tight', dpi=120)
    plt.close(fig)

    log_info(f"已生成图表 {image_path}")