package log

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var logFile *os.File

func init() {
	// 创建或打开日志文件
	logFile, _ = os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
}

func Info(msg ...any) {
	message := formatMsg(msg...)
	_, _ = fmt.Fprintf(logFile, "%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), "INFO", message)
}

func Error(msg ...any) {
	message := formatMsg(msg...)
	_, _ = fmt.Fprintf(logFile, "%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), "ERROR", message)
}

func formatMsg(msg ...any) string {
	strMsg := make([]string, len(msg))
	for i, v := range msg {
		strMsg[i] = fmt.Sprint(v)
	}
	return strings.Join(strMsg, " ")
}
