package protocol

import (
	"bytes"
	"net"
	"encoding/binary"
)

var HEADER = []byte{0xF8, 0xE9, 0xDA, 0xCB}

type buffer struct {
	*bytes.Buffer
}

func (b *buffer) Remark() {
	lastData := b.Buffer.Bytes() // 获取未读数据
	b.Reset()                    // 清空缓冲区
	b.Write(lastData)            // 将未读数据写回
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

		buf.Write(in[:n])
		clear(in)

		for {
			// 如果所有数据都不包含头，那所有数据都是无效的
			index := bytes.Index(buf.Bytes(), HEADER)
			if index == -1 {
				buf.Reset()
				break
			}

			// 读完头之前的数据，丢弃
			buf.Next(index)
			buf.Remark()

			// buf中有头，但是长度不够，那数据不完整，等待下一个数据
			if buf.Len() < 8 {
				break
			}

			// 整个消息长度不够，那数据不完整，等待下一个数据
			msgLen := binary.BigEndian.Uint32(buf.Bytes()[4:8])
			if buf.Len()-8 < int(msgLen) {
				break
			}
			buf.Next(8)
			msg := make([]byte, msgLen)
			copy(msg, buf.Next(int(msgLen)))
			buf.Remark()
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
