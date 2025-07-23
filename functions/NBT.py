"""------------------------外部库------------------------"""
import os
import nbtlib
from nbtlib.tag import Compound
'''------------------------本地库------------------------'''
from functions.utils import TextFormat, log_info, log_error, log_warn

# NBT文件解析类
# noinspection PyTypeChecker
class NBTDataParser:
    def __init__(self, file_path: str):
        self.file_path = file_path
        self.player_data = None
        self.player_name = self._GetPlayerName()

    # 解析NBT文件
    def ParseNBTData(self) -> bool:
        try:
            nbt_data = nbtlib.load(self.file_path)
            root = nbt_data if isinstance(nbt_data, Compound) else nbt_data.root
            self.player_data = root
            # 尝试从bukkit字段解析玩家名称
            if 'bukkit' in root and 'lastKnownName' in root['bukkit']:
                self.player_name = str(root['bukkit']['lastKnownName'])
            elif 'Name' in root:
                self.player_name = str(root['Name'])

            return True
        except Exception as e:
            log_error(f"解析NBT文件时出错: {e}")
            return False

    # 显示玩家数据
    def ShowALLData(self):
        if not self.player_data:
            log_error("没有可用的玩家数据")
            return
        # 定义相关方法
        self._GetPlayerBasicDatas()
        self._GetPositionDatas()
        self._GetPotionEffectsDatas()
        self._GetAbilities()
        self._GetDimensionData()

        # 此处只调用了 _display_inventory_stats()，它会处理所有库存显示
        inventory = self.player_data.get('Inventory', [])
        ender_items = self.player_data.get('EnderItems', [])
        self._ShowAllInventoryItems(inventory, ender_items)

    # 解析关键字段，获取玩家基础信息
    def _GetPlayerBasicDatas(self):
        data = {
            "玩家名称": TextFormat.Colorize(self.player_name, TextFormat.GREEN),
            "UUID": TextFormat.Colorize(str(self.player_data.get('UUID', '未知')), TextFormat.YELLOW),
            "生命值": f"{TextFormat.Colorize(f'{float(self.player_data.get("Health", 20.0)):.1f}', TextFormat.RED)}/20.0",
            "饥饿度": f"{TextFormat.Colorize(int(self.player_data.get('foodLevel', 20)), TextFormat.YELLOW)}/20",
            "饱和度": f"{float(self.player_data.get('foodSaturationLevel', 5.0)):.1f}",
            "经验值": f"等级 {TextFormat.Colorize(int(self.player_data.get('XpLevel', 0)), TextFormat.YELLOW)} (总计 {TextFormat.Colorize(int(self.player_data.get('XpTotal', 0)), TextFormat.YELLOW)})",
            "游戏模式": self._ProcessGamemode(int(self.player_data.get('playerGameType', 0))),
            "游玩时长": self._GetPlayedTime(),
            "最后在线": self._ConversionDate(self.player_data.get('lastPlayed', 0)),
            "是否已看终末之诗": TextFormat.Colorize("是", TextFormat.GREEN) if self.player_data.get(
                'seenCredits', False) else TextFormat.Colorize("否", TextFormat.RED)
        }
        for key, value in data.items():
            log_info(f"{key}: {value}")

    # 解析玩家游玩时长并换算为现实时间
    def _GetPlayedTime(self) -> str:
        first_played = self.player_data.get('FirstPlayed',
                                            self.player_data.get('bukkit', {}).get('firstPlayed', 0))
        last_played = self.player_data.get('LastPlayed',
                                           self.player_data.get('bukkit', {}).get('lastPlayed', 0))

        if first_played == 0 or last_played == 0:
            return TextFormat.Colorize("未知", TextFormat.YELLOW)

        play_time_hours = (last_played - first_played) / (1000 * 60 * 60)

        if play_time_hours < 1:
            return TextFormat.Colorize(f"{play_time_hours * 60:.0f}分钟", TextFormat.BLUE)
        elif play_time_hours < 24:
            return TextFormat.Colorize(f"{play_time_hours:.1f}小时", TextFormat.BLUE)
        else:
            return TextFormat.Colorize(f"{play_time_hours / 24:.1f}天", TextFormat.BLUE)

    # 转换日期
    @staticmethod
    def _ConversionDate(timestamp: int) -> str:
        if timestamp == 0:
            return TextFormat.Colorize("未知", TextFormat.YELLOW)

        try:
            from datetime import datetime
            dt = datetime.fromtimestamp(timestamp / 1000)
            return TextFormat.Colorize(dt.strftime("%Y-%m-%d %H:%M:%S"), TextFormat.BLUE)
        except:
            return TextFormat.Colorize("无效时间", TextFormat.RED)

    # 处理玩家名称，如果获取不到则返回UUID
    def _GetPlayerName(self) -> str:
        uuid = os.path.basename(self.file_path).replace('.dat', '')
        return f"玩家({uuid})"

    # 解析玩家位置（维度，坐标，生成点，上次死亡点等）
    def _GetPositionDatas(self):
        pos = self.player_data.get('Pos', [0, 0, 0])
        rotation = self.player_data.get('Rotation', [0, 0])
        # 解析维度
        dimension = str(self.player_data.get('Dimension', '未知'))
        dimension_color = {
            "minecraft:overworld": TextFormat.GREEN,
            "minecraft:the_nether": TextFormat.RED,
            "minecraft:the_end": TextFormat.PURPLE
        }.get(dimension, TextFormat.YELLOW)
        # 返回维度，位置和角度
        log_info(f"当前维度: {TextFormat.Colorize(dimension, dimension_color)}")
        log_info(f"当前位置: X={TextFormat.Colorize(f'{float(pos[0]):.1f}', TextFormat.YELLOW)}, "
                 f"Y={TextFormat.Colorize(f'{float(pos[1]):.1f}', TextFormat.YELLOW)}, "
                 f"Z={TextFormat.Colorize(f'{float(pos[2]):.1f}', TextFormat.YELLOW)}")
        log_info(f"当前朝向: 偏航角={TextFormat.Colorize(f'{float(rotation[0]):.1f}', TextFormat.YELLOW)}, "
                 f"俯仰角={TextFormat.Colorize(f'{float(rotation[1]):.1f}', TextFormat.YELLOW)}")
        # 解析生成点
        spawn = {
            "x": self.player_data.get('SpawnX', 0),
            "y": self.player_data.get('SpawnY', 0),
            "z": self.player_data.get('SpawnZ', 0),
            "forced": self.player_data.get('SpawnForced', False)
        }
        # 返回生成点数据
        log_info(f"重生点: X={TextFormat.Colorize(spawn['x'], TextFormat.YELLOW)}, "
                 f"Y={TextFormat.Colorize(spawn['y'], TextFormat.YELLOW)}, "
                 f"Z={TextFormat.Colorize(spawn['z'], TextFormat.YELLOW)} "
                 f"({TextFormat.Colorize('是', TextFormat.GREEN) if spawn['forced'] else TextFormat.Colorize('否', TextFormat.RED)})")
        # 解析并返回死亡点数据
        if 'LastDeathLocation' in self.player_data:
            death = self.player_data['LastDeathLocation']
            log_info(f"上次死亡位置: {TextFormat.Colorize(death.get('dimension', '未知'), dimension_color)} "
                     f"X={TextFormat.Colorize(f'{float(death["pos"][0]):.1f}', TextFormat.YELLOW)}, "
                     f"Y={TextFormat.Colorize(f'{float(death["pos"][1]):.1f}', TextFormat.YELLOW)}, "
                     f"Z={TextFormat.Colorize(f'{float(death["pos"][2]):.1f}', TextFormat.YELLOW)}")

    # 处理玩家库存异常，并显示基本处理结果
    def _ShowAllInventoryItems(self, inventory: list, ender_items: list):
        log_info(f"=============== {TextFormat.GREEN}库存统计{TextFormat.CLEAR} ===============")

        # 显示物品栏
        if not inventory:
            log_warn("物品栏为空")
        else:
            log_info(f"{TextFormat.YELLOW}[物品栏]")
            self._ShowPlayerInventoryItems(inventory)  # 这里传入inventory

        # 显示末影箱
        if not ender_items:
            log_warn("末影箱为空")
        else:
            log_info(f"{TextFormat.PURPLE}[末影箱]")
            self._ShowEnderChestItems(ender_items)  # 这里传入ender_items
        # 隔离
        log_info(f"========================================")

    # 解析玩家物品栏/热键栏/副手栏/末影箱/装备栏物品
    def _ShowPlayerInventoryItems(self, inventory: list):
        # 分类物品
        hotbar = []
        main_inventory = []
        armor = []
        offhand = []
        # 解析物品
        for item in inventory:
            slot = int(item['Slot'])
            item_info = self._ParseItem(item)

            if 0 <= slot < 9:  # 快捷栏
                hotbar.append((slot + 1, item_info))
            elif 9 <= slot < 36:  # 主物品栏
                main_inventory.append((slot - 8, item_info))
            elif 100 <= slot < 104:  # 盔甲栏
                armor_slots = {100: "靴子", 101: "护腿", 102: "胸甲", 103: "头盔"}
                armor.append((armor_slots.get(slot, "未知"), item_info))
            elif slot == -106:  # 副手
                offhand.append((f"副手", item_info))

        # 显示分类后的物品
        if hotbar: # 热键栏
            log_info(f"{TextFormat.GREEN}快捷栏:")
            for slot, item in sorted(hotbar, key=lambda x: x[0]):
                log_info(f"  [{slot}] {item}")

        if main_inventory: # 背包（库存/主物品栏）
            log_info(f"{TextFormat.GREEN}主物品栏:")
            for slot, item in sorted(main_inventory, key=lambda x: x[0]):
                log_info(f"  [{slot}] {item}")

        if armor: # 护甲栏
            log_info(f"{TextFormat.BLUE}装备:")
            for slot, item in armor:
                log_info(f"  [{slot}] {item}")

        if offhand: # 副手栏
            log_info(f"{TextFormat.PURPLE}副手:")
            for slot, item in offhand:
                log_info(f"  [{slot}] {item}")

    # 解析并显示末影箱物品
    def _ShowEnderChestItems(self, ender_items: list):
        for i, item in enumerate(ender_items, 1):
            item_info = self._ParseItem(item)
            log_info(f"  [{i}] {item_info}")

    # 处理玩家药水效果
    def _GetPotionEffectsDatas(self):
        effects = self.player_data.get('ActiveEffects', [])
        if not effects:
            log_warn("没有激活的药水效果")
            return
        # 显示药水数据
        for effect in effects:
            effect_info = {
                "存在药水效果": self._ProcessPotionEffectName(int(effect['Id'])) +
                        f" (等级 {int(effect['Amplifier']) + 1})",
                "剩余时间": f"{int(effect['Duration']) / 20:.1f}秒",
                "来源": "环境" if effect.get('Ambient', False) else "药水",
                "显示粒子": "是" if effect.get('ShowParticles', True) else "否"
            }
            print(f"{TextFormat.BLUE}[BACK]{TextFormat.CLEAR} " + ", ".join(f"{k}: {v}" for k, v in effect_info.items()))

    # 处理玩家在游戏中取得的能力
    def _GetAbilities(self):
        abilities = self.player_data.get('abilities', {})
        if not abilities:
            log_warn("无特殊能力数据")
            return
        # 显示能力数据
        ability_info = {
            "行走速度": f"{float(abilities.get('walkSpeed', 0.1)) * 100:.0f}%",
            "飞行速度": f"{float(abilities.get('flySpeed', 0.05)) * 100:.0f}%",
            "允许飞行": "是" if abilities.get('mayfly', False) else "否",
            "正在飞行": "是" if abilities.get('flying', False) else "否",
            "无敌": "是" if abilities.get('invulnerable', False) else "否",
            "可以建造": "是" if abilities.get('mayBuild', True) else "否",
            "即时建造": "是" if abilities.get('instabuild', False) else "否"
        }
        for key, value in ability_info.items():
            log_info(f"{key}: {value}")

    # 处理玩家在各个维度的坐标数据和传送门数据
    def _GetDimensionData(self):
        portal_cooldown = int(self.player_data.get('PortalCooldown', 0))
        if portal_cooldown > 0:
            log_info(f"传送门冷却: {portal_cooldown / 20:.1f}秒")
        else:
            log_info("传送门冷却: 无")

        if 'enteredNetherPosition' in self.player_data:
            pos = self.player_data['enteredNetherPosition']['pos']
            log_info(f"进入下界时的位置: X={float(pos[0]):.1f}, Y={float(pos[1]):.1f}, Z={float(pos[2]):.1f}")

        if 'enteredEndPosition' in self.player_data:
            pos = self.player_data['enteredEndPosition']['pos']
            log_info(f"进入末地时的位置: X={float(pos[0]):.1f}, Y={float(pos[1]):.1f}, Z={float(pos[2]):.1f}")

    # 解析单个物品信息
    @staticmethod
    def _ParseItem(item: dict) -> str:
        item_id = str(item['id']).replace('minecraft:', '')
        count = int(item['Count'])
        damage = int(item.get('Damage', 0))

        item_info = f"{item_id} x{count}"
        if damage > 0:
            item_info += f" (耐久 {damage})"

        # 添加自定义名称
        if 'tag' in item and 'display' in item['tag'] and 'Name' in item['tag']['display']:
            item_info += f" - '{str(item['tag']['display']['Name'])}'"

        # 添加附魔信息
        if 'tag' in item and 'Enchantments' in item['tag']:
            enchants = [f"{str(e['id']).replace('minecraft:', '')} {e['lvl']}"
                        for e in item['tag']['Enchantments']]
            item_info += f" [附魔: {', '.join(enchants)}]"

        return item_info

    # 处理游戏模式名
    @staticmethod
    def _ProcessGamemode(mode_id: int) -> str:
        modes = {
            0: "生存模式",
            1: "创造模式",
            2: "冒险模式",
            3: "旁观模式"
        }
        return modes.get(mode_id, f"{log_warn} 未知模式({mode_id})")

    # 此处为硬编码的药水名称，参考了zh.minecraft.wiki中的药水效果列表
    # 该方法被上面的_GetPotionEffectsDatas方法调用
    @staticmethod
    def _ProcessPotionEffectName(effect_id: int) -> str:
        effects = {
    1: "迅捷",
    2: "缓慢",
    3: "急迫",
    4: "挖掘疲劳",
    5: "力量",
    6: "瞬间治疗",
    7: "瞬间伤害",
    8: "跳跃提升",
    9: "反胃",
    10: "生命恢复",
    11: "抗性提升",
    12: "抗火",
    13: "水下呼吸",
    14: "隐身",
    15: "失明",
    16: "夜视",
    17: "饥饿",
    18: "虚弱",
    19: "中毒",
    20: "凋零",
    21: "生命提升",
    22: "伤害吸收",
    23: "饱和",
    24: "飘浮",
    25: "中毒",
    26: "潮涌能量",
    27: "缓降",
    28: "不祥之兆",
    29: "村庄英雄",
    30: "黑暗",
    31: "试炼之兆",
    32: "蓄风",
    33: "盘丝",
    34: "渗浆",
    35: "寄生",
    36: "袭击之兆"
            }

        return effects.get(effect_id, f"未知效果({effect_id})")

# 主函数
def ParseNBTFile(file_path: str):
    parser = NBTDataParser(file_path)
    if parser.ParseNBTData():
        parser.ShowALLData()
    else:
        log_error("无法解析NBT文件")