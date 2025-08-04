package main

import (
	"encoding/json"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
	"log/slog"
	"net"
	"talk/common/consts"
	"talk/common/model"
	"time"
)

var conn *net.TCPConn
var myName string

var app *tview.Application
var messagesView *tview.TextView
var textArea *tview.TextArea
var emojiButton *tview.Button
var emojiTable *tview.Table
var emojiVisible = false

var emojis = [][]string{
	{"😀", "😁", "😂", "😊", "😄", "😉", "😋", "😎", "😍", "😘", "🥰", "🥲", "😚", "🙂",
		"🤗", "🤔", "🤨", "😐", "😑", "🤡", "🤥", "🙂", "🙂", "🤫", "🤭", "🫣", "🧐", "🤓", "👻", "💩", "🥳", "🥸"},
	{"🙄", "😏", "😣", "😥", "🤐", "😯", "😫", "🥱", "😴", "😌", "🤤", "😒", "😓", "😔",
		"😕", "🫤", "🙃", "🫠", "😲", "🙁", "😖", "😞", "😟", "😤", "😢", "🥹", "😺", "🐰", "🐻", "🐽", "❤", "💔"},
	{"😭", "😦", "😧", "😨", "😩", "😬", "😮‍💨", "😰", "😱", "😳", "🤪", "😵", "😵‍💫", "🥴",
		"😠", "😡", "🤬", "😷", "🤒", "🤕", "🤮", "🤧", "😇", "", "", "", "", "", "", "", "", ""},
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
	var err error
	slog.Info("开始连接服务器...")
	for {
		remoteAddr := net.TCPAddr{
			IP:   net.ParseIP("serverIP"),
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
				sendTime := time.Now().Format("15:04")
				err := sendMsg(msg, sendTime)
				if err != nil {
					addMessage("[red]系统[white]", "发送失败: "+err.Error(), sendTime)
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
		SetFixed(len(emojis), len(emojis[0])).
		SetBorders(true)

	// 聊天区包含表情按钮和文本输入框
	chatBox := tview.NewFlex().
		AddItem(emojiButton, 3, 1, false).
		AddItem(textArea, 0, 1, true)

	// 主布局
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(messagesView, 0, 1, false).
		AddItem(chatBox, 3, 1, false)

	// 表情按钮点击处理
	emojiButton.SetSelectedFunc(func() {
		if emojiVisible {
			flex.RemoveItem(emojiTable)
			emojiVisible = false
		} else {
			flex.AddItem(emojiTable, 7, 1, true)
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
		AddItem("rabbit", "", 'r', nil).
		AddItem("bear", "", 'b', nil).
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
	showStr := fmt.Sprintf("[ %s ] [#464142]%s[white] : %s\n", sender, sendTime, message)

	_, err := messagesView.Write([]byte(showStr))
	if err != nil {
		slog.Error("write error", "error", err)
		return
	}
	messagesView.ScrollToEnd()
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

	msg := model.Msg{
		Data:    loginData,
		MsgType: consts.LoginMsgType,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}
	_, err = conn.Write(msgData)
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

	msg := model.Msg{
		Data:    chatData,
		MsgType: consts.ChatMsgType,
	}

	msgData, err := json.Marshal(msg)
	if err != nil {
		slog.Error("json marshal error", "error", err)
		return
	}
	_, err = conn.Write(msgData)
	return
}

func handleConn() {
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			app.QueueUpdateDraw(func() {
				addMessage("[red]系统[white]", "服务器故障，5秒后退出程序", "")
			})
			time.Sleep(5 * time.Second)
			app.QueueUpdateDraw(func() {
				app.Stop()
			})
		}
		chatMsg := model.Chat{}
		err = json.Unmarshal(buf[:n], &chatMsg)
		if err != nil {
			slog.Error("json unmarshal error", "error", err)
			continue
		}
		clear(buf)

		// 使用 QueueUpdateDraw 安全地更新 UI
		app.QueueUpdateDraw(func() {
			addMessage("[green]"+chatMsg.MyName+"[white]", chatMsg.Data, chatMsg.SendTime)
		})
		_ = beeep.Notify("新消息", "请看消息哦~🤗", "")
	}
}
