# -*- coding: utf-8 -*-
import os
import nbtlib
from functions.utils import log_info, log_warn, log_error


def find_world_dirs(server_root):
    """自动寻找可能的世界目录"""
    possible_paths = [
        os.path.join(server_root, "world"),
        os.path.join(server_root, "worlds"),
        os.path.join(server_root, "world_nether"),
        os.path.join(server_root, "world_the_end"),
        os.path.join(server_root, "DIM-1"),  # 下界(旧版)
        os.path.join(server_root, "DIM1")  # 末地(旧版)
    ]

    found_worlds = {}
    for path in possible_paths:
        level_dat = os.path.join(path, "level.dat")
        if os.path.exists(level_dat):
            # 自动识别维度
            if "nether" in path.lower() or "dim-1" in path.lower():
                dim = "下界"
            elif "end" in path.lower() or "dim1" in path.lower():
                dim = "末地"
            else:
                dim = "主世界"

            found_worlds[path] = dim

    return found_worlds


def check_world_integrity(server_root):
    log_info(f"正在扫描服务器目录: {server_root}")

    world_dirs = find_world_dirs(server_root)
    if not world_dirs:
        log_error("未找到任何有效的世界目录！")
        return False

    log_info("找到以下世界目录:")
    for path, dim in world_dirs.items():
        log_info(f"- [{dim}] {path}")

    log_info("开始完整性检查...")
    all_ok = True

    for world_path, dim in world_dirs.items():
        level_dat = os.path.join(world_path, "level.dat")
        log_info(f"正在检查 [{dim}] 世界: {level_dat}")

        try:
            # 检查level.dat基本完整性
            nbt_data = nbtlib.load(level_dat)
            if 'Data' not in nbt_data:
                raise ValueError("缺少Data标签")

            log_info(f"[{dim}] level.dat 基本结构正常")

            # 检查重要数据是否存在
            required_tags = ['Time', 'LastPlayed', 'RandomSeed']
            for tag in required_tags:
                if tag not in nbt_data['Data']:
                    log_warn(f"[{dim}] 缺少重要标签: {tag}")

        except Exception as e:
            log_error(f"[{dim}] level.dat 损坏: {str(e)}")
            all_ok = False
            continue

        # 检查region文件(仅主世界)
        if dim == "主世界":
            region_path = os.path.join(world_path, "region")
            if os.path.exists(region_path):
                region_files = [f for f in os.listdir(region_path) if f.endswith(".mca")]
                if not region_files:
                    log_warn("[主世界] region目录为空")
                else:
                    log_info(f"[主世界] 发现 {len(region_files)} 个region文件")
            else:
                log_warn("[主世界] 缺少region目录")

    if all_ok:
        log_info("所有世界完整性检查通过！")
    else:
        log_error("发现损坏的世界数据！")

    return all_ok