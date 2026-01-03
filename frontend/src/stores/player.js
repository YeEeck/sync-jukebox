import {defineStore} from 'pinia';
// api 和 websocketService 的导入保持不变
import api from '@/api';
import {websocketService} from '@/services/websocket';

const VOLUME_STORAGE_KEY = 'jukebox_volume';
const AUTH_HEADER_STORAGE_KEY = 'jukebox_auth_header';

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
