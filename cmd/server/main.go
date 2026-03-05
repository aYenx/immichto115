package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/aYenx/immichto115/internal/api"
	"github.com/aYenx/immichto115/internal/config"
)

//go:embed all:dist
var staticFS embed.FS

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "config file path (default: ./config/config.yaml)")
	port := flag.Int("port", 0, "server listen port (overrides config)")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[immichto115] starting...")

	// 确定配置文件路径
	cfgPath := *configPath
	if cfgPath == "" {
		// 优先检查环境变量
		cfgPath = os.Getenv("IMMICHTO115_CONFIG")
	}
	if cfgPath == "" {
		// 默认路径：可执行文件同级 config/config.yaml
		exe, err := os.Executable()
		if err != nil {
			log.Printf("[immichto115] warning: failed to get executable path: %v, using current dir", err)
			exe = "."
		}
		cfgPath = filepath.Join(filepath.Dir(exe), "config", "config.yaml")
	}

	log.Printf("[immichto115] config path: %s", cfgPath)

	// 初始化配置管理器
	cfgMgr, err := config.NewManager(cfgPath)
	if err != nil {
		log.Fatalf("[immichto115] failed to init config: %v", err)
	}

	// 确定端口
	listenPort := cfgMgr.Get().Server.Port
	if *port > 0 {
		listenPort = *port
	}
	if listenPort == 0 {
		listenPort = 8096
	}

	// 创建 Server
	srv := api.NewServer(cfgMgr)

	// 初始化定时任务
	srv.InitCron()

	// 设置路由
	router := srv.SetupRouter()

	// 提供前端静态资源（内嵌的 Vue dist）
	serveFrontend(router)

	addr := fmt.Sprintf(":%d", listenPort)
	log.Printf("[immichto115] server listening on http://0.0.0.0%s", addr)

	if err := router.Run(addr); err != nil {
		log.Fatalf("[immichto115] server failed: %v", err)
	}
}

// serveFrontend 将内嵌的前端静态文件挂载到路由上。
func serveFrontend(r *gin.Engine) {
	distFS, err := fs.Sub(staticFS, "dist")
	if err != nil {
		log.Printf("[immichto115] warning: no embedded frontend found: %v", err)
		return
	}

	httpFS := http.FS(distFS)

	// 静态资源直接服务
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, httpFS)
	})

	// favicon
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.FileFromFS("favicon.ico", httpFS)
	})

	// favicon.svg (Vite 生成的 index.html 引用的是 svg)
	r.GET("/favicon.svg", func(c *gin.Context) {
		c.FileFromFS("favicon.svg", httpFS)
	})

	// SPA 回退：所有非 API/WS 路径都返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// 不拦截 API 和 WebSocket 请求
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if len(path) >= 3 && path[:3] == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.FileFromFS("index.html", httpFS)
	})
}
