package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type WebSocketServer struct {
	id        string
	clients   map[*websocket.Conn]bool
	rooms     map[*websocket.Conn]string
	broadcast chan Broadcast
	tpl       *template.Template
}

type Broadcast struct {
	Msg    *Message
	Sender *websocket.Conn
}

func NewWebSocket() *WebSocketServer {
	tpl := template.Must(template.ParseFiles("views/message.html"))
	return &WebSocketServer{
		id:        uuid.New().String(),
		clients:   make(map[*websocket.Conn]bool),
		rooms:     make(map[*websocket.Conn]string),
		broadcast: make(chan Broadcast, 32),
		tpl:       tpl,
	}
}

func (s *WebSocketServer) HandleWebSocket(ctx *websocket.Conn) {
	s.clients[ctx] = true
	s.rooms[ctx] = "chatroom"

	defer func() {
		delete(s.clients, ctx)
		delete(s.rooms, ctx)
		ctx.Close()
	}()

	for {
		_, msg, err := ctx.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}

		var m Message
		if err := json.Unmarshal(msg, &m); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		// Normalize fields
		m.Text = strings.TrimSpace(m.Text)
		m.Smt = strings.TrimSpace(m.Smt)
		m.ClientName = strings.TrimSpace(m.ClientName)
		m.Room = strings.TrimSpace(m.Room)

		// Ignore empty messages
		if m.Text == "" {
			continue
		}

		// Fallback name
		if m.ClientName == "" && m.Smt != "" {
			m.ClientName = m.Smt
		}
		if m.ClientName == "" {
			m.ClientName = "Anon"
		}

		// Room handling
		if m.Room != "" {
			s.rooms[ctx] = m.Room
		} else {
			m.Room = s.rooms[ctx]
			if m.Room == "" {
				m.Room = "chatroom"
				s.rooms[ctx] = m.Room
			}
		}

		if m.Timestamp == "" {
			m.Timestamp = time.Now().Format("15:04")
		}

		s.broadcast <- Broadcast{Msg: &m, Sender: ctx}
	}
}

func (s *WebSocketServer) HandleMessages() {
	for b := range s.broadcast {
		msg := b.Msg

		msgRoom := msg.Room
		if msgRoom == "" {
			msgRoom = "chatroom"
		}

		for client := range s.clients {
			clientRoom := s.rooms[client]
			if clientRoom == "" {
				clientRoom = "chatroom"
			}
			if clientRoom != msgRoom {
				continue
			}

			var buf bytes.Buffer
			data := struct {
				Msg       *Message
				SelfClass string
			}{
				Msg:       msg,
				SelfClass: ternary(client == b.Sender, "me", "them"),
			}

			if err := s.tpl.Execute(&buf, data); err != nil {
				log.Printf("template execution error: %v", err)
				continue
			}
			if err := client.WriteMessage(websocket.TextMessage, buf.Bytes()); err != nil {
				log.Printf("write error: %v", err)
				client.Close()
				delete(s.clients, client)
				delete(s.rooms, client)
			}
		}
	}
}

// small helper since Go templates don't handle logic well
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
