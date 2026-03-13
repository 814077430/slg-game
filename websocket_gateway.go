// WebSocket 网关服务
// 将 WebSocket 连接转发到 TCP 游戏服务器

package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	TCP_SERVER_ADDR = "localhost:8080"
	WS_PORT         = ":8081"
	
	// 协议常量
	HeaderSize     = 12
	MagicNumber    = 0x534C
	ProtocolVer    = 1
)

// WebSocket 到 TCP 的桥接连接
type BridgeConn struct {
	wsConn   *websocket.Conn
	tcpConn  net.Conn
	quitChan chan struct{}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源（开发环境）
	},
}

func main() {
	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║     SLG Game WebSocket Gateway                         ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Printf("WebSocket 端口: %s", WS_PORT)
	log.Printf("TCP 服务器地址: %s", TCP_SERVER_ADDR)
	log.Println()

	// WebSocket 处理
	http.HandleFunc("/ws", handleWebSocket)
	
	log.Println("WebSocket 网关启动成功!")
	log.Printf("客户端连接地址: ws://localhost%s/ws", WS_PORT)
	log.Fatal(http.ListenAndServe(WS_PORT, nil))
}

// 处理 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 到 WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	defer wsConn.Close()

	log.Printf("WebSocket 客户端连接: %s", wsConn.RemoteAddr())

	// 连接到 TCP 游戏服务器
	tcpConn, err := net.Dial("tcp", TCP_SERVER_ADDR)
	if err != nil {
		log.Printf("连接 TCP 服务器失败: %v", err)
		wsConn.WriteJSON(map[string]interface{}{
			"type": "error",
			"msg":  "无法连接到游戏服务器",
		})
		return
	}
	defer tcpConn.Close()

	log.Printf("已连接到 TCP 服务器: %s", TCP_SERVER_ADDR)

	bridge := &BridgeConn{
		wsConn:   wsConn,
		tcpConn:  tcpConn,
		quitChan: make(chan struct{}),
	}

	// 启动双向转发
	go bridge.wsToTCP()
	go bridge.tcpToWS()

	// 等待连接关闭
	<-bridge.quitChan
	log.Printf("连接关闭: %s", wsConn.RemoteAddr())
}

// WebSocket → TCP
func (b *BridgeConn) wsToTCP() {
	defer func() {
		close(b.quitChan)
	}()

	for {
		// 读取 WebSocket 消息
		_, message, err := b.wsConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 读取错误: %v", err)
			}
			return
		}

		// 解析 JSON 消息
		var wsMsg struct {
			Magic   uint16      `json:"magic"`
			Version uint8       `json:"version"`
			MsgID   uint32      `json:"msgID"`
			Data    interface{} `json:"data"`
		}
		
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("解析 WebSocket 消息失败: %v", err)
			continue
		}

		// 转换为 TCP 二进制协议
		tcpData, err := json.Marshal(wsMsg.Data)
		if err != nil {
			log.Printf("序列化数据失败: %v", err)
			continue
		}

		// 构建 TCP 数据包
		packet := make([]byte, HeaderSize+len(tcpData))
		binary.BigEndian.PutUint16(packet[0:2], MagicNumber)
		packet[2] = ProtocolVer
		packet[3] = 0 // Flags
		binary.BigEndian.PutUint32(packet[4:8], wsMsg.MsgID)
		binary.BigEndian.PutUint32(packet[8:12], uint32(len(tcpData)))
		copy(packet[12:], tcpData)

		// 发送到 TCP 服务器
		if _, err := b.tcpConn.Write(packet); err != nil {
			log.Printf("写入 TCP 失败: %v", err)
			return
		}
	}
}

// TCP → WebSocket
func (b *BridgeConn) tcpToWS() {
	defer func() {
		close(b.quitChan)
	}()

	header := make([]byte, HeaderSize)
	
	for {
		// 读取 TCP 包头
		if _, err := b.tcpConn.Read(header); err != nil {
			log.Printf("读取 TCP 包头失败: %v", err)
			return
		}

		magic := binary.BigEndian.Uint16(header[0:2])
		if magic != MagicNumber {
			log.Printf("无效的 magic number: %x", magic)
			continue
		}

		msgID := binary.BigEndian.Uint32(header[4:8])
		dataLen := binary.BigEndian.Uint32(header[8:12])

		// 读取数据体
		data := make([]byte, dataLen)
		if _, err := b.tcpConn.Read(data); err != nil {
			log.Printf("读取 TCP 数据失败: %v", err)
			return
		}

		// 解析 JSON 数据
		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err != nil {
			log.Printf("解析 JSON 数据失败: %v", err)
			jsonData = string(data)
		}

		// 构建 WebSocket 消息
		wsMsg := map[string]interface{}{
			"magic":   MagicNumber,
			"version": ProtocolVer,
			"msgID":   msgID,
			"data":    jsonData,
		}

		// 发送到 WebSocket 客户端
		if err := b.wsConn.WriteJSON(wsMsg); err != nil {
			log.Printf("写入 WebSocket 失败: %v", err)
			return
		}
	}
}
