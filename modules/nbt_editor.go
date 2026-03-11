package modules

import (
	"bex/utils"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	TagEnd       = 0
	TagByte      = 1
	TagShort     = 2
	TagInt       = 3
	TagLong      = 4
	TagFloat     = 5
	TagDouble    = 6
	TagByteArray = 7
	TagString    = 8
	TagList      = 9
	TagCompound  = 10
	TagIntArray  = 11
	TagLongArray = 12
)

var tagNames = map[byte]string{
	TagEnd: "end", TagByte: "byte", TagShort: "short",
	TagInt: "int", TagLong: "long", TagFloat: "float",
	TagDouble: "double", TagByteArray: "byte[]",
	TagString: "string", TagList: "list",
	TagCompound: "compound", TagIntArray: "int[]",
	TagLongArray: "long[]",
}

var tagIcons = map[byte]string{
	TagEnd: "∅", TagByte: "B", TagShort: "S", TagInt: "I", TagLong: "L",
	TagFloat: "F", TagDouble: "D", TagByteArray: "[B]", TagString: `"`,
	TagList: "[ ]", TagCompound: "{ }", TagIntArray: "[I]", TagLongArray: "[L]",
}

var tagColors = map[byte]string{
	TagByte:      utils.ColorYellow,
	TagShort:     utils.ColorBrightYellow,
	TagInt:       utils.ColorGreen,
	TagLong:      utils.ColorBrightGreen,
	TagFloat:     utils.ColorCyan,
	TagDouble:    utils.ColorBrightCyan,
	TagString:    utils.ColorBlue,
	TagByteArray: utils.ColorPurple,
	TagIntArray:  utils.ColorBrightPurple,
	TagLongArray: utils.ColorRed,
	TagList:      utils.ColorBrightBlue,
	TagCompound:  utils.ColorBrightBlue,
}

func nodeIcon(node *NBTNode) string {
	tc := tagColors[node.Type]
	if node.Type == TagList {
		if l, ok := node.Value.(*NBTList); ok {
			empty := len(l.Items) == 0
			emptyMark := " [∅]"
			if !empty {
				emptyMark = " [ ]"
			}
			if l.ElemType == TagEnd {
				return tc + utils.StyleBold + "[∅]" + utils.ColorClear
			}
			elemIcon := tagIcons[l.ElemType]
			return tc + utils.StyleBold + elemIcon + emptyMark + utils.ColorClear
		}
	}
	return tc + utils.StyleBold + tagIcons[node.Type] + utils.ColorClear
}

func ansiLen(s string) int {
	n, ine := 0, false
	for _, r := range s {
		if r == '\x1b' {
			ine = true
			continue
		}
		if ine {
			if r == 'm' {
				ine = false
			}
			continue
		}
		n++
	}
	return n
}

func padRight(s string, width int) string {
	v := ansiLen(s)
	if v >= width {
		return s
	}
	return s + strings.Repeat(" ", width-v)
}

func truncVis(s string, maxW int) string {
	n, ine := 0, false
	var out strings.Builder
	for _, r := range s {
		if r == '\x1b' {
			ine = true
			out.WriteRune(r)
			continue
		}
		if ine {
			out.WriteRune(r)
			if r == 'm' {
				ine = false
			}
			continue
		}
		if n >= maxW {
			break
		}
		out.WriteRune(r)
		n++
	}
	return out.String()
}

type NBTNode struct {
	Type     byte
	Name     string
	Value    interface{}
	Parent   *NBTNode
	Expanded bool
	ListType byte
}

type NBTList struct {
	ElemType byte
	Items    []*NBTNode
}

type nbtR struct {
	data []byte
	pos  int
}

func (r *nbtR) b() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	v := r.data[r.pos]
	r.pos++
	return v, nil
}
func (r *nbtR) bs(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, io.ErrUnexpectedEOF
	}
	v := r.data[r.pos : r.pos+n]
	r.pos += n
	return v, nil
}
func (r *nbtR) i16() (int16, error) {
	b, e := r.bs(2)
	if e != nil {
		return 0, e
	}
	return int16(binary.BigEndian.Uint16(b)), nil
}
func (r *nbtR) i32() (int32, error) {
	b, e := r.bs(4)
	if e != nil {
		return 0, e
	}
	return int32(binary.BigEndian.Uint32(b)), nil
}
func (r *nbtR) i64() (int64, error) {
	b, e := r.bs(8)
	if e != nil {
		return 0, e
	}
	return int64(binary.BigEndian.Uint64(b)), nil
}
func (r *nbtR) f32() (float32, error) {
	b, e := r.bs(4)
	if e != nil {
		return 0, e
	}
	return math.Float32frombits(binary.BigEndian.Uint32(b)), nil
}
func (r *nbtR) f64() (float64, error) {
	b, e := r.bs(8)
	if e != nil {
		return 0, e
	}
	return math.Float64frombits(binary.BigEndian.Uint64(b)), nil
}
func (r *nbtR) str() (string, error) {
	l, e := r.i16()
	if e != nil {
		return "", e
	}
	if l < 0 {
		return "", fmt.Errorf("negative string length")
	}
	b, e := r.bs(int(l))
	if e != nil {
		return "", e
	}
	return string(b), nil
}

func (r *nbtR) payload(t byte, parent *NBTNode) (*NBTNode, error) {
	n := &NBTNode{Type: t, Parent: parent}
	switch t {
	case TagByte:
		b, e := r.b()
		if e != nil {
			return nil, e
		}
		n.Value = int8(b)
	case TagShort:
		v, e := r.i16()
		if e != nil {
			return nil, e
		}
		n.Value = v
	case TagInt:
		v, e := r.i32()
		if e != nil {
			return nil, e
		}
		n.Value = v
	case TagLong:
		v, e := r.i64()
		if e != nil {
			return nil, e
		}
		n.Value = v
	case TagFloat:
		v, e := r.f32()
		if e != nil {
			return nil, e
		}
		n.Value = v
	case TagDouble:
		v, e := r.f64()
		if e != nil {
			return nil, e
		}
		n.Value = v
	case TagString:
		s, e := r.str()
		if e != nil {
			return nil, e
		}
		n.Value = s
	case TagByteArray:
		l, e := r.i32()
		if e != nil {
			return nil, e
		}
		arr := make([]int8, l)
		for i := range arr {
			b, e := r.b()
			if e != nil {
				return nil, e
			}
			arr[i] = int8(b)
		}
		n.Value = arr
	case TagIntArray:
		l, e := r.i32()
		if e != nil {
			return nil, e
		}
		arr := make([]int32, l)
		for i := range arr {
			v, e := r.i32()
			if e != nil {
				return nil, e
			}
			arr[i] = v
		}
		n.Value = arr
	case TagLongArray:
		l, e := r.i32()
		if e != nil {
			return nil, e
		}
		arr := make([]int64, l)
		for i := range arr {
			v, e := r.i64()
			if e != nil {
				return nil, e
			}
			arr[i] = v
		}
		n.Value = arr
	case TagList:
		et, e := r.b()
		if e != nil {
			return nil, e
		}
		l, e := r.i32()
		if e != nil {
			return nil, e
		}
		list := &NBTList{ElemType: et}
		n.ListType = et
		for i := 0; i < int(l); i++ {
			c, e := r.payload(et, n)
			if e != nil {
				return nil, e
			}
			c.Name = fmt.Sprintf("[%d]", i)
			list.Items = append(list.Items, c)
		}
		n.Value = list
	case TagCompound:
		var ch []*NBTNode
		for {
			ct, e := r.b()
			if e != nil {
				return nil, e
			}
			if ct == TagEnd {
				break
			}
			name, e := r.str()
			if e != nil {
				return nil, e
			}
			c, e := r.payload(ct, n)
			if e != nil {
				return nil, e
			}
			c.Name = name
			ch = append(ch, c)
		}
		n.Value = ch
	}
	return n, nil
}

func parseNBT(data []byte) (*NBTNode, error) {
	r := &nbtR{data: data}
	t, e := r.b()
	if e != nil {
		return nil, e
	}
	if t == TagEnd {
		return nil, fmt.Errorf("unexpected TAG_End at root")
	}
	name, e := r.str()
	if e != nil {
		return nil, e
	}
	n, e := r.payload(t, nil)
	if e != nil {
		return nil, e
	}
	n.Name = name
	return n, nil
}

type ErrCorrupt struct{ reason string }

func (e *ErrCorrupt) Error() string { return "NBT格式错误: " + e.reason }

type ErrEncrypted struct{}

func (e *ErrEncrypted) Error() string { return "文件疑似加密，无法打开" }

type compressionType int

const (
	compNone compressionType = iota
	compGzip
	compZlib
	compRegion
)

type fileFormat int

const (
	fmtNBT fileFormat = iota
	fmtRegion
)

func detectCompression(data []byte, ext string) (compressionType, error) {

	if ext == ".mca" || ext == ".mcr" {
		if len(data) < 8192 {
			return 0, &ErrCorrupt{"Region文件过短（需≥8KB头部）"}
		}
		return compRegion, nil
	}

	if len(data) < 2 {
		return 0, &ErrCorrupt{"文件过短"}
	}
	b0, b1 := data[0], data[1]

	if b0 == 0x1f && b1 == 0x8b {
		return compGzip, nil
	}

	if b0 == 0x78 && (b1 == 0x01 || b1 == 0x5e || b1 == 0x9c || b1 == 0xda) {
		return compZlib, nil
	}

	if b0 >= 1 && b0 <= 12 {
		return compNone, nil
	}

	printable := 0
	check := len(data)
	if check > 256 {
		check = 256
	}
	for _, c := range data[:check] {
		if c >= 0x20 && c < 0x7f {
			printable++
		}
	}
	if printable*100/check > 80 {
		return 0, &ErrCorrupt{fmt.Sprintf("非NBT文件（首字节 0x%02x）", b0)}
	}
	return 0, &ErrEncrypted{}
}

func decompressPayload(raw []byte, comp compressionType) ([]byte, error) {
	switch comp {
	case compGzip:
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, &ErrCorrupt{"gzip头损坏: " + err.Error()}
		}
		out, err := io.ReadAll(gz)
		_ = gz.Close()
		if err != nil {
			return nil, &ErrCorrupt{"gzip解压失败: " + err.Error()}
		}
		return out, nil
	case compZlib:
		zr, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, &ErrCorrupt{"zlib头损坏: " + err.Error()}
		}
		out, err := io.ReadAll(zr)
		_ = zr.Close()
		if err != nil {
			return nil, &ErrCorrupt{"zlib解压失败: " + err.Error()}
		}
		return out, nil
	default:
		return raw, nil
	}
}

const regionSector = 4096

func parseRegion(raw []byte) (*NBTNode, error) {
	if len(raw) < regionSector*2 {
		return nil, &ErrCorrupt{"Region文件头不完整"}
	}

	root := &NBTNode{
		Type:  TagCompound,
		Name:  "region",
		Value: []*NBTNode{},
	}

	for i := 0; i < 1024; i++ {

		entry := binary.BigEndian.Uint32(raw[i*4 : i*4+4])
		sectorOffset := int(entry >> 8)
		sectorCount := int(entry & 0xff)
		if sectorOffset == 0 && sectorCount == 0 {
			continue
		}

		x := i % 32
		z := i / 32
		chunkName := fmt.Sprintf("chunk[%d,%d]", x, z)

		byteOffset := sectorOffset * regionSector
		if byteOffset+5 > len(raw) {

			continue
		}

		chunkLen := int(binary.BigEndian.Uint32(raw[byteOffset : byteOffset+4]))
		if chunkLen < 1 || byteOffset+4+chunkLen > len(raw) {
			continue
		}
		compScheme := raw[byteOffset+4]
		chunkData := raw[byteOffset+5 : byteOffset+4+chunkLen]

		if compScheme >= 128 {
			continue
		}

		var payload []byte
		var err error
		switch compScheme {
		case 1:
			payload, err = decompressPayload(chunkData, compGzip)
		case 2:
			payload, err = decompressPayload(chunkData, compZlib)
		case 3:
			payload = chunkData
		default:
			continue
		}
		if err != nil {
			continue
		}

		chunkNode, err := parseNBT(payload)
		if err != nil {
			continue
		}
		chunkNode.Name = chunkName
		chunkNode.Parent = root
		root.Value = append(root.Value.([]*NBTNode), chunkNode)
	}

	if len(root.Value.([]*NBTNode)) == 0 {
		return nil, &ErrCorrupt{"Region文件中没有有效的chunk数据"}
	}
	return root, nil
}

func saveRegion(root *NBTNode, path string) error {
	chunks := root.Value.([]*NBTNode)

	header := make([]byte, regionSector*2)

	var sectorBufs [][]byte
	nextSector := 2

	for _, chunk := range chunks {
		var cx, cz int
		_, _ = fmt.Sscanf(chunk.Name, "chunk[%d,%d]", &cx, &cz)
		i := cz*32 + cx

		wr := &nbtW{}
		wr.b(chunk.Type)
		wr.str(chunk.Name)
		wr.node(chunk)

		var zbuf bytes.Buffer
		zw := zlib.NewWriter(&zbuf)
		_, _ = zw.Write(wr.buf)
		_ = zw.Close()
		compressed := zbuf.Bytes()

		totalLen := 4 + 1 + len(compressed)

		sectors := (totalLen + regionSector - 1) / regionSector
		block := make([]byte, sectors*regionSector)
		binary.BigEndian.PutUint32(block[0:4], uint32(len(compressed)+1))
		block[4] = 2
		copy(block[5:], compressed)

		entry := uint32(nextSector<<8) | uint32(sectors)
		binary.BigEndian.PutUint32(header[i*4:i*4+4], entry)

		sectorBufs = append(sectorBufs, block)
		nextSector += sectors
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	if _, err = f.Write(header); err != nil {
		return err
	}
	for _, buf := range sectorBufs {
		if _, err = f.Write(buf); err != nil {
			return err
		}
	}
	return nil
}

func extFormat(ext string) (string, fileFormat) {
	switch ext {
	case ".mca":
		return "Region(.mca)", fmtRegion
	case ".mcr":
		return "Region(.mcr)", fmtRegion
	case ".litematic":
		return "Litematic", fmtNBT
	case ".schematic":
		return "Schematic", fmtNBT
	case ".nbt":
		return "NBT", fmtNBT
	default:
		return "NBT", fmtNBT
	}
}

func loadNBT(path string) (*NBTNode, compressionType, fileFormat, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, err
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("读取文件失败: %w", err)
	}

	ext := ""
	if idx := strings.LastIndexByte(path, '.'); idx >= 0 {
		ext = strings.ToLower(path[idx:])
	}
	_, fmtType := extFormat(ext)

	comp, err := detectCompression(raw, ext)
	if err != nil {
		return nil, 0, 0, err
	}

	if comp == compRegion {
		node, err := parseRegion(raw)
		if err != nil {
			return nil, 0, 0, err
		}
		return node, compRegion, fmtRegion, nil
	}

	payload, err := decompressPayload(raw, comp)
	if err != nil {
		return nil, 0, 0, err
	}

	n, err := parseNBT(payload)
	if err != nil {
		return nil, 0, 0, &ErrCorrupt{"解析失败: " + err.Error()}
	}
	return n, comp, fmtType, nil
}

type nbtW struct{ buf []byte }

func (w *nbtW) b(v byte)     { w.buf = append(w.buf, v) }
func (w *nbtW) bs(v []byte)  { w.buf = append(w.buf, v...) }
func (w *nbtW) i16(v int16)  { b := make([]byte, 2); binary.BigEndian.PutUint16(b, uint16(v)); w.bs(b) }
func (w *nbtW) i32(v int32)  { b := make([]byte, 4); binary.BigEndian.PutUint32(b, uint32(v)); w.bs(b) }
func (w *nbtW) i64(v int64)  { b := make([]byte, 8); binary.BigEndian.PutUint64(b, uint64(v)); w.bs(b) }
func (w *nbtW) str(s string) { w.i16(int16(len(s))); w.bs([]byte(s)) }

func (w *nbtW) node(n *NBTNode) {
	switch n.Type {
	case TagByte:
		w.b(byte(n.Value.(int8)))
	case TagShort:
		w.i16(n.Value.(int16))
	case TagInt:
		w.i32(n.Value.(int32))
	case TagLong:
		w.i64(n.Value.(int64))
	case TagFloat:
		w.i32(int32(math.Float32bits(n.Value.(float32))))
	case TagDouble:
		w.i64(int64(math.Float64bits(n.Value.(float64))))
	case TagString:
		w.str(n.Value.(string))
	case TagByteArray:
		arr := n.Value.([]int8)
		w.i32(int32(len(arr)))
		for _, v := range arr {
			w.b(byte(v))
		}
	case TagIntArray:
		arr := n.Value.([]int32)
		w.i32(int32(len(arr)))
		for _, v := range arr {
			w.i32(v)
		}
	case TagLongArray:
		arr := n.Value.([]int64)
		w.i32(int32(len(arr)))
		for _, v := range arr {
			w.i64(v)
		}
	case TagList:
		list := n.Value.(*NBTList)
		w.b(list.ElemType)
		w.i32(int32(len(list.Items)))
		for _, c := range list.Items {
			w.node(c)
		}
	case TagCompound:
		for _, c := range n.Value.([]*NBTNode) {
			w.b(c.Type)
			w.str(c.Name)
			w.node(c)
		}
		w.b(TagEnd)
	}
}

func saveNBT(n *NBTNode, path string, comp compressionType) error {

	if comp == compRegion {
		return saveRegion(n, path)
	}
	wr := &nbtW{}
	wr.b(n.Type)
	wr.str(n.Name)
	wr.node(n)
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)
	switch comp {
	case compGzip:
		gz := gzip.NewWriter(f)
		_, e = gz.Write(wr.buf)
		if e != nil {
			return e
		}
		return gz.Close()
	case compZlib:
		zw := zlib.NewWriter(f)
		_, e = zw.Write(wr.buf)
		if e != nil {
			return e
		}
		return zw.Close()
	default:
		_, e = f.Write(wr.buf)
		return e
	}
}

type FlatItem struct {
	Node     *NBTNode
	Depth    int
	Prefix   string
	ChildPfx string
}

func buildFlat(root *NBTNode) []FlatItem {
	var result []FlatItem
	var walk func(n *NBTNode, prefix, childPfx string, depth int)
	walk = func(n *NBTNode, prefix, childPfx string, depth int) {
		result = append(result, FlatItem{Node: n, Depth: depth, Prefix: prefix, ChildPfx: childPfx})
		if !n.Expanded {
			return
		}
		var children []*NBTNode
		switch n.Type {
		case TagCompound:
			if ch, ok := n.Value.([]*NBTNode); ok {
				children = ch
			}
		case TagList:
			if list, ok := n.Value.(*NBTList); ok {
				for _, c := range list.Items {
					children = append(children, c)
				}
			}
		}
		for i, c := range children {
			last := i == len(children)-1
			var cp, cc string
			if last {
				cp = childPfx + "└─ "
				cc = childPfx + "   "
			} else {
				cp = childPfx + "├─ "
				cc = childPfx + "│  "
			}
			walk(c, cp, cc, depth+1)
		}
	}
	walk(root, "", "", 0)
	return result
}

func valueStr(n *NBTNode) string {
	switch n.Type {
	case TagByte:
		return fmt.Sprintf("%d", n.Value.(int8))
	case TagShort:
		return fmt.Sprintf("%d", n.Value.(int16))
	case TagInt:
		return fmt.Sprintf("%d", n.Value.(int32))
	case TagLong:
		return fmt.Sprintf("%dL", n.Value.(int64))
	case TagFloat:
		return fmt.Sprintf("%gf", n.Value.(float32))
	case TagDouble:
		return fmt.Sprintf("%g", n.Value.(float64))
	case TagString:
		return n.Value.(string)
	case TagByteArray:
		arr := n.Value.([]int8)
		if len(arr) == 0 {
			return "[B; ]"
		}
		preview := make([]string, min8(3, len(arr)))
		for i := range preview {
			preview[i] = fmt.Sprintf("%d", arr[i])
		}
		suffix := ""
		if len(arr) > 3 {
			suffix = ", ..."
		}
		return fmt.Sprintf("[B; %s%s]  (%d)", strings.Join(preview, ", "), suffix, len(arr))
	case TagIntArray:
		arr := n.Value.([]int32)
		if len(arr) == 0 {
			return "[I; ]"
		}
		preview := make([]string, min8(3, len(arr)))
		for i := range preview {
			preview[i] = fmt.Sprintf("%d", arr[i])
		}
		suffix := ""
		if len(arr) > 3 {
			suffix = ", ..."
		}
		return fmt.Sprintf("[I; %s%s]  (%d)", strings.Join(preview, ", "), suffix, len(arr))
	case TagLongArray:
		arr := n.Value.([]int64)
		if len(arr) == 0 {
			return "[L; ]"
		}
		preview := make([]string, min8(3, len(arr)))
		for i := range preview {
			preview[i] = fmt.Sprintf("%d", arr[i])
		}
		suffix := ""
		if len(arr) > 3 {
			suffix = ", ..."
		}
		return fmt.Sprintf("[L; %s%s]  (%d)", strings.Join(preview, ", "), suffix, len(arr))
	case TagList:
		if l, ok := n.Value.(*NBTList); ok {
			return fmt.Sprintf("%d 项", len(l.Items))
		}
	case TagCompound:
		if ch, ok := n.Value.([]*NBTNode); ok {
			return fmt.Sprintf("%d 项", len(ch))
		}
	}
	return ""
}

func editableValueStr(n *NBTNode) string {
	switch n.Type {
	case TagByte:
		return fmt.Sprintf("%d", n.Value.(int8))
	case TagShort:
		return fmt.Sprintf("%d", n.Value.(int16))
	case TagInt:
		return fmt.Sprintf("%d", n.Value.(int32))
	case TagLong:
		return fmt.Sprintf("%d", n.Value.(int64))
	case TagFloat:
		return fmt.Sprintf("%g", n.Value.(float32))
	case TagDouble:
		return fmt.Sprintf("%g", n.Value.(float64))
	case TagString:
		return n.Value.(string)
	}
	return ""
}

func parseValue(t byte, s string) (interface{}, error) {
	switch t {
	case TagByte:
		v, e := strconv.ParseInt(s, 10, 8)
		return int8(v), e
	case TagShort:
		v, e := strconv.ParseInt(s, 10, 16)
		return int16(v), e
	case TagInt:
		v, e := strconv.ParseInt(s, 10, 32)
		return int32(v), e
	case TagLong:
		v, e := strconv.ParseInt(s, 10, 64)
		return v, e
	case TagFloat:
		v, e := strconv.ParseFloat(s, 32)
		return float32(v), e
	case TagDouble:
		v, e := strconv.ParseFloat(s, 64)
		return v, e
	case TagString:
		return s, nil
	case TagByteArray:
		return parseByteArray(s)
	case TagIntArray:
		return parseIntArray(s)
	case TagLongArray:
		return parseLongArray(s)
	}
	return nil, fmt.Errorf("unsupported type")
}

func parseByteArray(s string) ([]int8, error) {
	if strings.TrimSpace(s) == "" {
		return []int8{}, nil
	}
	tokens := strings.Split(s, ",")
	out := make([]int8, 0, len(tokens))
	for _, tok := range tokens {
		v, e := strconv.ParseInt(strings.TrimSpace(tok), 10, 8)
		if e != nil {
			return nil, fmt.Errorf("byte value %q: %v", strings.TrimSpace(tok), e)
		}
		out = append(out, int8(v))
	}
	return out, nil
}

func parseIntArray(s string) ([]int32, error) {
	if strings.TrimSpace(s) == "" {
		return []int32{}, nil
	}
	tokens := strings.Split(s, ",")
	out := make([]int32, 0, len(tokens))
	for _, tok := range tokens {
		v, e := strconv.ParseInt(strings.TrimSpace(tok), 10, 32)
		if e != nil {
			return nil, fmt.Errorf("int value %q: %v", strings.TrimSpace(tok), e)
		}
		out = append(out, int32(v))
	}
	return out, nil
}

func parseLongArray(s string) ([]int64, error) {
	if strings.TrimSpace(s) == "" {
		return []int64{}, nil
	}
	tokens := strings.Split(s, ",")
	out := make([]int64, 0, len(tokens))
	for _, tok := range tokens {
		v, e := strconv.ParseInt(strings.TrimSpace(tok), 10, 64)
		if e != nil {
			return nil, fmt.Errorf("long value %q: %v", strings.TrimSpace(tok), e)
		}
		out = append(out, v)
	}
	return out, nil
}

func isContainer(n *NBTNode) bool { return n.Type == TagCompound || n.Type == TagList }
func isEditable(n *NBTNode) bool {
	switch n.Type {
	case TagByte, TagShort, TagInt, TagLong, TagFloat, TagDouble, TagString:
		return true
	}
	return false
}

func insertChild(parent *NBTNode, child *NBTNode, idx int) {
	child.Parent = parent
	switch parent.Type {
	case TagCompound:
		ch := parent.Value.([]*NBTNode)
		if idx < 0 || idx >= len(ch) {
			parent.Value = append(ch, child)
			return
		}
		newCh := make([]*NBTNode, 0, len(ch)+1)
		newCh = append(newCh, ch[:idx]...)
		newCh = append(newCh, child)
		newCh = append(newCh, ch[idx:]...)
		parent.Value = newCh
	case TagList:
		l := parent.Value.(*NBTList)
		if idx < 0 || idx >= len(l.Items) {
			l.Items = append(l.Items, child)
		} else {
			newItems := make([]*NBTNode, 0, len(l.Items)+1)
			newItems = append(newItems, l.Items[:idx]...)
			newItems = append(newItems, child)
			newItems = append(newItems, l.Items[idx:]...)
			l.Items = newItems
		}
		for j, item := range l.Items {
			item.Name = fmt.Sprintf("[%d]", j)
		}
	}
}

func removeChildAt(parent *NBTNode, idx int) *NBTNode {
	switch parent.Type {
	case TagCompound:
		ch := parent.Value.([]*NBTNode)
		if idx < 0 || idx >= len(ch) {
			return nil
		}
		node := ch[idx]
		parent.Value = append(ch[:idx], ch[idx+1:]...)
		node.Parent = nil
		return node
	case TagList:
		l := parent.Value.(*NBTList)
		if idx < 0 || idx >= len(l.Items) {
			return nil
		}
		node := l.Items[idx]
		l.Items = append(l.Items[:idx], l.Items[idx+1:]...)
		for j, item := range l.Items {
			item.Name = fmt.Sprintf("[%d]", j)
		}
		node.Parent = nil
		return node
	}
	return nil
}

func childIndex(parent *NBTNode, child *NBTNode) int {
	switch parent.Type {
	case TagCompound:
		if ch, ok := parent.Value.([]*NBTNode); ok {
			for i, c := range ch {
				if c == child {
					return i
				}
			}
		}
	case TagList:
		if l, ok := parent.Value.(*NBTList); ok {
			for i, c := range l.Items {
				if c == child {
					return i
				}
			}
		}
	}
	return -1
}

var termOldState *term.State

func termInit() {
	st, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	termOldState = st
}
func termRestore() {
	if termOldState != nil {
		_ = term.Restore(int(os.Stdin.Fd()), termOldState)
	}
}
func termSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func readKey() string {
	keyCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 32)
		n, _ := os.Stdin.Read(buf)
		s := string(buf[:n])

		if n >= 6 && s[:3] == "\x1b[<" {
			rest := s[3:]
			if len(rest) > 0 {
				var btn int
				_, _ = fmt.Sscanf(rest, "%d;", &btn)
				if btn == 64 {
					keyCh <- "\x1b[A"
					return
				}
				if btn == 65 {
					keyCh <- "\x1b[B"
					return
				}
			}
			keyCh <- ""
			return
		}
		if n >= 6 && buf[0] == 0x1b && buf[1] == '[' && buf[2] == 'M' {
			btn := buf[3] & 0x7f
			if btn == 64 {
				keyCh <- "\x1b[A"
				return
			}
			if btn == 65 {
				keyCh <- "\x1b[B"
				return
			}
			keyCh <- ""
			return
		}
		keyCh <- s
	}()

	select {
	case key := <-keyCh:
		return key
	case <-resizeCh:

		for {
			select {
			case <-resizeCh:
			default:
				return "\x00resize"
			}
		}
	}
}

func insRune(s string, pos int, r rune) (string, int) {
	rr := []rune(s)
	out := make([]rune, 0, len(rr)+1)
	out = append(out, rr[:pos]...)
	out = append(out, r)
	out = append(out, rr[pos:]...)
	return string(out), pos + 1
}
func delBack(s string, pos int) (string, int) {
	rr := []rune(s)
	if pos == 0 {
		return s, 0
	}
	return string(append(append([]rune{}, rr[:pos-1]...), rr[pos:]...)), pos - 1
}
func delFwd(s string, pos int) string {
	rr := []rune(s)
	if pos >= len(rr) {
		return s
	}
	return string(append(append([]rune{}, rr[:pos]...), rr[pos+1:]...))
}

type Mode int

const (
	ModeNormal Mode = iota
	ModeEdit
	ModeRename
	ModeAddInline
	ModeAddName
	ModeAddType
	ModeAddValue
	ModeHelp
	ModeConfirmQuit
)

var addTypeIDs = []byte{
	TagByte, TagShort, TagInt, TagLong,
	TagFloat, TagDouble, TagString,
	TagList, TagCompound,
}
var addTypeLabels = []string{
	"byte", "short", "int", "long",
	"float", "double", "string",
	"list", "compound",
}

type Editor struct {
	root, addParent *NBTNode
	flat            []FlatItem
	cursor, scroll  int
	mode            Mode

	editBuf string
	editPos int

	addInlineBuf string
	addInlinePos int
	addName      string
	addNamePos   int
	addType      byte
	addValue     string
	addValPos    int

	message     string
	msgIsErr    bool
	modified    bool
	filepath    string
	compression compressionType
	format      fileFormat

	helpPage int
}

func NewEditor(root *NBTNode, fp string, comp compressionType, fmtType fileFormat) *Editor {
	root.Expanded = true
	e := &Editor{root: root, filepath: fp, compression: comp, format: fmtType}
	e.rebuild()
	return e
}

func (e *Editor) rebuild() {
	e.flat = buildFlat(e.root)
	if e.cursor >= len(e.flat) {
		e.cursor = imax(0, len(e.flat)-1)
	}
}

func (e *Editor) selected() *NBTNode {
	if e.cursor < len(e.flat) {
		return e.flat[e.cursor].Node
	}
	return nil
}

func (e *Editor) setMsg(msg string, isErr bool) { e.message = msg; e.msgIsErr = isErr }

func renderCursor(buf string, pos int, color string) string {
	rr := []rune(buf)
	var sb strings.Builder
	sb.WriteString(color)
	rev := utils.TC.Reverse()
	for i, r := range rr {
		if i == pos {
			sb.WriteString(rev + string(r) + utils.ColorClear + color)
		} else {
			sb.WriteRune(r)
		}
	}
	if pos >= len(rr) {
		sb.WriteString(rev + " " + utils.ColorClear)
	}
	sb.WriteString(utils.ColorClear)
	return sb.String()
}

func (e *Editor) render() {
	w, h := termSize()
	bodyH := h - 3
	if bodyH < 1 {
		bodyH = 1
	}

	if e.cursor < e.scroll {
		e.scroll = e.cursor
	}
	if e.cursor >= e.scroll+bodyH {
		e.scroll = e.cursor - bodyH + 1
	}

	if maxScroll := imax(0, len(e.flat)-bodyH); e.scroll > maxScroll {
		e.scroll = maxScroll
	}

	var sb strings.Builder

	filename := e.filepath
	if idx := strings.LastIndexByte(e.filepath, '/'); idx >= 0 {
		filename = e.filepath[idx+1:]
	} else if idx := strings.LastIndexByte(e.filepath, '\\'); idx >= 0 {
		filename = e.filepath[idx+1:]
	}
	title := " NBT 编辑器 | " + filename
	if e.modified {
		title += " [!]"
	}

	rev := utils.TC.Reverse()
	sb.WriteString("\x1b[1;1H\x1b[2K" + rev + truncVis(title, w-1) + utils.ColorClear + utils.TC.FillReverse())

	for row := 0; row < bodyH; row++ {
		idx := e.scroll + row
		sb.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[2K", row+2))
		if idx >= len(e.flat) {
			continue
		}

		item := e.flat[idx]
		node := item.Node
		sel := idx == e.cursor
		editing := sel && (e.mode == ModeEdit || e.mode == ModeRename)

		line := utils.ColorBrightBlack + item.Prefix + utils.ColorClear

		if isContainer(node) {
			if node.Expanded {
				line += utils.StyleBold + "- " + utils.ColorClear
			} else {
				line += utils.StyleBold + "+ " + utils.ColorClear
			}
		} else {
			line += "  "
		}

		tc := tagColors[node.Type]
		line += nodeIcon(node)
		if node.Name != "" {
			line += " " + utils.StyleBold + node.Name + utils.ColorClear
		}

		if editing {
			typePart := utils.ColorBrightBlack + "  [" + tagNames[node.Type] + ": " + utils.ColorClear
			if e.mode == ModeRename {
				typePart = utils.ColorBrightBlack + "  [rename: " + utils.ColorClear
			}
			line += typePart + renderCursor(e.editBuf, e.editPos, utils.ColorCyan) + utils.ColorBrightBlack + "]" + utils.ColorClear
		} else {
			switch node.Type {
			case TagCompound, TagList:
				line += utils.ColorBrightBlack + "  " + utils.StyleDim + valueStr(node) + utils.ColorClear
			default:
				line += utils.ColorBrightBlack + ": " + utils.ColorClear + tc + valueStr(node) + utils.ColorClear
			}
		}

		line = truncVis(line, w-1)
		if sel && !editing {

			sb.WriteString(utils.TC.Reverse() + line + utils.ColorClear + utils.TC.FillReverse())
		} else {
			sb.WriteString(line)
		}
	}

	sb.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[2K", h-1))
	var statusLine string
	switch e.mode {
	case ModeAddInline:

		var hint string
		node := e.selected()
		parentNode := node
		if node != nil && !isContainer(node) && node.Parent != nil {
			parentNode = node.Parent
		}
		if parentNode != nil && parentNode.Type == TagList {
			list := parentNode.Value.(*NBTList)
			if list.ElemType == TagCompound {
				hint = "直接按Enter创建空复合元素"
			} else {
				hint = "输入值，如 42"
			}
		} else {
			hint = "类型:名称:值  |  compound:名称  |  list:元素类型:名称  (^H查看完整格式)"
		}
		statusLine = utils.StyleBold + " ^N 新建 " + utils.ColorBrightBlack + hint + ": " + utils.ColorClear + renderCursor(e.addInlineBuf, e.addInlinePos, utils.ColorCyan)
	case ModeAddName:
		statusLine = utils.StyleBold + " 新标签名称: " + utils.ColorClear + renderCursor(e.addName, e.addNamePos, utils.ColorCyan)
	case ModeAddType:
		var typeSel strings.Builder
		typeSel.WriteString(utils.StyleBold + " 选择类型 " + utils.ColorBrightBlack + "(←→切换, Enter确认): " + utils.ColorClear)
		for i, id := range addTypeIDs {
			if id == e.addType {
				typeSel.WriteString(utils.TC.Reverse() + " " + addTypeLabels[i] + " " + utils.ColorClear + " ")
			} else {
				typeSel.WriteString(utils.StyleDim + addTypeLabels[i] + utils.ColorClear + " ")
			}
		}
		statusLine = typeSel.String()
	case ModeAddValue:
		statusLine = utils.StyleBold + " 值 [" + tagNames[e.addType] + "]: " + utils.ColorClear + renderCursor(e.addValue, e.addValPos, utils.ColorCyan)
	case ModeConfirmQuit:
		statusLine = utils.ColorYellow + utils.StyleBold + " 有未保存的修改！^S=保存并退出  ^Q=放弃并退出  其他键=取消" + utils.ColorClear
	default:
		if e.message != "" {
			if e.msgIsErr {
				statusLine = utils.ColorRed + utils.StyleBold + "[×] " + e.message + utils.ColorClear
			} else {
				statusLine = utils.ColorGreen + utils.StyleBold + "[√] " + e.message + utils.ColorClear
			}
		} else {
			if node := e.selected(); node != nil {
				if node.Type == TagCompound || node.Type == TagList {
					statusLine = utils.ColorBrightBlack + fmt.Sprintf("  %s:%s", tagNames[node.Type], node.Name) + utils.ColorClear
				} else {
					statusLine = utils.ColorBrightBlack + fmt.Sprintf("  %s:%s:%s", tagNames[node.Type], node.Name, valueStr(node)) + utils.ColorClear
				}
			}
		}
	}
	sb.WriteString(truncVis(statusLine, w-1) + "\x1b[K")

	sb.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[2K", h))
	hints := " ^E:编辑  ^R:命名  ^N:新建  ^D:删除  ^S:保存  ^Q:退出  ^H:帮助"
	sb.WriteString(utils.StyleDim + truncVis(hints, w-1) + utils.ColorClear + "\x1b[K")

	sb.WriteString("\x1b[2;1H")

	fmt.Print(sb.String())
}

func helpAllPages() [][]string {
	return [][]string{
		{
			utils.StyleBold + "+ 界面导航" + utils.ColorClear,
			"  ↑ / ↓           移动光标选择节点",
			"  → / Enter       展开/折叠容器节点（Compound / List）",
			"  ←               折叠当前容器 / 跳回父节点",
			"  Home / End      （普通模式）跳转到首个 / 最后一个节点",
			"  ^A / ^S         （帮助界面）上一页 / 下一页",
			"  鼠标滚轮         滚动视图",
			"",
			utils.StyleBold + "+ 节点操作" + utils.ColorClear,
			"  ^E (Ctrl+E)     编辑当前节点的值",
			"                   支持类型: byte / short / int / long / float / double / string",
			"                   内联编辑：Enter 确认，Esc 取消",
			"  ^R (Ctrl+R)     重命名当前节点",
			"                   限制：根节点和列表元素不可重命名",
			"  ^N (Ctrl+N)     新建节点（详见第 2 页）",
			"  ^D (Ctrl+D)     删除当前节点（根节点不可删除）",
			"",
			utils.StyleBold + "+ 文件操作" + utils.ColorClear,
			"  ^S (Ctrl+S)     保存修改到原文件",
			"  ^Q (Ctrl+Q)     退出编辑器",
			"                   若有未保存修改，会提示：",
			"                   ^S=保存并退出  ^Q=放弃并退出  其他=取消",
		},
		{
			utils.StyleBold + "+ ^N 新建节点功能详解" + utils.ColorClear,
			"",
			utils.StyleBold + "  [1] 根据光标位置决定插入策略" + utils.ColorClear,
			"  • 光标位于容器（Compound / List）→ 作为子节点插入容器末尾",
			"  • 光标位于叶子节点              → 作为同级节点插入当前节点之后",
			"",
			utils.StyleBold + "  [2] 在 Compound 中新建节点" + utils.ColorClear,
			"  格式: 类型:名称[:值]",
			"",
			"  标量类型（直接输入值）:",
			"    byte       范围 -128 ～ 127         示例: byte:Health:20",
			"    short      范围 -32768 ～ 32767     示例: short:Level:5",
			"    int        范围 -2^31 ～ 2^31-1     示例: int:Score:0",
			"    long       范围 -2^63 ～ 2^63-1     示例: long:Time:12000",
			"    float      单精度浮点数              示例: float:Speed:1.5",
			"    double     双精度浮点数              示例: double:X:128.0",
			"    string     UTF-8 字符串             示例: string:Name:Steve",
			"",
			"  数组类型（值可选，逗号分隔）:",
			"    byte[]     字节数组   示例: byte[]:Data:1,2,3  或  byte[]:Data",
			"    int[]      整数数组   示例: int[]:Pos:100,64,200",
			"    long[]     长整数数组 示例: long[]:UUIDs",
		},
		{
			utils.StyleBold + "  在 Compound 中新建容器类型" + utils.ColorClear,
			"",
			"  容器类型（无需值）:",
			"    compound   复合标签   示例: compound:Attributes",
			"               别名: comp / group",
			"    list       列表       示例: list:int:Scores",
			"               格式: list:元素类型:名称",
			"               元素类型: byte / short / int / long / float / double / string / compound",
			"",
			utils.StyleBold + "  [3] 在 List 中新建元素" + utils.ColorClear,
			"  List 的插入行为由其元素类型决定：",
			"",
			"  • 基本类型列表 (byte / short / int / long / float / double / string)",
			"    直接输入值即可，例如: 42  或  Hello",
			"    （值将根据列表的元素类型自动解析）",
			"",
			"  • 复合标签列表 (List of Compound)",
			"    直接按 Enter 创建空复合元素，之后可用 ^N 继续添加子节点",
			"",
			"  • 嵌套列表 (List of List)",
			"    直接输入元素类型:名称:值 格式（待完善）",
			"",
			utils.StyleBold + "+ 键盘快捷键速查" + utils.ColorClear,
			"  ^E 编辑   ^R 命名   ^N 新建   ^D 删除",
			"  ^S 保存   ^Q 退出   ^H / ? 帮助   Esc 取消操作",
		},
	}
}

func (e *Editor) renderHelp() {
	w, h := termSize()
	pages := helpAllPages()
	totalPages := len(pages)

	if e.helpPage < 0 {
		e.helpPage = 0
	}
	if e.helpPage >= totalPages {
		e.helpPage = totalPages - 1
	}

	pageInfo := fmt.Sprintf(" 第 %d/%d 页 ", e.helpPage+1, totalPages)
	titleVis := " NBT Editor — 帮助文档 " + strings.Repeat(" ", imax(0, w-len(" NBT Editor — 帮助文档 ")-len("^A上页 ^S下页  ^H退出 "+pageInfo))) + "^A上页 ^S下页  ^H退出 " + pageInfo

	var sb strings.Builder
	sb.WriteString("\x1b[2J")
	sb.WriteString("\x1b[1;1H\x1b[2K")
	sb.WriteString(utils.StyleBold + utils.TC.Reverse() + truncVis(titleVis, w) + utils.ColorClear + utils.TC.FillReverse())

	content := pages[e.helpPage]
	bodyH := h - 2
	if bodyH < 1 {
		bodyH = 1
	}

	for row := 0; row < bodyH; row++ {
		sb.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[2K", row+2))
		if row < len(content) {
			sb.WriteString(truncVis(padRight(content[row], w), w))
		}
	}

	var navHint string
	if e.helpPage == 0 {
		navHint = utils.StyleDim + "  ^S 下一页  |  ^H 返回编辑器" + utils.ColorClear
	} else if e.helpPage == totalPages-1 {
		navHint = utils.StyleDim + "  ^A 上一页  |  ^H 返回编辑器" + utils.ColorClear
	} else {
		navHint = utils.StyleDim + "  ^A 上一页   ^S 下一页  |  ^H 返回编辑器" + utils.ColorClear
	}
	sb.WriteString(fmt.Sprintf("\x1b[%d;1H\x1b[2K", h))
	sb.WriteString(truncVis(navHint, w))

	fmt.Print(sb.String())
}

func (e *Editor) handleHelp(key string) {
	pages := helpAllPages()
	switch key {
	case "\x08":
		e.mode = ModeNormal
		e.helpPage = 0
	case "\x01":
		if e.helpPage > 0 {
			e.helpPage--
		}
	case "\x13":
		if e.helpPage < len(pages)-1 {
			e.helpPage++
		}
	}
}

func (e *Editor) Run() {
	utils.InitTermCap()
	termInit()
	defer termRestore()

	defer fmt.Print(utils.TC.DisableMouse + utils.TC.LeaveAltScreen + utils.TC.ShowCursor)

	fmt.Print(utils.TC.EnterAltScreen + utils.TC.HideCursor + utils.TC.EnableMouse)

	if !utils.TC.AltScreenOK {
		fmt.Print("\x1b[2J\x1b[1;1H")
	}

	startResizeWatcher()

	for {
		if e.mode == ModeHelp {
			e.renderHelp()
		} else {
			e.render()
		}

		e.message = ""
		key := readKey()
		if key == "\x00resize" {
			continue
		}
		switch e.mode {
		case ModeNormal:
			e.handleNormal(key)
		case ModeEdit:
			e.handleInlineEdit(key, false)
		case ModeRename:
			e.handleInlineEdit(key, true)
		case ModeAddInline:
			e.handleAddInline(key)
		case ModeAddName:
			e.handleAddName(key)
		case ModeAddType:
			e.handleAddType(key)
		case ModeAddValue:
			e.handleAddValue(key)
		case ModeHelp:
			e.handleHelp(key)
		case ModeConfirmQuit:
			e.handleConfirmQuit(key)
		}
	}
}

func (e *Editor) handleNormal(key string) {
	switch key {
	case "\x11":
		if e.modified {
			e.mode = ModeConfirmQuit
		} else {
			e.quit()
		}
	case "\x13":
		if err := saveNBT(e.root, e.filepath, e.compression); err != nil {
			e.setMsg("保存失败: "+err.Error(), true)
		} else {
			e.modified = false
			e.setMsg("已保存 → "+e.filepath, false)
		}
	case "\x05":
		node := e.selected()
		if node == nil {
			break
		}
		if !isEditable(node) {
			e.setMsg("容器类型请用 ^N 添加子标签。", true)
			break
		}
		e.editBuf = editableValueStr(node)
		e.editPos = len([]rune(e.editBuf))
		e.mode = ModeEdit
	case "\x12":
		node := e.selected()
		if node == nil {
			break
		}
		if node.Parent == nil {
			e.setMsg("根节点不可重命名。", true)
			break
		}
		if node.Parent.Type == TagList {
			e.setMsg("列表元素不可重命名。", true)
			break
		}
		e.editBuf = node.Name
		e.editPos = len([]rune(e.editBuf))
		e.mode = ModeRename
	case "\x0e":
		node := e.selected()
		if node == nil {
			break
		}
		e.addParent = node
		e.addInlineBuf = ""
		e.addInlinePos = 0
		e.mode = ModeAddInline
	case "\x04":
		node := e.selected()
		if node == nil || node.Parent == nil {
			e.setMsg("根节点不可删除。", true)
			break
		}
		parent := node.Parent
		idx := childIndex(parent, node)
		if idx < 0 {
			e.setMsg("在父节点中找不到该节点。", true)
			break
		}
		removeChildAt(parent, idx)
		e.modified = true
		e.rebuild()
		if e.cursor >= len(e.flat) {
			e.cursor = imax(0, len(e.flat)-1)
		}
	case "\x08":
		e.helpPage = 0
		e.mode = ModeHelp
	case "?":
		e.helpPage = 0
		e.mode = ModeHelp

	case "\x1b[A":
		if e.cursor > 0 {
			e.cursor--
		}
	case "\x1b[B":
		if e.cursor < len(e.flat)-1 {
			e.cursor++
		}
	case "\x1b[H", "\x1b[1~":
		e.cursor = 0
	case "\x1b[F", "\x1b[4~":
		e.cursor = len(e.flat) - 1
	case "\x1b[C", "\r", "\n":
		if node := e.selected(); node != nil && isContainer(node) {
			node.Expanded = !node.Expanded
			e.rebuild()
		}
	case "\x1b[D":
		node := e.selected()
		if node == nil {
			break
		}
		if isContainer(node) && node.Expanded {
			node.Expanded = false
			e.rebuild()
		} else if node.Parent != nil {
			for i, item := range e.flat {
				if item.Node == node.Parent {
					e.cursor = i
					break
				}
			}
		}
	}
}

func (e *Editor) handleInlineEdit(key string, rename bool) {
	commit := func() {
		node := e.selected()
		if node == nil {
			e.mode = ModeNormal
			return
		}
		if rename {
			oldName := node.Name
			desc := fmt.Sprintf("%s:%s → %s:%s", tagNames[node.Type], oldName, tagNames[node.Type], e.editBuf)
			node.Name = e.editBuf
			e.modified = true
			e.setMsg("已重命名: "+desc, false)
		} else {
			val, err := parseValue(node.Type, e.editBuf)
			if err != nil {
				e.setMsg("值无效: "+err.Error(), true)
				return
			}
			oldStr := valueStr(node)
			desc := fmt.Sprintf("%s:%s:%s → %s:%s:%s", tagNames[node.Type], node.Name, oldStr, tagNames[node.Type], node.Name, e.editBuf)
			node.Value = val
			e.modified = true
			e.setMsg("已更新: "+desc, false)
		}
		e.mode = ModeNormal
	}
	switch key {
	case "\r", "\n":
		commit()
	case "\x1b":
		e.mode = ModeNormal
		e.setMsg("已取消。", false)
	case "\x05":
		commit()
	case "\x12":
		commit()
	case "\x7f", "\x08":
		e.editBuf, e.editPos = delBack(e.editBuf, e.editPos)
	case "\x1b[C":
		rr := []rune(e.editBuf)
		if e.editPos < len(rr) {
			e.editPos++
		}
	case "\x1b[D":
		if e.editPos > 0 {
			e.editPos--
		}
	case "\x1b[H":
		e.editPos = 0
	case "\x1b[F":
		e.editPos = len([]rune(e.editBuf))
	case "\x1b[3~":
		e.editBuf = delFwd(e.editBuf, e.editPos)
	default:
		if len(key) >= 1 {
			r, size := utf8.DecodeRuneInString(key)
			if size > 0 && r != utf8.RuneError && key[0] >= 0x20 {
				e.editBuf, e.editPos = insRune(e.editBuf, e.editPos, r)
			}
		}
	}
}

var typeAliases = map[string]byte{
	"byte": TagByte, "b": TagByte,
	"short": TagShort, "s": TagShort,
	"int": TagInt, "i": TagInt,
	"long": TagLong, "l": TagLong,
	"float": TagFloat, "f": TagFloat,
	"double": TagDouble, "d": TagDouble,
	"string": TagString, "str": TagString,
	"list":     TagList,
	"compound": TagCompound, "group": TagCompound, "comp": TagCompound,
	"byte[]": TagByteArray, "int[]": TagIntArray, "long[]": TagLongArray,
}

func (e *Editor) handleAddInline(key string) {
	switch key {
	case "\r", "\n", "\x0e":
		e.commitAddInline()
	case "\x1b":
		e.mode = ModeNormal
		e.setMsg("已取消。", false)
	case "\x7f", "\x08":
		e.addInlineBuf, e.addInlinePos = delBack(e.addInlineBuf, e.addInlinePos)
	case "\x1b[C":
		rr := []rune(e.addInlineBuf)
		if e.addInlinePos < len(rr) {
			e.addInlinePos++
		}
	case "\x1b[D":
		if e.addInlinePos > 0 {
			e.addInlinePos--
		}
	case "\x1b[H":
		e.addInlinePos = 0
	case "\x1b[F":
		e.addInlinePos = len([]rune(e.addInlineBuf))
	case "\x1b[3~":
		e.addInlineBuf = delFwd(e.addInlineBuf, e.addInlinePos)
	default:
		if len(key) >= 1 {
			r, size := utf8.DecodeRuneInString(key)
			if size > 0 && r != utf8.RuneError && key[0] >= 0x20 {
				e.addInlineBuf, e.addInlinePos = insRune(e.addInlineBuf, e.addInlinePos, r)
			}
		}
	}
}

func (e *Editor) commitAddInline() {
	input := strings.TrimSpace(e.addInlineBuf)
	if input == "" {
		e.setMsg("输入不能为空。", true)
		return
	}

	node := e.selected()

	var parent *NBTNode
	var insertAfter *NBTNode

	if node != nil && isContainer(node) {

		parent = node
		node.Expanded = true
	} else if node != nil && node.Parent != nil {

		parent = node.Parent
		insertAfter = node
	} else {
		e.setMsg("无法确定插入位置。", true)
		return
	}

	parts := strings.SplitN(input, ":", 3)
	typeStr := strings.ToLower(strings.TrimSpace(parts[0]))

	var tagType byte
	var name string
	var val interface{}

	if parent.Type == TagList {
		list := parent.Value.(*NBTList)
		tagType = list.ElemType

		if tagType == TagCompound {
			val = []*NBTNode{}
		} else if tagType == TagList {
			val = &NBTList{ElemType: TagString}
		} else {
			var err error
			val, err = parseValue(tagType, input)
			if err != nil {
				e.setMsg(tagNames[tagType]+" 值无效: "+err.Error(), true)
				return
			}
		}

	} else {

		switch typeStr {

		case "compound", "group", "comp":

			if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
				e.setMsg("复合标签格式: compound:名称", true)
				return
			}
			tagType = TagCompound
			name = strings.TrimSpace(parts[1])
			val = []*NBTNode{}

		case "list":

			if len(parts) < 3 {
				e.setMsg("列表格式: list:元素类型:名称  (例: list:int:Scores)", true)
				return
			}
			elemTypeStr := strings.ToLower(strings.TrimSpace(parts[1]))
			elemType, ok := typeAliases[elemTypeStr]
			if !ok {
				e.setMsg("未知的元素类型 \""+elemTypeStr+"\"。", true)
				return
			}
			tagType = TagList
			name = strings.TrimSpace(parts[2])
			if name == "" {
				e.setMsg("列表名称不能为空。", true)
				return
			}
			val = &NBTList{ElemType: elemType}

		default:

			var ok bool
			tagType, ok = typeAliases[typeStr]
			if !ok {
				e.setMsg("未知类型 \""+typeStr+"\"，可用: byte/short/int/long/float/double/string/compound/list/byte[]/int[]/long[]", true)
				return
			}
			isArray := tagType == TagByteArray || tagType == TagIntArray || tagType == TagLongArray

			if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
				e.setMsg("格式: 类型:名称:值  (例: int:Score:0)", true)
				return
			}
			if !isArray && len(parts) < 3 {
				e.setMsg("格式: 类型:名称:值  (例: int:Score:0)", true)
				return
			}
			name = strings.TrimSpace(parts[1])
			if name == "" {
				e.setMsg("名称不能为空。", true)
				return
			}
			valStr := ""
			if len(parts) == 3 {
				valStr = strings.TrimSpace(parts[2])
			}
			var err error
			val, err = parseValue(tagType, valStr)
			if err != nil {
				e.setMsg("值无效: "+err.Error(), true)
				return
			}
		}
	}

	newNode := &NBTNode{Type: tagType, Name: name, Value: val, Parent: parent}

	var insertIdx int
	switch parent.Type {
	case TagCompound:
		ch := parent.Value.([]*NBTNode)
		insertIdx = len(ch)
		if insertAfter != nil {
			for i, c := range ch {
				if c == insertAfter {
					insertIdx = i + 1
					break
				}
			}
		}
	case TagList:
		l := parent.Value.(*NBTList)
		insertIdx = len(l.Items)
		if insertAfter != nil {
			for i, c := range l.Items {
				if c == insertAfter {
					insertIdx = i + 1
					break
				}
			}
		}
	}

	insertChild(parent, newNode, insertIdx)

	valDesc := valueStr(newNode)
	if valDesc == "" {
		valDesc = "(空)"
	}
	desc := fmt.Sprintf("add %s:%s:%s", tagNames[newNode.Type], newNode.Name, valDesc)
	e.modified = true
	e.mode = ModeNormal
	e.rebuild()
	for i, item := range e.flat {
		if item.Node == newNode {
			e.cursor = i
			break
		}
	}
	e.setMsg("已添加: "+desc, false)
}

func (e *Editor) handleAddName(key string) {
	switch key {
	case "\r", "\n":
		if e.addName == "" {
			e.setMsg("名称不能为空。", true)
			return
		}
		e.mode = ModeAddType
	case "\x1b":
		e.mode = ModeNormal
		e.setMsg("已取消。", false)
	case "\x7f", "\x08":
		e.addName, e.addNamePos = delBack(e.addName, e.addNamePos)
	case "\x1b[C":
		rr := []rune(e.addName)
		if e.addNamePos < len(rr) {
			e.addNamePos++
		}
	case "\x1b[D":
		if e.addNamePos > 0 {
			e.addNamePos--
		}
	default:
		if len(key) >= 1 {
			r, size := utf8.DecodeRuneInString(key)
			if size > 0 && r != utf8.RuneError && key[0] >= 0x20 {
				e.addName, e.addNamePos = insRune(e.addName, e.addNamePos, r)
			}
		}
	}
}

func (e *Editor) handleAddType(key string) {
	idx := 0
	for i, id := range addTypeIDs {
		if id == e.addType {
			idx = i
			break
		}
	}
	switch key {
	case "\x1b[C":
		if idx < len(addTypeIDs)-1 {
			e.addType = addTypeIDs[idx+1]
		}
	case "\x1b[D":
		if idx > 0 {
			e.addType = addTypeIDs[idx-1]
		}
	case "\r", "\n":
		if e.addType == TagCompound || e.addType == TagList {
			e.finishAdd(nil)
		} else {
			e.mode = ModeAddValue
		}
	case "\x1b":
		e.mode = ModeNormal
		e.setMsg("已取消。", false)
	}
}

func (e *Editor) handleAddValue(key string) {
	switch key {
	case "\r", "\n":
		val, err := parseValue(e.addType, e.addValue)
		if err != nil {
			e.setMsg("值无效: "+err.Error(), true)
			return
		}
		e.finishAdd(val)
	case "\x1b":
		e.mode = ModeNormal
		e.setMsg("已取消。", false)
	case "\x7f", "\x08":
		e.addValue, e.addValPos = delBack(e.addValue, e.addValPos)
	case "\x1b[C":
		rr := []rune(e.addValue)
		if e.addValPos < len(rr) {
			e.addValPos++
		}
	case "\x1b[D":
		if e.addValPos > 0 {
			e.addValPos--
		}
	case "\x1b[3~":
		e.addValue = delFwd(e.addValue, e.addValPos)
	default:
		if len(key) >= 1 {
			r, size := utf8.DecodeRuneInString(key)
			if size > 0 && r != utf8.RuneError && key[0] >= 0x20 {
				e.addValue, e.addValPos = insRune(e.addValue, e.addValPos, r)
			}
		}
	}
}

func (e *Editor) finishAdd(val interface{}) {
	parent := e.addParent
	if parent == nil {
		e.mode = ModeNormal
		return
	}
	if e.addType == TagCompound && val == nil {
		val = []*NBTNode{}
	}
	if e.addType == TagList && val == nil {
		val = &NBTList{ElemType: TagString}
	}
	newNode := &NBTNode{Type: e.addType, Name: e.addName, Value: val, Parent: parent}
	var insertIdx int
	switch parent.Type {
	case TagCompound:
		ch := parent.Value.([]*NBTNode)
		insertIdx = len(ch)
	case TagList:
		l := parent.Value.(*NBTList)
		insertIdx = len(l.Items)
	}
	insertChild(parent, newNode, insertIdx)
	valDesc := valueStr(newNode)
	if valDesc == "" {
		valDesc = "(空)"
	}
	desc := fmt.Sprintf("add %s:%s:%s", tagNames[newNode.Type], newNode.Name, valDesc)
	e.modified = true
	e.mode = ModeNormal
	e.rebuild()
	for i, item := range e.flat {
		if item.Node == newNode {
			e.cursor = i
			break
		}
	}
	e.setMsg("已添加: "+desc, false)
}

func (e *Editor) handleConfirmQuit(key string) {
	switch key {
	case "\x13":
		if err := saveNBT(e.root, e.filepath, e.compression); err != nil {
			e.setMsg("保存失败: "+err.Error(), true)
			e.mode = ModeNormal
		} else {
			e.quit()
		}
	case "\x11":
		e.quit()
	default:
		e.mode = ModeNormal
		e.setMsg("已取消退出。", false)
	}
}

func (e *Editor) quit() {
	termRestore()
	fmt.Print(utils.TC.DisableMouse + utils.TC.LeaveAltScreen + utils.TC.ShowCursor)
	os.Exit(0)
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min8(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NBTEditor(path string) {
	root, comp, fmtType, err := loadNBT(path)
	if err != nil {
		var errEncrypted *ErrEncrypted
		var errCorrupt *ErrCorrupt
		switch {
		case errors.As(err, &errEncrypted):
			utils.LogError("%v", err)
			return
		case errors.As(err, &errCorrupt):
			utils.LogError("%v", err)
			return
		default:
			if os.IsNotExist(err) {

				ext := ""
				if idx := strings.LastIndexByte(path, '.'); idx >= 0 {
					ext = strings.ToLower(path[idx:])
				}
				if ext == ".mca" || ext == ".mcr" {
					utils.LogError("Region文件不存在，无法新建: %s", path)
					return
				}
				root = &NBTNode{Type: TagCompound, Name: "root", Value: []*NBTNode{}}
				comp = compGzip
				fmtType = fmtNBT
			} else {
				utils.LogError("加载NBT文件失败: %v", err)
				return
			}
		}
	}
	NewEditor(root, path, comp, fmtType).Run()
}
