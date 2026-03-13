<div align="center">

<img src="winres/ICON.png" alt="Logo" width="160" height="160">

# BeaconEX - 我们Minecraft也要有自己的[图吧工具箱](#)

[![GitHub release](https://img.shields.io/github/v/release/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Downloads](https://img.shields.io/github/downloads/GongSunFangYun/BeaconEX/total?style=flat-square)]()
[![Stars](https://img.shields.io/github/stars/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Forks](https://img.shields.io/github/forks/GongSunFangYun/BeaconEX?style=flat-square)]()
[![Issues](https://img.shields.io/github/issues/GongSunFangYun/BeaconEX?style=flat-square)]()
[![License](https://img.shields.io/github/license/GongSunFangYun/BeaconEX?style=flat-square)]()
![简体中文支持](https://img.shields.io/badge/简体中文-支持-ff8c00?style=flat-square&labelColor=ff8c00&color=ffd700)

</div>

## 项目简介

BeaconEX 是一个用 Go 语言编写的 Minecraft 服务器管理工具箱，集成了服务器状态查询、日志分析、NBT文件处理、RCON远程控制、世界文件备份与分析等多种功能。得益于 Go 语言的跨平台特性，BeaconEX 可以在 Windows、Linux、macOS 等多个操作系统上运行。

## 核心特性

### 跨平台支持
- 基于 Go 语言开发，编译生成单一可执行文件
- 支持 Windows (7/10/11)、Linux、macOS、FreeBSD
- 支持 x86、x86_64、ARMv7、ARM64、RISC-V 等多种架构
- 无需安装依赖，开箱即用

### 主要功能模块
- **服务器查询**：自动识别 Java/基岩版，查询服务器状态、MOTD、玩家在线情况
- **Ping测试**：网络连通性测试，支持持续 Ping 模式，ICMP 不可用时自动降级 TCP
- **RCON远程控制**：通过 RCON 协议远程执行服务器指令，支持交互式 Shell
- **日志分析**：AI 辅助分析服务器日志，提取错误信息并给出解决方案
- **NBT文件处理**：查看 Minecraft NBT 格式数据（玩家存档、level.dat等）
- **NBT编辑器**：交互式 NBT 编辑器，支持增删改查，全面支持 Region 文件格式
- **启动脚本生成**：AI 根据需求生成优化的服务器启动脚本
- **热力图生成**：分析 playerdata 目录，生成玩家游玩时长热力图
- **世界分析**：扫描并统计世界文件信息，检测可能损坏的世界
- **DLL注入**：Windows 平台专用，将 DLL 注入 Minecraft 进程
- **服务器图标生成**：将图片转换为 64x64 的 server-icon.png
- **世界备份**：支持定时、循环备份世界文件

### 待实现特性
- [ ] 优化 world_backup 模块的输出（低优先级|不影响功能）
- [ ] 更新软件徽标（低优先级|闲的屁吃更新的）

## 编译指南

### 环境要求
- Go 1.16 或更高版本
- Git

### 获取源码
```bash
git clone https://github.com/GongSunFangYun/BeaconEX.git
cd BeaconEX
```

### 安装依赖
```bash
go mod download
```

### 编译项目

项目提供了编译脚本，会自动编译 Windows、Linux、macOS、FreeBSD 等多个平台的可执行文件，以及 Windows 平台的安装程序。

#### Windows 系统
双击运行 `build.bat` 或在命令行执行：
```batch
build.bat
```

#### Linux/macOS 系统
```bash
chmod +x build.sh
./build.sh
```

编译完成后，所有可执行文件会保存在 `build` 目录下。

### 编译产物

| 文件 | 平台 | 架构 | 说明 |
|------|------|------|------|
| `beaconex-windows-x86_64-v302.exe` | Windows | x86_64 | 主程序 |
| `beaconex-windows-x86-v302.exe` | Windows | x86 | 主程序 |
| `beaconex-windows-arm64-v302.exe` | Windows | ARM64 | 主程序 |
| `beaconex-linux-x86_64-v302` | Linux | x86_64 | 主程序 |
| `beaconex-linux-x86-v302` | Linux | x86 | 主程序 |
| `beaconex-linux-arm64-v302` | Linux | ARM64 | 主程序 |
| `beaconex-linux-armv7-v302` | Linux | ARMv7 | 主程序 |
| `beaconex-linux-riscv64-v302` | Linux | RISC-V | 主程序 |
| `beaconex-darwin-x86_64-v302` | macOS | Intel | 主程序 |
| `beaconex-darwin-arm64-v302` | macOS | Apple Silicon | 主程序 |
| `beaconex-freebsd-x86_64-v302` | FreeBSD | x86_64 | 主程序 |
| `beaconex-freebsd-arm64-v302` | FreeBSD | ARM64 | 主程序 |

注：下列产物由 **Advanced Installer** 构建
| 文件 | 平台 | 架构 | 说明 |
|------|------|------|------|
| `beaconex-windows-x86_64-v302-setup.exe` | Windows | x86_64 | 安装程序 |
| `beaconex-windows-x86-v302-setup.exe` | Windows | x86 | 安装程序 |
| `beaconex-windows-arm64-v302-setup.exe` | Windows | ARM64 | 安装程序 |

## 安装说明

### Windows 用户
推荐使用安装程序（带 `-setup.exe` 后缀的文件）进行安装：
1. 下载对应系统架构的安装程序（x86_64/x86/arm64）
2. 双击运行安装程序
3. 安装程序会将 `bex.exe` 添加到系统 PATH，你可以在任何地方直接使用 `bex` 命令

如果下载的是主程序文件，可以将其重命名为 `bex.exe` 并放置在任意目录，建议将该目录添加到系统 PATH 环境变量中。

### Linux/macOS 用户
```bash
# 下载对应平台的可执行文件
# 重命名为 bex 并添加执行权限
chmod +x bex
# 移动到系统 PATH 目录（可选）
sudo mv bex /usr/local/bin/
```

## 使用方法

### 通用格式
```bash
bex <模块> [参数...]
bex <模块> ?          查看模块详细帮助
```

### 模块列表
| 命令 | 简写 | 功能描述 |
|------|------|----------|
| query | q | 查询服务器状态 |
| ping | p | 网络连通性测试 |
| rcon | r | RCON 远程控制 |
| log | l | 服务器日志分析 |
| nbt | n | NBT 文件查看 |
| editnbt | e | 交互式 NBT 编辑器 |
| script | s | 生成服务器启动脚本 |
| heatmap | h | 玩家游玩时长热力图 |
| world | w | 世界文件分析 |
| dll | d | DLL 注入（仅 Windows） |
| icon | i | 生成服务器图标 |
| backup | b | 世界文件备份 |
| about | ! | 关于 BeaconEX |
| help | ? | 显示帮助信息 |

## 模块详细说明

### 1. 服务器查询模块 (query/q)
自动识别 Java/基岩版并查询服务器状态，包括 MOTD、版本、协议号、在线玩家等。

示例：
```bash
# 查询 Java 版服务器
bex query mc.hypixel.net

# 指定端口查询基岩版服务器
bex query play.cubecraft.net:19132

# 查询本地服务器（简写）
bex q 127.0.0.1:25565
```

### 2. Ping 测试模块 (ping/p)
对目标主机执行网络连通性测试，支持 ICMP 协议，当 ICMP 不可用时自动降级到 TCP 443/80 端口测量。

参数：
- `-f <次数>` Ping 次数（默认 4）
- `-v <秒>` Ping 间隔（默认 1.0）
- `-r` 持续 Ping 模式

示例：
```bash
# 基本 Ping 测试
bex ping 8.8.8.8

# 指定 Ping 10 次
bex ping mc.hypixel.net -f 10

# 持续 Ping 模式
bex p 127.0.0.1 -r
```

### 3. RCON 远程控制模块 (rcon/r)
通过 RCON 协议远程执行服务器指令，支持交互式 Shell。

格式：
```
bex rcon server@<地址>[:端口]
```

示例：
```bash
# 连接本地服务器（默认端口 25575）
bex rcon server@127.0.0.1

# 指定端口连接
bex r server@127.0.0.1:25575
```

连接后输入密码进入交互模式，支持的命令：
- 输入任意服务器指令执行
- `exit` 或 `quit` 退出会话
- Ctrl+C 强制退出

### 4. 日志分析模块 (log/l)
使用 AI 分析服务器日志，提取错误信息并给出解决方案。

示例：
```bash
# 分析日志文件
bex log server.log

# 分析最新日志
bex l ./logs/latest.log
```

### 5. NBT 文件查看模块 (nbt/n)
查看 Minecraft NBT 格式数据文件的内容，以树形结构展示。

示例：
```bash
# 查看 level.dat 文件
bex nbt level.dat

# 查看玩家存档
bex n world/playerdata/xxxxxxxx.dat
```

### 6. NBT 编辑器模块 (editnbt/e)
交互式 NBT 编辑器，支持增删改查操作，全面支持 Region 文件格式。

支持的文件格式：
- .dat / .nbt（NBT 文件）
- .schematic（建筑结构）
- .litematic（Litematica 投影）
- .mca / .mcr（Region 区域文件）

编辑器快捷键：
- ↑/↓ 导航
- Enter/Space 展开/折叠节点
- e 编辑当前节点值
- r 重命名当前节点
- a 新增子节点
- d/Delete 删除当前节点
- Ctrl+S 保存更改
- Ctrl+Q 退出编辑器
- ? 显示帮助

示例：
```bash
# 编辑 level.dat
bex editnbt level.dat

# 编辑玩家存档
bex e world/playerdata/xxxxxxxx.dat

# 编辑 Region 文件
bex e world/region/r.0.0.mca
```

### 7. 启动脚本生成模块 (script/s)
通过 AI 根据需求生成优化的服务器启动脚本，自动适配操作系统类型。

参数：
- `-o <路径>` 脚本输出目录（默认当前目录）

示例：
```bash
# 生成 Paper 1.20.1 服务器启动脚本
bex script "1.20.1 Paper，4G 内存"

# 生成原版服务器脚本并指定输出目录
bex s "原版 1.21" -o ./server
```

### 8. 热力图生成模块 (heatmap/h)
分析 playerdata 目录，生成玩家游玩时长热力图。

参数：
- `-o <路径>` 同时将结果保存为文本文件

颜色说明：
- 蓝色：< 1 天（新玩家）
- 绿色：1~7 天（普通玩家）
- 黄色：7~30 天（活跃玩家）
- 红色：> 30 天（核心玩家）

示例：
```bash
# 显示热力图
bex heatmap ./world/playerdata

# 显示并保存结果
bex h ./playerdata -o ./output
```

### 9. 世界分析模块 (world/w)
扫描指定目录下的所有 Minecraft 世界文件，统计文件数量、大小、维度等信息，检测可能损坏的世界。

示例：
```bash
# 分析服务器目录下的所有世界
bex world ./server

# 分析指定世界目录
bex w ./worlds
```

### 10. DLL 注入模块 (dll/d)
Windows 平台专用，将 DLL 注入到 Minecraft 进程（需要管理员权限）。

参数：
- `<DLL路径>` 要注入的 DLL 文件路径（第一个参数）
- `-p <进程名>` 目标进程名（默认 Minecraft.Windows.exe）
- `-i` 使用上次保存的 DLL 路径和进程名直接注入
- `-c` 重置配置文件中的 DLL 路径和进程名

示例：
```bash
# 基本用法
bex dll mod.dll

# 指定目标进程
bex dll mod.dll -p javaw.exe

# 使用上次的配置注入
bex dll -i

# 重置配置文件
bex dll -c
```

### 11. 服务器图标生成模块 (icon/i)
将图片转换为 64×64 的 server-icon.png。

参数：
- `-o <路径>` 输出目录（默认当前目录）
- `-n <名称>` 输出文件名（默认 server-icon.png）

支持的输入格式：PNG、JPG、BMP、GIF

示例：
```bash
# 生成服务器图标
bex icon ./logo.png

# 指定输出目录和文件名
bex i ./logo.jpg -o ./server -n icon.png
```

### 12. 世界备份模块 (backup/b)
支持冷/热备份，定时定量备份世界、模组、插件等文件夹。

参数：
- `<备份目录>` 要备份的文件夹路径（必需）
- `-o <路径>` 指定备份输出路径（默认保存至目标目录上一级的 bex_backup 文件夹）
- `-v <时间>` 备份间隔，支持格式：30m、1h30m
- `-r` 循环执行模式（需配合 -v 使用）
- `-x <次数>` 最大保留的备份数量

示例：
```bash
# 单次备份
bex backup ./server/world

# 定时循环备份（每小时备份一次，最多10次）
bex backup ./server/world -v 1h -r -x 10

# 指定输出目录
bex backup ./server/world -o D:/backups
```

### 13. 关于模块 (about/!)
显示程序信息、版本、开发者、开源协议等。

示例：
```bash
bex about
bex !
```

## 配置文件

BeaconEX 会在程序所在目录自动生成 `config.json` 配置文件：

```json
{
  "last_check_update": "26-03-11 12:00",
  "disable_update": false,
  "dll_injector_target_file_path": "C:\\path\\to\\mod.dll",
  "dll_injector_target_process_name": "Minecraft.Windows.exe"
}
```

- `last_check_update`：上次检查更新的时间
- `disable_update`：是否禁用自动更新检查（设置为 true 可完全禁用更新提醒）
- `dll_injector_target_file_path`：上次成功注入的 DLL 路径（用于 -i 参数）
- `dll_injector_target_process_name`：上次注入的目标进程名（用于 -i 参数）

## 注意事项

1. DLL 注入功能仅支持 Windows 系统，且需要以管理员身份运行
2. RCON 功能需要服务器开启 enable-rcon=true 并设置 rcon.password
3. 日志分析和启动脚本生成功能需要联网使用 AI 服务
4. NBT 编辑器支持 Region 文件格式（.mca/.mcr），可以进行查看和修改
5. 建议在 PowerShell、CMD、Terminal 中运行，避免双击执行
6. Windows 用户推荐使用安装程序，会自动配置环境变量
7. 如遇到任何问题，请提交 [issues](https://github.com/GongSunFangYun/BeaconEX/issues/new)

## 许可证

本项目采用 GNU Lesser General Public License v3.0 开源协议。

## 版权信息

- 开发者：GongSunFangYun
- 项目地址：https://github.com/GongSunFangYun/BeaconEX
- 反馈邮箱：misakifeedback@outlook.com
- 计算机软件著作权登记：2025SR203****

## 星标历史

<a href="https://www.star-history.com/#GongSunFangYun/BeaconEX&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=GongSunFangYun/BeaconEX&type=Date" />
 </picture>
</a>