package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
	pb "slg-game/protocol"
)

const (
	HeaderSize = 12 // Magic(2) + Version(1) + Flags(1) + MsgID(4) + DataLen(4)
)

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

// Packet 网络包
type Packet struct {
	MsgID uint32
	Data  []byte
}

// Encode 编码
func (p *Packet) Encode() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))

	binary.BigEndian.PutUint16(buf[0:2], 0x534C) // "SL"
	buf[2] = 1                                    // Version
	buf[3] = 0                                    // Flags
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
	if magic != 0x534C {
		return nil, fmt.Errorf("invalid magic number")
	}

	msgID := binary.BigEndian.Uint32(header[4:8])
	dataLen := binary.BigEndian.Uint32(header[8:12])

	data := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	return &Packet{
		MsgID: msgID,
		Data:  data,
	}, nil
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
		return fmt.Errorf("connection closed")
	}

	data := packet.Encode()
	_, err := c.writer.Write(data)
	return err
}

// Send sends a protobuf message
func (c *Client) Send(msgID uint32, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	packet := &Packet{
		MsgID: msgID,
		Data:  data,
	}

	select {
	case c.sendChan <- packet:
		return nil
	default:
		return fmt.Errorf("send queue full")
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
		return nil, fmt.Errorf("timeout")
	case <-c.closeChan:
		return nil, fmt.Errorf("connection closed")
	}
}

// RecvProto receives and unmarshals a protobuf message
func (c *Client) RecvProto(msg proto.Message, timeout time.Duration) error {
	packet, err := c.RecvWithTimeout(timeout)
	if err != nil {
		return err
	}
	return proto.Unmarshal(packet.Data, msg)
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

// TestClient is a test client for the game server
type TestClient struct {
	client     *Client
	playerId   uint64
	username   string
	isLoggedIn bool
}

// NewTestClient creates a new test client
func NewTestClient(serverAddr string) *TestClient {
	return &TestClient{
		client: NewClient(serverAddr),
	}
}

// Connect connects to the server
func (tc *TestClient) Connect() error {
	return tc.client.Connect()
}

// Register registers a new account
func (tc *TestClient) Register(username, password, email string) (*pb.S2C_RegisterResponse, error) {
	req := &pb.C2S_RegisterRequest{
		Username: username,
		Password: password,
		Email:    email,
	}

	if err := tc.client.Send(1002, req); err != nil {
		return nil, err
	}

	resp := &pb.S2C_RegisterResponse{}
	if err := tc.client.RecvProto(resp, 10*time.Second); err != nil {
		return nil, err
	}

	if resp.Success {
		tc.playerId = resp.PlayerId
		tc.username = username
	}

	return resp, nil
}

// Login logs in to the server
func (tc *TestClient) Login(username, password string) (*pb.S2C_LoginResponse, error) {
	req := &pb.C2S_LoginRequest{
		Username: username,
		Password: password,
	}

	if err := tc.client.Send(1001, req); err != nil {
		return nil, err
	}

	resp := &pb.S2C_LoginResponse{}
	if err := tc.client.RecvProto(resp, 10*time.Second); err != nil {
		return nil, err
	}

	if resp.Success {
		tc.playerId = resp.PlayerId
		tc.username = username
		tc.isLoggedIn = true
	}

	return resp, nil
}

// Move sends a move request
func (tc *TestClient) Move(x, y int32) (*pb.S2C_MoveResponse, error) {
	if !tc.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	req := &pb.C2S_MoveRequest{X: x, Y: y}

	if err := tc.client.Send(1003, req); err != nil {
		return nil, err
	}

	resp := &pb.S2C_MoveResponse{}
	if err := tc.client.RecvProto(resp, 10*time.Second); err != nil {
		return nil, err
	}

	return resp, nil
}

// Build sends a build request
func (tc *TestClient) Build(buildingType string, x, y int32) (*pb.S2C_BuildResponse, error) {
	if !tc.isLoggedIn {
		return nil, fmt.Errorf("not logged in")
	}

	req := &pb.C2S_BuildRequest{
		BuildingType: buildingType,
		X:            x,
		Y:            y,
	}

	if err := tc.client.Send(1004, req); err != nil {
		return nil, err
	}

	resp := &pb.S2C_BuildResponse{}
	if err := tc.client.RecvProto(resp, 10*time.Second); err != nil {
		return nil, err
	}

	return resp, nil
}

// Close closes the connection
func (tc *TestClient) Close() {
	tc.client.Close()
}
