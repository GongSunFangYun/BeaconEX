# BeaconEX - Minecraft 伺服器命令行工具箱

[! [GitHub release](https://img.shields.io/github/v/release/GongSunFangYun/BeaconEX?style=flat-square)]()
[! [Downloads](https://img.shields.io/github/downloads/GongSunFangYun/BeaconEX/total?style=flat-square)]()
[! [Stars](https://img.shields.io/github/stars/GongSunFangYun/BeaconEX?style=flat-square)]()
[! [Forks](https://img.shields.io/github/forks/GongSunFangYun/BeaconEX?style=flat-square)]()
[! [Issues](https://img.shields.io/github/issues/GongSunFangYun/BeaconEX?style=flat-square)]()
[! [License](https://img.shields.io/github/license/GongSunFangYun/BeaconEX?style=flat-square)]()
! [简体中文支持]（https://img.shields.io/badge/ 简体中文-支持-ff8c00？style=flat-square&labelColor=ff8c00&color=ffd700）

## 它能做什麽？

>BeaconEX 是一个专为 Minecraft 服务器管理员设计的多功能命令行工具，提供以下核心功能：
>- 服务器状态查询：支持 Java 版和基岩版服务器的基本信息查询
>- 网络诊断工具：Ping 测试服务器响应能力
>-远程控制：通过 RCON 协议远程执行指令
>- 数据分析：玩家 NBT 解析、世界档案检查、日志分析
>- 实用工具：自动生成启动脚本、玩家活动热力图生成、DLL 注入、P2P 联机、世界备份、图标生成

## 构建帮助

- 目前为该程序的构建分为[Installer-Build]（https://github.com/FanYaRou/BeaconEX/releases/tag/INSTALLER-BUILD）和[Release-Build]（https://github.com/FanYaRou/BeaconEX/releases）两种方式
- Installer-Build 专门为安装程序进行发布
- Release-Build 专门为程式本体进行发布，以方便'''BeaconEX'''更新

## 系统需求

-作系统：Windows 10 或更新版本（不太支持 Windows 7）
- 运行环境：已配置 Java 运行环境
- 权限要求：建议以管理员身份运行
- 不要使用反向代理程序（steamcommunity302，steam++等）安装本地证书，否则会影响到更新功能

##用户需求
- 懂得'''Windows Powershell''' / '''Windows Command Prompt'''用法
- 了解终端程序中的主命令，参数和语法分别是什么

## 安装说明

1. 下载 BeaconEX.msi 安装包
2. 以管理员身份运行安装程序
3. 按照指引完成安装（程序会自动添加至系统环境变量）
4. 如环境变量未正确配置，请手动将安装目录添加至 Path 变量

## 命令使用指南
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
4. 如遇到任何问题，请提交 [issues]（https://github.com/FanYaRou/BeaconEX/issues/new）

## Star History

<a href="https://www.star-history.com/#GongSunFangYun/BeaconEX&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
 </picture>
</a>
