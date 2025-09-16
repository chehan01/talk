package main

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net"
	"talk/common/consts"
	"talk/common/model"
	"talk/common/protocol"
	"time"
)

var userConn = make(map[string]net.Conn)
var connUser = make(map[net.Conn]string)

// cache 键是消息接收方，值是消息
var cache = make(map[string][][]byte)

func main() {
	listener, err := net.Listen("tcp", ":82")
	if err != nil {
		slog.Error("listen error", "error", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("accept error", "error", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	slog.Info("new connection", "remote address", conn.RemoteAddr())
	defer func(conn net.Conn) {
		slog.Info("connection closed", "remote address", conn.RemoteAddr())
		_ = conn.Close()
	}(conn)

	err := protocol.Decoder(conn, handleMsg)
	if err != nil {
		slog.Error("客户端断开连接，清除连接映射", "error", err)
		delete(userConn, connUser[conn])
		delete(connUser, conn)
		return
	}
}

func handleMsg(msgBytes []byte, conn net.Conn) {
	msg := model.Msg{}
	err := json.Unmarshal(msgBytes, &msg)
	if err != nil {
		slog.Error("json 反序列化外层消息错误:", err)
		return
	}
	switch msg.MsgType {
	case consts.LoginMsgType:
		loginSt := model.Login{}
		err = json.Unmarshal(msg.Data, &loginSt)
		if err != nil {
			slog.Error("json 反序列化登录消息错误:", err)
			return
		}
		connUser[conn] = loginSt.MyName
		userConn[loginSt.MyName] = conn
		slog.Info("用户登录", "username", loginSt.MyName)

		// 方法2：使用索引遍历
		for len(cache[loginSt.MyName]) > 0 {
			cacheMsg := cache[loginSt.MyName][0]
			// 把缓存的消息发送给用户
			slog.Info("要发给用户", "username", loginSt.MyName, "的消息:", hex.EncodeToString(cacheMsg))
			_, _ = conn.Write(protocol.Encoder(cacheMsg))
			cache[loginSt.MyName] = cache[loginSt.MyName][1:]
		}
	case consts.ChatMsgType:
		chatSt := model.Chat{}
		err = json.Unmarshal(msg.Data, &chatSt)
		if err != nil {
			slog.Error("json 反序列化消息详情错误:", "error", err)
			return
		}

		var to string
		if chatSt.MyName == "晗" {
			to = "勋"
		} else {
			to = "晗"
		}

		toConn, ok := userConn[to]
		if !ok {
			slog.Error("用户不在线:", "username", to)
			sendMsg("SYSTEM", time.Now().Format("01-02 15:04"), "对方不在线", conn)
			cache[to] = append(cache[to], msg.Data)
			slog.Info("缓存消息", "username", to, "message", hex.EncodeToString(msg.Data))
			return
		}

		sendMsg(chatSt.MyName, chatSt.SendTime, chatSt.Data, toConn)
	default:
	}
}

func sendMsg(sender, sendTime, data string, conn net.Conn) {
	chatSt := model.Chat{
		Data:     data,
		SendTime: sendTime,
		MyName:   sender,
	}

	msg, err := json.Marshal(chatSt)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}

	finalMsg := protocol.Encoder(msg)

	slog.Info("发送消息", "username", sender, "message", hex.EncodeToString(finalMsg))

	_, err = conn.Write(finalMsg)
	if err != nil {
		slog.Error("write error", "error", err)
		return
	}
}
