package network

import (
	"encoding/binary"
	"io"

	"slg-game/protocol"
)

const (
	HeaderSize = 12 // Magic(2) + Version(1) + Flags(1) + MsgID(4) + DataLen(4)
)

// Packet represents a network packet
type Packet struct {
	MsgID uint32
	Data  []byte
}

// Encode serializes the packet with protobuf header
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))

	binary.BigEndian.PutUint16(buf[0:2], protocol.MagicNumber)
	buf[2] = protocol.ProtocolVersion
	buf[3] = 0 // Flags
	binary.BigEndian.PutUint32(buf[4:8], p.MsgID)
	binary.BigEndian.PutUint32(buf[8:12], uint32(len(p.Data)))

	copy(buf[12:], p.Data)
	return buf
}

// Decode reads a packet from reader
func Decode(reader io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	magic := binary.BigEndian.Uint16(header[0:2])
	if magic != protocol.MagicNumber {
		return nil, protocol.ErrInvalidMagic
	}

	version := header[2]
	if version != protocol.ProtocolVersion {
		return nil, protocol.ErrInvalidVersion
	}

	msgID := binary.BigEndian.Uint32(header[4:8])
	dataLen := binary.BigEndian.Uint32(header[8:12])

	if dataLen > protocol.MaxMessageSize {
		return nil, protocol.ErrMessageTooLarge
	}

	data := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	return &Packet{
		MsgID: msgID,
		Data:  data,
	}, nil
}
