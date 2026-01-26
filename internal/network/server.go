package network

import (
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
	"vanished-rooms/internal/storage"
	"vanished-rooms/internal/ui"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

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

	http.HandleFunc("/ws", requestLogger(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[-] Upgrade error:", err)
			return
		}
		go sv.HandleConnection(conn)
	}))

	http.HandleFunc("/", requestLogger(func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles("./web/index.html")
		if err != nil {
			log.Printf("[!] Error cargando template: %v. Verifica que ./web/index.html existe.", err)
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
			log.Printf("[!] Error al renderizar banners: %v", err)
		}
	}))

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("[+] Vanished Rooms server listening on port: %s (Tor/Onion Mode)...\n", port)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("[-] Error starting server:", err)
	}
}

func generateUUID() string {
	return uuid.New().String()
}

func requestLogger(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Solo registramos la hora y la ruta (ej: / o /ws)
		// No usamos r.RemoteAddr para proteger el anonimato del usuario
		log.Printf("[TRAFFIC] %s - Request: %s %s",
			time.Now().Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
		)
		next(w, r)
	}
}
