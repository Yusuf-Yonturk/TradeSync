package main

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/twmb/franz-go/pkg/kgo"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS her isteğe açık
	},
}

type WSHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]bool
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[*websocket.Conn]bool),
	}
}

func (h *WSHub) AddClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
}

func (h *WSHub) RemoveClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		conn.Close()
		delete(h.clients, conn)
	}
}

func (h *WSHub) Broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *WSHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade hatasi:", err)
		return
	}
	h.AddClient(conn)

	// Bağlantı açık kaldığı sürece dinle, kopunca temizle
	go func() {
		defer h.RemoveClient(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

func startKafkaTradeConsumer(brokers []string, hub *WSHub) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics("trades", "depth"),
		// Sadece anlık olarak en son düşen verileri almak için AtEnd offset kullanıyoruz
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()),
	)
	if err != nil {
		log.Fatalf("ws kafka consumer hatasi: %v", err)
	}

	go func() {
		defer client.Close()
		log.Println("ws için kafka trade ve depth consumer basladi")
		for {
			fetches := client.PollFetches(context.Background())
			if fetches.IsClientClosed() {
				break
			}
			fetches.EachError(func(t string, p int32, err error) {
				log.Printf("ws fetch hatasi topic=%s: %v", t, err)
			})
			fetches.EachRecord(func(r *kgo.Record) {
				// r.Value halihazırda trade veya depth JSON'ı, direkt broadcast edebiliriz
				hub.Broadcast(r.Value)
			})
		}
	}()
}
