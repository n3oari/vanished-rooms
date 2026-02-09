package network

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"vanished-rooms/internal/logger"
	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var l = logger.New()

type ClientSession struct {
	wsConn    *websocket.Conn
	ID        string
	Username  string
	PublicKey string
	Room      string
	writeMu   sync.Mutex // ← Añadido
}

func (c *ClientSession) Send(msg []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.wsConn.WriteMessage(websocket.TextMessage, msg)
}

type Server struct {
	Clients          map[string]*ClientSession
	mu               sync.RWMutex
	SQLiteRepository *storage.SQLiteRepository
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func StartServer(port string, repository *storage.SQLiteRepository) {
	ui.PrintRandomBanner()
	l.Log(logger.WARN, "System boot: Purging existing ephemeral data...")
	err := repository.PurgeEverything()
	if err != nil {
		l.Log(logger.ERROR, "Startup purge failed: "+err.Error())
	} else {
		l.Log(logger.INFO, "Database is clean. Ready for new connections.")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	sv := &Server{
		Clients:          make(map[string]*ClientSession),
		SQLiteRepository: repository,
	}

	http.HandleFunc("/ws", l.MiddlewareRequest(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			l.Log(logger.ERROR, "WS Upgrade failed: "+err.Error())
			return
		}
		go sv.HandleConnection(conn)
	}))

	http.HandleFunc("/", l.MiddlewareRequest(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./web/index.html")
		if err != nil {
			l.Log(logger.ERROR, "Template missing: ./web/index.html")
			http.Error(w, "Internal Error: Could not load the portal.", 500)
			return
		}
		data := struct {
			Banner1 string
			Banner2 string
			Banner3 string
		}{
			Banner1: ui.Banner1,
			Banner2: ui.Banner2,
			Banner3: ui.Banner3,
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			l.Log(logger.ERROR, "Render error: "+err.Error())
		}
	}))

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	go func() {
		l.Log(logger.WARN, "Connecting to the wire...")
		l.Log(logger.INFO, "Vanished Rooms listening on port: "+port+" (Tor Mode)")
		err := http.ListenAndServe(":"+port, nil)
		if err != nil && err != http.ErrServerClosed {
			l.Log(logger.ERROR, "Fatal Server Error: "+err.Error())
		}
	}()

	<-stop
	l.Log(logger.INFO, "Shutting down... Purging database.")
	err = sv.SQLiteRepository.PurgeEverything()
	if err != nil {
		l.Log(logger.ERROR, "Failed to purge database: "+err.Error())
	}
	l.Log(logger.INFO, "Vanished successfully.")
}

func generateUUID() string {
	return uuid.New().String()
}

func (sv *Server) sendAutoRooms(conn *websocket.Conn) {
	rooms, err := sv.SQLiteRepository.ListPublicRooms()
	if err != nil || len(rooms) == 0 {
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s:\n\n=== PUBLIC ROOMS AVAILABLE ;)) ===\n", EvSystemInfo))
	for _, name := range rooms {
		fmt.Fprintf(&sb, " • %s\n", name)
	}
	sb.WriteString("================================")
	conn.WriteMessage(websocket.TextMessage, []byte(sb.String()))
}
