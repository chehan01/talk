package main

import (
	"encoding/json"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
	"log/slog"
	"net"
	"talk/common/consts"
	"talk/common/model"
	"time"
	"github.com/gdamore/tcell/v2"
	"talk/common/protocol"
	"os"
)

var conn *net.TCPConn
var myName string

var app *tview.Application
var messagesView *tview.TextView
var msgViewTable *tview.Table

// 添加全局行计数器
var messageRow = 0
var textArea *tview.TextArea
var emojiButton *tview.Button
var emojiTable *tview.Table
var emojiVisible = false

var emojis = [][]string{
	{"😊", "😁", "😂", "😀", "😄", "😉", "😋", "😎", "😍", "😘", "🥰", "🥲", "😚", "🙂",
		"🤗", "🤔", "🤨", "😐", "😑", "🤡", "🤥", "🙂", "🙂", "🤫", "🤭", "🫣", "🧐", "🤓", "🥳"},
	{"🙄", "😏", "😣", "😥", "🤐", "😯", "😫", "🥱", "😴", "😌", "🤤", "😒", "😓", "😔",
		"😕", "🫤", "🙃", "🫠", "😲", "🙁", "😖", "😞", "😟", "😤", "😢", "🥹", "😺", "💖", "💔"},
	{"😭", "😦", "😧", "😨", "😩", "😬", "😮‍💨", "😰", "😱", "😳", "🤪", "😵", "😵‍💫", "🥴",
		"😠", "😡", "🤬", "😷", "🤒", "🤕", "🤮", "🤧", "🥸", "😇", "👻", "💩", "🐰", "🐻", "🐽"},
}

type emojiData struct {
	tview.TableContentReadOnly
}

func (e emojiData) GetCell(row, column int) *tview.TableCell {
	// 检查行和列索引是否有效
	if row < 0 || row >= e.GetRowCount() || column < 0 || column >= e.GetColumnCount() {
		return tview.NewTableCell("") // 返回空单元格
	}
	return tview.NewTableCell(emojis[row][column])
}

func (e emojiData) GetRowCount() int {
	return len(emojis)
}

func (e emojiData) GetColumnCount() int {
	return len(emojis[0])
}

func main() {
	beeep.AppName = "Talk"

	// 创建或打开日志文件
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// 如果无法创建日志文件，可以考虑使用默认行为或其他处理方式
		slog.Error("failed to open log file", "error", err)
		return
	}

	defer logFile.Close()

	// 创建使用文件作为输出的 slog handler
	handler := slog.NewTextHandler(logFile, nil)
	logger := slog.New(handler)

	// 设置全局 logger（可选，如果不设置则需要在各处使用 logger 而不是 slog）
	slog.SetDefault(logger)

	slog.Info("开始连接服务器...")
	for {
		remoteAddr := net.TCPAddr{
			IP:   net.ParseIP("localhost"),
			Port: 82,
		}
		conn, err = net.DialTCP("tcp", nil, &remoteAddr)

		if err != nil {
			slog.Error("连接服务端失败", "error", err)
			slog.Info("3秒后重连...")
			time.Sleep(3 * time.Second)
			continue
		}
		err = conn.SetKeepAlive(true)
		if err != nil {
			slog.Error("设置 KeepAlive 失败", "error", err)
		}
		// 设置Keep-Alive探测间隔（可选）
		err = conn.SetKeepAlivePeriod(30 * time.Second)
		if err != nil {
			slog.Error("设置Keep-Alive周期失败", "error", err)
		}
		slog.Info("连接服务端成功")
		break
	}
	go handleConn()

	// 在创建 tview.Application 之前设置 tCell 的字符集
	tcell.SetEncodingFallback(tcell.EncodingFallbackUTF8)
	// 创建app
	app = tview.NewApplication()

	// 创建消息展示区
	messagesView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true)

	msgViewTable = tview.NewTable().
		SetSelectable(true, true).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorNone).
			Foreground(tcell.ColorNone))

	// 创建消息输入区
	// 替换 inputField 的创建
	textArea = tview.NewTextArea().
		SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack))
	textArea.SetBorder(true).SetBorderColor(tcell.ColorDimGrey).
		SetTitleAlign(tview.AlignLeft).SetTitleColor(tcell.ColorDimGrey)
	// 添加按键处理
	textArea.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			msg := textArea.GetText()
			if msg != "" {
				sendTime := time.Now().Format("01-02 15:04")
				err = sendMsg(msg, sendTime)
				if err != nil {
					addMessage("SYSTEM", "发送失败: "+err.Error(), sendTime)
				} else {
					addMessage(myName, msg, sendTime)
					textArea.SetText("", true)
				}
			}
			return nil
		}
		return event
	})

	// 创建表情按钮和表情表格
	emojiButton = tview.NewButton("😊").
		SetStyle(tcell.StyleDefault.
			Background(tcell.ColorBlack).
			Foreground(tcell.ColorBlack))
	// 创建表格
	emojiTable = tview.NewTable().
		SetSelectable(true, true).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorBlue).
			Foreground(tcell.ColorWhite)).
		SetContent(emojiData{}).
		SetFixed(len(emojis), len(emojis[0]))

	// 聊天区包含表情按钮和文本输入框
	chatBox := tview.NewFlex().
		AddItem(emojiButton, 3, 1, false).
		AddItem(textArea, 0, 1, true)

	// 主布局
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		//AddItem(messagesView, 0, 1, false).
		AddItem(msgViewTable, 0, 1, false).
		AddItem(chatBox, 3, 1, false)

	// 表情按钮点击处理
	emojiButton.SetSelectedFunc(func() {
		if emojiVisible {
			flex.RemoveItem(emojiTable)
			emojiVisible = false
		} else {
			flex.AddItem(emojiTable, 3, 1, true)
			emojiVisible = true
		}
	})

	// 表情选择处理
	emojiTable.SetSelectionChangedFunc(func(row, column int) {
		// 检查索引有效性
		if row >= 0 && row < len(emojis) && column >= 0 && column < len(emojis[0]) {
			selectedEmoji := emojis[row][column]

			// 将选中的表情插入到文本区域
			currentText := textArea.GetText()
			textArea.SetText(currentText+selectedEmoji, true)

			flex.RemoveItem(emojiTable)
			emojiVisible = false

			app.SetFocus(textArea)
		}
	})

	list := tview.NewList().
		AddItem("晗", "Miss Rabbit 最最最亲爱的宝贝兔兔", 'H', nil).
		AddItem("勋", "", 'X', nil).
		SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
			myName = mainText
			textArea.SetTitle("[ " + mainText + " ]")
			err := login(mainText)
			if err != nil {
				return
			}
			app.SetRoot(flex, true).SetFocus(textArea)
		})

	if err := app.SetRoot(list, true).EnableMouse(true).SetFocus(textArea).Run(); err != nil {
		panic(err)
	}
}

func addMessage(sender, message, sendTime string) {
	// 消息
	messageCell := tview.NewTableCell(message).SetExpansion(1)
	// 发送者
	senderCell := tview.NewTableCell("").SetExpansion(1)

	if sender == "SYSTEM" {
		messageCell.SetTextColor(tcell.ColorRed).SetAlign(tview.AlignCenter)
		msgViewTable.SetCell(messageRow, 0, messageCell)
		messageRow++
		return
	} else if sender == myName {
		senderCell.SetText("[#464142]" + sendTime + " [green]" + sender).SetAlign(tview.AlignRight)
		messageCell.SetAlign(tview.AlignRight)
	} else {
		messageCell.SetAlign(tview.AlignLeft)
		senderCell.SetText("[green]" + sender + " [#464142]" + sendTime)
	}

	// 添加到表格
	msgViewTable.SetCell(messageRow, 0, senderCell)
	messageRow++

	msgViewTable.SetCell(messageRow, 0, messageCell)
	messageRow++
}

func login(name string) (err error) {
	loginMsg := model.Login{
		MyName: name,
	}

	loginData, err := json.Marshal(loginMsg)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}

	msgData := model.Msg{
		Data:    loginData,
		MsgType: consts.LoginMsgType,
	}

	msg, err := json.Marshal(msgData)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}

	finalMsg := protocol.Encoder(msg)

	_, err = conn.Write(finalMsg)
	return
}

func sendMsg(data string, sendTime string) (err error) {
	chatMsg := model.Chat{
		Data:     data,
		SendTime: sendTime,
		MyName:   myName,
	}
	chatData, err := json.Marshal(chatMsg)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}

	msgData := model.Msg{
		Data:    chatData,
		MsgType: consts.ChatMsgType,
	}

	msg, err := json.Marshal(msgData)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}

	finalMsg := protocol.Encoder(msg)

	_, err = conn.Write(finalMsg)
	return
}

func handleConn() {
	err := protocol.Decoder(conn, handleMsg)
	if err != nil {
		app.QueueUpdateDraw(func() {
			addMessage("SYSTEM", "服务器故障，5秒后退出程序", "")
		})
		time.Sleep(5 * time.Second)
		app.QueueUpdateDraw(func() {
			app.Stop()
		})
	}
}

func handleMsg(msgBytes []byte, conn net.Conn) {
	chatMsg := model.Chat{}
	err := json.Unmarshal(msgBytes, &chatMsg)
	if err != nil {
		slog.Error("json 反序列化消息错误", "error", err)
		return
	}

	slog.Info("收到消息", "message", chatMsg)

	// 使用 QueueUpdateDraw 安全地更新 UI
	app.QueueUpdateDraw(func() {
		addMessage(chatMsg.MyName, chatMsg.Data, chatMsg.SendTime)
	})
	_ = beeep.Notify("新消息", "请看消息哦~🤗", "")
}
