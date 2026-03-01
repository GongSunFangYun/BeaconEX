package modules

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/term"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"bex/utils"
)

// 错误代码
const (
	ErrRuntimeRconConnection = iota + 1
	ErrRuntimeRconAuth
	ErrRuntimeRconCommand
)

var errorMessages = map[int]string{
	ErrRuntimeRconConnection: "RCON 连接失败",
	ErrRuntimeRconAuth:       "RCON 认证失败",
	ErrRuntimeRconCommand:    "RCON 命令执行失败",
}

type RuntimeError struct {
	Code    int
	Message string
	Details string
}

func (e *RuntimeError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

type RCONClient struct {
	mu        sync.Mutex
	conn      net.Conn
	host      string
	port      int
	password  string
	timeout   time.Duration
	requestID int32
}

func NewRCONClient(host string, port int, password string, timeout time.Duration) *RCONClient {
	return &RCONClient{
		host:      host,
		port:      port,
		password:  password,
		timeout:   timeout,
		requestID: 0,
	}
}

func (c *RCONClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

func (c *RCONClient) connectLocked() error {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return &RuntimeError{
			Code:    ErrRuntimeRconConnection,
			Message: errorMessages[ErrRuntimeRconConnection],
			Details: err.Error(),
		}
	}
	c.conn = conn

	_, err = c.sendPacketLocked(3, c.password)
	if err != nil {
		_ = c.conn.Close()
		c.conn = nil
		return &RuntimeError{
			Code:    ErrRuntimeRconAuth,
			Message: errorMessages[ErrRuntimeRconAuth],
			Details: err.Error(),
		}
	}

	return nil
}

func (c *RCONClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *RCONClient) sendPacketLocked(pktType int32, payload string) (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("连接未建立")
	}

	if err := c.conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", err
	}

	c.requestID++
	reqID := c.requestID
	payloadBytes := []byte(payload)

	body := make([]byte, 0, 8+len(payloadBytes)+2)
	body = binary.LittleEndian.AppendUint32(body, uint32(reqID))
	body = binary.LittleEndian.AppendUint32(body, uint32(pktType))
	body = append(body, payloadBytes...)
	body = append(body, 0, 0)

	lengthBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBuf, uint32(len(body)))

	if _, err := c.conn.Write(lengthBuf); err != nil {
		return "", fmt.Errorf("发送失败: %v", err)
	}
	if _, err := c.conn.Write(body); err != nil {
		return "", fmt.Errorf("发送失败: %v", err)
	}

	if pktType == 3 {
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
			return "", fmt.Errorf("读取认证响应失败: %v", err)
		}
		pktLen := binary.LittleEndian.Uint32(lenBuf)
		if pktLen < 10 {
			return "", fmt.Errorf("认证响应包太短: %d", pktLen)
		}
		pktBuf := make([]byte, pktLen)
		if _, err := io.ReadFull(c.conn, pktBuf); err != nil {
			return "", fmt.Errorf("读取认证响应包失败: %v", err)
		}
		respID := int32(binary.LittleEndian.Uint32(pktBuf[0:4]))
		if respID == -1 {
			return "", fmt.Errorf("密码错误")
		}
		return "", nil
	}

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
		if err == io.EOF {
			return "", io.EOF
		}
		return "", fmt.Errorf("读取响应长度失败: %v", err)
	}

	pktLen := binary.LittleEndian.Uint32(lenBuf)
	if pktLen < 10 {
		return "", fmt.Errorf("响应包太短: %d", pktLen)
	}

	pktBuf := make([]byte, pktLen)
	if _, err := io.ReadFull(c.conn, pktBuf); err != nil {
		return "", fmt.Errorf("读取响应包失败: %v", err)
	}

	respID := int32(binary.LittleEndian.Uint32(pktBuf[0:4]))
	respPayload := pktBuf[8 : pktLen-2]
	padding := pktBuf[pktLen-2:]

	if padding[0] != 0 || padding[1] != 0 {
		return "", fmt.Errorf("无效的包填充")
	}
	if respID == -1 {
		return "", fmt.Errorf("登录失败")
	}
	if respID != reqID {
		return "", fmt.Errorf("请求ID不匹配: 发送 %d, 收到 %d", reqID, respID)
	}

	return string(respPayload), nil
}

func (c *RCONClient) Command(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		if err := c.connectLocked(); err != nil {
			return "", fmt.Errorf("重连失败: %v", err)
		}
	}

	response, err := c.sendPacketLocked(2, cmd)
	if err != nil {
		c.conn = nil
		if err := c.connectLocked(); err != nil {
			return "", fmt.Errorf("重连失败: %v", err)
		}
		response, err = c.sendPacketLocked(2, cmd)
		if err != nil {
			c.conn = nil
			return "", err
		}
	}

	return response, nil
}

func RconExecutorEntry(loginStr string) {
	username, host, port := parseLoginString(loginStr)

	if username != "server" {
		utils.LogError("RCON 只允许 server 用户登录")
		return
	}

	fmt.Printf("输入 %s 的 RCON 密码: ", username)
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		utils.LogError("读取密码失败: %s", err)
		return
	}
	password := string(passwordBytes)
	if password == "" {
		utils.LogError("密码不能为空")
		return
	}

	client := NewRCONClient(host, port, password, 5*time.Second)
	if err := client.Connect(); err != nil {
		var rerr *RuntimeError
		if errors.As(err, &rerr) {
			switch rerr.Code {
			case ErrRuntimeRconAuth:
				utils.LogError("连接失败: 密码错误")
			case ErrRuntimeRconConnection:
				utils.LogError("连接失败: 服务器未启动/未配置 enable-rcon=true/rcon-port 错误")
			default:
				utils.LogError("连接失败: %s", rerr.Message)
			}
		}
		return
	}
	defer func() { _ = client.Close() }()

	fmt.Printf("欢迎使用 BeaconEX RCON Shell！\n")
	fmt.Printf("\n")
	fmt.Printf("[安全警告]\n")
	fmt.Printf("• 已知 RCON 密码 + 启用 RCON 功能 = 服务器最高权限\n")
	fmt.Printf("• 在开启 RCON 的情况下泄露密码或者服务器 IP + RCON 端口都可能导致服务器被完全控制\n")
	fmt.Printf("• 您可以执行任何控制台权限级别的命令，包括但不限于 stop/op/ban/ban-ip 等\n")
	fmt.Printf("• 请务必在确认命令效果后再执行，否则可能会造成服务器维度损坏及玩家数据丢失\n")
	fmt.Printf("\n")
	fmt.Printf("[操作提示]\n")
	fmt.Printf("• 输入命令后按回车(Enter)键即可执行命令\n")
	fmt.Printf("• 输入 'exit' 或 'quit' 终止会话\n")
	fmt.Printf("• 按下 'Ctrl+C' 强制关闭会话\n")
	fmt.Printf("\n")

	rconInteractive(client, username)
}

func rconInteractive(client *RCONClient, username string) {
	prompt := fmt.Sprintf("%s%s@rcon-session:~#%s ",
		utils.ColorGreen, username, utils.ColorClear)

	scanner := bufio.NewScanner(os.Stdin)
	inputChan := make(chan string)

	go func() {
		for scanner.Scan() {
			inputChan <- scanner.Text()
		}
		close(inputChan)
	}()

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	lastActivity := time.Now()

	fmt.Print(prompt)

	for {
		select {
		case input, ok := <-inputChan:
			if !ok {
				return
			}

			input = strings.TrimSpace(input)
			lastActivity = time.Now()

			if input == "" {
				fmt.Print(prompt)
				continue
			}

			if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
				fmt.Println("正在关闭本 RCON 会话...")
				return
			}

			response, err := client.Command(input)
			if err != nil {
				if strings.Contains(err.Error(), "重连失败") {
					fmt.Println("\nRCON 连接已断开！")
					return
				}
				fmt.Printf("命令执行失败: %v\n", err)
			} else {
				if strings.HasPrefix(strings.ToLower(input), "say ") {
					fmt.Printf("[Rcon] %s\n", input[4:])
				} else if response != "" {
					fmt.Printf("%s\n", utils.ParseMinecraftFormat(response))
				}
			}

			fmt.Print(prompt)

		case <-keepAlive.C:
			if time.Since(lastActivity) > 20*time.Second {
				_, _ = client.Command("list")
			}
		}
	}
}

func parseLoginString(loginStr string) (username string, host string, port int) {
	port = 25575

	if loginStr == "" {
		return "server", "localhost", port
	}

	if strings.Contains(loginStr, "@") {
		parts := strings.SplitN(loginStr, "@", 2)
		username = parts[0]
		hostPort := parts[1]

		if strings.Contains(hostPort, ":") {
			hp := strings.SplitN(hostPort, ":", 2)
			host = hp[0]
			if host == "" {
				host = "localhost"
			}
			if p, err := strconv.Atoi(hp[1]); err == nil && p > 0 && p < 65536 {
				port = p
			}
		} else {
			host = hostPort
			if host == "" {
				host = "localhost"
			}
		}
	} else {
		username = "server"
		if strings.Contains(loginStr, ":") {
			hp := strings.SplitN(loginStr, ":", 2)
			host = hp[0]
			if host == "" {
				host = "localhost"
			}
			if p, err := strconv.Atoi(hp[1]); err == nil && p > 0 && p < 65536 {
				port = p
			}
		} else {
			host = loginStr
			if host == "" {
				host = "localhost"
			}
		}
	}

	return username, host, port
}
