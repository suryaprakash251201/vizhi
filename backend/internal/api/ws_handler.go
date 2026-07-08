package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"vizhi/backend/internal/monitor"
	"vizhi/backend/internal/process"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // restrict in production via CORS/origin header
	},
}

type WSHandler struct {
	mon       *monitor.Monitor
	appMgr    *process.AppManager
	interval  time.Duration
	clients   map[*websocket.Conn]struct{}
	mu        sync.RWMutex
}

func NewWSHandler(mon *monitor.Monitor, appMgr *process.AppManager, intervalSec int) *WSHandler {
	h := &WSHandler{
		mon:      mon,
		appMgr:   appMgr,
		interval: time.Duration(intervalSec) * time.Second,
		clients:  make(map[*websocket.Conn]struct{}),
	}
	go h.broadcastLoop()
	return h
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	log.Printf("ws client connected (%d total)", len(h.clients))

	// Read loop to detect disconnection
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			conn.Close()
			log.Printf("ws client disconnected (%d remaining)", len(h.clients))
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *WSHandler) broadcastLoop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for range ticker.C {
		h.broadcastStats()
	}
}

func (h *WSHandler) broadcastStats() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := h.mon.Gather(ctx)
	if err != nil {
		log.Printf("ws broadcast: gather stats: %v", err)
		return
	}

	msg := WSMessage{
		Type: "system_stats",
		Data: stats,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws broadcast: marshal: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Printf("ws set deadline: %v", err)
			continue
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws write: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
}
