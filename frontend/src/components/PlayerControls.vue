<template>
  <div class="controls-wrapper">
    <!-- 歌曲信息 -->
    <div class="song-info">
      <div class="info-text">
        <p class="song-title">{{ store.currentSong?.title || 'No song selected' }}</p>
        <small class="song-artist">{{ store.currentSong?.artist || '...' }}</small>
      </div>
    </div>

    <!-- 主控制区 -->
    <div class="main-controls">
      <div class="buttons">
        <!-- 上一首 -->
        <button class="control-btn secondary" @click="store.prev()">
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 6h2v12H6zm3.5 6l8.5 6V6z"/></svg>
        </button>

        <!-- 播放/暂停 (大按钮) -->
        <button class="control-btn primary" @click="togglePlayPause">
          <svg v-if="store.isPlaying" viewBox="0 0 24 24" fill="currentColor"><path d="M6 19h4V5H6v14zm8-14v14h4V5h-4z"/></svg>
          <svg v-else viewBox="0 0 24 24" fill="currentColor"><path d="M8 5v14l11-7z"/></svg>
        </button>

        <!-- 下一首 -->
        <button class="control-btn secondary" @click="store.next()">
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M6 18l8.5-6L6 6v12zM16 6v12h2V6h-2z"/></svg>
        </button>
      </div>

      <!-- 播放进度条 -->
      <div class="progress-bar">
        <span class="time-text">{{ formatTime(isSeeking ? localProgressValue : store.progressMs) }}</span>
        
        <input 
          type="range" 
          class="custom-range"
          min="0" 
          :max="store.currentSong?.duration_ms || 0"
          :value="isSeeking ? localProgressValue : store.progressMs" 
          :style="getSliderStyle(isSeeking ? localProgressValue : store.progressMs, store.currentSong?.duration_ms || 0)"
          @mousedown="handleSeekStart" 
          @input="handleSeeking"
          @change="handleSeekEnd" 
        />
        
        <span class="time-text">{{ formatTime(store.currentSong?.duration_ms || 0) }}</span>
      </div>
    </div>

    <!-- 音量控制 -->
    <div class="volume-control">
      <div class="volume-icon" @click="toggleMute">
        <svg v-if="store.localVolume === 0" viewBox="0 0 24 24" fill="currentColor"><path d="M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73 4.27 3zM12 4L9.91 6.09 12 8.18V4z"/></svg>
        <svg v-else viewBox="0 0 24 24" fill="currentColor"><path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/></svg>
      </div>
      <input 
        type="range" 
        class="custom-range volume-range"
        min="0" 
        max="1" 
        step="0.01" 
        :value="store.localVolume" 
        :style="getSliderStyle(store.localVolume, 1)"
        @input="onVolumeChange" 
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { usePlayerStore } from '@/stores/player.js';
const store = usePlayerStore();

// --- 样式辅助函数: 动态计算背景渐变 ---
const getSliderStyle = (current, max) => {
  const percentage = max > 0 ? (current / max) * 100 : 0;
  return {
    background: `linear-gradient(to right, #1DB954 0%, #1DB954 ${percentage}%, #535353 ${percentage}%, #535353 100%)`
  };
};

const onVolumeChange = (event) => {
  const newVolume = parseFloat(event.target.value);
  store.setLocalVolume(newVolume);
};

// 切换静音方法
const toggleMute = () => {
  store.toggleMute();
};

const togglePlayPause = () => {
  if (store.isPlaying) {
    store.pause();
  } else {
    store.play();
  }
};

// --- 用于拖动逻辑的本地状态 ---
const isSeeking = ref(false);
const localProgressValue = ref(0);
const handleSeekStart = () => {
  isSeeking.value = true;
};
const handleSeeking = (event) => {
  localProgressValue.value = parseInt(event.target.value, 10);
};
const handleSeekEnd = (event) => {
  const newPositionMs = parseInt(event.target.value, 10);
  store.seekTo(newPositionMs);
  isSeeking.value = false;
};

const formatTime = (ms) => {
  if (!ms) return '0:00';
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
};
</script>

<style scoped>
.controls-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  background-color: #181818; /* 底栏背景 */
  border-top: 1px solid #282828;
  color: #fff;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
  height: 90px;
  box-sizing: border-box;
}

/* --- 歌曲信息 --- */
.song-info {
  flex: 1;
  min-width: 180px;
  display: flex;
  align-items: center;
}

.info-text {
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.song-title {
  font-size: 0.9rem;
  font-weight: 600;
  margin: 0;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.song-artist {
  font-size: 0.75rem;
  color: #b3b3b3;
  margin-top: 4px;
}

.song-artist:hover {
  color: #fff;
  text-decoration: underline;
  cursor: pointer;
}

/* --- 主控制区 --- */
.main-controls {
  flex: 2;
  max-width: 722px; /* 限制最大宽度以免拉得太长 */
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.buttons {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  margin-bottom: 4px;
}

.control-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #b3b3b3;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  padding: 0;
}

.control-btn svg {
  width: 24px; /* 默认图标大小 */
  height: 24px;
}

.control-btn:hover {
  color: #fff;
}

.control-btn:active {
  transform: scale(0.95);
}

/* 播放暂停主按钮样式 */
.control-btn.primary {
  color: #000;
  background-color: #fff;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  transition: transform 0.1s ease;
}

.control-btn.primary:hover {
  transform: scale(1.05);
  color: #000;
}
.control-btn.primary svg {
  width: 16px; /* 内部图标稍小一点 */
  height: 16px;
}

.control-btn.secondary svg {
  width: 18px; /* 前后切歌按钮大小 */
  height: 18px;
}


/* --- 进度条区域 --- */
.progress-bar {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 0.5rem;
}

.time-text {
  font-size: 0.7rem;
  color: #a7a7a7;
  font-variant-numeric: tabular-nums;
  min-width: 35px;
  text-align: center;
}

/* --- 自定义 Input Range 样式 (核心) --- */
.custom-range {
  appearance: none; /* 清除默认样式 */
  width: 100%;
  height: 4px;
  border-radius: 2px;
  background: #535353; /* 默认未填充背景，但在 JS 中会被覆盖 */
  outline: none;
  cursor: pointer;
  flex-grow: 1;
}

/* 滑块头 (Thumb) - Chrome/Safari/Edge */
.custom-range::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  cursor: pointer;
  opacity: 0; /* 默认隐藏滑块，像 Spotify 一样 */
  transition: opacity 0.2s;
}

/* 鼠标悬停进度条时显示滑块头 */
.custom-range:hover::-webkit-slider-thumb {
  opacity: 1;
}

/* 轨道进度颜色 - Firefox (Firefox需要单独设置) */
.custom-range::-moz-range-progress {
  background-color: #1DB954; 
  height: 4px;
  border-radius: 2px;
}
.custom-range::-moz-range-track {
  background-color: #535353;
  height: 4px;
  border-radius: 2px;
}
.custom-range::-moz-range-thumb {
  width: 12px;
  height: 12px;
  border: none;
  border-radius: 50%;
  background: #fff;
  opacity: 0;
  transition: opacity 0.2s;
}
.custom-range:hover::-moz-range-thumb {
  opacity: 1;
}

/* --- 音量控制 --- */
.volume-control {
  flex: 1;
  min-width: 180px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 0.5rem;
}

.volume-icon {
  color: #b3b3b3;
  width: 18px;
  height: 18px;
  /* 鼠标手势与悬停效果 */
  cursor: pointer;
  transition: color 0.2s;
}

.volume-icon:hover {
  color: #fff;
}

.volume-icon svg {
  width: 100%;
  height: 100%;
}

.volume-range {
  max-width: 100px; /* 音量条不需要太长 */
}

/* 响应式适配 */
@media (max-width: 768px) {
  .song-info, .volume-control {
    display: none; /* 移动端可以隐藏左右两侧，只留中间 */
  }
}
</style>
