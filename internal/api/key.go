package api

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
)

// InvitationKeyManager 负责生成、存储和验证注册邀请密钥。
// 它的字段是小写的，意味着它们是私有的，只能通过导出的方法访问。
type InvitationKeyManager struct {
	mu       sync.RWMutex
	key      string
	filePath string
}

// NewInvitationKeyManager 创建一个新的密钥管理器实例。
func NewInvitationKeyManager(filePath string) *InvitationKeyManager {
	km := &InvitationKeyManager{
		filePath: filePath,
	}
	// 尝试从文件加载现有密钥
	if err := km.loadKeyFromFile(); err != nil {
		// 如果加载失败（例如，文件不存在），则生成一个新密钥
		log.Printf("Could not load key from file ('%s'). Generating a new one.", err)
		if _, genErr := km.GenerateNewKey(); genErr != nil {
			// 这是一个严重问题，如果连初始密钥都无法生成和保存，程序应该中止
			log.Fatalf("FATAL: Failed to generate and save initial invitation key: %v", genErr)
		}
	} else {
		log.Printf("🔑 Invitation key successfully loaded from %s", filePath)
	}
	return km
}

// GenerateNewKey 生成一个新的、安全的随机密钥并存储它。
// 它会覆盖任何现有的密钥。
func (km *InvitationKeyManager) GenerateNewKey() (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	newKey := base64.URLEncoding.EncodeToString(bytes)
	km.key = newKey
	// --- 新增: 将新密钥保存到文件 ---
	if err := km.saveKeyToFile(newKey); err != nil {
		// 返回错误，让调用者知道持久化失败
		return "", err
	}
	log.Printf("🔑 New invitation key generated and saved: %s", newKey)
	return newKey, nil
}

// ValidateAndConsumeKey 验证提交的密钥。
// 如果验证成功，它会返回 true 并立即在后台生成一个新密钥，使旧密钥失效（实现“一次性”使用）。
func (km *InvitationKeyManager) ValidateAndConsumeKey(submittedKey string) bool {
	km.mu.Lock()
	defer km.mu.Unlock()
	// 检查密钥是否匹配
	if submittedKey == "" || submittedKey != km.key {
		return false
	}
	// 密钥正确！立即生成一个新密钥以使旧的失效
	log.Printf("🔑 Invitation key '%s' consumed.", submittedKey)
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("CRITICAL: Failed to generate random bytes for new key after consumption: %v", err)
		// 在这种罕见的失败情况下，我们保留旧密钥以避免系统没有密钥
		return true // 尽管生成失败，但本次验证是成功的
	}
	newKey := base64.URLEncoding.EncodeToString(bytes)
	km.key = newKey
	// --- 新增: 将消耗后生成的新密钥保存到文件 ---
	if err := km.saveKeyToFile(newKey); err != nil {
		log.Printf("CRITICAL: Failed to save new key after consumption: %v", err)
	}
	log.Printf("🔑 New key generated and saved after consumption: %s", newKey)
	return true
}

// --- 从文件加载密钥的私有方法 ---
func (km *InvitationKeyManager) loadKeyFromFile() error {
	km.mu.Lock()
	defer km.mu.Unlock()
	data, err := os.ReadFile(km.filePath)
	if err != nil {
		return err // 例如 os.ErrNotExist
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return errors.New("key file is empty")
	}
	km.key = key
	return nil
}

// --- 将密钥保存到文件的私有方法 ---
func (km *InvitationKeyManager) saveKeyToFile(key string) error {
	// 使用 0600 权限，确保只有所有者可以读写该文件
	return os.WriteFile(km.filePath, []byte(key), 0600)
}
