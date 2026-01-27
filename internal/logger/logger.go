package logger

import (
	"fmt"
	"net/http"
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
	file *os.File
}

func New() *CustomLogger {
	f, err := os.OpenFile("vanished_rooms.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[!] No se pudo crear el archivo de log: %v. Usando solo consola.\n", err)
	}
	return &CustomLogger{
		file: f,
	}
}

func (l *CustomLogger) Log(level LogLevel, message string) {
	now := time.Now().Format("2006/01/02 15:04:05")

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
		color = colorBlue
	}

	consoleMsg := fmt.Sprintf("%s%s%s %s[%s]%s %s\n",
		colorYellow, now, colorReset,
		color, level, colorReset,
		message,
	)
	fmt.Print(consoleMsg)

	// 2. Log para Archivo (SIN colores)
	if l.file != nil {
		fileMsg := fmt.Sprintf("%s [%s] %s\n", now, level, message)
		l.file.WriteString(fileMsg)
	}
}

func (l *CustomLogger) LogRoom(roomName string, user string, message string) {
	now := time.Now().Format("2006/01/02 15:04:05")

	// 1. Consola (Colorido)
	consoleMsg := fmt.Sprintf("%s[%s]%s%s[%s]%s %s%s%s: %s\n",
		colorYellow, now, colorReset,
		colorPurple, roomName, colorReset,
		colorCyan, user, colorReset,
		message,
	)
	fmt.Print(consoleMsg)

	// 2. Archivo (Plano)
	if l.file != nil {
		fileMsg := fmt.Sprintf("%s [%s] %s: %s\n", now, roomName, user, message)
		l.file.WriteString(fileMsg)
	}
}

func (l *CustomLogger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *CustomLogger) MiddlewareRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Log(ONION_INFO, fmt.Sprintf("%s %s", r.Method, r.URL.Path))

		next(w, r)
	}
}
