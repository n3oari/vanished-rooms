package logger

import (
	"log"
	"os"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorWhite  = "\033[37m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorGray   = "\033[90m"
	colorPurple = "\033[35m"
)

type LogLevel string

const (
	INFO       LogLevel = "INFO"
	DEBUG      LogLevel = "DEBUG"
	WARN       LogLevel = "WARN"
	ERROR      LogLevel = "ERROR"
	ONION_INFO LogLevel = "TRAFFIC-ONION"
)

type CustomLogger struct {
	logger *log.Logger
}

func New() *CustomLogger {
	return &CustomLogger{
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *CustomLogger) Log(level LogLevel, message string) {
	var color string
	switch level {
	case INFO:
		color = colorCyan
	case WARN:
		color = colorYellow
	case ERROR:
		color = colorRed
	case ONION_INFO:
		color = colorPurple
	default:
		color = colorBlue // El azul para DEBUG
	}

	now := time.Now().Format("2006/01/02 15:04:05")

	l.logger.Printf("%s%s%s %s[%s]%s %s\n",
		colorYellow, now, colorReset,
		color, level, colorReset,
		message,
	)
}

func (l *CustomLogger) LogRoom(roomName string, user string, message string) {
	now := time.Now().Format("2006/01/02 15:04:05")

	l.logger.Printf("%s[%s]%s%s[%s]%s %s%s%s: %s\n",
		colorYellow, now, colorReset,
		colorPurple, roomName, colorReset,
		colorCyan, user, colorReset,
		message,
	)
}
