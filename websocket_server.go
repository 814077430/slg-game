// WebSocket 游戏服务器
// 原生支持 WebSocket 连接

package main

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"slg-game/config"
	"slg-game/database"
	"slg-game/game/core"
)

const (
	WS_PORT = ":8080"
	
	HeaderSize     = 12
	MagicNumber    = 0x534C
	ProtocolVer    = 1
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

type WebSocketServer struct {
	gameServer *core.GameServer
}

func NewWebSocketServer(db database.DB, cfg *config.Config) *WebSocketServer {
	return &WebSocketServer{
		gameServer: core.NewGameServer(db, cfg),
	}
}

func (wss *WebSocketServer) Start() {
	log.Println("╔════════════════════════════════════════════════════════╗")
	log.Println("║     SLG Game WebSocket Server                          ║")
	log.Println("╚════════════════════════════════════════════════════════╝")
	log.Printf("WebSocket 端口: %s", WS_PORT)
	log.Println()

	// WebSocket 处理
	http.HandleFunc("/ws", wss.handleWebSocket)
	
	log.Println("WebSocket 服务器启动成功!")
	log.Printf("客户端连接地址: ws://localhost%s/ws", WS_PORT)
	log.Fatal(http.ListenAndServe(WS_PORT, nil))
}

func (wss *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级 HTTP 到 WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket 升级失败: %v", err)
		return
	}
	defer wsConn.Close()

	log.Printf("WebSocket 客户端连接: %s", wsConn.RemoteAddr())

	// 创建 WebSocket 连接包装器
	wsWrapper := NewWebSocketWrapper(wsConn)
	
	// 使用现有的游戏服务器处理连接
	wss.gameServer.HandleWebSocketClient(wsWrapper)
	
	log.Printf("WebSocket 客户端断开: %s", wsConn.RemoteAddr())
}

// WebSocket 连接包装器，实现 net.Conn 接口
type WebSocketWrapper struct {
	conn     *websocket.Conn
	readBuf  []byte
	writeBuf []byte
}

func NewWebSocketWrapper(conn *websocket.Conn) *WebSocketWrapper {
	return &WebSocketWrapper{
		conn: conn,
	}
}

func (w *WebSocketWrapper) Read(b []byte) (n int, err error) {
	// 读取 WebSocket 消息
	_, message, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	// 解析 JSON 消息
	var wsMsg struct {
		Magic   uint16      `json:"magic"`
		Version uint8       `json:"version"`
		MsgID   uint32      `json:"msgID"`
		Data    interface{} `json:"data"`
	}
	
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		return 0, err
	}

	// 转换为二进制协议
	data, _ := json.Marshal(wsMsg.Data)
	packet := make([]byte, HeaderSize+len(data))
	binary.BigEndian.PutUint16(packet[0:2], MagicNumber)
	packet[2] = ProtocolVer
	packet[3] = 0
	binary.BigEndian.PutUint32(packet[4:8], wsMsg.MsgID)
	binary.BigEndian.PutUint32(packet[8:12], uint32(len(data)))
	copy(packet[12:], data)

	n = copy(b, packet)
	return n, nil
}

func (w *WebSocketWrapper) Write(b []byte) (n int, err error) {
	if len(b) < HeaderSize {
		return 0, nil
	}

	magic := binary.BigEndian.Uint16(b[0:2])
	if magic != MagicNumber {
		return len(b), nil
	}

	msgID := binary.BigEndian.Uint32(b[4:8])
	dataLen := binary.BigEndian.Uint32(b[8:12])

	var data interface{}
	if dataLen > 0 && len(b) >= int(HeaderSize+dataLen) {
		if err := json.Unmarshal(b[12:12+dataLen], &data); err != nil {
			data = string(b[12 : 12+dataLen])
		}
	}

	wsMsg := map[string]interface{}{
		"magic":   MagicNumber,
		"version": ProtocolVer,
		"msgID":   msgID,
		"data":    data,
	}

	if err := w.conn.WriteJSON(wsMsg); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (w *WebSocketWrapper) Close() error {
	return w.conn.Close()
}

func (w *WebSocketWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WebSocketWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}
