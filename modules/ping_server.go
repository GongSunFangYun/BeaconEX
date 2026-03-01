package modules

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"bex/utils"
)

func PingServer(
	target string,
	count int,
	interval float64,
	repeat bool,
) {
	server := &PingServerInstance{
		Target:   target,
		Count:    count,
		Interval: interval,
		Repeat:   repeat,
	}

	server.Execute()
}

type PingServerInstance struct {
	Target   string
	Count    int
	Interval float64
	Repeat   bool
}

func (p *PingServerInstance) Execute() {
	// 验证参数
	if !p.validateArgs() {
		return
	}

	p.Target = extractHost(p.Target)

	if p.Repeat {
		p.loopPing()
	} else {
		p.singlePing()
	}
}

func (p *PingServerInstance) validateArgs() bool {
	if p.Target == "" {
		utils.LogError("Ping 目标不能为空")
		return false
	}

	if p.Count < 1 || p.Count > 1000 {
		utils.LogError("Ping 次数必须在 1-1000 之间")
		return false
	}

	if p.Interval < 0.1 || p.Interval > 60 {
		utils.LogError("间隔时间必须在 0.1-60 秒之间")
		return false
	}

	return true
}

func (p *PingServerInstance) singlePing() {
	host := extractHost(p.Target)
	ip, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		utils.LogError("解析目标地址失败: %s", err)
		return
	}

	utils.LogInfo("正在 Ping %s [%s] 具有 32 字节的数据:", host, ip.String())

	var successCount int
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration
	var minInitialized bool
	timeout := time.Duration(p.Interval * 2 * float64(time.Second))

	for i := 1; i <= p.Count; i++ {
		conn, err := net.DialIP("ip4:icmp", nil, ip)
		if err != nil {
			utils.LogWarn("[%d/%d] 正在 Ping %s: %s×%s 创建连接失败",
				i, p.Count, host, utils.ColorRed, utils.ColorClear)
			if i < p.Count {
				time.Sleep(time.Duration(p.Interval * float64(time.Second)))
			}
			continue
		}

		// 构建 ICMP Echo 请求
		msg := make([]byte, 40)
		msg[0] = 8 // Echo Request
		msg[1] = 0
		msg[4] = byte(i >> 8)
		msg[5] = byte(i)
		checksum := computeChecksum(msg)
		msg[2] = byte(checksum >> 8)
		msg[3] = byte(checksum)

		start := time.Now()
		_, err = conn.Write(msg)
		if err != nil {
			_ = conn.Close()
			utils.LogWarn("[%d/%d] 正在 Ping %s: %s×%s 发送失败",
				i, p.Count, host, utils.ColorRed, utils.ColorClear)
			if i < p.Count {
				time.Sleep(time.Duration(p.Interval * float64(time.Second)))
			}
			continue
		}

		reply := make([]byte, 1024)
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		_, _, readErr := conn.ReadFrom(reply)
		latency := time.Since(start)
		_ = conn.Close()

		if readErr != nil {
			utils.LogWarn("[%d/%d] 正在 Ping %s: %s×%s 响应超时",
				i, p.Count, host, utils.ColorRed, utils.ColorClear)
		} else {
			successCount++
			totalLatency += latency
			if !minInitialized || latency < minLatency {
				minLatency = latency
				minInitialized = true
			}
			if latency > maxLatency {
				maxLatency = latency
			}
			utils.LogInfo("[%d/%d] 正在 Ping %s: 延迟 %s%.2fms%s",
				i, p.Count, host,
				utils.ColorPurple, float64(latency)/float64(time.Millisecond), utils.ColorClear)
		}

		if i < p.Count {
			time.Sleep(time.Duration(p.Interval * float64(time.Second)))
		}
	}

	utils.LogInfo("Ping 结果:")
	utils.LogInfo("已对 %s [%s] 进行 %s%d%s 次 Ping",
		host, ip.String(), utils.ColorYellow, p.Count, utils.ColorClear)
	utils.LogInfo("成功接收: %s%d%s 次", utils.ColorGreen, successCount, utils.ColorClear)
	lossRate := float64(p.Count-successCount) / float64(p.Count) * 100
	utils.LogInfo("丢包率: %s%.2f%%%s", utils.ColorRed, lossRate, utils.ColorClear)
	if successCount > 0 {
		avgLatency := totalLatency / time.Duration(successCount)
		utils.LogInfo("延迟统计: 平均 %s%.2fms%s | 最小 %s%.2fms%s | 最大 %s%.2fms%s",
			utils.ColorBlue, float64(avgLatency)/float64(time.Millisecond), utils.ColorClear,
			utils.ColorGreen, float64(minLatency)/float64(time.Millisecond), utils.ColorClear,
			utils.ColorYellow, float64(maxLatency)/float64(time.Millisecond), utils.ColorClear)
	}
}

func (p *PingServerInstance) loopPing() {
	utils.LogInfo("开始以 %s%.1f%s 秒间隔持续 Ping %s",
		utils.ColorYellow, p.Interval, utils.ColorClear, p.Target)
	utils.LogInfo("按 Ctrl+C 停止测试")

	host := extractHost(p.Target)
	ip, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		utils.LogError("解析目标地址失败: %s", err)
		return
	}

	utils.LogInfo("正在 Ping %s [%s] 具有 32 字节的数据:",
		host, ip.String())

	var successCount, totalCount int
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration
	var minInitialized bool
	var lastLatency time.Duration
	var consecutiveTimeouts int

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ticker := time.NewTicker(time.Duration(p.Interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-c:
			goto showStats

		case <-ticker.C:
			totalCount++
			sequence := totalCount

			startTime := time.Now()

			conn, err := net.DialIP("ip4:icmp", nil, ip)
			if err != nil {
				utils.LogWarn("[%d] 创建连接失败: %s", sequence, err)
				consecutiveTimeouts++
				continue
			}

			msg := make([]byte, 32+8)
			msg[0] = 8
			msg[1] = 0
			msg[4] = byte(sequence >> 8)
			msg[5] = byte(sequence)
			msg[6] = 0
			msg[7] = 0

			checksum := computeChecksum(msg)
			msg[2] = byte(checksum >> 8)
			msg[3] = byte(checksum)

			_, err = conn.Write(msg)
			if err != nil {
				err := conn.Close()
				if err != nil {
					return
				}
				utils.LogWarn("[%d] 发送失败: %s", sequence, err)
				consecutiveTimeouts++
				continue
			}

			reply := make([]byte, 1024)
			err = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				return
			}
			_, _, err = conn.ReadFrom(reply)
			err = conn.Close()
			if err != nil {
				return
			}

			if err != nil {
				consecutiveTimeouts++
				lossRate := float64(totalCount-successCount) / float64(totalCount) * 100
				utils.LogWarn("[%d] %s×%s 响应超时 | 连续超时: %s%d次%s | 丢包率 %s%.1f%%%s",
					sequence,
					utils.ColorRed, utils.ColorClear,
					utils.ColorYellow, consecutiveTimeouts, utils.ColorClear,
					utils.ColorRed, lossRate, utils.ColorClear)
				continue
			}

			latency := time.Since(startTime)
			successCount++
			totalLatency += latency

			if !minInitialized || latency < minLatency {
				minLatency = latency
				minInitialized = true
			}
			if latency > maxLatency {
				maxLatency = latency
			}

			jitter := latency - lastLatency
			lastLatency = latency
			consecutiveTimeouts = 0

			jitterStr := formatJitter(jitter)

			avgLatency := totalLatency / time.Duration(successCount)
			lossRate := float64(totalCount-successCount) / float64(totalCount) * 100

			utils.LogInfo("[%d] 延迟: %s%.2fms%s | 抖动: %s | 平均: %s%.2fms%s | 丢包率 %s%.1f%%%s",
				sequence,
				utils.ColorGreen, float64(latency)/float64(time.Millisecond), utils.ColorClear,
				jitterStr,
				utils.ColorBlue, float64(avgLatency)/float64(time.Millisecond), utils.ColorClear,
				utils.ColorRed, lossRate, utils.ColorClear)
		}
	}

showStats:
	if totalCount > 0 {
		lossRate := float64(totalCount-successCount) / float64(totalCount) * 100
		avgLatency := totalLatency / time.Duration(successCount)

		utils.LogInfo("已结束持续 Ping 测试，结果统计如下：")
		utils.LogInfo("总测试次数: %s%d%s",
			utils.ColorYellow, totalCount, utils.ColorClear)
		utils.LogInfo("成功响应: %s%d%s 次",
			utils.ColorGreen, successCount, utils.ColorClear)
		utils.LogInfo("丢包率: %s%.2f%%%s",
			utils.ColorRed, lossRate, utils.ColorClear)

		if successCount > 0 {
			if !minInitialized {
				minLatency = 0
			}
			utils.LogInfo("最小延迟: %s%.2fms%s",
				utils.ColorGreen, float64(minLatency)/float64(time.Millisecond), utils.ColorClear)
			utils.LogInfo("最大延迟: %s%.2fms%s",
				utils.ColorYellow, float64(maxLatency)/float64(time.Millisecond), utils.ColorClear)
			utils.LogInfo("平均延迟: %s%.2fms%s",
				utils.ColorBlue, float64(avgLatency)/float64(time.Millisecond), utils.ColorClear)
		}
	}
}

func computeChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}
	for (sum >> 16) > 0 {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}
	return ^uint16(sum)
}

func formatJitter(jitter time.Duration) string {
	jitterMs := float64(jitter) / float64(time.Millisecond)
	if jitter > 0 {
		return fmt.Sprintf("%s+%.2fms%s",
			utils.ColorRed, jitterMs, utils.ColorClear)
	} else if jitter < 0 {
		return fmt.Sprintf("%s%.2fms%s",
			utils.ColorGreen, jitterMs, utils.ColorClear)
	}
	return fmt.Sprintf("%s±0ms%s",
		utils.ColorPurple, utils.ColorClear)
}

func extractHost(target string) string {
	if strings.Contains(target, ":") {
		return strings.Split(target, ":")[0]
	}
	return target
}
