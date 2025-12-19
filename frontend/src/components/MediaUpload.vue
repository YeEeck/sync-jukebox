<template>
  <div class="upload-section">
    <!-- 隐藏的文件输入框 -->
    <input 
      type="file" 
      id="file-upload" 
      class="hidden-input" 
      ref="fileInput"
      @change="handleFileUpload" 
      accept="audio/*" 
      :disabled="isUploading"
    />

    <!-- 默认显示上传按钮 -->
    <label 
      v-if="!isUploading" 
      for="file-upload" 
      class="upload-btn"
    >
      <span class="icon">☁️</span>
      <span>Upload New Song</span>
    </label>

    <!-- 上传中/进度条显示 -->
    <div v-else class="upload-status-panel">
      <div class="status-text">
        <span class="file-name">{{ currentFileName }}</span>
        <span class="status-label">Uploading...</span>
      </div>
      <div class="progress-track">
        <div class="progress-bar" :style="{ width: progress + '%' }"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { usePlayerStore } from '@/stores/player';

const store = usePlayerStore();
const fileInput = ref(null);

// 状态管理
const isUploading = ref(false);
const progress = ref(0);
const currentFileName = ref('');

const handleFileUpload = async (event) => {
  const file = event.target.files[0];
  if (!file) return;

  // 初始化状态
  isUploading.value = true;
  progress.value = 0;
  currentFileName.value = file.name;

  // 模拟进度条动画 (为了更好的 UX，因为通常后端 API 不一定会返回实时进度)
  // 如果你的 store.uploadSong 支持 axios onUploadProgress，可以替换这里的逻辑
  const progressInterval = setInterval(() => {
    if (progress.value < 90) {
      // 前期快，后期慢的模拟效果
      const increment = (90 - progress.value) / 10; 
      progress.value += Math.max(1, increment); 
    }
  }, 200);

  try {
    // 调用 Store 的上传方法
    // store.uploadSong 返回一个 Promise
    await store.uploadSong(file);
    
    // 上传完成，瞬间填满进度条
    progress.value = 100;
    
    // 给用户展示一小会儿"100%"的状态，然后复原
    setTimeout(() => {
      resetUploadState();
    }, 800);

  } catch (error) {
    console.error("Upload failed:", error);
    alert("Upload failed. Please try again.");
    resetUploadState();
  } finally {
    clearInterval(progressInterval);
    // 清空 input，防止无法重复上传同一个文件
    if (fileInput.value) fileInput.value.value = '';
  }
};

const resetUploadState = () => {
  isUploading.value = false;
  progress.value = 0;
  currentFileName.value = '';
};
</script>

<style scoped>
.upload-section {
  margin-top: 1rem;
  margin-bottom: 1rem;
  padding: 0 0.25rem;
}

.hidden-input {
  display: none;
}

/* --- 默认按钮样式 (保持原有风格但微调) --- */
.upload-btn {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  padding: 0.75rem;
  background-color: #282828;
  color: #fff;
  border: 1px dashed #535353;
  border-radius: 4px; /* Spotify 风格通常圆角较小或全圆，这里用小圆角 */
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.9rem;
  font-weight: 700;
  gap: 0.5rem;
  box-sizing: border-box;
}

.upload-btn:hover {
  background-color: #333;
  border-color: #1db954; /* Spotify Green */
  transform: scale(1.01); /* 微小的放大效果 */
}

.upload-btn:active {
  background-color: #222;
  transform: scale(0.99);
}

/* --- 上传中状态样式 --- */
.upload-status-panel {
  width: 100%;
  padding: 0.75rem;
  background-color: #282828;
  border: 1px solid #282828; /* 实线边框代替虚线 */
  border-radius: 4px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.5rem;
}

.status-text {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.8rem;
  color: #b3b3b3;
}

.file-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 70%;
  font-weight: 500;
  color: #fff;
}

.status-label {
  font-size: 0.75rem;
  color: #1db954; /* 文字也用绿色提示 */
}

/* 进度条轨道 */
.progress-track {
  width: 100%;
  height: 4px;
  background-color: #404040;
  border-radius: 2px;
  overflow: hidden;
}

/* 进度条本体 */
.progress-bar {
  height: 100%;
  background-color: #1db954; /* Spotify Green */
  border-radius: 2px;
  transition: width 0.2s ease-out; /* 平滑过渡 */
}
</style>