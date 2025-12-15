package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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

func New(db *db.DB, state *state.Manager, hub *websocket.Hub, mediaDir string) *API {
	return &API{db, state, hub, mediaDir}
}

func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// Web Sockets
	mux.HandleFunc("/ws", a.handleWebSocket)

	// API
	mux.HandleFunc("/api/validate-token", a.handleValidateToken)
	mux.HandleFunc("/api/library", a.handleGetLibrary)
	mux.HandleFunc("/api/library/upload", a.handleUpload)
	mux.HandleFunc("/api/library/remove", a.handleLibraryRemove) // 新增路由
	mux.HandleFunc("/api/playlist/add", a.handlePlaylistAdd)
	// ... 其他播放控制API
	mux.HandleFunc("/api/player/play", a.handlePlay)
	mux.HandleFunc("/api/player/pause", a.handlePause)
	mux.HandleFunc("/api/player/next", a.handleNext)
	mux.HandleFunc("/api/player/prev", a.handlePrev)
	mux.HandleFunc("/api/player/seek", a.handleSeek)

	// Static files
	mux.Handle("/static/audio/", http.StripPrefix("/static/audio/", http.FileServer(http.Dir(a.mediaDir))))
}

func (a *API) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 传递一个函数，当新用户连接时，会调用此函数获取当前状态并发送
	a.hub.ServeWs(w, r, a.state.GetFullState)
}

func (a *API) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}
	valid, err := a.db.IsTokenValid(token)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
}

func (a *API) handleGetLibrary(w http.ResponseWriter, r *http.Request) {
	songs, err := a.db.GetAllSongs()
	if err != nil {
		http.Error(w, "Failed to get library", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(songs)
}

func (a *API) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseMultipartForm(100 << 20) // 100 MB max
	file, handler, err := r.FormFile("audioFile")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	songUUID, _ := uuid.NewV4()
	ext := filepath.Ext(handler.Filename)
	newFileName := songUUID.String() + ext
	filePath := filepath.Join(a.mediaDir, newFileName)
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	// --- TODO 实现部分 ---
	// 使用 ffprobe 读取元数据
	title, artist, album, durationMs, err := getAudioMetadata(filePath)
	if err != nil {
		log.Printf("Warning: Failed to get metadata for %s: %v. Using filename as title.", handler.Filename, err)
		// 即使读取失败，也继续，只是元数据不完整
		title = strings.TrimSuffix(handler.Filename, ext)
		durationMs = 0 // 或者一个默认值
	}
	// 如果 ffprobe 没有读到标题，则使用文件名作为后备
	if title == "" {
		title = strings.TrimSuffix(handler.Filename, ext)
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
		http.Error(w, "Error adding song to database", http.StatusInternalServerError)
		os.Remove(filePath) // 如果数据库失败，删除文件
		return
	}

	log.Printf("New song uploaded: %s (Artist: %s, Duration: %dms)", song.Title, song.Artist, song.DurationMs)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(song)
}

// handleLibraryRemove 处理删除歌曲的请求
func (a *API) handleLibraryRemove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload struct {
		SongID string `json:"songId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if payload.SongID == "" {
		http.Error(w, "songId is required", http.StatusBadRequest)
		return
	}
	// 在状态管理器操作之前获取文件路径，因为之后记录就没了
	song, err := a.db.GetSong(payload.SongID)
	if err != nil {
		// 如果歌曲本就不存在，可以认为删除成功
		log.Printf("Attempted to delete non-existent song %s", payload.SongID)
		w.WriteHeader(http.StatusOK)
		return
	}
	// 调用状态管理器来处理所有逻辑
	if err := a.state.RemoveSongFromLibrary(payload.SongID); err != nil {
		http.Error(w, "Failed to remove song: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// 从文件系统删除文件
	filePath := filepath.Join(a.mediaDir, song.FilePath)
	if err := os.Remove(filePath); err != nil {
		// 即使文件删除失败，数据库和状态也已更新，所以只记录错误
		log.Printf("Warning: failed to delete audio file %s: %v", filePath, err)
	}
	// 状态管理器已经广播了状态更新，这里只需返回成功
	w.WriteHeader(http.StatusOK)
}

func (a *API) handlePlaylistAdd(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SongID string `json:"songId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := a.state.AddToPlaylist(payload.SongID); err != nil {
		http.Error(w, "Failed to add song to playlist", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// --- Player Controls ---
func (a *API) handlePlay(w http.ResponseWriter, r *http.Request) {
	a.state.Play()
	w.WriteHeader(http.StatusAccepted)
}
func (a *API) handlePause(w http.ResponseWriter, r *http.Request) {
	a.state.Pause()
	w.WriteHeader(http.StatusAccepted)
}
func (a *API) handleNext(w http.ResponseWriter, r *http.Request) {
	a.state.NextSong()
	w.WriteHeader(http.StatusAccepted)
}
func (a *API) handlePrev(w http.ResponseWriter, r *http.Request) {
	a.state.PrevSong()
	w.WriteHeader(http.StatusAccepted)
}

func (a *API) handleSeek(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload SeekPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.state.Seek(payload.PositionMs); err != nil {
		// This error is returned if no song is playing.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}
