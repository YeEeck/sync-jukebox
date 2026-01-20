import { defineStore } from 'pinia';
import api from '@/api';
import { websocketService } from '@/services/websocket';

const VOLUME_STORAGE_KEY = 'jukebox_volume';

// --- 辅助函数：从 localStorage 安全地加载音量 ---
const loadInitialVolume = () => {
  const savedVolume = localStorage.getItem(VOLUME_STORAGE_KEY);
  // 检查 savedVolume 是否为 null 或 undefined。
  // 如果直接用 parseFloat(savedVolume) || 0.5，当保存的音量是 0 时，会被错误地重置为 0.5。
  if (savedVolume !== null) {
    return parseFloat(savedVolume);
  }
  return 0.5; // 默认音量
};

export const usePlayerStore = defineStore('player', {
  // State: 镜像后端的 GlobalState，并增加一些前端自身的状态
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
    isAuthenticated: false,
    mediaLibrary: [],
    localVolume: loadInitialVolume(),
    // 用于记录静音前的音量
    previousVolume: null,
    playbackError: null,
  }),

  getters: {
    // 计算当前歌曲的音频文件URL
    currentSongUrl: (state) => {
      // 确保 currentSong 存在且有 filePath 属性
      // console.log('Current Song:', state.currentSong);
      if (state.currentSong && state.currentSong.id) {
        return `/static/audio/${state.currentSong.id}/index.m3u8`;
      }
      return null;
    },
  },

  actions: {
    // --- 核心 Action，用于接收后端推送 ---
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
    async validateTokenAndConnect(token) {
      try {
        const response = await api.validateToken(token);
        if (response.data.valid) {
          this.isAuthenticated = true;
          localStorage.setItem('jukebox_token', token); // 持久化token
          websocketService.connect();
          this.fetchLibrary(); // 连接成功后，获取一次媒体库
          return true;
        }
      } catch (error) {
        console.error('Token validation failed:', error);
      }
      this.isAuthenticated = false;
      return false;
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
