package network

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type Connection struct {
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	sendChan  chan *Packet
	closeChan chan struct{}
	isClosed  bool
	mutex     sync.Mutex
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		conn:      conn,
		reader:    bufio.NewReader(conn),
		writer:    bufio.NewWriter(conn),
		sendChan:  make(chan *Packet, 100),
		closeChan: make(chan struct{}),
	}
}

func (c *Connection) ReadPacket() (*Packet, error) {
	return Decode(c.reader)
}

func (c *Connection) SendPacket(packet *Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isClosed {
		return ErrConnectionClosed
	}

	select {
	case c.sendChan <- packet:
		return nil
	default:
		return ErrSendQueueFull
	}
}

func (c *Connection) Start() {
	go c.sendLoop()
}

func (c *Connection) sendLoop() {
	ticker := time.NewTicker(time.Millisecond * 10)
	defer ticker.Stop()

	for {
		select {
		case packet := <-c.sendChan:
			if err := c.writePacket(packet); err != nil {
				c.Close()
				return
			}
		case <-ticker.C:
			// Flush writer periodically
			c.writer.Flush()
		case <-c.closeChan:
			return
		}
	}
}

func (c *Connection) writePacket(packet *Packet) error {
	data := packet.Encode()
	_, err := c.writer.Write(data)
	return err
}

func (c *Connection) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isClosed {
		return
	}

	c.isClosed = true
	close(c.closeChan)
	c.conn.Close()
}

var (
	ErrConnectionClosed = &ConnectionError{"connection closed"}
	ErrSendQueueFull    = &ConnectionError{"send queue full"}
)

type ConnectionError struct {
	Message string
}

func (e *ConnectionError) Error() string {
	return e.Message
}