"""------------------------外部库------------------------"""
import argparse
import ssl
ssl._create_default_https_context = ssl._create_unverified_context
'''------------------------本地库------------------------'''
from functions.ping import *
from functions.LogAnalysis import *
from functions.RCON import *
from functions.update import *
from functions.query import *
from functions.NBT import *
from functions.HeatMap import *
from functions.LaunchServer import *
from functions.WorldChecker import *
from functions.utils import *

# 命令参数构建
# noinspection PyUnboundLocalVariable
def main():
    parser = argparse.ArgumentParser(
        formatter_class=argparse.RawTextHelpFormatter,
        add_help=False,
        usage=argparse.SUPPRESS,
        epilog=f"""
{TextFormat.YELLOW}请注意！如果您的服务器为本地部署
{TextFormat.RED}请不要使用任何反向代理软件提供的IP进行RCON远程控制！
{TextFormat.YELLOW}您只需要在RCON远程控制模式下指定IP为127.0.0.1即可！{TextFormat.CLEAR}
{TextFormat.BLUE}值得一提的一点是，以下牵涉到本地路径的命令，其路径如果包含中文请使用" "包装为字符串，否则大概率出现路径错误！{TextFormat.CLEAR}
{TextFormat.GREEN}比较邪门的一点就是如果你在检查非原版level.dat(例如NukkitX端自生成的level.dat)，大概率出现缺东少西的问题，无需在意({TextFormat.CLEAR}
{TextFormat.BLUE}================================================== {TextFormat.YELLOW}完整使用示例{TextFormat.BLUE} =================================================={TextFormat.CLEAR}
{TextFormat.GREEN}1. 查询Java版服务器:{TextFormat.CLEAR}
  bex -java -ip mc.example.com -p 25565 (如果端口默认为25565则可以省略-p参数)

{TextFormat.GREEN}2. 查询基岩版服务器:{TextFormat.CLEAR}  
  bex -bedrock -ip mc.example.com -p 19132 (如果端口默认为19132则可以省略-p参数)
  
{TextFormat.CYAN}3. Ping测试:{TextFormat.CLEAR}
  bex -ping mc.example.com -pc 1~50 (可选项，Ping执行次数。不指定则默认Ping4次)

{TextFormat.PURPLE}4. RCON执行单个命令:{TextFormat.CLEAR}
  bex -rcon -ip mc.example.com -p 服务器端口 (如果端口默认为25565则可以省略-p参数) -rp RCON端口 -pw RCON密码 -cmd "你要执行的命令"

{TextFormat.PURPLE}5. RCON交互模式:{TextFormat.CLEAR}
  bex -rcon -ip mc.example.com -p 服务器端口 (如果端口默认为25565则可以省略-p参数) -rp RCON端口 -pw RCON密码 -cg

{TextFormat.RED}6. 日志分析:{TextFormat.CLEAR}
  bex -la -lp 目标日志文件路径(例如C:/Server/MyServer/logs/latest.log)
  
{TextFormat.RED}7. 玩家NBT分析:{TextFormat.CLEAR}
  bex -nbt -np 目标NBT文件路径(例如C:/Server/MyServer/world/playerdata/XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX.dat)  
  
{TextFormat.RED}8. 服务器启动脚本生成:{TextFormat.CLEAR}
  bex -genbat -rq "你的需求"(需要声明核心完整名称，其他参数选择性需求就行了)  
  
{TextFormat.RED}9. 玩家游玩天数热力图生成:{TextFormat.CLEAR}
  bex -hp -np 目标NBT文件夹路径(例如C:/Server/MyServer/world/playerdata/) [可选项: -mp 指定一张图表中最多存在多少个数据，默认为15个/张，超出的额外生成新表]
  
{TextFormat.RED}10. level.dat完整性检查:{TextFormat.CLEAR}  
  bex -wc -np 服务器根目录路径(例如C:/Server/MyServer/)
  
{TextFormat.YELLOW}12. 检查版本:{TextFormat.CLEAR}
  bex -v

{TextFormat.YELLOW}13. 程序更新:{TextFormat.CLEAR}
  bex -update

{TextFormat.YELLOW}14. 关于我们:{TextFormat.CLEAR}
  bex -about 
{TextFormat.BLUE}=================================================================================================================={TextFormat.CLEAR} 
"""
    )
    
    # 主帮助信息
    parser._positionals.title = f"{TextFormat.YELLOW}基本命令{TextFormat.CLEAR}"
    parser._optionals.title = f"{TextFormat.YELLOW}其他选项{TextFormat.CLEAR}"
    
    # 服务器查询组
    server_group = parser.add_argument_group(
        f"{TextFormat.GREEN}服务器查询可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}查询Minecraft服务器{TextFormat.CLEAR}"
    )
    server_group.add_argument('-java', '--java', action='store_true', 
                            help='查询Java版服务器')
    server_group.add_argument('-bedrock', '--bedrock', action='store_true',
                            help='查询基岩版服务器')
    
    # RCON组
    rcon_group = parser.add_argument_group(
        f"{TextFormat.PURPLE}RCON远程控制可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}远程执行Minecraft服务器命令{TextFormat.CLEAR}"
    )
    rcon_group.add_argument('-rcon', '--rcon', action='store_true',
                          help='启用RCON模式')
    rcon_group.add_argument('-rp', '--rcon-port', type=int, default=25575,
                            help=f'RCON端口 {TextFormat.YELLOW}(默认: 25575){TextFormat.CLEAR}')
    rcon_group.add_argument('-pw', '--password', 
                          help=f'RCON密码')
    rcon_group.add_argument('-cmd', '--command', 
                          help='执行单个RCON命令')
    rcon_group.add_argument('-cg', '--command-group', action='store_true',
                          help='进入交互式命令组模式')

    # 网络测试组
    network_group = parser.add_argument_group(
        f"{TextFormat.CYAN}网络测试可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}测试服务器连接{TextFormat.CLEAR}"
    )
    network_group.add_argument('-ping', '--ping', metavar='IP', nargs='?', const=None,
                               help='执行Ping测试，参数为IP或域名')
    network_group.add_argument('-pc', '--ping-count', type=int, default=4,
                               help=f'Ping次数 (默认: 4，范围1-50)')

    # 日志分析组
    log_group = parser.add_argument_group(
        f"{TextFormat.RED}日志分析可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}分析Minecraft服务器日志{TextFormat.CLEAR}"
    )
    log_group.add_argument('-la', '--log-analysis', action='store_true',
                         help='分析日志文件')
    log_group.add_argument('-lp', '--log-path',
                         help='指定日志文件路径')
    
    # 程序更新组
    update_group = parser.add_argument_group(
        f"{TextFormat.GREEN}检查版本&更新可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}检查并更新工具{TextFormat.CLEAR}"
    )
    update_group.add_argument('-v', '--version', action='store_true',
                           help='显示版本并检查更新')
    update_group.add_argument('-update', '--update', action='store_true',
                           help='下载并安装最新版本')
    
    # 公用参数
    common_group = parser.add_argument_group(
        f"{TextFormat.YELLOW}通用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}适用于大多数命令{TextFormat.CLEAR}"
    )
    common_group.add_argument('-ip', '--ip', 
                            help=f'服务器IP/域名')
    common_group.add_argument('-p', '--port', type=int,
                            help=f'服务器端口')
    common_group.add_argument('-h', '--help', action='help',
                              help=f'{TextFormat.GREEN}显示此帮助信息{TextFormat.CLEAR}')
    # NBT解析组
    nbt_group = parser.add_argument_group(
        f"{TextFormat.PURPLE}NBT解析可用参数{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}解析Minecraft NBT数据文件{TextFormat.CLEAR}"
    )
    nbt_group.add_argument('-nbt', '--nbt', action='store_true',
                           help='解析NBT数据文件')
    nbt_group.add_argument('-np', '--nbt-path',
                           help='指定NBT文件路径')
    # 生成脚本组
    genbat_group = parser.add_argument_group(
        f"{TextFormat.GREEN}启动脚本生成{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}通过AI生成优化后的启动脚本{TextFormat.CLEAR}"
    )
    # 修改 genbat 参数组定义
    genbat_group.add_argument('-genbat', '--generate-bat', action='store_true',
                              help='生成启动脚本(start.bat)')
    genbat_group.add_argument('-rq', '--request',
                              help='生成要求，例如："1.20.1原版服务器，4G内存"')
    # 关于信息
    about_group = parser.add_argument_group(
        f"{TextFormat.CYAN}关于{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}程序信息{TextFormat.CLEAR}"
    )
    about_group.add_argument('-about', '--about', action='store_true',
                             help='显示程序信息和版权声明')

    # 热力图组
    heatmap_group = parser.add_argument_group(
        f"{TextFormat.CYAN}玩家热力图分析{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}生成玩家活动热力图(PNG格式){TextFormat.CLEAR}"
    )
    heatmap_group.add_argument('-hp', '--heatmap', action='store_true',
                               help='生成玩家游玩时长热力图(PNG)')
    heatmap_group.add_argument('-mp', '--max-player', type=int, default=15,
                               help=f'每张图表显示的最大玩家数 {TextFormat.YELLOW}(默认: 15){TextFormat.CLEAR}')

    # 世界校验组
    world_group = parser.add_argument_group(
        f"{TextFormat.CYAN}世界完整性检查{TextFormat.CLEAR}",
        f"{TextFormat.BLUE}检查Minecraft世界数据完整性{TextFormat.CLEAR}"
    )
    world_group.add_argument('-wc', '--world-check', action='store_true',
                             help='检查世界完整性')
    world_group.add_argument('-wp', '--world-path',
                             help='指定服务器根目录路径')

    args = parser.parse_args()

    '''------------------------下为参数检查部分------------------------'''

    # 更新检查
    if args.version:
        CheckForUpdates()
        sys.exit(0)
    elif args.update:
        if not DownloadUpdate():
            sys.exit(1)
        sys.exit(0)

    # 关于我们
    if args.about:
        output(f"""\n
        {TextFormat.CYAN}==================================================
        {TextFormat.GREEN} BeaconEX - A Minecraft Server Toolbox
        {TextFormat.CYAN}=================================================={TextFormat.CLEAR}
        Version: {TextFormat.GREEN}{Version()}{TextFormat.CLEAR}
        Authors: GongSunFangYun & MCAST-Team
        
        E-mail: <misakifeedback@outlook.com>
        GitHub: https://github.com/GongSunFangYun/BeaconEX

        {TextFormat.YELLOW}Copyright © 2024-2025 GongSunFangYun & MCAST-Team
        The software is a temporary {TextFormat.RED}closed-source{TextFormat.YELLOW} software, 
        and will be open-sourced under the GPL-V3 license after improvements.
        {TextFormat.CLEAR}
        """)
        sys.exit(0)

    # 参数完整性检查
    if not (args.java or args.bedrock or args.rcon or args.log_analysis
            or args.nbt or args.generate_bat or args.version
            or args.update or args.about or args.heatmap
            or args.world_check or args.ping):
        log_error("""Exception in thread "main" java.lang.IllegalArgumentException: No operation mode specified
                                        at cn.gongsunqiluo.bex.parseArguments(bex.java:229)
                                        at cn.gongsunqiluo.bex.main(bex.java:18)
                                    Caused by: MissingOperationModeException: Must specify at least one operation mode
                                        ... 2 more
        """)
        log_info("请使用bex.exe -h/--help 查看参数帮助信息，然后再次尝试输入命令。")
        input()
        sys.exit(1)

    # Java/Bedrock服务器检查
    if args.java or args.bedrock:
        if not args.ip:
            log_error("查询服务器必须指定IP地址(-ip)")
            sys.exit(1)

    # 世界检查
    if args.world_check:
        if not args.world_path:
            log_error("世界检查必须指定服务器路径(-wp)")
            sys.exit(1)
        if not os.path.exists(args.world_path):
            log_error(f"指定路径不存在: {args.world_path}")
            sys.exit(1)

    # RCON检查
    if args.rcon:
        if not args.ip:
            log_error("RCON模式必须指定IP地址(-ip)")
            sys.exit(1)
        if not args.password:
            log_error("RCON模式必须指定密码(-pw)")
            sys.exit(1)
        if not (args.command or args.command_group):
            log_error("RCON模式必须指定命令(-cmd)或使用交互模式(-cg)")
            sys.exit(1)

    # 日志分析检查
    if args.log_analysis:
        if not args.log_path:
            log_error("日志分析必须指定日志路径(-lp)")
            sys.exit(1)

    # NBT分析检查
    if args.nbt:
        if not args.nbt_path:
            log_error("NBT分析必须指定文件路径(-np)")
            sys.exit(1)

    # 启动脚本生成检查
    if args.generate_bat:
        if not args.request:
            log_error("生成启动脚本必须指定需求描述(-rq)")
            sys.exit(1)

    # 校验区分-ping和-java/-bedrock参数
    if args.ping is not None:  # 明确检查是否为None而不是简单的if args.ping
        if args.ping and (args.java or args.bedrock or args.rcon or args.log_analysis
                          or args.nbt or args.generate_bat or args.heatmap or args.world_check):
            log_error("不能同时使用-ping和其他参数")
            if args.ping.isdigit():  # 如果用户直接输入数字
                log_error("""参数格式错误！正确用法：
      bex -ping <IP或域名> [可选参数: -pc 次数]
    示例:
      bex -ping example.com
      bex -ping 1.1.1.1 -pc 10""")
            sys.exit(1)

        # 校验ping次数
        if args.ping_count < 1 or args.ping_count > 50:
            log_error("Ping次数必须在1-50之间")
            sys.exit(1)

        # 确保没有同时使用查询模式
        if args.java or args.bedrock:
            log_error("错误：不能同时使用-ping和-java/-bedrock参数")
            sys.exit(1)

    # 日志分析参数检查
    if args.log_analysis:
        if not args.log_path:  # 检查是否提供了日志路径
            log_error("日志分析必须指定日志文件路径，请使用 -lp 参数")
            sys.exit(1)

        # 检查文件是否存在且可读
        if not os.path.exists(args.log_path):
            log_error(f"指定的日志文件不存在: {args.log_path}")
            sys.exit(1)
            # 检查文件是否有权限读取
        if not os.access(args.log_path, os.R_OK):
            log_error(f"没有读取日志文件的权限: {args.log_path}")
            sys.exit(1)

        # 检查文件扩展名是否合理
        if not args.log_path.lower().endswith(('.log', '.txt', '.gz')):
            log_warn("日志文件扩展名非常规，建议使用 .log 或 .txt 格式文件")

        # 实际执行日志分析
        PerformLogAnalysis(args)  # 该函数接受日志路径作为参数
        sys.exit(0)  # 分析完成后退出

    # 设置RCON默认端口
    if getattr(args, 'rcon', False):
        game_port = args.p if hasattr(args, 'p') else (args.port if hasattr(args, 'port') else 25565)
        rcon_port = args.rp if hasattr(args, 'rp') else (args.rcon_port if hasattr(args, 'rcon_port') else 25575)
    else:
        game_port = 25565 if args.java else 19132
        port = None
        if hasattr(args, 'p') and args.p:
            port = args.p
        elif hasattr(args, 'port') and args.port:
            port = args.port
        if port:
            game_port = port

    # 热力图分析参数检查
    if args.heatmap:
        if not args.nbt_path:  # 复用nbt_path参数
            log_error("热力图分析必须指定NBT文件路径(-np)")
            sys.exit(1)
        if args.max_player < 1 or args.max_player > 50:
            log_error("每页玩家数必须在1-50之间")
            sys.exit(1)

    # 综合性校验，检查和调用各自的函数
    if args.ping:
        Ping(args.ping, args.ping_count)
    elif getattr(args, 'rcon', False):
        if args.command_group:
            InteractiveRCON(args.ip, game_port, rcon_port, args.password)
        else:
            RCONExecute(args.ip, game_port, rcon_port, args.password, args.command)
    elif args.java:
        CheckJavaServer(args.ip, game_port)
    elif args.bedrock:
        CheckBedrockServer(args.ip, game_port)
    elif args.nbt:
        ParseNBTFile(args.nbt_path)
    elif args.generate_bat:
        GenerateLaunchBat(request=args.request)
    elif args.heatmap:
        try:
            ProcessHeatMap(
                nbt_path=args.nbt_path,
                players_per_chart=args.max_player
            )
        except FileNotFoundError as e:
            log_error(f"NBT文件夹不存在: {str(e)}")
            sys.exit(1)
        except Exception as e:
            log_error(f"热力图生成失败: {str(e)}")
            sys.exit(1)
    elif args.world_check:
        check_world_integrity(args.world_path)
    elif args.ping:
        if not args.ping:
            parser.error("必须指定要ping的目标地址")
        Ping(args.ping, args.ping_count)

# 入口点
if __name__ == "__main__":
     main()