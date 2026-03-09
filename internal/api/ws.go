package api

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/aYenx/immichto115/internal/rclone"
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

// Hub 管理所有活跃的 WebSocket 连接，负责广播日志。
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

// NewHub 创建一个新的 WebSocket Hub。
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Register 注册一个新的 WebSocket 连接。
func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = true
	log.Printf("[ws] client connected, total: %d", len(h.clients))
}

// Unregister 移除一个 WebSocket 连接。
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
		conn.Close()
		log.Printf("[ws] client disconnected, total: %d", len(h.clients))
	}
}

// Broadcast 向所有已连接的客户端广播一条日志消息。
// 使用非阻塞写入避免慢客户端阻塞 rclone 进程输出；
// 收集断开的连接在释放读锁后统一清理，避免 RLock/Lock 交叉死锁。
func (h *Hub) Broadcast(line rclone.LogLine) {
	h.mu.RLock()
	var dead []*websocket.Conn
	for conn := range h.clients {
		err := conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err != nil {
			dead = append(dead, conn)
			continue
		}
		err = conn.WriteJSON(line)
		if err != nil {
			log.Printf("[ws] write error: %v, removing client", err)
			dead = append(dead, conn)
		}
	}
	h.mu.RUnlock()

	// 在读锁释放后统一清理断开的连接
	for _, conn := range dead {
		h.Unregister(conn)
	}
}

// BroadcastFromChannel 从 Rclone 输出 channel 中读取日志并广播。
// 此函数会阻塞直到 channel 关闭（即 Rclone 进程结束）。
func (h *Hub) BroadcastFromChannel(logCh <-chan rclone.LogLine) {
	for line := range logCh {
		h.Broadcast(line)
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

		hub.Register(conn)

		// 发送欢迎消息
		_ = conn.WriteJSON(rclone.LogLine{
			Stream: "stdout",
			Text:   "[immichto115] 已连接到日志流",
		})

		// 持续读取客户端消息（主要用于检测断开连接）
		// 同时通过 Pong handler 实现心跳检测
		go func() {
			defer hub.Unregister(conn)

			const pongWait = 60 * time.Second
			conn.SetReadDeadline(time.Now().Add(pongWait))
			conn.SetPongHandler(func(string) error {
				conn.SetReadDeadline(time.Now().Add(pongWait))
				return nil
			})

			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()

		// 定时发送 Ping 帧，触发客户端 Pong 回复
		go func() {
			const pingInterval = 30 * time.Second
			ticker := time.NewTicker(pingInterval)
			defer ticker.Stop()

			for range ticker.C {
				hub.mu.RLock()
				_, connected := hub.clients[conn]
				hub.mu.RUnlock()
				if !connected {
					return
				}
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}()
	}
}
