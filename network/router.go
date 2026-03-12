package network

import (
	"slg-game/session"
)

// MessageHandler 消息处理器接口
type MessageHandler interface {
	Handle(sess session.Session, packet *Packet) *Packet
}

// Router 统一消息路由器
type Router struct {
	handlers map[uint32]MessageHandler
}

// NewRouter 创建消息路由器
func NewRouter() *Router {
	return &Router{
		handlers: make(map[uint32]MessageHandler),
	}
}

// RegisterHandler 注册消息处理器
func (r *Router) RegisterHandler(msgID uint32, handler MessageHandler) {
	r.handlers[msgID] = handler
}

// RegisterRangeHandler 注册消息范围处理器
func (r *Router) RegisterRangeHandler(start, end uint32, handler MessageHandler) {
	for msgID := start; msgID <= end; msgID++ {
		r.handlers[msgID] = handler
	}
}

// Route 根据 MsgID 路由消息到对应的处理器
func (r *Router) Route(sess session.Session, packet *Packet) *Packet {
	handler, exists := r.handlers[packet.MsgID]
	if !exists {
		return nil
	}
	return handler.Handle(sess, packet)
}
