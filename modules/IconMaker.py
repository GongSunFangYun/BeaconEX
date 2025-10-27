"""------------------------外部库------------------------"""
import argparse
import os
import sys

from PIL import Image, ImageOps
'''------------------------本地库------------------------'''
from bexlib2.lg4pb import TextFormat, log_info, log_error, log_warn


class ServerIconProcessor:
    def __init__(self, picture_path: str, output_dir: str = None, picture_name: str = None):
        # 原始图片文件路径
        self.picture_path = picture_path
        # 输出目录路径
        self.output_dir = output_dir
        # 输出图片文件名（默认：server-icon.png）
        self.picture_name = picture_name or "server-icon.png"
        # 处理后的图片对象
        self.processed_image = None

    def ValidateInput(self) -> bool: # 检验传入的路径/文件格式是否正确
        if not os.path.exists(self.picture_path):
            log_error(f"图片文件不存在: {self.picture_path}")
            return False

        if not self.picture_path.lower().endswith(('.png', '.jpg', '.jpeg', '.bmp', '.gif')):
            log_error("不支持的文件格式，请使用 PNG、JPG、BMP 或 GIF 格式")
            return False

        if self.output_dir and not os.path.isdir(self.output_dir):
            log_info(f"创建输出目录: {self.output_dir}")
            os.makedirs(self.output_dir, exist_ok=True)

        return True

    def ProcessImage(self) -> bool: # 将图片处理成64x64尺寸大小，确保符合mc服务器规范
        try:
            log_info(f"正在处理图片: {self.picture_path}")

            # 打开图片
            with Image.open(self.picture_path) as img:
                log_info(f"原始图片尺寸: {img.size} | 格式: {img.format}")

                # 转换为 RGB 模式（处理 PNG 透明度）
                if img.mode in ('RGBA', 'LA', 'P'):
                    background = Image.new('RGB', img.size, (255, 255, 255))
                    if img.mode == 'P':
                        img = img.convert('RGBA')
                    background.paste(img, mask=img.split()[-1] if img.mode == 'RGBA' else None)
                    img = background
                elif img.mode != 'RGB':
                    img = img.convert('RGB')

                # 调整尺寸为 64x64
                self.processed_image = ImageOps.fit(img, (64, 64), method=Image.Resampling.LANCZOS)
                log_info(f"调整后尺寸: {self.processed_image.size}")

            return True

        except Exception as e:
            log_error(f"图片处理失败: {str(e)}")
            return False

    def SaveImage(self) -> bool:
        if not self.processed_image:
            log_error("没有可保存的图片数据")
            return False

        try:
            if self.output_dir:
                output_path = os.path.join(self.output_dir, self.picture_name)

                # 确保文件扩展名为 .png
                if not output_path.lower().endswith('.png'):
                    output_path += '.png'

                self.processed_image.save(output_path, 'PNG')
                log_info(f"{TextFormat.GREEN}✓ 图片已保存: {output_path}{TextFormat.CLEAR}")

                # 显示文件信息
                file_size = os.path.getsize(output_path)
                log_info(f"文件大小: {self._FormatFileSize(file_size)}")

            else:
                log_warn(f"未指定图片输出目录，请自行保存图片并重命名！")

            return True

        except Exception as e:
            log_error(f"图片保存失败: {str(e)}")
            return False

    def PreviewImage(self) -> bool: # 使用PIL打开图片
        if not self.processed_image:
            return False

        try:
            self.processed_image.show()
            return True
        except Exception as e:
            log_warn(f"图片预览失败: {str(e)}")
            return False

    @staticmethod
    def _FormatFileSize(size_bytes: int) -> str: # 格式化文件大小
        if size_bytes == 0:
            return "0 B"

        for unit in ['B', 'KB', 'MB']:
            if size_bytes < 1024.0:
                return f"{size_bytes:.2f} {unit}"
            size_bytes /= 1024.0
        return f"{size_bytes:.2f} GB"

    def execute(self) -> bool:
        # 验证输入
        if not self.ValidateInput():
            return False

        # 处理图片
        if not self.ProcessImage():
            return False

        # 保存图片
        save_success = self.SaveImage()

        # 预览图片
        preview_success = self.PreviewImage()

        return save_success or preview_success


def ProcessServerIcon(picture_path: str, output_dir: str = None, picture_name: str = None): # 处理参数
    processor = ServerIconProcessor(picture_path, output_dir, picture_name)
    return processor.execute()


def main():
    parser = argparse.ArgumentParser(
        description='Minecraft服务器图标处理工具 - 将图片转换为64x64服务器图标',
        formatter_class=argparse.RawTextHelpFormatter
    )

    # 必需参数
    parser.add_argument('-pp', '--picture-path', required=True,
                        help='指定原始图片路径\n'
                             '支持格式: PNG, JPG, JPEG, BMP, GIF\n'
                             '示例: -pp /path/to/image.jpg')

    # 可选参数
    parser.add_argument('-od', '--output-dir',
                        help='指定输出目录 (默认: 仅预览不保存)\n'
                             '示例: -od ./server_files')

    parser.add_argument('-pn', '--picture-name', default='server-icon.png',
                        help='指定输出图片名称 (默认: server-icon.png)\n'
                             '示例: -pn my-server-icon.png')

    parser.add_argument('-about', '--about', action='version',
                        version='BeaconEX 服务器图标处理工具\n源仓库：https://github.com/GongSunFangYun/BeaconEX',
                        help='显示关于信息')

    args = parser.parse_args()

    try:
        # 参数验证
        if not os.path.exists(args.picture_path):
            log_error(f"指定的图片文件不存在: {args.picture_path}")
            sys.exit(1)

        # 执行图片处理
        success = ProcessServerIcon(
            picture_path=args.picture_path,
            output_dir=args.output_dir,
            picture_name=args.picture_name
        )

        if success:
            log_info(f"服务器图标处理完成")
        else:
            log_error("服务器图标处理失败")
            sys.exit(1)

    except Exception as e:
        log_error(f"服务器图标处理失败: {str(e)}")
        sys.exit(1)


if __name__ == "__main__":
    main()