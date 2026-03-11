package protocol

import (
	"encoding/binary"
	"io"

	"google.golang.org/protobuf/proto"
)

const (
	// 协议魔术数字 "SL" = SLG
	MagicNumber = 0x534C

	// 协议版本
	ProtocolVersion = 1

	// 头部大小：Magic(2) + Version(1) + Flags(1) + MsgID(4) + DataLen(4) = 12
	HeaderSize = 12

	// 最大消息大小
	MaxMessageSize = 1024 * 1024 // 1MB
)

// Flags 标志位
type Flags uint8

const (
	FlagCompress Flags = 1 << iota
	FlagEncrypt
)

// Packet 网络包
type Packet struct {
	Magic   uint16
	Version uint8
	Flags   Flags
	MsgID   uint32
	Data    []byte
}

// Encode 编码
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))

	binary.BigEndian.PutUint16(buf[0:2], p.Magic)
	buf[2] = p.Version
	buf[3] = uint8(p.Flags)
	binary.BigEndian.PutUint32(buf[4:8], p.MsgID)
	binary.BigEndian.PutUint32(buf[8:12], uint32(len(p.Data)))

	copy(buf[12:], p.Data)
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
	msgID := binary.BigEndian.Uint32(header[4:8])
	dataLen := binary.BigEndian.Uint32(header[8:12])

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
		MsgID:   msgID,
		Data:    data,
	}, nil
}

// Marshal 序列化 protobuf 消息
func Marshal(msg proto.Message) ([]byte, error) {
	return proto.Marshal(msg)
}

// Unmarshal 反序列化 protobuf 消息
func Unmarshal(data []byte, msg proto.Message) error {
	return proto.Unmarshal(data, msg)
}

// 错误
var (
	ErrInvalidMagic   = &ProtocolError{"invalid magic number"}
	ErrInvalidVersion = &ProtocolError{"invalid protocol version"}
	ErrMessageTooLarge = &ProtocolError{"message too large"}
)

type ProtocolError struct {
	Message string
}

func (e *ProtocolError) Error() string {
	return e.Message
}
