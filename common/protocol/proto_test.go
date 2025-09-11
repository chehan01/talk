package protocol

import (
	"testing"
	"talk/common/log"
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

func TestDecoder(t *testing.T) {

	data, _ := hex.DecodeString("f8e9dacb000000347b224d794e616d65223a22e58b8b222c2253656e6454696d65223a2230392d31312031343a3537222c2244617461223a2232227df8e9dacb000000347b224d794e616d65223a22e58b8b222c2253656e6454696d65223a2230392d31312031343a3537222c2244617461223a2233227d")

	buf.Write(data)

	for {
		log.Info("buf中的数据:", "data", hex.EncodeToString(buf.Bytes()))
		// 如果所有数据都不包含头，那所有数据都是无效的
		index := bytes.Index(buf.Bytes(), HEADER)
		if index == -1 {
			log.Error("没有数据头，数据不完整")
			buf.Reset()
			break
		}
		// 读完头之前的数据，丢弃
		log.Info("头位置:", index)

		buf.Next(index)
		buf.Remark()

		// buf中有头，但是长度不够，那数据不完整，等待下一个数据
		if buf.Len() < 8 {
			//log.Error("数据长度不够")
			break
		}

		// 整个消息长度不够，那数据不完整，等待下一个数据
		msgLen := binary.BigEndian.Uint32(buf.Bytes()[4:8])
		if buf.Len()-8 < int(msgLen) {
			//log.Error("消息长度小于msgLen")
			break
		}
		buf.Next(8)
		msg := buf.Next(int(msgLen))
		log.Info("decode完成，处理数据:", "data:", string(msg))
		buf.Remark()
		log.Info("buf剩余的数据:", "data", hex.EncodeToString(buf.Bytes()))
	}
}
