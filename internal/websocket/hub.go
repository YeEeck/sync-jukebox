package websocket

import (
	"encoding/json"
	"fmt" // --- NEW ---
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/yeeeck/sync-jukebox/internal/db" // --- NEW ---
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// --- MODIFIED ---
// Client 是一个websocket连接的封装
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	User *db.User // 新增: 存储此连接对应的用户信息
}

// Hub 维护了所有活跃的客户端，并向他们广播消息
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run 启动Hub的事件循环
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			// --- MODIFIED ---
			if client.User != nil {
				log.Printf("New client registered: %s", client.User.Username)
			} else {
				log.Println("New anonymous client registered")
			}
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// --- MODIFIED ---
				if client.User != nil {
					log.Printf("Client unregistered: %s", client.User.Username)
				} else {
					log.Println("Anonymous client unregistered")
				}
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast 广播消息给所有客户端
func (h *Hub) Broadcast(message interface{}) {
	// ... (此函数内容不变)
	jsonMsg, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshalling broadcast message: %v", err)
		return
	}
	h.broadcast <- jsonMsg
}

// --- NEW ---
// UserInfo 是用于API响应的简化用户结构
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// --- NEW ---
// GetOnlineUsers 返回所有在线用户的列表（去重后）
func (h *Hub) GetOnlineUsers() []UserInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 使用 map 按用户ID去重，因为一个用户可能打开多个tab (多个连接)
	uniqueUsers := make(map[uint]UserInfo)

	for client := range h.clients {
		if client.User != nil {
			username := client.User.Username
			// 检查用户名是否包含"@"，如果包含，则只取"@"之前的部分
			if strings.Contains(username, "@") {
				username = strings.Split(username, "@")[0]
			}
			// 用 User.ID 作为 key 来去重
			uniqueUsers[client.User.ID] = UserInfo{
				ID:       fmt.Sprintf("%d", client.User.ID),
				Username: username, // 使用处理后的用户名
			}
		}
	}

	// 将 map 的值转换为切片
	users := make([]UserInfo, 0, len(uniqueUsers))
	for _, user := range uniqueUsers {
		users = append(users, user)
	}

	return users
}

// --- MODIFIED ---
// ServeWs 处理websocket请求，现在需要传入认证后的用户信息
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request, user *db.User, onConnect func() interface{}) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	// --- MODIFIED ---
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256), User: user}
	h.register <- client

	initialState := onConnect()
	if initialState != nil {
		jsonState, err := json.Marshal(initialState)
		if err == nil {
			client.send <- jsonState
		}
	}

	go client.writePump()
	go client.readPump()
}

// ... readPump 和 writePump 函数保持不变 ...
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}
