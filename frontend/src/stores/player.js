import {defineStore} from 'pinia';
import api from '@/api'; // 假设你的api服务在这里
import {websocketService} from '@/services/websocket'; // 假设你的websocket服务在这里

const VOLUME_STORAGE_KEY = 'jukebox_volume';
const AUTH_HEADER_STORAGE_KEY = 'jukebox_auth_header'; // 新增：用于存储认证头

const loadInitialVolume = () => {
    const savedVolume = localStorage.getItem(VOLUME_STORAGE_KEY);
    if (savedVolume !== null) {
        return parseFloat(savedVolume);
    }
    return 0.5;
};

export const usePlayerStore = defineStore('player', {
    state: () => ({
        // 后端同步的状态
        isPlaying: false,
        currentSongId: null,
        currentSong: null,
        playlist: [],
        currentPlaylistIdx: -1,
        progressMs: 0,
        playMode: 'REPEAT_ALL',
        // 前端自身的状态
        isAuthenticated: !!localStorage.getItem(AUTH_HEADER_STORAGE_KEY), // 根据是否存在authHeader初始化
        authHeader: localStorage.getItem(AUTH_HEADER_STORAGE_KEY) || null, // 从localStorage加载
        authError: null, // 用于存储登录/注册错误信息
        mediaLibrary: [],
        localVolume: loadInitialVolume(),
        previousVolume: null,
        playbackError: null,
    }),

    getters: {
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

        async register(username, password) {
            this.authError = null;
            try {
                // 注意：注册是一个公开接口，不需要认证头
                const response = await fetch('/api/register', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({username, password}),
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

        async loginAndConnect(username, password) {
            this.authError = null;
            // 1. 创建 Basic Auth 凭证
            const credentials = btoa(`${username}:${password}`); // Base64 编码
            const authHeader = `Basic ${credentials}`;

            try {
                // 2. 使用 /api/login 验证凭证
                const response = await fetch('/api/login', {
                    method: 'POST',
                    headers: {'Authorization': authHeader},
                });

                if (!response.ok) {
                    const data = await response.json();
                    throw new Error(data.error || 'Authentication failed');
                }

                // 3. 认证成功, 保存状态并连接
                this.authHeader = authHeader;
                this.isAuthenticated = true;
                localStorage.setItem(AUTH_HEADER_STORAGE_KEY, authHeader);

                // **重要**: 通知你的 api 和 websocket 服务更新认证信息
                websocketService.connect(credentials); // 传递凭证给websocket服务

                this.fetchLibrary();
                return true;

            } catch (error) {
                this.authError = error.message;
                this.logout(); // 登录失败时清理状态
                return false;
            }
        },

        logout() {
            this.isAuthenticated = false;
            this.authHeader = null;
            this.authError = null;
            localStorage.removeItem(AUTH_HEADER_STORAGE_KEY);
            websocketService.disconnect();
            // 可以在这里添加路由跳转到登录页
        },

        async initializeAuthAndConnect() {
            // 这个 action 在应用加载时被调用，以恢复会话
            if (!this.authHeader) {
                this.isAuthenticated = false;
                return false;
            }
            try {
                console.log('Initializing session with stored credentials...');
                const base64Credentials = this.authHeader.split(' ')[1];
                websocketService.connect(base64Credentials);
                // 2. 通过获取媒体库来验证凭证是否仍然有效
                await this.fetchLibrary();

                // 3. 如果成功，确认状态
                this.isAuthenticated = true;
                console.log('Session restored successfully.');
                return true;
            } catch (error) {
                console.error('Failed to restore session:', error);
                // 如果凭证失效（例如，服务器重启、用户被删除等），则登出
                this.logout();
                return false;
            }
        },


        // --- 调用 HTTP API 的 Actions  ---
        // ... 通过已配置认证的 api 服务自动发送 auth header ...
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
