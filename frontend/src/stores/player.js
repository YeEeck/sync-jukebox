import { defineStore } from 'pinia';
import api from '@/api';
import { websocketService } from '@/services/websocket';

const VOLUME_STORAGE_KEY = 'jukebox_volume';
const AUTH_HEADER_STORAGE_KEY = 'jukebox_auth_header';

// --- 辅助函数：从 localStorage 安全地加载音量 ---
const loadInitialVolume = () => {
    const savedVolume = localStorage.getItem(VOLUME_STORAGE_KEY);
    return savedVolume !== null ? parseFloat(savedVolume) : 0.5;
};

export const usePlayerStore = defineStore('player', {
    state: () => ({
        // ... 其他状态保持不变 ...
        isPlaying: false,
        currentSongId: null,
        currentSong: null,
        playlist: [],
        currentPlaylistIdx: -1,
        progressMs: 0,
        playMode: 'REPEAT_ALL',
        isAuthenticated: !!localStorage.getItem(AUTH_HEADER_STORAGE_KEY),
        authHeader: localStorage.getItem(AUTH_HEADER_STORAGE_KEY) || null,
        authError: null,
        mediaLibrary: [],
        localVolume: loadInitialVolume(),
        previousVolume: null,
        playbackError: null,
    }),

    getters: {
        // ... getters 保持不变 ...
        currentSongUrl: (state) => {
            if (state.currentSong && state.currentSong.id) {
                return `/static/audio/${state.currentSong.id}/index.m3u8`;
            }
            return null;
        },
    },

    actions: {
        setGlobalState(newState) {
            this.isPlaying = newState.isPlaying;
            this.currentSongId = newState.currentSongId;
            this.currentSong = newState.currentSong;
            this.playlist = newState.playlist;
            this.currentPlaylistIdx = newState.currentPlaylistIdx;
            this.progressMs = newState.progressMs;
            this.playMode = newState.playMode;
        },

        // --- 认证与连接 ---

        // 修改: 接收 invitationKey
        async register(username, password, invitationKey) {
            this.authError = null;
            try {
                const response = await fetch('/api/register', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    // 修改: 在请求体中包含 key
                    body: JSON.stringify({username, password, key: invitationKey}),
                });
                const data = await response.json();
                if (!response.ok) {
                    throw new Error(data.error || 'Registration failed');
                }
                return {success: true, message: data.message};
            } catch (error) {
                this.authError = error.message;
                return {success: false, message: error.message};
            }
        },

        // loginAndConnect, logout, initializeAuthAndConnect 等其他 actions 保持不变
        async loginAndConnect(username, password) {
            this.authError = null;
            const credentials = btoa(`${username}:${password}`);
            const authHeader = `Basic ${credentials}`;

            try {
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {'Authorization': authHeader},
                });
                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.error || 'Authentication failed');
                }
                this.authHeader = authHeader;
                this.isAuthenticated = true;
                localStorage.setItem(AUTH_HEADER_STORAGE_KEY, authHeader);
                websocketService.connect(credentials);
                this.fetchLibrary();
                return true;
            } catch (error) {
                this.authError = error.message;
                this.logout();
                return false;
            }
        },
        logout() {
            this.isAuthenticated = false;
            this.authHeader = null;
            this.authError = null;
            localStorage.removeItem(AUTH_HEADER_STORAGE_KEY);
            websocketService.disconnect();
        },
        async initializeAuthAndConnect() {
            if (!this.authHeader) {
                this.isAuthenticated = false;
                return false;
            }
            try {
                console.log('Initializing session with stored credentials...');
                const base64Credentials = this.authHeader.split(' ')[1];
                websocketService.connect(base64Credentials);
                await this.fetchLibrary();
                this.isAuthenticated = true;
                console.log('Session restored successfully.');
                return true;
            } catch (error) {
                console.error('Failed to restore session:', error);
                this.logout();
                return false;
            }
        },

    // --- 调用 HTTP API 的 Actions (Fire and Forget) ---
    play() { api.play(); },
    // 播放指定 ID 的歌曲
    async playSpecificSong(songId) {
      try {
        await api.playSpecific(songId);
      } catch (error) {
        console.error('Failed to play specific song:', error);
      }
    },
    pause() { api.pause(); },
    next() { api.next(); },
    prev() { api.prev(); },
    seekTo(positionMs) {
      // "Fire and Forget"
      // 我们发送指令，然后等待 WebSocket 推送校准后的进度
      api.seek(positionMs);
    },

    async addToPlaylist(songId) {
      try {
        await api.addToPlaylist(songId);
        // 无需手动更新 state，等待 WebSocket 推送
      } catch (error) {
        console.error('Failed to add song to playlist:', error);
      }
    },

    // 将歌曲添加到当前播放歌曲的下一首
    async addSongNextInPlaylist(songId) {
      try {
        // 假设存在一个 API 端点，它接收歌曲 ID
        // 后端逻辑会找到当前播放歌曲的索引，并将新歌曲插入到 그 索引 + 1 的位置
        await api.addNextToPlaylist(songId);
      } catch (error) {
        console.error('Failed to add song next in playlist:', error);
      }
    },

    // 调整播放列表顺序
    async movePlaylistItem(songId, newIndex) {
      try {
        // 先调用 API，状态更新依赖 WebSocket 推送，保持前端状态单一数据源
        await api.movePlaylistItem(songId, newIndex);
      } catch (error) {
        console.error('Failed to reorder playlist:', error);
      }
    },

    async shufflePlaylist() {
      try {
        await api.shufflePlaylist();
        // 同样不需要手动更新 state，等待 WebSocket 推送新的 GlobalState
      } catch (error) {
        console.error('Failed to shuffle playlist:', error);
      }
    },

    async removeSongFromPlaylist(songId) {
      try {
        await api.removeFromPlaylist(songId);
        // 无需手动更新 state.playlist，依赖 WebSocket 推送
      } catch (error) {
        console.error('Failed to remove song from playlist:', error);
      }
    },

    async fetchLibrary() {
      try {
        const response = await api.getLibrary();
        this.mediaLibrary = response.data;
      } catch (error) {
        console.error('Failed to fetch library:', error);
      }
    },

    async uploadSong(file) {
      const formData = new FormData();
      formData.append('audioFile', file);
      try {
        // 原来的代码在这里会调用 api.uploadSong 和 this.fetchLibrary()
        // 我们将 fetchLibrary() 移除，让调用方（组件）来决定何时刷新
        await api.uploadSong(formData);
        // this.fetchLibrary(); // <-- 移除这一行
      } catch (error) {
        console.error('Failed to upload song:', error);
        // 抛出错误，以便组件可以捕获并处理
        throw error;
      }
    },

    async removeSongFromLibrary(songId) {
      try {
        await api.removeSong(songId);
        // "Fire and Forget" - 无需手动修改 state
        // 后端会处理删除，并通过 WebSocket 推送最新的 mediaLibrary 和 playlist
      } catch (error) {
        console.error('Failed to remove song:', error);
        // 可以在此添加用户错误提示
      }
    },

    setLocalVolume(newVolume) {
      // 增加一个安全边界，确保音量值在 0 和 1 之间
      const clampedVolume = Math.max(0, Math.min(1, newVolume));

      this.localVolume = clampedVolume;

      // --- 将新音量保存到 localStorage ---
      localStorage.setItem(VOLUME_STORAGE_KEY, clampedVolume.toString());
    },

    // 切换静音/恢复音量
    toggleMute() {
      if (this.localVolume > 0) {
        // 当前有声音，记录当前音量并静音
        this.previousVolume = this.localVolume;
        this.setLocalVolume(0);
      } else {
        // 当前是静音，恢复音量
        // 如果有记录且记录大于0，则恢复记录值；否则默认恢复到 0.5
        const targetVolume = (this.previousVolume && this.previousVolume > 0)
          ? this.previousVolume
          : 0.5;
        this.setLocalVolume(targetVolume);
      }
    }
  },
});
