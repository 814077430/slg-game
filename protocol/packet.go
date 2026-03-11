package protocol

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"sync"
	"time"
)

const (
	// 协议魔术数字
	MagicNumber = 0x534C // "SL" = SLG

	// 协议版本
	ProtocolVersion = 1

	// 头部大小：Magic(2) + Version(1) + Flags(1) + SeqID(4) + MsgID(4) + TokenLen(1) = 13
	HeaderSize = 13

	// Token 大小
	TokenSize = 16

	// 最大消息大小
	MaxMessageSize = 1024 * 1024 // 1MB
)

// Flags 标志位
type Flags uint8

const (
	FlagCompress Flags = 1 << iota // 压缩
	FlagEncrypt                    // 加密
	FlagRequest                    // 请求
	FlagResponse                   // 响应
)

// Packet 网络包
type Packet struct {
	Magic   uint16
	Version uint8
	Flags   Flags
	SeqID   uint32    // 序列号（请求 ID）
	MsgID   uint32    // 消息类型
	Token   []byte    // 认证 Token
	Data    json.RawMessage // 数据体
}

// Encode 编码
func (p *Packet) Encode() []byte {
	tokenLen := len(p.Token)
	if tokenLen > 255 {
		tokenLen = 255
	}

	buf := make([]byte, HeaderSize+tokenLen+len(p.Data))

	binary.BigEndian.PutUint16(buf[0:2], p.Magic)
	buf[2] = p.Version
	buf[3] = uint8(p.Flags)
	binary.BigEndian.PutUint32(buf[4:8], p.SeqID)
	binary.BigEndian.PutUint32(buf[8:12], p.MsgID)
	buf[12] = uint8(tokenLen)

	copy(buf[13:13+tokenLen], p.Token[:tokenLen])
	copy(buf[13+tokenLen:], p.Data)

	return buf
}

// Decode 解码
func Decode(reader io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	magic := binary.BigEndian.Uint16(header[0:2])
	if magic != MagicNumber {
		return nil, ErrInvalidMagic
	}

	version := header[2]
	if version != ProtocolVersion {
		return nil, ErrInvalidVersion
	}

	flags := Flags(header[3])
	seqID := binary.BigEndian.Uint32(header[4:8])
	msgID := binary.BigEndian.Uint32(header[8:12])
	tokenLen := int(header[12])

	if tokenLen > TokenSize {
		return nil, ErrInvalidToken
	}

	token := make([]byte, tokenLen)
	if tokenLen > 0 {
		if _, err := io.ReadFull(reader, token); err != nil {
			return nil, err
		}
	}

	// 读取数据长度
	var dataLenBuf [4]byte
	if _, err := io.ReadFull(reader, dataLenBuf[:]); err != nil {
		return nil, err
	}
	dataLen := binary.BigEndian.Uint32(dataLenBuf[:])

	if dataLen > MaxMessageSize {
		return nil, ErrMessageTooLarge
	}

	data := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	return &Packet{
		Magic:   magic,
		Version: version,
		Flags:   flags,
		SeqID:   seqID,
		MsgID:   msgID,
		Token:   token,
		Data:    data,
	}, nil
}

// Marshal 序列化数据
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 反序列化数据
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Request 请求
type Request struct {
	SeqID uint32          `json:"-"`
	MsgID uint32          `json:"-"`
	Token string          `json:"token,omitempty"`
	Data  json.RawMessage `json:"data"`
}

// Response 响应
type Response struct {
	SeqID   uint32      `json:"-"`
	MsgID   uint32      `json:"-"`
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewRequest 创建请求
func NewRequest(msgID uint32, data interface{}) (*Request, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Request{
		MsgID: msgID,
		Data:  jsonData,
	}, nil
}

// NewResponse 创建响应
func NewResponse(seqID, msgID uint32, success bool, code int, message string, data interface{}) *Response {
	return &Response{
		SeqID:   seqID,
		MsgID:   msgID,
		Success: success,
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// 错误
var (
	ErrInvalidMagic   = &ProtocolError{"invalid magic number"}
	ErrInvalidVersion = &ProtocolError{"invalid protocol version"}
	ErrInvalidToken   = &ProtocolError{"invalid token"}
	ErrMessageTooLarge = &ProtocolError{"message too large"}
)

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}

// SequenceGenerator 序列号生成器
type SequenceGenerator struct {
	current uint32
	mutex   sync.Mutex
}

// NewSequenceGenerator 创建序列号生成器
func NewSequenceGenerator() *SequenceGenerator {
	return &SequenceGenerator{
		current: uint32(time.Now().UnixNano()),
	}
}

// Next 获取下一个序列号
func (s *SequenceGenerator) Next() uint32 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.current++
	return s.current
}
