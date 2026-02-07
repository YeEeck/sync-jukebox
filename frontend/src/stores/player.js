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
        onlineUsers: [],
        onlineUsersIntervalId: null, // 用于存储定时器ID
    }),

    getters: {
        // ... getters 保持不变 ...
        currentSongUrl: (state) => {
            if (state.currentSong && state.currentSong.id) {
                return `/static/audio/${state.currentSong.id}/index.m3u8`;
            }
            return null;
        },
        onlineUserCount: (state) => state.onlineUsers.length,
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

        async checkUsernameExists(username) {
            try {
                // 调用新的后端端点
                const response = await fetch(`/api/users/check?username=${encodeURIComponent(username)}`);
                if (!response.ok) {
                    // 如果请求失败，为简单起见，我们假设用户名不可用或检查失败
                    console.error('Username check failed:', response.statusText);
                    return true; // 谨慎起见，返回 true 以阻止注册
                }
                const data = await response.json();
                return data.exists; // 返回后端返回的 boolean 值
            } catch (error) {
                console.error('Error checking username:', error);
                return true; // 网络错误等，同样返回 true 阻止注册
            }
        },
        
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
                websocketService.connect(username, password);
                this.fetchLibrary();
                this.startFetchingOnlineUsers();
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
            this.stopFetchingOnlineUsers();
        },
        async initializeAuthAndConnect() {
            if (!this.authHeader) {
                this.isAuthenticated = false;
                return false;
            }
            try {
                console.log('Initializing session with stored credentials...');
                const base64Credentials = this.authHeader.split(' ')[1];
                if (!base64Credentials) {
                    throw new Error("Invalid auth header in storage");
                }
                const decodedCredentials = atob(base64Credentials);
                const [username, password] = decodedCredentials.split(':', 2); // split into max 2 parts
                if (!username || password === undefined) {
                    throw new Error("Could not decode credentials from storage");
                }

                // Pass the decoded credentials to the websocket service
                websocketService.connect(username, password);
                
                await this.fetchLibrary();
                this.isAuthenticated = true;
                this.startFetchingOnlineUsers();
                console.log('Session restored successfully.');
                return true;
            } catch (error) {
                console.error('Failed to restore session:', error);
                this.logout();
                return false;
            }
        },

        // --- 新增 Actions ---
        async fetchOnlineUsers() {
            try {
                const response = await api.getOnlineUsers();
                this.onlineUsers = response.data;
            } catch (error) {
                console.error('Failed to fetch online users:', error);
                this.onlineUsers = []; // 出错时清空列表
            }
        },
        startFetchingOnlineUsers() {
            // 先立即执行一次
            this.fetchOnlineUsers();
            // 如果已有定时器，先清除
            if (this.onlineUsersIntervalId) {
                clearInterval(this.onlineUsersIntervalId);
            }
            // 设置定时器，每15秒获取一次
            this.onlineUsersIntervalId = setInterval(this.fetchOnlineUsers, 15000);
        },
        stopFetchingOnlineUsers() {
            if (this.onlineUsersIntervalId) {
                clearInterval(this.onlineUsersIntervalId);
                this.onlineUsersIntervalId = null;
            }
            this.onlineUsers = []; // 清空用户列表
        },

        // ... 其他所有 API 调用 actions 保持不变 ...
        play() {
            api.play();
        },
        async playSpecificSong(songId) {
            try {
                await api.playSpecific(songId);
            } catch (error) {
                console.error('Failed to play specific song:', error);
            }
        },
        pause() {
            api.pause();
        },
        next() {
            api.next();
        },
        prev() {
            api.prev();
        },
        seekTo(positionMs) {
            api.seek(positionMs);
        },
        async addToPlaylist(songId) {
            try {
                await api.addToPlaylist(songId);
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
        async movePlaylistItem(songId, newIndex) {
            try {
                await api.movePlaylistItem(songId, newIndex);
            } catch (error) {
                console.error('Failed to reorder playlist:', error);
            }
        },
        async shufflePlaylist() {
            try {
                await api.shufflePlaylist();
            } catch (error) {
                console.error('Failed to shuffle playlist:', error);
            }
        },
        async removeSongFromPlaylist(songId) {
            try {
                await api.removeFromPlaylist(songId);
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
                await api.uploadSong(formData);
            } catch (error) {
                console.error('Failed to upload song:', error);
                throw error;
            }
        },
        async removeSongFromLibrary(songId) {
            try {
                await api.removeSong(songId);
            } catch (error) {
                console.error('Failed to remove song:', error);
            }
        },
        setLocalVolume(newVolume) {
            const clampedVolume = Math.max(0, Math.min(1, newVolume));
            this.localVolume = clampedVolume;
            localStorage.setItem(VOLUME_STORAGE_KEY, clampedVolume.toString());
        },
        toggleMute() {
            if (this.localVolume > 0) {
                this.previousVolume = this.localVolume;
                this.setLocalVolume(0);
            } else {
                const targetVolume = (this.previousVolume && this.previousVolume > 0) ? this.previousVolume : 0.5;
                this.setLocalVolume(targetVolume);
            }
        }
    },
});
