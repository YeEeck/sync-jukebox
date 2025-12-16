package state

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/yeeeck/sync-jukebox/internal/db"
	"github.com/yeeeck/sync-jukebox/internal/websocket"
)

// PlayMode 定义播放模式
type PlayMode string

const (
	RepeatAll PlayMode = "REPEAT_ALL"
	RepeatOne PlayMode = "REPEAT_ONE"
	Shuffle   PlayMode = "SHUFFLE"
)

// GlobalState 是应用唯一的实时状态来源
type GlobalState struct {
	IsPlaying          bool              `json:"isPlaying"`
	CurrentSongID      string            `json:"currentSongId"`
	CurrentSong        *db.Song          `json:"currentSong"`
	Playlist           []db.PlaylistItem `json:"playlist"`
	CurrentPlaylistIdx int               `json:"currentPlaylistIdx"`
	ProgressMs         int64             `json:"progressMs"` // 当前歌曲播放进度
	LastUpdate         time.Time         `json:"-"`          // 服务端进度更新时间
	PlayMode           PlayMode          `json:"playMode"`
}

// Manager 封装了状态以及其依赖
type Manager struct {
	State  *GlobalState
	db     *db.DB
	hub    *websocket.Hub
	mu     sync.RWMutex
	ticker *time.Ticker
}

// NewManager 创建并从数据库加载状态
func NewManager(db *db.DB, hub *websocket.Hub) (*Manager, error) {
	m := &Manager{
		State: &GlobalState{
			IsPlaying: false,
			PlayMode:  RepeatAll,
		},
		db:  db,
		hub: hub,
	}
	if err := m.loadFromDB(); err != nil {
		return nil, err
	}
	log.Println("State manager initialized and loaded from DB.")
	return m, nil
}

func (m *Manager) loadFromDB() error {
	// 加载播放列表
	playlist, err := m.db.GetPlaylistItems()
	if err != nil {
		return err
	}
	m.State.Playlist = playlist

	// 加载系统状态
	m.State.CurrentSongID, _ = m.db.GetSystemState("current_song_id")
	isPlayingStr, _ := m.db.GetSystemState("is_playing")
	m.State.IsPlaying = isPlayingStr == "true"

	progressStr, _ := m.db.GetSystemState("progress_ms")
	progress, _ := strconv.ParseInt(progressStr, 10, 64)
	m.State.ProgressMs = progress

	lastUpdateStr, _ := m.db.GetSystemState("last_update_unix")
	lastUpdateUnix, _ := strconv.ParseInt(lastUpdateStr, 10, 64)

	// 计算自上次保存以来的进度
	if m.State.IsPlaying && lastUpdateUnix > 0 {
		elapsed := time.Now().Unix() - lastUpdateUnix
		m.State.ProgressMs += elapsed * 1000
	}

	// 找到当前歌曲在播放列表中的索引
	for i, item := range m.State.Playlist {
		if item.SongID == m.State.CurrentSongID {
			m.State.CurrentPlaylistIdx = i
			m.State.CurrentSong = item.Song
			break
		}
	}

	if m.State.IsPlaying {
		m.startProgressTicker()
	}

	return nil
}

// GetFullState 返回当前状态的副本，用于新连接
func (m *Manager) GetFullState() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.State
}

// --- 核心操作方法 ---
// 遵循 "更新内存 -> 更新DB -> 触发广播" 的原子流程

func (m *Manager) Play() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.State.IsPlaying {
		return
	}
	if len(m.State.Playlist) == 0 {
		return
	}

	// 将 IsPlaying 状态设置为 true
	m.State.IsPlaying = true
	// 重置 LastUpdate 时间戳，从现在开始计算播放时长
	m.State.LastUpdate = time.Now()
	// 重新启动进度更新定时器
	m.startProgressTicker()
	// 持久化当前状态到数据库
	m.db.SetSystemState("is_playing", "true")
	m.db.SetSystemState("last_update_unix", strconv.FormatInt(m.State.LastUpdate.Unix(), 10))
	// 通过 WebSocket 广播状态更新
	m.hub.Broadcast(m.State)
	log.Println("Action: Play")
}

func (m *Manager) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 如果当前没有在播放，则直接返回
	if !m.State.IsPlaying {
		return
	}
	// 停止进度更新定时器
	m.stopProgressTicker() // 假设存在一个停止定时器的函数
	// 核心修复：
	// 1. 计算从上次更新到现在的增量时间并累加到进度中
	elapsed := time.Since(m.State.LastUpdate).Milliseconds()
	m.State.ProgressMs += elapsed
	// 2. 将 IsPlaying 状态设置为 false
	m.State.IsPlaying = false
	// 3. 更新 LastUpdate 时间戳，为下一次播放做准备
	m.State.LastUpdate = time.Now()
	// 持久化当前状态到数据库
	m.db.SetSystemState("is_playing", "false")
	m.db.SetSystemState("progress_ms", strconv.FormatInt(m.State.ProgressMs, 10))
	m.db.SetSystemState("last_update_unix", strconv.FormatInt(m.State.LastUpdate.Unix(), 10))
	// 通过 WebSocket 广播状态更新
	m.hub.Broadcast(m.State)
	log.Println("Action: Pause")
}

func (m *Manager) NextSong() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.State.Playlist) == 0 {
		m.stopPlayback()
		return
	}

	// TODO: 实现不同播放模式的逻辑
	nextIdx := (m.State.CurrentPlaylistIdx + 1) % len(m.State.Playlist)

	m.changeSong(nextIdx)
	log.Println("Action: Next Song")
}

func (m *Manager) PrevSong() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.State.Playlist) == 0 {
		m.stopPlayback()
		return
	}

	nextIdx := (m.State.CurrentPlaylistIdx - 1 + len(m.State.Playlist)) % len(m.State.Playlist)

	m.changeSong(nextIdx)
	log.Println("Action: Previous Song")
}

func (m *Manager) AddToPlaylist(songID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	song, err := m.db.GetSong(songID)
	if err != nil {
		return err
	}

	// 检查是否已在播放列表
	for _, item := range m.State.Playlist {
		if item.SongID == songID {
			return nil // 已存在，不重复添加
		}
	}

	newOrderItem := db.PlaylistItem{
		SongID: songID,
		Order:  len(m.State.Playlist),
		Song:   song,
	}
	m.State.Playlist = append(m.State.Playlist, newOrderItem)

	// 更新数据库
	var songIDs []string
	for _, item := range m.State.Playlist {
		songIDs = append(songIDs, item.SongID)
	}
	m.db.UpdatePlaylist(songIDs)

	// 如果这是第一首歌，自动开始播放
	if len(m.State.Playlist) == 1 {
		m.changeSong(0)
	}

	m.hub.Broadcast(m.State)
	log.Printf("Action: Add to playlist, songId: %s", songID)
	return nil
}

// RemoveFromPlaylist removes a song from the playlist and updates the state
func (m *Manager) RemoveFromPlaylist(songID string) error {
	m.mu.Lock()
	isPlayingDeletedSong := m.State.CurrentSong != nil && m.State.CurrentSong.ID == songID
	m.mu.Unlock()

	if isPlayingDeletedSong {
		// 调用播放器的 Next 方法切歌
		// 如果切歌失败（比如列表只有这一首了），Next() 通常会处理为停止播放
		m.NextSong()
		// 切歌后，稍微等待一下或确认状态更新，确保 CurrentSong 已经变了
	}

	// 从数据库删除
	if err := m.db.RemoveSongFromPlaylist(songID); err != nil {
		return err
	}

	// 更新内存状态
	m.mu.Lock()
	defer m.mu.Unlock()
	newPlaylist := make([]db.PlaylistItem, 0)
	for _, item := range m.State.Playlist {
		// 过滤掉匹配 songID 的项
		if item.SongID != songID {
			newPlaylist = append(newPlaylist, item)
		}
	}
	m.State.Playlist = newPlaylist

	// 更新最后修改时间，触发前端同步（假设有相关逻辑）
	m.State.LastUpdate = time.Now()

	return nil
}

// --- 内部辅助方法 ---

func (m *Manager) changeSong(playlistIndex int) {
	// 这个方法假设锁已经被持有
	item := m.State.Playlist[playlistIndex]
	m.State.CurrentPlaylistIdx = playlistIndex
	m.State.CurrentSongID = item.SongID
	m.State.CurrentSong = item.Song
	m.State.ProgressMs = 0
	m.State.LastUpdate = time.Now()

	if !m.State.IsPlaying {
		m.State.IsPlaying = true
		m.startProgressTicker()
	}

	// 持久化
	m.db.SetSystemState("current_song_id", m.State.CurrentSongID)
	m.db.SetSystemState("progress_ms", "0")
	m.db.SetSystemState("last_update_unix", strconv.FormatInt(m.State.LastUpdate.Unix(), 10))
	m.db.SetSystemState("is_playing", "true")

	m.hub.Broadcast(m.State)
}

func (m *Manager) stopPlayback() {
	// 假设锁已被持有
	m.stopProgressTicker()
	m.State.IsPlaying = false
	m.State.CurrentSongID = ""
	m.State.CurrentSong = nil
	m.State.ProgressMs = 0

	m.db.SetSystemState("is_playing", "false")
	m.db.SetSystemState("current_song_id", "")
	m.db.SetSystemState("progress_ms", "0")

	m.hub.Broadcast(m.State)
}

func (m *Manager) startProgressTicker() {
	if m.ticker != nil {
		return
	}
	m.ticker = time.NewTicker(1 * time.Second)
	go func() {
		for range m.ticker.C {
			m.mu.Lock()
			if !m.State.IsPlaying {
				m.mu.Unlock()
				return
			}
			m.State.ProgressMs += 1000

			// 如果歌曲结束，自动下一首
			if m.State.CurrentSong != nil && m.State.ProgressMs >= int64(m.State.CurrentSong.DurationMs) {
				// 调用内部的next方法，避免死锁
				if len(m.State.Playlist) > 0 {
					nextIdx := (m.State.CurrentPlaylistIdx + 1) % len(m.State.Playlist)
					m.changeSong(nextIdx)
				} else {
					m.stopPlayback()
				}
			}
			m.mu.Unlock()

			// 定期广播，减少频率以降低网络负载
			// 这里我们每秒都广播，以便进度条平滑
			m.hub.Broadcast(m.State)
		}
	}()
}

func (m *Manager) stopProgressTicker() {
	if m.ticker != nil {
		m.ticker.Stop()
		m.ticker = nil
	}
}

// RemoveSongFromLibrary 处理从媒体库删除歌曲的逻辑
func (m *Manager) RemoveSongFromLibrary(songID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 1. 从数据库删除
	// 我们需要先获取文件路径，以便稍后删除文件
	_, err := m.db.GetSong(songID)
	if err != nil {
		return fmt.Errorf("song not found in db: %w", err)
	}
	if err := m.db.DeleteSong(songID); err != nil {
		return fmt.Errorf("failed to delete song from db: %w", err)
	}
	// 2. 从文件系统删除
	// 注意：这里的 filePath 是相对路径，需要拼接
	// 我们将在 API handler 中处理文件删除，因为它持有 mediaDir 的路径
	// 3. 更新内存中的播放列表状态
	var newPlaylist []db.PlaylistItem
	var wasPlayingRemoved bool
	var songIDs []string
	for _, item := range m.State.Playlist {
		if item.SongID != songID {
			newPlaylist = append(newPlaylist, item)
			songIDs = append(songIDs, item.SongID)
		} else {
			// 标记被删除的歌曲是否是当前正在播放的
			if m.State.CurrentSongID == songID {
				wasPlayingRemoved = true
			}
		}
	}
	// 如果播放列表发生了变化
	if len(newPlaylist) != len(m.State.Playlist) {
		m.State.Playlist = newPlaylist
		// 更新数据库中的播放列表
		m.db.UpdatePlaylist(songIDs)
		if wasPlayingRemoved {
			// 如果被删除的是当前歌曲，则播放下一首
			if len(m.State.Playlist) > 0 {
				// 播放当前索引的歌曲（它现在是新的歌曲了），或者从头开始
				nextIdx := m.State.CurrentPlaylistIdx
				if nextIdx >= len(m.State.Playlist) {
					nextIdx = 0
				}
				m.changeSong(nextIdx)
			} else {
				// 播放列表空了，停止播放
				m.stopPlayback()
			}
		} else {
			// 如果删除的不是当前歌曲，只需更新当前播放索引
			newIdx := -1
			for i, item := range m.State.Playlist {
				if item.SongID == m.State.CurrentSongID {
					newIdx = i
					break
				}
			}
			m.State.CurrentPlaylistIdx = newIdx
			m.hub.Broadcast(m.State) // 广播播放列表的变化
		}
	}
	log.Printf("Action: Removed song %s from library.", songID)
	// 因为状态可能已在 changeSong 或 stopPlayback 中广播，这里可以不重复广播
	// 但为了确保，广播一次总是安全的
	m.hub.Broadcast(m.State)

	return nil
}

func (m *Manager) Seek(positionMs int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.State.CurrentSong == nil {
		return fmt.Errorf("no song is currently playing")
	}
	// Clamp the position to be within the song's duration
	if positionMs < 0 {
		positionMs = 0
	}
	// Ensure duration is not zero to avoid division by zero or invalid seek
	if m.State.CurrentSong.DurationMs > 0 && positionMs > int64(m.State.CurrentSong.DurationMs) {
		positionMs = int64(m.State.CurrentSong.DurationMs)
	}
	m.State.ProgressMs = positionMs
	m.State.LastUpdate = time.Now()
	// Persist the new progress and update time
	if err := m.db.SetSystemState("progress_ms", strconv.FormatInt(positionMs, 10)); err != nil {
		// Log the error but continue to broadcast, as the in-memory state is updated
		// log.Printf("Warning: failed to persist seek progress: %v", err)
	}
	if err := m.db.SetSystemState("last_update_unix", strconv.FormatInt(m.State.LastUpdate.Unix(), 10)); err != nil {
		// log.Printf("Warning: failed to persist seek update time: %v", err)
	}
	// Broadcast the new state to all clients
	m.hub.Broadcast(m.State)
	return nil
}
