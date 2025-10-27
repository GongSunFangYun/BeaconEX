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

## 命令使用指南（待更新）
### 查询功能
```bash
# 查询 Java 版服务器（默认端口可省略 -p 参数）
bex -java -ip mc.example.com -p 25565
```
```bash
# 查询基岩版服务器（默认端口可省略 -p 参数）
bex -bedrock -ip mc.example.com -p 19132
```
### 远程控制功能
```bash
# RCON 执行单个命令（默认端口可省略 -p 参数）
bex -rcon -ip mc.example.com -p 25565 -rp 25575 -pw password -cmd "say Hello"
```
```bash
# RCON 交互模式（默认端口可省略 -p 参数）
bex -rcon -ip mc.example.com -p 25565 -rp 25575 -pw password -cg
```
### 数据包测试功能
```bash
# 网络测试
bex -ping mc.example.com -pc Ping 次数（选填，默认 4 次）
```
### 数据分析功能
```bash
# 日志分析（指定文档路径）
bex -la -lp "C:/Server/logs/latest.log"
```
```bash
# 玩家 NBT 分析（指定文档路径）
bex -nbt -np "C:/Server/world/playerdata/XXXXXXXX.dat"
```
```bash
# level.dat 完整性检查（指定文件夹路径）
bex -wc -np "C:/Server/"
```
### 生成功能
```bash
# 生成启动脚本（指定文档路径）
bex -genbat -rq “paper1.20.1 核心，最大分配 4G，最小分配 2G。”
```
```bash
# 生成玩家热力图（指定文件夹路径）
bex -hp -np "C:/Server/world/playerdata/"
```
### 杂项
```bash
# 检查版本
bex -v
```
```bash
# 程序更新
bex -update
```
```bash
# 关于我们
bex -about
```
## 注意事项

1. 使用 RCON 功能时请确保在'''server.properties'''正确配置了以下内容：
```bash
enable-rcon=true
rcon.password=你的 RCON 连接密码
rcon.port=你的 RCON 端口
```
4. 如遇到任何问题，请提交 [issues](https://github.com/GongSunFangYun/BeaconEX/issues/new)

## Star History

<a href="https://www.star-history.com/#GongSunFangYun/BeaconEX&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
 </picture>
</a>
