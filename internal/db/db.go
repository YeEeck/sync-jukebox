package db

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- 定义数据模型 ---

// Song 歌曲模型
type Song struct {
	ID         string `gorm:"primaryKey;type:text" json:"id"` // 对应原代码 ID TEXT PRIMARY KEY
	Title      string `gorm:"not null" json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int    `json:"duration_ms"`
	Source     string `json:"source"`
	FilePath   string `gorm:"not null;unique" json:"-"` // unique 对应原代码 UNIQUE
}

// PlaylistItem 播放列表项模型
type PlaylistItem struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	SongID string `gorm:"not null;index" json:"song_id"`  // 外键
	Order  int    `gorm:"column:item_order" json:"order"` // item_order 对应原代码 item_order

	// 关联关系：属于 Song，外键是 SongID，引用 Song 的 ID
	// OnDelete:CASCADE 对应原代码 FOREIGN KEY... ON DELETE CASCADE
	Song *Song `gorm:"foreignKey:SongID;references:ID;constraint:OnDelete:CASCADE" json:"song,omitempty"`
}

// Token 认证模型
type Token struct {
	Token     string    `gorm:"primaryKey" json:"token"`
	IsActive  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"` // 对应 DEFAULT CURRENT_TIMESTAMP
}

// SystemState 系统状态模型
type SystemState struct {
	Key   string `gorm:"primaryKey" json:"key"`
	Value string `json:"value"`
}

// --- DB 封装 ---

// DB 是数据库操作的封装
type DB struct {
	*gorm.DB
}

// New 初始化并返回一个数据库连接
func New(dataSourceName string) (*DB, error) {
	// 连接 SQLite
	gormDB, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{
		// 可选：禁用默认事务以提高性能，如果你的逻辑不需要强事务
		// SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{gormDB}

	// 自动迁移模式 (AutoMigrate)
	// GORM 会自动创建表、缺失的列和索引
	err = db.AutoMigrate(&Song{}, &PlaylistItem{}, &Token{}, &SystemState{})
	if err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return db, nil
}

// Close 获取底层连接并关闭
func (db *DB) Close() error {
	// 这里的 db.DB 指的是嵌入的 *gorm.DB
	// GORM v2 中，获取底层 sql.DB 需要调用 .DB() 方法
	// 但因为匿名嵌入，直接调用可能产生歧义。
	// 我们显式调用 gorm.DB 实例上的 DB() 方法
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// --- Token 操作 ---

func (db *DB) IsTokenValid(tokenStr string) (bool, error) {
	var token Token
	// SELECT is_active FROM tokens WHERE token = ?
	result := db.Select("is_active").First(&token, "token = ?", tokenStr)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return false, nil // Token不存在
	}
	if result.Error != nil {
		return false, result.Error
	}
	return token.IsActive, nil
}

func (db *DB) AddToken(tokenStr string) error {
	token := Token{Token: tokenStr, IsActive: true}
	// INSERT INTO tokens ...
	return db.Create(&token).Error
}

func (db *DB) SetTokenState(tokenStr string, isActive bool) error {
	// UPDATE tokens SET is_active = ? WHERE token = ?
	return db.Model(&Token{}).Where("token = ?", tokenStr).Update("is_active", isActive).Error
}

// --- System State 操作 ---

func (db *DB) GetSystemState(key string) (string, error) {
	var state SystemState
	err := db.First(&state, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil // Key不存在
	}
	return state.Value, err
}

func (db *DB) SetSystemState(key, value string) error {
	// GORM 的 Upsert (OnConflict)
	// INSERT OR REPLACE INTO system_state ...
	state := SystemState{Key: key, Value: value}
	return db.Clauses(clause.OnConflict{
		UpdateAll: true, // 如果主键冲突，更新所有列
	}).Create(&state).Error
}

// --- Song 操作 ---

func (db *DB) AddSong(song *Song) error {
	// INSERT INTO songs ...
	return db.Create(song).Error
}

func (db *DB) GetSong(id string) (*Song, error) {
	var song Song
	// SELECT * FROM songs WHERE id = ?
	err := db.First(&song, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &song, nil
}

func (db *DB) GetAllSongs() ([]Song, error) {
	var songs []Song
	// SELECT * FROM songs ORDER BY title
	result := db.Order("title").Find(&songs)
	return songs, result.Error
}

func (db *DB) DeleteSong(id string) error {
	// DELETE FROM songs WHERE id = ?
	// 注意：由于我们在 PlaylistItem 设置了 CASCADE，GORM/SQLite 会自动处理级联删除
	return db.Delete(&Song{}, "id = ?", id).Error
}

// --- Playlist 操作 ---

func (db *DB) GetPlaylistItems() ([]PlaylistItem, error) {
	var items []PlaylistItem
	// Preload("Song"): 预加载 Song 关联，相当于 SQL Join 或者先查列表再查详情
	// Order("item_order"): 按顺序排序
	err := db.Preload("Song").Order("item_order").Find(&items).Error

	if err != nil {
		return nil, err
	}

	// 过滤掉 Song 为 nil 的情况 (类似原代码中的逻辑，如果在库里找不到歌曲)
	// 虽然有了 CASCADE 外键，这种情况理论上很少发生，但为了保持逻辑一致：
	validItems := make([]PlaylistItem, 0, len(items))
	for _, item := range items {
		if item.Song != nil {
			validItems = append(validItems, item)
		} else {
			log.Printf("Warning: song %s in playlist not found in library", item.SongID)
		}
	}

	return validItems, nil
}

// UpdatePlaylist 完全重写播放列表
func (db *DB) UpdatePlaylist(songIDs []string) error {
	// 使用 GORM 的事务闭包
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. 清空当前列表
		// exec: DELETE FROM playlist_items
		// 使用 Where("1 = 1") 这是一个防止 GORM 警告全局删除的小技巧，或者使用 AllowGlobalUpdate 模式
		if err := tx.Exec("DELETE FROM playlist_items").Error; err != nil {
			return err
		}

		// 2. 批量插入
		if len(songIDs) == 0 {
			return nil
		}

		items := make([]PlaylistItem, len(songIDs))
		for i, songID := range songIDs {
			items[i] = PlaylistItem{
				SongID: songID,
				Order:  i,
			}
		}

		// INSERT INTO playlist_items ... VALUES ...
		// GORM 支持批量插入，性能较好
		if err := tx.Create(&items).Error; err != nil {
			return err
		}

		return nil // 提交事务
	})
}

// RemoveSongFromPlaylist removes a song from the playlist by its SongID
func (db *DB) RemoveSongFromPlaylist(songID string) error {
	// 假设播放列表表名为 playlist_items，模型为 PlaylistItem
	// 根据 song_id 字段删除
	return db.Where("song_id = ?", songID).Delete(&PlaylistItem{}).Error
}
