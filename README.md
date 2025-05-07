# BeaconEX - Minecraft 伺服器命令行工具箱

[![GitHub release](https://img.shields.io/github/v/release/FanYaRou/BeaconEX?style=flat-square)]()
[![Downloads](https://img.shields.io/github/downloads/FanYaRou/BeaconEX/total?style=flat-square)]()
[![Stars](https://img.shields.io/github/stars/FanYaRou/BeaconEX?style=flat-square)]()
[![Forks](https://img.shields.io/github/forks/FanYaRou/BeaconEX?style=flat-square)]()
[![Issues](https://img.shields.io/github/issues/FanYaRou/BeaconEX?style=flat-square)]()
![简体中文支持](https://img.shields.io/badge/简体中文-支持-ff8c00?style=flat-square&labelColor=ff8c00&color=ffd700)

## 它能做什麽？

>BeaconEX 是一個專為 Minecraft 伺服器管理員設計的多功能命令行工具，提供以下核心功能：
>- 伺服器狀態查詢：支援 Java 版和基岩版伺服器的基本信息查詢
>- 網路診斷工具：TCPing/UDPing 測試伺服器響應能力
>- 遠程控制：通過 RCON 協議遠程執行指令
>- 數據分析：玩家 NBT 解析、世界檔案檢查、日誌分析
>- 實用工具：自動生成啟動腳本、玩家活動熱力圖生成

## 系統需求

- 作業系統：Windows 10 或更新版本（不太支援 Windows 7）
- 運行環境：已配置 Java 運行環境
- 權限要求：建議以管理員身份運行

## 使用者需求
- 懂得```Windows Powershell``` / ```Windows Command Prompt```用法
- 了解終端程序中的主命令，參數和語法分別是什麽

## 安裝說明

1. 下載 BeaconEX.msi 安裝包
2. 以管理員身份運行安裝程序
3. 按照指引完成安裝（程式會自動添加至系統環境變量）
4. 如環境變量未正確配置，請手動將安裝目錄添加至 Path 變量

## 命令使用指南
### 查詢功能
```bash
# 查詢 Java 版伺服器（默認端口可省略 -p 參數）
bex -java -ip mc.example.com -p 25565
```
```bash
# 查詢基岩版伺服器（默認端口可省略 -p 參數）
bex -bedrock -ip mc.example.com -p 19132
```
### 遠程控制功能
```bash
# RCON 執行單個命令（默認端口可省略 -p 參數）
bex -rcon -ip mc.example.com -p 25565 -rp 25575 -pw password -cmd "say Hello"
```
```bash
# RCON 交互模式（默認端口可省略 -p 參數）
bex -rcon -ip mc.example.com -p 25565 -rp 25575 -pw password -cg
```
### 數據包測試功能
```bash
### 網路測試
# TCPing 測試（Java 版，默認端口可省略 -p 參數）
bex -java -ip mc.example.com -p 25565 -ping Ping次數（選填，默認4次）
```
```bash
# UDPing 測試（基岩版，默認端口可省略 -p 參數）
bex -bedrock -ip mc.example.com -p 19132 -ping Ping次數（選填，默認4次）
```
### 數據分析功能
```bash
# 日誌分析（指定文件路徑）
bex -la -lp "C:/Server/logs/latest.log"
```
```bash
# 玩家 NBT 分析（指定文件路徑）
bex -nbt -np "C:/Server/world/playerdata/XXXXXXXX.dat"
```
```bash
# level.dat 完整性檢查（指定文件夾路徑）
bex -wc -np "C:/Server/"
```
### 生成功能
```bash
# 生成啟動腳本（指定文件路徑）
bex -genbat -rq "paper1.20.1核心，最大分配4G，最小分配2G。"
```
```bash
# 生成玩家熱力圖（指定文件夾路徑）
bex -hp -np "C:/Server/world/playerdata/"
```
### 雜項
```bash
# 檢查版本
bex -v
```
```bash
# 程式更新
bex -update
```
```bash
# 關於我們
bex -about
```
## 注意事項

1. 使用 RCON 功能時請確保在```server.properties```正確配置了以下内容：
```bash
enable-rcon=true
rcon.password=你的RCON連接密碼
rcon.port=你的RCON端口
```
4. 如遇到任何問題，請提交 [issues](https://github.com/FanYaRou/BeaconEX/issues/new) 至我們的代碼倉庫

## 未來計劃

- 增加對 ```Linux``` 和 ```MacOS``` 平台的支援
- 添加更多數據可視化功能

## 授權協議

BeaconEX 使用臨時閉源分發政策。在未經授權的情況下，禁止對該程序進行任何反向操作！
