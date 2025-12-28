"""------------------------外部库------------------------"""
import os
import subprocess
import time
import argparse
import sys
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import log_error, log_info

def EditNBTFile(nbt_path):
    try:
        log_info(f"正在将 {nbt_path} 导入至 NBTExplorer...")

        # 获取脚本所在目录（_internal文件夹）
        script_dir = os.path.dirname(os.path.abspath(__file__))
        editor_path = os.path.join(script_dir, "NBTExplorer", "NBTExplorerCN.exe")

        # 检查编辑器是否存在
        if not os.path.exists(editor_path):
            log_error("未找到NBTExplorer编辑器")
            log_error(f"请在 {script_dir} 目录下检查NBTExplorer文件夹")
            return False

        # 检查NBT文件是否存在
        if not os.path.exists(nbt_path):
            log_error(f"指定的NBT文件不存在: {nbt_path}")
            return False
        time.sleep(0.2)

        # 启动编辑器
        try:
            subprocess.Popen([editor_path, nbt_path])
            log_info("NBTExplorer已启动，请在编辑器中修改NBT文件")
            return True

        except Exception as e:
            log_error(f"启动NBTExplorer失败: {str(e)}")
            return False

    except Exception as e:
        log_error(f"启动NBT编辑器失败: {str(e)}")
        return False


def main():
    parser = argparse.ArgumentParser(
        description='使用NBTExplorer编辑NBT文件',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-fp', '--file-path', required=True,
                        help='指定要编辑的NBT文件路径')

    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX NBT编辑器\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 执行NBT编辑
        success = EditNBTFile(args.file_path)

        if not success:
            log_error("NBT编辑器操作失败")
            sys.exit(1)

    except Exception as e:
        log_error(f"NBT编辑失败: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()