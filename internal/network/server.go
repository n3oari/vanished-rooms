package network

import (
	"html/template"
	"net/http"
	"sync"
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
			http.Error(w, "Error interno: No se pudo cargar el portal.", 500)
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

	l.Log(logger.INFO, "Vanished Rooms listening on port: "+port+" (Tor Mode)")

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		l.Log(logger.ERROR, "Fatal Server Error: "+err.Error())
	}
}

func generateUUID() string {
	return uuid.New().String()
}
