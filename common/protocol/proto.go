package protocol

import (
	"bytes"
	"net"
	"encoding/binary"
	"log/slog"
	"encoding/hex"
)

var HEADER = []byte{0xF8, 0xE9, 0xDA, 0xCB}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Remark() {
	buf := b.Buffer.Bytes() // 获取未读数据
	b.Reset()               // 清空缓冲区
	b.Write(buf)            // 将未读数据写回
}

var buf = &buffer{Buffer: bytes.NewBuffer(nil)}

func Decoder(conn net.Conn, handle func(data []byte, conn net.Conn)) (err error) {
	in := make([]byte, 1024)
	for {
		// 收到数据写进buf
		var n int
		n, err = conn.Read(in)
		if err != nil {
			return
		}

		slog.Info("收到数据:", hex.EncodeToString(in[:n]))

		buf.Write(in[:n])
		clear(in)

		for {
			// 如果所有数据都不包含头，那所有数据都是无效的
			index := bytes.Index(buf.Bytes(), HEADER)
			if index == -1 {
				//slog.Error("没有数据头，数据不完整")
				buf.Reset()
				break
			}
			// 读完头之前的数据，丢弃
			buf.Next(index)
			buf.Remark()

			// buf中有头，但是长度不够，那数据不完整，等待下一个数据
			if buf.Len() < 8 {
				//slog.Error("数据长度不够")
				break
			}

			// 整个消息长度不够，那数据不完整，等待下一个数据
			msgLen := binary.BigEndian.Uint32(buf.Bytes()[4:8])
			if buf.Len()-8 < int(msgLen) {
				//slog.Error("消息长度小于msgLen")
				break
			}
			buf.Next(8)
			msg := buf.Next(int(msgLen))
			buf.Remark()
			slog.Info("decode完成，处理数据:", "data:", hex.EncodeToString(msg))
			handle(msg, conn)
		}
	}
}

func Encoder(msg []byte) (finalMsg []byte) {
	finalMsg = make([]byte, 0)
	finalMsg = append(finalMsg, HEADER...)
	finalMsg = binary.BigEndian.AppendUint32(finalMsg, uint32(len(msg)))
	finalMsg = append(finalMsg, msg...)
	return
}
