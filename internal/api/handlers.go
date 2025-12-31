package api

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/yeeeck/sync-jukebox/internal/db"
	"github.com/yeeeck/sync-jukebox/internal/state"
	"github.com/yeeeck/sync-jukebox/internal/websocket"
)

type API struct {
	db       *db.DB
	state    *state.Manager
	hub      *websocket.Hub
	mediaDir string
}

type SeekPayload struct {
	PositionMs int64 `json:"positionMs"`
}

type PlaySpecificPayload struct {
	SongID string `json:"songId"`
}
type ReorderPlaylistPayload struct {
	SongID   string `json:"songId"`
	NewIndex int    `json:"newIndex"`
}

type AuthPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func New(db *db.DB, state *state.Manager, hub *websocket.Hub, mediaDir string) *API {
	return &API{db, state, hub, mediaDir}
}

// RegisterRoutes 注册 Gin 路由
func (a *API) RegisterRoutes(router *gin.Engine) {

	// Static files
	router.Static("/static/audio", a.mediaDir)

	// API Group
	apiGroup := router.Group("/api")
	{
		// Web Sockets
		// WebSocket 通常需要直接操作 http.ResponseWriter 和 *http.Request
		router.GET("/ws", a.handleWebSocket)

		// --- 公开路由 (无需认证) ---
		apiGroup.POST("/register", a.handleRegister)
		apiGroup.POST("/login", a.handleLogin) // 用于前端验证凭证
		// --- 受保护的路由组 ---
		// 使用 BasicAuthMiddleware 中间件
		protected := apiGroup.Group("")
		protected.Use(a.BasicAuthMiddleware())
		{
			libraryGroup := apiGroup.Group("/library")
			{
				libraryGroup.GET("", a.handleGetLibrary)
				libraryGroup.POST("/upload", a.handleUpload)
				libraryGroup.POST("/remove", a.handleLibraryRemove)
			}

			playlistGroup := apiGroup.Group("/playlist")
			{
				playlistGroup.POST("/add", a.handlePlaylistAdd)
				playlistGroup.POST("/remove", a.handlePlaylistRemove)
				// 移动播放列表中的歌曲位置
				playlistGroup.POST("/move", a.handlePlaylistMove)
				// 打乱播放列表
				playlistGroup.POST("/shuffle", a.handlePlaylistShuffle)
			}

			playerGroup := apiGroup.Group("/player")
			{
				playerGroup.POST("/play", a.handlePlay)
				// 播放列表中指定的歌曲
				playerGroup.POST("/play-specific", a.handlePlaySpecific)
				playerGroup.POST("/pause", a.handlePause)
				playerGroup.POST("/next", a.handleNext)
				playerGroup.POST("/prev", a.handlePrev)
				playerGroup.POST("/seek", a.handleSeek)
			}
		}

	}
}

func (a *API) handleWebSocket(c *gin.Context) {
	// Gin 的 Context 提供了 Writer 和 Request，可以直接传递给 WebSocket 升级器
	// 传递一个函数，当新用户连接时，会调用此函数获取当前状态并发送
	a.hub.ServeWs(c.Writer, c.Request, a.state.GetFullState)
}

//func (a *API) handleValidateToken(c *gin.Context) {
//	token := c.Query("token")
//	if token == "" {
//		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
//		return
//	}
//	valid, err := a.db.IsTokenValid(token)
//	if err != nil {
//		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
//		return
//	}
//	c.JSON(http.StatusOK, gin.H{"valid": valid})
//}

// --- 认证处理 ---
// BasicAuthMiddleware 是一个 Gin 中间件，用于验证 Basic Authentication
func (a *API) BasicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, ok := c.Request.BasicAuth()
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
			return
		}
		dbUser, err := a.db.GetUserByUsername(user)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		if !dbUser.CheckPassword(pass) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		// 可选：将用户信息存入 context
		c.Set("username", dbUser.Username)
		c.Next()
	}
}

// handleRegister 处理用户注册
func (a *API) handleRegister(c *gin.Context) {
	var payload AuthPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}
	// 检查用户名是否已存在
	_, err := a.db.GetUserByUsername(payload.Username)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	// 创建用户
	_, err = a.db.CreateUser(payload.Username, payload.Password)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}

// handleLogin 验证用户凭证 (主要用于前端检查)
func (a *API) handleLogin(c *gin.Context) {
	// 复用中间件的逻辑
	user, pass, ok := c.Request.BasicAuth()
	if !ok {
		c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header not provided"})
		return
	}
	dbUser, err := a.db.GetUserByUsername(user)
	if err != nil || !dbUser.CheckPassword(pass) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func (a *API) handleGetLibrary(c *gin.Context) {
	songs, err := a.db.GetAllSongs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get library"})
		return
	}
	c.JSON(http.StatusOK, songs)
}

func (a *API) handleUpload(c *gin.Context) {
	// 1. 获取上传的文件
	fileHeader, err := c.FormFile("audioFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving the file"})
		return
	}
	songUUID, _ := uuid.NewV4()
	songID := songUUID.String()
	// 2. 保存原始文件到临时路径 (例如 media/temp_<uuid>.mp3)
	tempFileName := fmt.Sprintf("temp_%s%s", songID, filepath.Ext(fileHeader.Filename))
	tempFilePath := filepath.Join(a.mediaDir, tempFileName)
	if err := c.SaveUploadedFile(fileHeader, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving temporary file"})
		return
	}
	// 确保函数退出时删除临时文件
	defer os.Remove(tempFilePath)
	// 3. 提取元数据 (Duration, Title, Artist)
	// 在转换前从源文件提取通常更准确
	title, artist, album, durationMs, err := getAudioMetadata(tempFilePath)
	if err != nil {
		log.Printf("Warning: Metadata extraction failed: %v", err)
		durationMs = 0 // 转换失败降级处理
	}
	// 如果元数据中没有标题，使用文件名
	if title == "" {
		title = strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	}
	// 4. 创建该歌曲的 HLS 输出目录 (media/<uuid>/)
	songDir := filepath.Join(a.mediaDir, songID)
	if err := os.MkdirAll(songDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create song directory"})
		return
	}
	// 5. 执行 FFmpeg 转换为 HLS
	// output: media/<uuid>/index.m3u8
	hlsFileName := "index.m3u8"
	hlsFilePath := filepath.Join(songDir, hlsFileName)
	if err := convertToHLS(tempFilePath, hlsFilePath); err != nil {
		// 失败时清理创建的目录
		os.RemoveAll(songDir)
		log.Printf("FFmpeg conversion failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert audio to HLS"})
		return
	}
	// 6. 存入数据库
	// FilePath 存储相对路径: <uuid>/index.m3u8
	relativeFilePath := filepath.Join(songID, hlsFileName)
	// 注意：Windows 下 Join 会用反斜杠，web 访问需要正斜杠，这里做个替换以防万一
	relativeFilePath = filepath.ToSlash(relativeFilePath)
	song := &db.Song{
		ID:         songID,
		Title:      title,
		Artist:     artist,
		Album:      album,
		DurationMs: durationMs,
		Source:     "local",
		FilePath:   relativeFilePath, // 指向 .m3u8
	}
	if err := a.db.AddSong(song); err != nil {
		os.RemoveAll(songDir) // 数据库失败，清理目录
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding song to database"})
		return
	}
	log.Printf("New song uploaded and converted to HLS: %s (%dms)", song.Title, song.DurationMs)
	c.JSON(http.StatusCreated, song)
}

func convertToHLS(inputFile, outputFile string) error {
	// ffmpeg 命令参数：
	// -i input.mp3    : 输入
	// -c:a aac        : 音频编码 AAC (HLS 标准)
	// -b:a 192k       : 码率
	// -vn             : 不处理视频流
	// -hls_time 10    : 每个切片约 10 秒
	// -hls_list_size 0: 索引文件包含所有切片（不覆盖）
	// -f hls          : 输出格式
	cmd := exec.Command("ffmpeg",
		"-i", inputFile,
		"-c:a", "aac",
		"-b:a", "320k",
		"-vn",
		"-hls_time", "10",
		"-hls_list_size", "0",
		"-f", "hls",
		outputFile,
	)
	// 将 stderr 输出到日志以便调试 ffmpeg 错误
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// handleLibraryRemove 处理删除歌曲的请求
func (a *API) handleLibraryRemove(c *gin.Context) {
	var payload struct {
		SongID string `json:"songId"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if payload.SongID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	song, err := a.db.GetSong(payload.SongID)
	if err != nil {
		log.Printf("Attempted to delete non-existent song %s", payload.SongID)
		c.Status(http.StatusOK)
		return
	}
	if err := a.state.RemoveSongFromLibrary(payload.SongID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove song: " + err.Error()})
		return
	}
	// 关键修改：因为现在每个歌曲是一个目录，不仅是 .m3u8 文件
	// 数据库存的是 "uuid/index.m3u8"，我们需要删除 "media/uuid"
	relDir := filepath.Dir(song.FilePath) // 获取 "uuid"
	absDir := filepath.Join(a.mediaDir, relDir)
	// 使用 RemoveAll 递归删除目录及其内容 (.m3u8 和 .ts)
	if err := os.RemoveAll(absDir); err != nil {
		log.Printf("Warning: failed to delete audio directory %s: %v", absDir, err)
	}
	c.Status(http.StatusOK)
}

func (a *API) handlePlaylistAdd(c *gin.Context) {
	var payload struct {
		SongID string `json:"songId"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := a.state.AddToPlaylist(payload.SongID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add song to playlist"})
		return
	}
	c.Status(http.StatusOK)
}

// handlePlaylistRemove 处理从播放列表中移除歌曲的请求
func (a *API) handlePlaylistRemove(c *gin.Context) {
	var payload struct {
		SongID string `json:"songId"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if payload.SongID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	if err := a.state.RemoveFromPlaylist(payload.SongID); err != nil {
		// 记录错误日志
		log.Printf("Failed to remove song from playlist: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove song from playlist"})
		return
	}
	// 成功返回 200 OK
	c.Status(http.StatusOK)
}

// --- Player Controls ---

func (a *API) handlePlay(c *gin.Context) {
	a.state.Play()
	c.Status(http.StatusAccepted)
}

func (a *API) handlePause(c *gin.Context) {
	a.state.Pause()
	c.Status(http.StatusAccepted)
}

func (a *API) handleNext(c *gin.Context) {
	a.state.NextSong()
	c.Status(http.StatusAccepted)
}

func (a *API) handlePrev(c *gin.Context) {
	a.state.PrevSong()
	c.Status(http.StatusAccepted)
}

func (a *API) handleSeek(c *gin.Context) {
	var payload SeekPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := a.state.SeekTo(payload.PositionMs); err != nil {
		// This error is returned if no song is playing.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

// handlePlaySpecific 处理播放指定歌曲的请求
func (a *API) handlePlaySpecific(c *gin.Context) {
	var payload PlaySpecificPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if payload.SongID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	if err := a.state.PlaySpecificSong(payload.SongID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}

// handlePlaylistMove 处理移动播放列表项的请求
func (a *API) handlePlaylistMove(c *gin.Context) {
	var payload ReorderPlaylistPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if payload.SongID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "songId is required"})
		return
	}
	// index 校验在 state 逻辑中处理，但这里可以做一个基本防守
	if payload.NewIndex < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "newIndex must be >= 0"})
		return
	}
	if err := a.state.ReorderPlaylist(payload.SongID, payload.NewIndex); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// handlePlaylistShuffle 处理打乱播放列表的请求
func (a *API) handlePlaylistShuffle(c *gin.Context) {
	// 该接口不需要请求体参数
	if err := a.state.ShufflePlaylist(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to shuffle playlist"})
		return
	}
	c.Status(http.StatusOK)
}
