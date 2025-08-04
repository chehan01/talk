package model

type Msg struct {
	MsgType int
	Data    []byte
}

type Login struct {
	MyName string
}

type Chat struct {
	MyName   string
	SendTime string
	Data     string
}
