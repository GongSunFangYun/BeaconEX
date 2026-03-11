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

func pingOnce(host string, seq int, timeout time.Duration) (latency time.Duration, method string, err error) {
	ip, resolveErr := net.ResolveIPAddr("ip", host)
	if resolveErr != nil {
		return 0, "", resolveErr
	}

	conn, icmpErr := net.DialIP("ip4:icmp", nil, ip)
	if icmpErr == nil {
		msg := make([]byte, 40)
		msg[0] = 8
		msg[1] = 0
		msg[4] = byte(seq >> 8)
		msg[5] = byte(seq)
		checksum := computeChecksum(msg)
		msg[2] = byte(checksum >> 8)
		msg[3] = byte(checksum)

		start := time.Now()
		_, writeErr := conn.Write(msg)
		if writeErr != nil {
			_ = conn.Close()
			return 0, "icmp", writeErr
		}

		reply := make([]byte, 1024)
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		_, _, readErr := conn.ReadFrom(reply)
		elapsed := time.Since(start)
		_ = conn.Close()

		if readErr != nil {
			return 0, "icmp", readErr
		}
		return elapsed, "icmp", nil
	}

	start := time.Now()
	tcpConn, tcpErr := net.DialTimeout("tcp", fmt.Sprintf("%s:443", host), timeout)
	elapsed := time.Since(start)
	if tcpErr == nil {
		_ = tcpConn.Close()
		return elapsed, "tcp", nil
	}

	start = time.Now()
	tcpConn, tcpErr = net.DialTimeout("tcp", fmt.Sprintf("%s:80", host), timeout)
	elapsed = time.Since(start)
	if tcpErr == nil {
		_ = tcpConn.Close()
		return elapsed, "tcp", nil
	}

	return 0, "tcp", fmt.Errorf("ICMP 需要 root 权限，TCP 降级也失败 (443/80 均不可达)")
}

func (p *PingServerInstance) singlePing() {
	host := extractHost(p.Target)
	ip, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		utils.LogError("解析目标地址失败: %s", err)
		return
	}

	utils.LogInfo("正在 Ping %s [%s] 具有 32 字节的数据:", host, ip.String())

	timeout := time.Duration(p.Interval * 2 * float64(time.Second))

	var successCount int
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration
	var minInitialized bool
	var shownMethodNote bool

	for i := 1; i <= p.Count; i++ {
		latency, method, err := pingOnce(host, i, timeout)

		if !shownMethodNote && method == "tcp" {
			utils.LogWarn("ICMP 不可用（需要 root/管理员权限），已自动切换到 TCP 延迟测量模式")
			shownMethodNote = true
		}

		if err != nil {
			utils.LogWarn("[%d/%d] 正在 Ping %s: %s无响应%s",
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

	utils.LogInfo("正在 Ping %s [%s] 具有 32 字节的数据:", host, ip.String())

	var successCount, totalCount int
	var totalLatency time.Duration
	var minLatency, maxLatency time.Duration
	var minInitialized bool
	var lastLatency time.Duration
	var consecutiveTimeouts int
	var shownMethodNote bool

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

			latency, method, err := pingOnce(host, totalCount, 2*time.Second)

			if !shownMethodNote && method == "tcp" {
				utils.LogWarn("ICMP 不可用（需要 root/管理员权限），已自动切换到 TCP 延迟测量模式")
				shownMethodNote = true
			}

			if err != nil {
				consecutiveTimeouts++
				lossRate := float64(totalCount-successCount) / float64(totalCount) * 100
				utils.LogWarn("[%d] %s无响应%s | 连续超时: %s%d次%s | 丢包率 %s%.1f%%%s",
					totalCount,
					utils.ColorRed, utils.ColorClear,
					utils.ColorYellow, consecutiveTimeouts, utils.ColorClear,
					utils.ColorRed, lossRate, utils.ColorClear)
				continue
			}

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
				totalCount,
				utils.ColorGreen, float64(latency)/float64(time.Millisecond), utils.ColorClear,
				jitterStr,
				utils.ColorBlue, float64(avgLatency)/float64(time.Millisecond), utils.ColorClear,
				utils.ColorRed, lossRate, utils.ColorClear)
		}
	}

showStats:
	if totalCount > 0 {
		lossRate := float64(totalCount-successCount) / float64(totalCount) * 100
		utils.LogInfo("已结束持续 Ping 测试，结果统计如下：")
		utils.LogInfo("总测试次数: %s%d%s", utils.ColorYellow, totalCount, utils.ColorClear)
		utils.LogInfo("成功响应: %s%d%s 次", utils.ColorGreen, successCount, utils.ColorClear)
		utils.LogInfo("丢包率: %s%.2f%%%s", utils.ColorRed, lossRate, utils.ColorClear)

		if successCount > 0 {
			if !minInitialized {
				minLatency = 0
			}
			avgLatency := totalLatency / time.Duration(successCount)
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
		return fmt.Sprintf("%s+%.2fms%s", utils.ColorRed, jitterMs, utils.ColorClear)
	} else if jitter < 0 {
		return fmt.Sprintf("%s%.2fms%s", utils.ColorGreen, jitterMs, utils.ColorClear)
	}
	return fmt.Sprintf("%s+/-0ms%s", utils.ColorPurple, utils.ColorClear)
}

func extractHost(target string) string {
	if strings.Contains(target, ":") {
		return strings.Split(target, ":")[0]
	}
	return target
}
