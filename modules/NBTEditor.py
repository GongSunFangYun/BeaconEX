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
        # 获取NBTExplorer.exe的路径
        log_info(f"正在将 {nbt_path} 导入至 NBTExplorer...")
        root_dir = os.path.dirname(os.path.dirname(__file__))  # 获取根目录
        editor_path = os.path.join(root_dir, "NBTExplorer", "NBTExplorerCN.exe")
        time.sleep(0.2)
        log_info("导入完成，请在NBTExplorer中编辑该NBT文件")

        if not os.path.exists(editor_path):
            log_error("找不到NBTExplorer编辑器，请确保已正确安装")
            return False

        # 启动NBTExplorer并打开指定文件
        time.sleep(0.1)
        subprocess.Popen([editor_path, nbt_path])
        return True

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
        # 参数验证
        if not os.path.exists(args.file_path):
            log_error(f"指定的NBT文件不存在: {args.file_path}")
            sys.exit(1)

        # 执行NBT编辑
        success = EditNBTFile(args.file_path)

        if success:
            log_info("NBT编辑器启动成功！")
        else:
            log_error("NBT编辑器启动失败")
            sys.exit(1)

    except Exception as e:
        log_error(f"NBT编辑失败: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()