<template>
  <div class="controls-wrapper">
    <!-- 快捷键提示模态框 -->
    <KeyboardShortcutsModal :visible="showShortcutsModal" @close="showShortcutsModal = false" />
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
        <!-- NEW: 切换播放模式按钮 -->
        <button class="control-btn secondary" :class="{ active: store.playMode && store.playMode !== 'REPEAT_ALL' }" @click="togglePlayMode" :title="playModeTitle">
          <!-- 随机播放图标 -->
          <svg v-if="store.playMode === 'SHUFFLE'" viewBox="0 0 24 24" fill="currentColor">
            <path d="M10.59 9.17 5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z"/>
          </svg>
          <!-- 单曲循环图标 -->
          <svg v-else-if="store.playMode === 'REPEAT_ONE'" viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v2.5H13z"/>
          </svg>
          <!-- 列表循环图标 (默认) -->
          <svg v-else viewBox="0 0 24 24" fill="currentColor">
            <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z"/>
          </svg>
        </button>

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
// --- MODIFIED: 引入 onMounted, onUnmounted ---
import { ref, computed, onMounted, onUnmounted } from 'vue';
import { usePlayerStore } from '@/stores/player.js';
// --- NEW: 引入快捷键提示组件 ---
import KeyboardShortcutsModal from './KeyboardShortcutsModal.vue';
const store = usePlayerStore();
// --- NEW: 控制快捷键提示模态框的显示状态 ---
const showShortcutsModal = ref(false);
// --- NEW: 全局键盘事件处理器 ---
const handleKeyDown = (event) => {
  // 如果焦点在输入框内，则不触发快捷键
  if (['INPUT', 'TEXTAREA'].includes(event.target.tagName)) {
    return;
  }
  // Shift + ? 显示帮助
  if (event.shiftKey && event.key === '?') {
    event.preventDefault(); // 防止浏览器默认行为
    showShortcutsModal.value = !showShortcutsModal.value;
    return;
  }
  // 如果模态框是打开的，则不触发其他快捷键
  if (showShortcutsModal.value) {
    return;
  }
  // 根据按键触发不同操作
  switch (event.key.toLowerCase()) {
    case 'p':
      togglePlayPause();
      break;
    case 'j':
      store.next();
      break;
    case 'k':
      store.prev();
      break;
    case 'm':
      toggleMute();
      break;
    case 'c':
      togglePlayMode();
      break;
  }
};
// --- NEW: 在组件挂载时添加监听器，卸载时移除 ---
onMounted(() => {
  window.addEventListener('keydown', handleKeyDown);
});
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown);
});

// --- 样式辅助函数: 动态计算背景渐变 ---
const getSliderStyle = (current, max) => {
  const percentage = max > 0 ? (current / max) * 100 : 0;
  return {
    background: `linear-gradient(to right, #1DB954 0%, #1DB954 ${percentage}%, #535353 ${percentage}%, #535353 100%)`
  };
};

// --- NEW: 播放模式相关 ---
const togglePlayMode = () => {
  // 假设 store 中有这个方法，它会调用后端 API
  store.togglePlayMode();
};

const playModeTitle = computed(() => {
  switch (store.playMode) {
    case 'SHUFFLE':
      return '随机播放';
    case 'REPEAT_ONE':
      return '单曲循环';
    case 'REPEAT_ALL':
      return '列表循环';
    default:
      return '切换播放模式';
  }
});

// --- 现有代码 ---
const onVolumeChange = (event) => {
  const newVolume = parseFloat(event.target.value);
  store.setLocalVolume(newVolume);
};

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
  background-color: #181818;
  border-top: 1px solid #282828;
  color: #fff;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif;
  height: 90px;
  box-sizing: border-box;
}

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

.main-controls {
  flex: 2;
  max-width: 722px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.buttons {
  display: flex;
  align-items: center;
  gap: 1rem; /* 减小按钮间距以容纳新按钮 */
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
  width: 24px;
  height: 24px;
}

.control-btn:hover {
  color: #fff;
}

.control-btn:active {
  transform: scale(0.95);
}

/* NEW: 激活状态的按钮样式 */
.control-btn.active {
  color: #1DB954;
}
.control-btn.active:hover {
  color: #1ED760;
}

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
  width: 16px;
  height: 16px;
}

.control-btn.secondary svg {
  width: 18px;
  height: 18px;
}

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

.custom-range {
  appearance: none;
  width: 100%;
  height: 4px;
  border-radius: 2px;
  background: #535353;
  outline: none;
  cursor: pointer;
  flex-grow: 1;
}

.custom-range::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s;
}

.custom-range:hover::-webkit-slider-thumb {
  opacity: 1;
}

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
  max-width: 100px;
}

@media (max-width: 768px) {
  .song-info, .volume-control {
    display: none;
  }
}
</style>
