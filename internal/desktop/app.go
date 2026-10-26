package desktop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type ServerEvent struct {
	Type    string `json:"type"`
	RawLine string `json:"raw"`
}

type App struct {
	mu          sync.Mutex
	conn        net.Conn
	playerName  string
	subMu       sync.Mutex
	subscribers map[chan string]bool
}

func NewApp() *App {
	return &App{
		subscribers: make(map[chan string]bool),
	}
}

func (a *App) emit(evType, rawLine string) {
	payload, err := json.Marshal(ServerEvent{
		Type:    evType,
		RawLine: rawLine,
	})
	if err != nil {
		return
	}

	a.subMu.Lock()
	defer a.subMu.Unlock()

	for ch := range a.subscribers {
		select {
		case ch <- string(payload):
		default:
		}
	}
}

// closeExisting закриває старе TCP-з'єднання перед новим логіном,
// щоб сервер звільнив нікнейм у своєму Registry.
func (a *App) closeExisting() {
	a.mu.Lock()
	oldConn := a.conn
	a.conn = nil
	a.playerName = ""
	a.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		_, _ = fmt.Fprintln(oldConn, "QUIT")
		_ = oldConn.Close()
		time.Sleep(80 * time.Millisecond) // Даємо серверу час видалити сесію з Registry
	}
}

// HandleSession перевіряє, чи вже є активне з'єднання (рятує при натисканні F5!)
func (a *App) HandleSession(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	connected := a.conn != nil && a.playerName != ""
	name := a.playerName
	a.mu.Unlock()

	writeJSON(w, map[string]interface{}{
		"connected": connected,
		"username":  name,
	})
}

func (a *App) HandleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"status": "ERR", "error": "Invalid JSON request"})
		return
	}

	// Якщо гравець оновив сторінку або перепідключається — спочатку звільняємо стару сесію!
	a.closeExisting()

	addr := net.JoinHostPort(strings.TrimSpace(req.Host), strings.TrimSpace(req.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		writeJSON(w, map[string]string{"status": "ERR", "error": "ERR 900 CONNECTION_FAILED"})
		return
	}

	reader := bufio.NewReader(conn)

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	greeting, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		writeJSON(w, map[string]string{"status": "ERR", "error": "Failed to read greeting"})
		return
	}
	if strings.TrimSpace(greeting) != "OK hello proto=1" {
		_ = conn.Close()
		writeJSON(w, map[string]string{"status": "ERR", "error": "Unexpected greeting: " + strings.TrimSpace(greeting)})
		return
	}

	username := strings.TrimSpace(req.Username)
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintf(conn, "CONNECT %s\n", username); err != nil {
		_ = conn.Close()
		writeJSON(w, map[string]string{"status": "ERR", "error": "ERR 901 SEND_FAILED"})
		return
	}

	resp, err := reader.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		writeJSON(w, map[string]string{"status": "ERR", "error": "Server closed connection"})
		return
	}
	_ = conn.SetReadDeadline(time.Time{})

	resp = strings.TrimSpace(resp)
	if resp != "OK connected" {
		_ = conn.Close()
		writeJSON(w, map[string]string{"status": "ERR", "error": resp})
		return
	}

	a.mu.Lock()
	a.conn = conn
	a.playerName = username
	a.mu.Unlock()

	go a.readLoop(conn, reader)

	writeJSON(w, map[string]string{
		"status":   "OK",
		"username": username,
		"message":  resp,
	})
}

func (a *App) readLoop(conn net.Conn, reader *bufio.Reader) {
	defer func() {
		_ = conn.Close()
		a.mu.Lock()
		if a.conn == conn {
			a.conn = nil
			a.playerName = ""
		}
		a.mu.Unlock()
		a.emit("DISCONNECTED", "Disconnected from server")
	}()

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			msg := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(msg, "EVT "):
				a.emit("EVT", msg)
			case strings.HasPrefix(msg, "ERR "):
				a.emit("ERR", msg)
			default:
				a.emit("OK", msg)
			}

			if msg == "OK bye" {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (a *App) HandleCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]string{"status": "ERR", "error": "Invalid JSON"})
		return
	}

	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()

	if conn == nil {
		writeJSON(w, map[string]string{"status": "ERR", "error": "Not connected"})
		return
	}

	cmd := strings.TrimSpace(req.Command)
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := fmt.Fprintf(conn, "%s\n", cmd); err != nil {
		writeJSON(w, map[string]string{"status": "ERR", "error": "ERR 901 SEND_FAILED"})
		return
	}

	writeJSON(w, map[string]string{"status": "OK"})
}

// Shutdown коректно відправляє QUIT серверу і завершує процес GUI-клієнта
func (a *App) Shutdown() {
	log.Println("Window closed. Disconnecting from server and exiting GUI client...")
	a.closeExisting()
	os.Exit(0)
}

func (a *App) HandleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Відправляємо початковий коментар, щоб браузер одразу зафіксував відкритий потік
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ch := make(chan string, 32)
	a.subMu.Lock()
	a.subscribers[ch] = true
	a.subMu.Unlock()

	defer func() {
		a.subMu.Lock()
		delete(a.subscribers, ch)
		remaining := len(a.subscribers)
		a.subMu.Unlock()

		// Якщо більше немає відкритих вікон — чекаємо 1.5 сек (на випадок F5 reload)
		// і якщо вікно не повернулося — чисто завершуємо програму!
		if remaining == 0 {
			go func() {
				time.Sleep(1500 * time.Millisecond)
				a.subMu.Lock()
				stillZero := len(a.subscribers) == 0
				a.subMu.Unlock()

				if stillZero {
					a.Shutdown()
				}
			}()
		}
	}()

	for {
		select {
		case data := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
