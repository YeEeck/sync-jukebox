// cmd/server/main.go

package main

import (
	"log"
	"mime"
	"os"
	"time"

	"github.com/gin-contrib/cors" // 1. 引入 Gin 的 CORS 库
	"github.com/gin-gonic/gin"    // 2. 引入 Gin
	"github.com/yeeeck/sync-jukebox/internal/api"
	"github.com/yeeeck/sync-jukebox/internal/db"
	"github.com/yeeeck/sync-jukebox/internal/state"
	"github.com/yeeeck/sync-jukebox/internal/websocket"
)

const (
	dbPath      = "./jukebox.db"
	mediaDir    = "./media"
	frontendDir = "./frontend/dist"
	serverAddr  = ":8880"
	keyFilePath = "./invitation.key"
)

func main() {
	// ... (数据库、Hub、状态管理器的初始化代码保持不变) ...
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		log.Fatalf("Failed to create media directory: %v", err)
	}

	// --- 注册 HLS MIME 类型 ---
	// 某些操作系统默认没有注册这些类型，会导致浏览器无法播放
	if err := mime.AddExtensionType(".m3u8", "application/vnd.apple.mpegurl"); err != nil {
		log.Printf("Warning: Failed to register .m3u8 mime type: %v", err)
	}
	if err := mime.AddExtensionType(".ts", "video/mp2t"); err != nil {
		log.Printf("Warning: Failed to register .ts mime type: %v", err)
	}

	// --- 初始化密钥管理器 ---
	keyManager := api.NewInvitationKeyManager(keyFilePath)

	if _, err := keyManager.GenerateNewKey(); err != nil {
		log.Fatalf("Failed to generate initial invitation key: %v", err)
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if _, err := keyManager.GenerateNewKey(); err != nil {
				log.Printf("Error in periodic key generation: %v", err)
			}
		}
	}()

	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("DB initialization failed: %v", err)
	}
	defer database.Close()

	hub := websocket.NewHub()
	go hub.Run()

	stateManager, err := state.NewManager(database, hub)
	if err != nil {
		log.Fatalf("State manager initialization failed: %v", err)
	}

	// 3. 初始化 Gin 引擎
	// gin.SetMode(gin.ReleaseMode) // 如果在生产环境，取消这行注释以关闭调试日志
	router := gin.Default()

	// 4. 配置 CORS 中间件 (gin-contrib/cors)
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://10.8.0.10:5173",
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:4200",
		// "https://your-production-frontend.com",
	}
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"Content-Type", "Authorization"}
	router.Use(cors.New(config))

	// 5. 注册 API 路由
	// 注意：这里需要根据之前修改的 api.go，传入 router 而不是 mux
	apiHandler := api.New(database, stateManager, hub, mediaDir, keyManager)
	apiHandler.RegisterRoutes(router)

	// 6. 服务前端静态文件
	// 注意：SPA (Vue/React) 需要特殊处理，不能简单使用 Static
	// 任何没有匹配到 API 或 websocket 的路由，都应该返回 index.html
	router.NoRoute(func(c *gin.Context) {
		// 尝试直接访问文件 (例如 .js, .css, .ico)
		path := frontendDir + c.Request.URL.Path
		if _, err := os.Stat(path); err == nil {
			c.File(path)
			return
		}
		// 如果文件不存在，或者是目录，则返回 index.html (SPA History Mode 支持)
		c.File(frontendDir + "/index.html")
	})

	// 启动服务器
	log.Printf("SyncJukebox v2.0 server starting on %s with Gin & CORS enabled", serverAddr)
	log.Printf("Serving frontend from: %s", frontendDir)
	log.Printf("Serving media from: %s", mediaDir)

	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
