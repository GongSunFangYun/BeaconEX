package modules

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"bex/utils"
	"github.com/miekg/dns"
)

type JavaServerInfo struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample"`
	} `json:"players"`
	Description interface{} `json:"description"`
	Favicon     string      `json:"favicon,omitempty"`
	ForgeData   interface{} `json:"forgeData,omitempty"`
	ModInfo     interface{} `json:"modinfo,omitempty"`
}

type BedrockServerInfo struct {
	ServerName string
	Protocol   int
	Version    string
	Online     int
	Max        int
	MapName    string
	GameMode   string
	Brand      string
}

func QueryServer(isJava bool, isBedrock bool, target string) {
	if isJava {
		utils.LogDebug("指定 Java 版查询，但将使用自动识别模式")
	} else if isBedrock {
		utils.LogDebug("指定基岩版查询，但将使用自动识别模式")
	}
	QueryServerAuto(target)
}

func QueryServerAuto(target string) {
	type result struct {
		info    interface{}
		latency time.Duration
		err     error
		version string
	}

	javaChan := make(chan result)
	bedrockChan := make(chan result)

	utils.LogInfo("正在查询目标服务器...")

	go func() {
		addr, err := LookupSRV(target, 25565)
		if err != nil {
			javaChan <- result{err: err, version: "java"}
			return
		}

		pinger, err := NewJavaPinger(addr, 5*time.Second)
		if err != nil {
			javaChan <- result{err: err, version: "java"}
			return
		}
		defer func(pinger *JavaPinger) {
			err := pinger.Close()
			if err != nil {

			}
		}(pinger)

		if err := pinger.Handshake(); err != nil {
			javaChan <- result{err: err, version: "java"}
			return
		}

		info, latency, err := pinger.ReadStatus()
		javaChan <- result{info: info, latency: latency, err: err, version: "java"}
	}()

	go func() {
		addr, err := ParseAddress(target, 19132)
		if err != nil {
			bedrockChan <- result{err: err, version: "bedrock"}
			return
		}

		pinger, err := NewBedrockPinger(addr, 5*time.Second)
		if err != nil {
			bedrockChan <- result{err: err, version: "bedrock"}
			return
		}
		defer func(pinger *BedrockPinger) {
			err := pinger.Close()
			if err != nil {

			}
		}(pinger)

		info, latency, err := pinger.ReadStatus()
		bedrockChan <- result{info: info, latency: latency, err: err, version: "bedrock"}
	}()

	var e []string
	timeout := time.After(6 * time.Second)

	var javaResult *result
	var bedrockResult *result

	for i := 0; i < 2; i++ {
		select {
		case res := <-javaChan:
			r := res
			if res.err == nil {
				javaResult = &r
			} else {
				e = append(e, fmt.Sprintf("Java版: %v", res.err))
			}

		case res := <-bedrockChan:
			r := res
			if res.err == nil {
				bedrockResult = &r
			} else {
				e = append(e, fmt.Sprintf("基岩版: %v", res.err))
			}

		case <-timeout:
			if javaResult != nil || bedrockResult != nil {
				break
			}
			utils.LogError("查询超时 (6秒)")
			return
		}
	}

	if javaResult != nil && bedrockResult != nil {
		utils.LogWarn("%s### 似乎双版本查询均出现了响应(这可能是目标主机开设了跨版本代理服务器) ###%s",
			utils.ColorBrightYellow, utils.ColorClear)
		utils.LogInfo("已返回汇总查询结果...")
		utils.LogInfo("━━━━━━━━━━━━ JE ━━━━━━━━━━━━")
		if javaInfo, ok := javaResult.info.(*JavaServerInfo); ok {
			displayServerInfo(target, javaInfo, javaResult.latency, true)
		}
		utils.LogInfo("━━━━━━━━━━━━ BE ━━━━━━━━━━━━")
		if bedrockInfo, ok := bedrockResult.info.(*BedrockServerInfo); ok {
			displayServerInfo(target, bedrockInfo, bedrockResult.latency, false)
		}
		return
	}

	if javaResult != nil {
		if javaInfo, ok := javaResult.info.(*JavaServerInfo); ok {
			displayServerInfo(target, javaInfo, javaResult.latency, true)
		}
		return
	}
	if bedrockResult != nil {
		if bedrockInfo, ok := bedrockResult.info.(*BedrockServerInfo); ok {
			displayServerInfo(target, bedrockInfo, bedrockResult.latency, false)
		}
		return
	}

	utils.LogError("所有查询方式都失败：")
	for _, err := range e {
		utils.LogError("  %s", err)
	}
}

func displayServerInfo(target string, info interface{}, latency time.Duration, isJava bool) {
	utils.LogInfo("状态: %s在线%s", utils.ColorBrightGreen, utils.ColorClear)
	utils.LogInfo("地址: %s", target)

	if isJava {
		displayJavaDetails(info.(*JavaServerInfo))
	} else {
		displayBedrockDetails(info.(*BedrockServerInfo))
	}

	utils.LogInfo("连接延迟: %s%.2f%s ms",
		utils.ColorBrightYellow, float64(latency)/float64(time.Millisecond), utils.ColorClear)
}

func displayJavaDetails(info *JavaServerInfo) {
	motdStr := utils.ParseMOTDFromJSON(info.Description)
	coloredMotd := utils.ParseMinecraftFormat(motdStr)
	lines := strings.Split(coloredMotd, "\n")

	utils.LogInfo("服务器名称:")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			utils.LogInfo("%s", line)
		}
	}

	utils.LogInfo("版本/协议: %s%s%s/%s%d",
		utils.ColorBrightYellow, info.Version.Name, utils.ColorClear,
		utils.ColorBrightBlue, info.Version.Protocol)
	utils.LogInfo("玩家数量: %s%d%s/%s%d%s",
		utils.ColorBrightGreen, info.Players.Online, utils.ColorClear,
		utils.ColorYellow, info.Players.Max, utils.ColorClear)

	if len(info.Players.Sample) > 0 {
		utils.LogInfo("玩家列表:")
		for i, player := range info.Players.Sample {
			if i < 10 {
				coloredName := utils.ParseMinecraftFormat(player.Name)
				utils.LogInfo("  %s•%s %s",
					utils.ColorPurple, utils.ColorClear, coloredName)
			}
		}
		if len(info.Players.Sample) > 10 {
			utils.LogInfo("  %s... 还有 %d 名玩家%s",
				utils.ColorCyan, len(info.Players.Sample)-10, utils.ColorClear)
		}
	} else {
		utils.LogInfo("玩家列表: 无")
	}
}

func displayBedrockDetails(info *BedrockServerInfo) {
	utils.LogInfo("名称: %s", utils.ParseMinecraftFormat(info.ServerName))
	utils.LogInfo("版本/协议: %s%s%s/%s%d",
		utils.ColorBrightYellow, info.Version, utils.ColorClear,
		utils.ColorBrightBlue, info.Protocol)
	utils.LogInfo("玩家数量: %s%d%s/%s%d%s",
		utils.ColorBrightGreen, info.Online, utils.ColorClear,
		utils.ColorYellow, info.Max, utils.ColorClear)

	if info.MapName != "" {
		utils.LogInfo("核心/代理层: %s%s%s",
			utils.ColorCyan, info.MapName, utils.ColorClear)
	}
	if info.GameMode != "" {
		utils.LogInfo("游戏模式: %s%s%s",
			utils.ColorPurple, info.GameMode, utils.ColorClear)
	}
	//if info.Brand != "" {
	//	utils.LogInfo("品牌: %s%s%s",
	//		utils.ColorBlue, info.Brand, utils.ColorClear)
	//}
}

type QueryServerInstance struct {
	IsJava    bool
	IsBedrock bool
	Target    string
	Timeout   time.Duration
}

func (q *QueryServerInstance) Execute() {
	QueryServerAuto(q.Target)
}

type ResolvedAddress struct {
	Host string
	Port int
}

func ParseAddress(address string, defaultPort int) (*ResolvedAddress, error) {
	host, port := parseHostPort(address)

	if port != 0 {
		if host == "" {
			return nil, fmt.Errorf("无效的地址: %s", address)
		}
		return &ResolvedAddress{
			Host: host,
			Port: port,
		}, nil
	}

	if host == "" {
		host = address
	}

	return &ResolvedAddress{
		Host: host,
		Port: defaultPort,
	}, nil
}

func LookupSRV(address string, defaultPort int) (*ResolvedAddress, error) {
	host, port := parseHostPort(address)

	if port != 0 {
		return &ResolvedAddress{Host: host, Port: port}, nil
	}

	srvHost, srvPort, err := lookupMinecraftSRV(host)
	if err == nil {
		return &ResolvedAddress{Host: srvHost, Port: srvPort}, nil
	}

	return &ResolvedAddress{Host: host, Port: defaultPort}, nil
}

func parseHostPort(address string) (string, int) {
	if strings.Contains(address, ":") {
		parts := strings.Split(address, ":")

		if strings.Count(parts[0], ":") > 0 || len(parts) > 2 {
			host := strings.Join(parts[:len(parts)-1], ":")
			portStr := parts[len(parts)-1]
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return host, 0
			}
			return host, port
		}

		if len(parts) == 2 {
			port, err := strconv.Atoi(parts[1])
			if err != nil {
				return parts[0], 0
			}
			return parts[0], port
		}
	}

	return address, 0
}

func lookupMinecraftSRV(host string) (string, int, error) {
	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	m := new(dns.Msg)
	m.SetQuestion("_minecraft._tcp."+host+".", dns.TypeSRV)

	r, _, err := c.Exchange(m, "8.8.8.8:53")
	if err != nil {
		return "", 0, err
	}

	for _, ans := range r.Answer {
		if srv, ok := ans.(*dns.SRV); ok {
			return srv.Target, int(srv.Port), nil
		}
	}

	return "", 0, fmt.Errorf("未找到 SRV 记录")
}

type JavaPinger struct {
	conn      net.Conn
	address   *ResolvedAddress
	version   int
	pingToken int64
}

func NewJavaPinger(addr *ResolvedAddress, timeout time.Duration) (*JavaPinger, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", addr.Host, addr.Port), timeout)
	if err != nil {
		return nil, err
	}

	return &JavaPinger{
		conn:      conn,
		address:   addr,
		version:   0,
		pingToken: time.Now().UnixNano(),
	}, nil
}

func (p *JavaPinger) Close() error {
	return p.conn.Close()
}

func writeVarint(w io.Writer, value int) error {
	for {
		b := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			b |= 0x80
		}
		if _, err := w.Write([]byte{b}); err != nil {
			return err
		}
		if value == 0 {
			break
		}
	}
	return nil
}

func readVarint(r io.Reader) (int, error) {
	var result int
	var shift uint
	for {
		b := make([]byte, 1)
		if _, err := r.Read(b); err != nil {
			return 0, err
		}
		result |= int(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			break
		}
		shift += 7
		if shift > 35 {
			return 0, fmt.Errorf("varint 太大")
		}
	}
	return result, nil
}

func readUTF(r io.Reader) (string, error) {
	length, err := readVarint(r)
	if err != nil {
		return "", err
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

func (p *JavaPinger) Handshake() error {
	buf := new(bytes.Buffer)

	err := writeVarint(buf, 0x00)
	if err != nil {
		return err
	}

	err = writeVarint(buf, p.version)
	if err != nil {
		return err
	}

	err = writeVarint(buf, len(p.address.Host))
	if err != nil {
		return err
	}
	buf.WriteString(p.address.Host)

	err = binary.Write(buf, binary.BigEndian, uint16(p.address.Port))
	if err != nil {
		return err
	}

	err = writeVarint(buf, 1)
	if err != nil {
		return err
	}

	packet := buf.Bytes()
	lengthBuf := new(bytes.Buffer)
	err = writeVarint(lengthBuf, len(packet))
	if err != nil {
		return err
	}

	if _, err := p.conn.Write(lengthBuf.Bytes()); err != nil {
		return err
	}
	if _, err := p.conn.Write(packet); err != nil {
		return err
	}

	return nil
}

func (p *JavaPinger) ReadStatus() (*JavaServerInfo, time.Duration, error) {
	start := time.Now()

	reqBuf := new(bytes.Buffer)
	err := writeVarint(reqBuf, 0x00)
	if err != nil {
		return nil, 0, err
	}

	lengthBuf := new(bytes.Buffer)
	err = writeVarint(lengthBuf, reqBuf.Len())
	if err != nil {
		return nil, 0, err
	}

	if _, err := p.conn.Write(lengthBuf.Bytes()); err != nil {
		return nil, 0, err
	}
	if _, err := p.conn.Write(reqBuf.Bytes()); err != nil {
		return nil, 0, err
	}

	respLength, err := readVarint(p.conn)
	if err != nil {
		return nil, 0, err
	}

	respData := make([]byte, respLength)
	if _, err := io.ReadFull(p.conn, respData); err != nil {
		return nil, 0, err
	}

	reader := bytes.NewReader(respData)
	packetID, err := readVarint(reader)
	if err != nil {
		return nil, 0, err
	}

	if packetID != 0x00 {
		return nil, 0, fmt.Errorf("收到无效的包 ID: %d", packetID)
	}

	jsonStr, err := readUTF(reader)
	if err != nil {
		return nil, 0, err
	}

	var info JavaServerInfo
	if err := json.Unmarshal([]byte(jsonStr), &info); err != nil {
		return nil, 0, err
	}

	latency := time.Since(start)

	return &info, latency, nil
}

var rakNetMagic = []byte{
	0x00, 0xff, 0xff, 0x00, 0xfe, 0xfe, 0xfe, 0xfe,
	0xfd, 0xfd, 0xfd, 0xfd, 0x12, 0x34, 0x56, 0x78,
}

type BedrockPinger struct {
	conn    *net.UDPConn
	address *ResolvedAddress
	timeout time.Duration
}

func NewBedrockPinger(addr *ResolvedAddress, timeout time.Duration) (*BedrockPinger, error) {
	raddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", addr.Host, addr.Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return nil, err
	}

	err = conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return nil, err
	}

	return &BedrockPinger{
		conn:    conn,
		address: addr,
		timeout: timeout,
	}, nil
}

func (p *BedrockPinger) Close() error {
	return p.conn.Close()
}

func (p *BedrockPinger) ReadStatus() (*BedrockServerInfo, time.Duration, error) {
	start := time.Now()

	pingData := make([]byte, 0, 33)
	pingData = append(pingData, 0x01)

	for i := 0; i < 8; i++ {
		pingData = append(pingData, 0x00)
	}

	pingData = append(pingData, rakNetMagic...)

	for i := 0; i < 8; i++ {
		pingData = append(pingData, 0x00)
	}

	_, err := p.conn.Write(pingData)
	if err != nil {
		return nil, 0, fmt.Errorf("发送 Ping 失败: %v", err)
	}

	buffer := make([]byte, 2048)
	n, err := p.conn.Read(buffer)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil, 0, fmt.Errorf("接收响应超时 (5秒)")
		}
		return nil, 0, fmt.Errorf("接收响应失败: %v", err)
	}

	if n < 2 {
		return nil, 0, fmt.Errorf("响应数据太短: %d 字节", n)
	}

	responseData := buffer[1:n]

	if len(responseData) < 34 {
		return nil, 0, fmt.Errorf("响应数据不完整: 需要至少34字节, 只有 %d 字节", len(responseData))
	}

	strLen := int(binary.BigEndian.Uint16(responseData[32:34]))

	if len(responseData) < 34+strLen {
		return nil, 0, fmt.Errorf("响应数据不完整: 需要 %d 字节, 只有 %d 字节", 34+strLen, len(responseData))
	}

	serverInfo := string(responseData[34 : 34+strLen])

	parts := strings.Split(serverInfo, ";")
	if len(parts) < 6 {
		return nil, 0, fmt.Errorf("无法解析服务器信息: %s", serverInfo)
	}

	info := &BedrockServerInfo{
		Brand:      parts[0],
		ServerName: parts[1],
		Protocol:   parseInt(parts[2]),
		Version:    parts[3],
		Online:     parseInt(parts[4]),
		Max:        parseInt(parts[5]),
	}

	if len(parts) > 7 {
		info.MapName = parts[7]
	}
	if len(parts) > 8 {
		info.GameMode = parts[8]
	}

	latency := time.Since(start)

	return info, latency, nil
}

func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}
