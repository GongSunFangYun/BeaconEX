<div align="center">

<img src="Resources/ICON.png" alt="Logo" width="160" height="160">

# BeaconEX - 我们Minecraft也要有自己的[图吧工具箱](https://tubatool.com.cn/?lang=ZH)

[![GitHub release](https://img.shields.io/github/v/release/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Downloads](https://img.shields.io/github/downloads/GongSunFangYun/BeaconEX/total?style=flat-square)]()
[![Stars](https://img.shields.io/github/stars/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Forks](https://img.shields.io/github/forks/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Issues](https://img.shields.io/github/issues/GongSunFangYun/BeaconEX?style=flat-square)]()
[![License](https://img.shields.io/github/license/GongSunFangYun/BeaconEX?style=flat-square)]()
![简体中文支持](https://img.shields.io/badge/简体中文-支持-ff8c00?style=flat-square&labelColor=ff8c00&color=ffd700)

</div>

## 它能做什么？

- BeaconEX的设计初衷就是为不同维度的Minecraft玩家提供服务，让大部分玩家都可以找到适合自己所使用的功能
- 不过主体功能更偏向于服务器运维管理，适合服主/服管/服内玩家使用
- 功能列表（v2.0.0）：
>- P2P联机功能
>- 热力图生成功能
>- 服务器启动脚本生成功能
>- 服务器徽标生成功能
>- 日志分析功能
>- 基岩版DLL注入功能
>- NBT分析功能
>- NBT编辑功能
>- Ping服务器功能
>- 查询服务器信息功能
>- RCON远程控制功能
>- 世界分析功能
>- 世界备份功能
>- 正在开发...
- **下为不同维度玩家推荐使用的功能：**
>- **对于服务器运维**：查询双版本服务器信息，生成服务器玩家游玩天数热力图，生成服务器启动脚本，世界自动备份，日志分析，世界分析，RCON远程控制，测试服务器延迟抖动...
>- **对于普通用户**：NBT编辑器，NBT分析器，P2P联机，基岩版客户端注入，查询自己游玩的服务器信息，测试自己游玩的服务器延迟抖动，分析客户端崩溃日志，为自己的存档备份...
>- **对于基岩版玩家**：基岩版DLL客户端注入，世界备份[需要管理员权限，因为存档位于AppData]，查询自己游玩的服务器信息，测试自己游玩的服务器延迟抖动...

## 系统需求
- 操作系统：当前仅支持Windows 7+(7/8/8.1/10/11)
- 运行环境：计算机已安装```命令提示符/Powershell/终端```
- 权限要求：除P2P联机模块需要管理员权限之外，其他一般不需要管理员权限

## 安装说明
1. 下载 BeaconEX-win-Installer-vX.X.X.msi 安装包
2. 以管理员身份运行安装程序
3. 按照指引完成安装（程序会自动将安装路径添加至系统环境变量）
4. 如环境变量未正确配置，请手动将安装目录添加至 Path 变量（例如直接使用zip）

## 命令使用指南
### 查询服务器相关
```bash
# 查询 Java 版服务器（支持解析A/AAA/SRV记录，支持解析纯IPV4/V6与纯主机名，不带端口默认为25565）
bex -query -java -t mc.example.com
bex -query -java -t mc.example.com:12345
bex -query -java -t XXX.XXX.XXX.XXX
bex -query -java -t XXX.XXX.XXX.XXX:12345
```
```bash
# 查询基岩版服务器（支持解析A/AAA/SRV记录，支持解析纯IPV4/V6与纯主机名，不带端口默认为19132）
bex -query -bedrock -t mc.example.com
bex -query -bedrock -t mc.example.com:12345
bex -query -bedrock -t XXX.XXX.XXX.XXX
bex -query -bedrock -t XXX.XXX.XXX.XXX:12345
```
### 远程控制相关
```bash
# RCON 执行单个命令（支持解析A/AAA/SRV记录，支持解析纯IPV4/V6与纯主机名，不带端口默认JE25565/BE19132，RCON不帶端口默认25575）
bex -rcon -t mc.example.com -rp 12345 -rpw 123456 -cmd "say hello world！"
bex -rcon -t mc.example.com:12345 -rpw 123456 -cmd "say hello world"
bex -rcon -t XXX.XXX.XXX.XXX:12345 -rpw 123456 -cmd "say hello world"
bex -rcon -t XXX.XXX.XXX.XXX -rpw 123456 -cmd "say hello world"
```
```bash
# RCON 交互模式（支持解析A/AAA/SRV记录，支持解析纯IPV4/V6与纯主机名，不带端口默认JE25565/BE19132，RCON不帶端口默认25575）
bex -rcon -t mc.example.com -rp 12345 -rpw 123456 -cg
bex -rcon -t mc.example.com:12345 -rpw 123456 -cg
bex -rcon -t XXX.XXX.XXX.XXX -rpw 123456 -cg
bex -rcon -t XXX.XXX.XXX.XXX:12345 -rpw 123456 -cg
```
```bash
# RCON 脚本解释器模式
bex -rcon -s
```
### 延迟测试相关
```bash
# Ping单次测试
bex -ping -t mc.example.com
bex -ping -t mc.example.com -pf 10
```
```bash
# Ping持续测试
bex -ping -t mc.example.com -r
bex -ping -t mc.example.com -r -pi 0.5
```
### 数据处理相关
```bash
# 日志分析（指定日志文件路径）
bex -log -lp "C:/Server/logs/latest.log"
```
```bash
# 玩家 NBT 分析（指定单个玩家NBT路径）
bex -nbt -np "C:/Server/world/playerdata/XXXXXXXX.dat"
```
```bash
# level.dat 完整性检查（如果世界文件夹是分散的，则只指定服务器根目录便可，模块会自动扫描 level.dat 文件位置）
bex -world -wp "C:/Server/worlds"
```
```bash
# NBT文件编辑器（使用自分支NBTExplorerCN）
bex -editnbt -np "C:/Server/worlds/overworld/level.dat"
```
### 生成数据相关
```bash
# 生成启动脚本（指定文档路径）
bex -genbat -rq “paper1.20.1 核心，最大分配 4G，最小分配 2G。”
```
```bash
# 生成玩家热力图（指定文件夹路径）
bex -hp -np "C:/Server/world/playerdata/"
```
```bash
# 生成处理后的服务器徽标
bex -icon -pp "C:/Picture/vanilla-icon.png"
bex -icon -pp "C:/Picture/vanilla-icon.png" -od "C:/Server"
bex -icon -pp "C:/Picture/vanilla-icon.png" -od "C:/Server" -pn "custom-name.png"
```
### P2P联机相关
```bash
# P2P创建网络/加入网络/列出用户
bex -p2p -cn -n "MyNetwork" -pw "MyPassword"  
bex -p2p -jn -n "MyNetwork" -pw "MyPassword"  
bex -p2p -l
```
### DLL注入器相关
```bash
# 注入DLL
bex -injector -dp  
bex -injector -dp "Latite.dll"  
bex -injector -dp "Latite.dll" -ct "Example.exe"
bex -injector -dp "Latite.dll" -ct "Example.exe" -tm 1m30s  
```
```bash
# 立即注入上次注入的DLL（保存到配置文件）
bex -injector -i
```
### 世界备份相关（生成的备份文件位于工作目录下的```BEX_Backups```文件夹）
```bash
# 定时备份与循环备份
bex.exe -backup -bp "C:/Server" -sd "worlds/*" -bt 1h30m -le -mx 10
bex.exe -backup -bp "C:/Server" -sd "worlds/*" -bt 1h30m -le
bex.exe -backup -bp "C:/Server" -sd "worlds/nether" -bt 1h30m
```
```bash
# 立即备份
bex.exe -backup -bp "C:/Server" -sd "worlds/nether"
```
### 杂项
```bash
# 关于我们
bex -about
```
```bash
# 取得帮助
bex ?
bex -help
bex -MODULE_NAME ?
```
## 注意事项  
1. 使用 RCON 功能时请确保在'''server.properties'''正确配置了以下内容：
```bash
enable-rcon=true
rcon.password=你的 RCON 连接密码
rcon.port=你的 RCON 端口
```
2. 如遇到任何问题，请提交 [issues](https://github.com/GongSunFangYun/BeaconEX/issues/new)

## 星标历史

<a href="https://www.star-history.com/#GongSunFangYun/BeaconEX&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
 </picture>
</a>
