package main

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"
	"time"
)

const (
	HeaderSize = 8 // msg_id(4) + msg_len(4)
)

// Encode serializes the packet
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))
	binary.LittleEndian.PutUint32(buf[0:4], p.MsgID)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(p.Data)))
	copy(buf[8:], p.Data)
	return buf
}

// Decode reads a packet from reader
func Decode(reader io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}

	msgID := binary.LittleEndian.Uint32(header[0:4])
	msgLen := binary.LittleEndian.Uint32(header[4:8])

	if msgLen > 1024*1024 {
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

// MarshalJSON wraps JSON marshal
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON wraps JSON unmarshal
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// Errors
var (
	ErrPacketTooLarge = &PacketError{"packet too large"}
	ErrConnectionClosed = &PacketError{"connection closed"}
)

type PacketError struct {
	Message string
}

func (e *PacketError) Error() string {
	return e.Message
}

// Client TCP 客户端
type Client struct {
	conn      net.Conn
	reader    io.Reader
	writer    io.Writer
	sendChan  chan *Packet
	recvChan  chan *Packet
	closeChan chan struct{}
	isClosed  bool
	mutex     sync.Mutex
	serverAddr string
}

// NewClient creates a new TCP client
func NewClient(serverAddr string) *Client {
	return &Client{
		serverAddr: serverAddr,
		sendChan:   make(chan *Packet, 100),
		recvChan:   make(chan *Packet, 100),
		closeChan:  make(chan struct{}),
	}
}

// Connect connects to the server
func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.serverAddr, 10*time.Second)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = conn
	c.writer = conn

	// Start send and receive loops
	go c.sendLoop()
	go c.recvLoop()

	return nil
}

// sendLoop sends packets to server
func (c *Client) sendLoop() {
	for {
		select {
		case packet := <-c.sendChan:
			if err := c.writePacket(packet); err != nil {
				c.Close()
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

// recvLoop receives packets from server
func (c *Client) recvLoop() {
	for {
		packet, err := Decode(c.reader)
		if err != nil {
			c.Close()
			return
		}

		select {
		case c.recvChan <- packet:
		case <-c.closeChan:
			return
		}
	}
}

// writePacket writes a packet to the connection
func (c *Client) writePacket(packet *Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isClosed {
		return ErrConnectionClosed
	}

	data := packet.Encode()
	_, err := c.writer.Write(data)
	return err
}

// Send sends a packet to the server
func (c *Client) Send(msgID uint32, data interface{}) error {
	jsonData, err := MarshalJSON(data)
	if err != nil {
		return err
	}

	packet := &Packet{
		MsgID: msgID,
		Data:  jsonData,
	}

	select {
	case c.sendChan <- packet:
		return nil
	default:
		return ErrSendQueueFull
	}
}

// Recv receives a packet from the server (blocking)
func (c *Client) Recv() (*Packet, error) {
	select {
	case packet := <-c.recvChan:
		return packet, nil
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	}
}

// RecvWithTimeout receives a packet with timeout
func (c *Client) RecvWithTimeout(timeout time.Duration) (*Packet, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case packet := <-c.recvChan:
		return packet, nil
	case <-timer.C:
		return nil, ErrTimeout
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	}
}

// Close closes the connection
func (c *Client) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	close(c.closeChan)
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return !c.isClosed && c.conn != nil
}

// Errors
var (
	ErrSendQueueFull = &ClientError{"send queue full"}
	ErrTimeout = &ClientError{"timeout"}
)

type ClientError struct {
	Message string
}

func (e *ClientError) Error() string {
	return e.Message
}
