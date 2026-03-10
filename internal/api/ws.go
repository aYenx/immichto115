package api

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aYenx/immichto115/internal/rclone"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 允许 same-origin 和本地开发环境
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // 无 Origin 头（非浏览器客户端）
		}
		host := r.Host
		// 允许 same-origin: http(s)://host
		if origin == "http://"+host || origin == "https://"+host {
			return true
		}
		// 允许本地开发常见端口
		for _, prefix := range []string{"http://localhost:", "http://127.0.0.1:", "http://[::1]:"} {
			if len(origin) > len(prefix) && origin[:len(prefix)] == prefix {
				return true
			}
		}
		log.Printf("[ws] rejected origin: %s (host: %s)", origin, host)
		return false
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// wsClient 代表一个 WebSocket 客户端连接，所有写操作通过 send channel 串行化。
type wsClient struct {
	conn *websocket.Conn
	send chan rclone.LogLine
}

// Hub 管理所有活跃的 WebSocket 连接，负责广播日志。
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]bool
}

// NewHub 创建一个新的 WebSocket Hub。
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*wsClient]bool),
	}
}

// register 注册一个新的 WebSocket 客户端。
func (h *Hub) register(client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
	log.Printf("[ws] client connected, total: %d", len(h.clients))
}

// unregister 移除一个 WebSocket 客户端并关闭连接。
func (h *Hub) unregister(client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		client.conn.Close()
		log.Printf("[ws] client disconnected, total: %d", len(h.clients))
	}
}

// Broadcast 向所有已连接的客户端广播一条日志消息。
// 使用非阻塞 channel 发送避免慢客户端阻塞 rclone 进程输出。
func (h *Hub) Broadcast(line rclone.LogLine) {
	h.mu.RLock()
	var dead []*wsClient
	for client := range h.clients {
		select {
		case client.send <- line:
		default:
			// send buffer 已满，说明客户端过慢，标记为断开
			dead = append(dead, client)
		}
	}
	h.mu.RUnlock()

	// 在读锁释放后统一清理断开的连接
	for _, client := range dead {
		log.Printf("[ws] client too slow, removing")
		h.unregister(client)
	}
}

// BroadcastFromChannel 从 Rclone 输出 channel 中读取日志并广播。
// 此函数会阻塞直到 channel 关闭（即 Rclone 进程结束）。
func (h *Hub) BroadcastFromChannel(logCh <-chan rclone.LogLine) {
	for line := range logCh {
		h.Broadcast(line)
	}
}

// writePump 是每个客户端独立的写 goroutine，串行化所有写操作。
// 这保证了 gorilla/websocket 的 Conn 不会被并发写入。
func (c *wsClient) writePump() {
	const (
		writeWait    = 5 * time.Second
		pingInterval = 30 * time.Second
	)
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Hub 已关闭 send channel，发送 close 帧后退出
				_ = c.conn.WriteControl(websocket.CloseMessage, nil, time.Now().Add(writeWait))
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}

// readPump 持续读取客户端消息（主要用于检测断开连接和处理 Pong）。
func (c *wsClient) readPump(hub *Hub) {
	defer hub.unregister(c)

	const pongWait = 60 * time.Second
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// HandleWebSocket 处理 WebSocket 连接升级请求。
// 对应路由: GET /ws/logs
func HandleWebSocket(hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[ws] upgrade error: %v", err)
			return
		}

		client := &wsClient{
			conn: conn,
			send: make(chan rclone.LogLine, 256),
		}
		hub.register(client)

		// 发送欢迎消息（在 writePump 启动前直接写，此时还没有并发）
		_ = conn.WriteJSON(rclone.LogLine{
			Stream: "stdout",
			Text:   "[immichto115] 已连接到日志流",
		})

		// 启动读写 pump
		go client.writePump()
		go client.readPump(hub)
	}
}
