package api

import (
	"log"
	"net/http"
	"os"
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

func New(db *db.DB, state *state.Manager, hub *websocket.Hub, mediaDir string) *API {
	return &API{db, state, hub, mediaDir}
}

// RegisterRoutes 注册 Gin 路由
func (a *API) RegisterRoutes(router *gin.Engine) {
	// Web Sockets
	// WebSocket 通常需要直接操作 http.ResponseWriter 和 *http.Request
	router.GET("/ws", a.handleWebSocket)

	// Static files
	router.Static("/static/audio", a.mediaDir)

	// API Group
	apiGroup := router.Group("/api")
	{
		apiGroup.GET("/validate-token", a.handleValidateToken)

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

func (a *API) handleWebSocket(c *gin.Context) {
	// Gin 的 Context 提供了 Writer 和 Request，可以直接传递给 WebSocket 升级器
	// 传递一个函数，当新用户连接时，会调用此函数获取当前状态并发送
	a.hub.ServeWs(c.Writer, c.Request, a.state.GetFullState)
}

func (a *API) handleValidateToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token is required"})
		return
	}
	valid, err := a.db.IsTokenValid(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": valid})
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
	// Gin 默认限制了请求体大小，如果需要更改，可以在 router 初始化时设置 router.MaxMultipartMemory
	// c.Request.ParseMultipartForm(100 << 20) // Gin 内部会自动处理 MultipartForm，通常不需要手动 Parse

	fileHeader, err := c.FormFile("audioFile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error retrieving the file"})
		return
	}

	songUUID, _ := uuid.NewV4()
	ext := filepath.Ext(fileHeader.Filename)
	newFileName := songUUID.String() + ext
	filePath := filepath.Join(a.mediaDir, newFileName)

	// 使用 Gin 的 SaveUploadedFile 简化保存过程
	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving the file"})
		return
	}

	// --- TODO 实现部分 ---
	// 使用 ffprobe 读取元数据 (假设 getAudioMetadata 函数存在于包中)
	title, artist, album, durationMs, err := getAudioMetadata(filePath)
	if err != nil {
		log.Printf("Warning: Failed to get metadata for %s: %v. Using filename as title.", fileHeader.Filename, err)
		// 即使读取失败，也继续，只是元数据不完整
		title = strings.TrimSuffix(fileHeader.Filename, ext)
		durationMs = 0 // 或者一个默认值
	}
	// 如果 ffprobe 没有读到标题，则使用文件名作为后备
	if title == "" {
		title = strings.TrimSuffix(fileHeader.Filename, ext)
	}
	// --- END TODO ---

	song := &db.Song{
		ID:         songUUID.String(),
		Title:      title,
		Artist:     artist,
		Album:      album,
		DurationMs: durationMs,
		Source:     "local",
		FilePath:   newFileName,
	}

	if err := a.db.AddSong(song); err != nil {
		os.Remove(filePath) // 如果数据库失败，删除文件
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding song to database"})
		return
	}

	log.Printf("New song uploaded: %s (Artist: %s, Duration: %dms)", song.Title, song.Artist, song.DurationMs)
	c.JSON(http.StatusCreated, song)
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

	// 在状态管理器操作之前获取文件路径，因为之后记录就没了
	song, err := a.db.GetSong(payload.SongID)
	if err != nil {
		// 如果歌曲本就不存在，可以认为删除成功
		log.Printf("Attempted to delete non-existent song %s", payload.SongID)
		c.Status(http.StatusOK)
		return
	}

	// 调用状态管理器来处理所有逻辑
	if err := a.state.RemoveSongFromLibrary(payload.SongID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove song: " + err.Error()})
		return
	}

	// 从文件系统删除文件
	filePath := filepath.Join(a.mediaDir, song.FilePath)
	if err := os.Remove(filePath); err != nil {
		// 即使文件删除失败，数据库和状态也已更新，所以只记录错误
		log.Printf("Warning: failed to delete audio file %s: %v", filePath, err)
	}

	// 状态管理器已经广播了状态更新，这里只需返回成功
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
