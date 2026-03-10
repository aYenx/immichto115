package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	appfs "github.com/aYenx/immichto115"
	"github.com/aYenx/immichto115/internal/api"
	"github.com/aYenx/immichto115/internal/config"
	"github.com/gin-gonic/gin"
)

// version 由构建时 -ldflags "-X main.version=vX.Y.Z" 注入
var version = "dev"

func main() {
	// 命令行参数
	configPath := flag.String("config", "", "config file path (default: ./config/config.yaml)")
	port := flag.Int("port", 0, "server listen port (overrides config)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("[immichto115] starting (version: %s)...", version)

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

	// 启动后台清理过期的 115 授权 session
	srv.StartAuthCleanup(context.Background())

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
	distFS, err := fs.Sub(appfs.WebDistFS, "web/dist")
	if err != nil {
		log.Printf("[immichto115] warning: no embedded frontend found: %v", err)
		return
	}

	httpFS := http.FS(distFS)

	// 预读 index.html 内容，避免 http.FileServer 对 /index.html 的自动 301 重定向
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		log.Printf("[immichto115] warning: index.html not found in dist: %v", err)
		return
	}

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
	// 注意：不能用 c.FileFromFS("index.html", httpFS)，
	// 因为 http.FileServer 内部会对 /index.html 路径 301 重定向到 /，导致死循环。
	// 改为直接写入预读的 index.html 字节内容。
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if len(path) >= 3 && path[:3] == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
