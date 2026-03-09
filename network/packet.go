package network

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	HeaderSize = 8 // msg_id(4) + msg_len(4)
)

// Packet represents a complete network packet with header and payload
type Packet struct {
	MsgID uint32
	Data  []byte
}

// Encode serializes the packet into bytes
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))
	binary.LittleEndian.PutUint32(buf[0:4], p.MsgID)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(p.Data)))
	copy(buf[8:], p.Data)
	return buf
}

// Decode reads a packet from the reader
func Decode(reader io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	msgID := binary.LittleEndian.Uint32(header[0:4])
	msgLen := binary.LittleEndian.Uint32(header[4:8])

	if msgLen > 1024*1024 { // 1MB limit
		return nil, ErrPacketTooLarge
	}

	data := make([]byte, msgLen)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	return &Packet{
		MsgID: msgID,
		Data:  data,
	}, nil
}

var (
	ErrPacketTooLarge = &PacketError{"packet too large"}
)

type PacketError struct {
	Message string
}

func (e *PacketError) Error() string {
	return e.Message
}