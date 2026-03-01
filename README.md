<div align="center">

<img src="winres/ICON.png" alt="Logo" width="160" height="160">

# BeaconEX - 我们Minecraft也要有自己的[图吧工具箱](https://tubatool.com.cn/?lang=ZH)

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
- 支持 Windows (7/10/11)、Linux、macOS
- 无需安装依赖，开箱即用
- Windows 旧终端（CMD/PowerShell）自动适配 ANSI 颜色支持

### 主要功能模块
- **服务器查询**：自动识别 Java/基岩版，查询服务器状态、MOTD、玩家在线情况
- **Ping测试**：网络连通性测试，支持持续 Ping 模式
- **RCON远程控制**：通过 RCON 协议远程执行服务器指令
- **日志分析**：AI 辅助分析服务器日志，提取错误信息并给出解决方案
- **NBT文件处理**：查看 Minecraft NBT 格式数据（玩家存档、level.dat等）
- **启动脚本生成**：AI 根据需求生成优化的服务器启动脚本
- **热力图生成**：分析 playerdata 目录，生成玩家游玩时长热力图
- **世界分析**：扫描并统计世界文件信息，检测可能损坏的世界
- **DLL注入**：Windows 平台专用，将 DLL 注入 Minecraft 进程
- **服务器图标生成**：将图片转换为 64x64 的 server-icon.png
- **世界备份**：支持定时、循环备份世界文件

### 编译指南

#### 环境要求
- Go 1.16 或更高版本
- Git

#### 获取源码
```bash
git clone https://github.com/GongSunFangYun/BeaconEX.git
cd BeaconEX
```

#### 安装依赖
```bash
go mod download
```

#### 编译项目

项目提供了编译脚本，会自动编译 Windows、Linux、macOS 等多个平台的可执行文件。

##### Windows 系统
双击运行 `build.bat` 或在命令行执行：
```batch
build.bat
```

##### Linux/macOS 系统
```bash
chmod +x build.sh
./build.sh
```

编译完成后，所有可执行文件会保存在 `build` 目录下。

#### 编译产物

| 文件 | 平台 |
|------|------|
| `bex_windows_amd64.exe` | Windows 64位 |
| `bex_windows_arm64.exe` | Windows ARM64 |
| `bex_linux_amd64` | Linux 64位 |
| `bex_linux_arm64` | Linux ARM64 |
| `bex_darwin_amd64` | macOS Intel |
| `bex_darwin_arm64` | macOS Apple Silicon |

## 使用方法

### Windows 用户
1. 双击运行 `build.bat`
2. 等待编译完成
3. 在 `build` 目录下获取编译好的可执行文件

### Linux/macOS 用户
```bash
# 添加执行权限
chmod +x build.sh

# 执行编译
./build.sh
```

## 编译产物说明

编译完成后，在 `build` 目录下会生成以下文件：

| 文件名 | 平台 | 架构 |
|--------|------|------|
| `bex_windows_amd64.exe` | Windows | 64位 (AMD64) |
| `bex_windows_arm64.exe` | Windows | ARM64 |
| `bex_linux_amd64` | Linux | 64位 (AMD64) |
| `bex_linux_arm64` | Linux | ARM64 (如树莓派) |
| `bex_darwin_amd64` | macOS | Intel 芯片 |
| `bex_darwin_arm64` | macOS | Apple Silicon (M1/M2/M3) |

## 手动编译（可选）

如果你只需要特定平台的版本，也可以手动编译：

```bash
# Windows 64位
GOOS=windows GOARCH=amd64 go build -o bex.exe bex.go

# Linux 64位
GOOS=linux GOARCH=amd64 go build -o bex bex.go

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o bex bex.go

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o bex bex.go
```

## 验证编译结果

编译完成后，可以运行以下命令验证：

```bash
# 查看版本信息
./bex about

# 查看帮助
./bex help
```

## 注意事项

1. **DLL注入模块**：仅支持 Windows 系统，在 Linux/macOS 编译时会自动跳过
2. **CGO 禁用**：脚本中设置了 `CGO_ENABLED=0`，确保编译出的二进制文件不依赖动态链接库
3. **文件大小优化**：使用 `-ldflags="-s -w"` 参数减小可执行文件体积
4. **版本信息注入**：编译时会自动注入当前版本号

## 使用说明

### 通用格式
```
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
| nbt | n | NBT 文件查看/编辑 |
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

**示例：**
```bash
# 查询 Java 版服务器
bex query mc.hypixel.net

# 指定端口查询基岩版服务器
bex query play.cubecraft.net:19132

# 查询本地服务器
bex q 127.0.0.1:11451
```

### 2. Ping 测试模块 (ping/p)
对目标主机执行网络连通性测试，支持 ICMP 协议。

**参数：**
- `-f <次数>` Ping 次数（默认 4）
- `-v <秒>` Ping 间隔（默认 1.0）
- `-r` 持续 Ping 模式

**示例：**
```bash
# 基本 Ping 测试
bex ping 8.8.8.8

# 指定 Ping 10 次
bex ping mc.hypixel.net -f 10

# 持续 Ping 模式
bex ping 127.0.0.1 -r
```

### 3. RCON 远程控制模块 (rcon/r)
通过 RCON 协议远程执行服务器指令，支持交互式 Shell。

**格式：**
```
bex rcon server@<地址>[:端口]
```

**示例：**
```bash
# 连接本地服务器（默认端口 25575）
bex rcon server@127.0.0.1

# 指定端口连接
bex rcon server@127.0.0.1:25575
```

连接后输入密码进入交互模式，支持的命令：
- 输入任意服务器指令执行
- `exit` 或 `quit` 退出会话
- Ctrl+C 强制退出

### 4. 日志分析模块 (log/l)
使用 AI 分析服务器日志，提取错误信息并给出解决方案。

**示例：**
```bash
# 分析日志文件
bex log server.log

# 分析最新日志
bex log ./logs/latest.log
```

### 5. NBT 文件处理模块 (nbt/n)
查看 Minecraft NBT 格式数据文件的内容。

**参数：**
- `-e` 编辑模式（开发中）

**示例：**
```bash
# 查看 level.dat 文件
bex nbt level.dat

# 查看玩家存档
bex nbt playerdata/xxxxxxxx.dat
```

### 6. 启动脚本生成模块 (script/s)
通过 AI 根据需求生成优化的服务器启动脚本，自动适配操作系统类型。

**参数：**
- `-o <路径>` 脚本输出目录（默认当前目录）

**示例：**
```bash
# 生成 Paper 1.20.1 服务器启动脚本
bex script "1.20.1 Paper，4G 内存"

# 生成原版服务器脚本并指定输出目录
bex script "原版 1.21" -o ./server
```

### 7. 热力图生成模块 (heatmap/h)
分析 playerdata 目录，生成玩家游玩时长热力图。

**参数：**
- `-o <路径>` 同时将结果保存为文本文件

**颜色说明：**
- 蓝色：< 1 天（萌新）
- 绿色：1~7 天（轻度）
- 黄色：7~30 天（中度）
- 红色：> 30 天（重度）

**示例：**
```bash
# 显示热力图
bex heatmap ./world/playerdata

# 显示并保存结果
bex heatmap ./playerdata -o ./output
```

### 8. 世界分析模块 (world/w)
扫描指定目录下的所有 Minecraft 世界文件，统计文件数量、大小、维度等信息，检测可能损坏的世界。

**示例：**
```bash
# 分析服务器目录下的所有世界
bex world ./server

# 分析指定世界目录
bex world ./worlds
```

### 9. DLL 注入模块 (injectdll/i)
Windows 平台专用，将 DLL 注入到 Minecraft 进程（需要管理员权限）。

**参数：**
- `-d <路径>` DLL 文件路径
- `-p <进程>` 注入目标进程名（默认 Minecraft.Windows.exe）
- `-t <时间>` 定时注入，如 -t 1m30s
- `-i` 使用上次保存的 DLL 路径直接注入
- `-c` 重置配置文件

**示例：**
```bash
# 注入 DLL
bex injectdll -d ./mod.dll

# 指定进程名注入
bex injectdll -d ./mod.dll -p Minecraft.Windows.exe

# 使用上次的配置注入
bex injectdll -i

# 重置配置文件
bex injectdll -c
```

### 10. 服务器图标生成模块 (icon/ic)
将图片转换为 64×64 的 server-icon.png。

**参数：**
- `-o <路径>` 输出目录（默认当前目录）
- `-n <名称>` 输出文件名（默认 server-icon.png）

**支持的输入格式：** PNG、JPG、BMP、GIF

**示例：**
```bash
# 生成服务器图标
bex icon ./logo.png

# 指定输出目录和文件名
bex icon ./logo.jpg -o ./server -n icon.png
```

### 11. 世界备份模块 (backup/b)
支持冷/热备份，定时定量备份世界、模组、插件等文件夹。

**参数：**
- `-b <目录>` 基础工作目录（必填）
- `-t <目录>` 要备份的子目录名（必填）
- `-v <时间>` 备份间隔，如 30m、1h30m
- `-x <次数>` 最大备份次数
- `-l` 循环执行模式

**示例：**
```bash
# 单次备份
bex backup -b ./server -t worlds

# 定时循环备份（每小时备份一次，最多10次）
bex backup -b ./server -t worlds -v 1h -l -x 10

# 备份多个目录
bex backup -b ./server -t worlds,mods,plugins
```

### 12. 关于模块 (about/!)
显示程序信息、版本、开发者、开源协议等。

**示例：**
```bash
bex about
bex !
```

## 配置文件

BeaconEX 会在程序所在目录自动生成 `config.json` 配置文件：

```json
{
  "last_check_update": "26-03-01 12:00",
  "disable_update": false
}
```

- `last_check_update`：上次检查更新的时间
- `disable_update`：是否禁用自动更新检查

## 注意事项

1. DLL 注入功能仅支持 Windows 系统，且需要以管理员身份运行
2. RCON 功能需要服务器开启 enable-rcon=true 并设置 rcon.password
3. 日志分析功能需要联网使用 AI 服务
4. 启动脚本生成功能需要联网使用 AI 服务
5. 建议在 PowerShell、CMD、Terminal 中运行，避免双击执行
6. 如遇到任何问题，请提交 [issues](https://github.com/GongSunFangYun/BeaconEX/issues/new)


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
