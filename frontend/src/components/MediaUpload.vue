<template>
  <div class="upload-section">
    <!-- 隐藏的文件输入框，增加了 multiple 属性 -->
    <input
        type="file"
        id="file-upload"
        class="hidden-input"
        ref="fileInput"
        @change="handleFileSelect"
        accept="audio/*"
        :disabled="isUploading"
        multiple
    />

    <!-- 默认显示上传按钮 -->
    <label
        v-if="!isUploading"
        for="file-upload"
        class="upload-btn"
    >
      <span class="icon">☁️</span>
      <span>Upload New Song(s)</span>
    </label>

    <!-- 上传中/进度条显示 -->
    <!-- 修改为上传列表容器 -->
    <div v-else class="upload-status-container">
      <!-- v-for 循环渲染每个上传项的进度 -->
      <div
          v-for="upload in uploads"
          :key="upload.id"
          class="upload-status-panel"
      >
        <div class="status-text">
          <span class="file-name">{{ upload.name }}</span>
          <!-- 根据状态显示不同文本 -->
          <span
              class="status-label"
              :class="{ 'status-error': upload.error, 'status-done': upload.progress === 100 }"
          >
            {{ upload.error ? 'Error' : (upload.progress === 100 ? 'Done' : 'Uploading...') }}
          </span>
        </div>
        <div class="progress-track">
          <!-- 绑定到每个上传项的进度 -->
          <div
              class="progress-bar"
              :style="{ width: upload.progress + '%' }"
              :class="{ 'progress-error': upload.error }"
          ></div>
        </div>
        <!-- 显示错误信息 -->
        <div v-if="upload.error" class="error-message">{{ upload.error }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { usePlayerStore } from '@/stores/player';

const store = usePlayerStore();
const fileInput = ref(null);

// 状态管理：从单个状态变为管理一个上传对象数组
const uploads = ref([]); // e.g., [{ id, file, name, progress, error }]

// 计算属性，判断当前是否有文件在上传
const isUploading = computed(() => uploads.value.length > 0);

// 当用户选择文件后触发
const handleFileSelect = (event) => {
  const files = event.target.files;
  if (!files.length) return;

  // 为每个选中的文件创建一个唯一的上传状态对象
  uploads.value = Array.from(files).map(file => ({
    id: Symbol('upload-id'), // 使用 Symbol 保证 key 的唯一性
    file: file,
    name: file.name,
    progress: 0,
    error: null,
  }));

  // 开始处理所有上传任务
  processUploadQueue();
};

// 处理整个上传队列
const processUploadQueue = async () => {
  // 为队列中的每个文件创建一个上传 Promise
  const uploadPromises = uploads.value.map(upload => uploadFile(upload));

  try {
    // 等待所有文件都处理完毕（无论成功或失败）
    await Promise.allSettled(uploadPromises);

    // 所有上传结束后，刷新一次媒体库
    // 只有在至少一个上传成功的情况下才刷新
    const hasSuccess = uploads.value.some(u => !u.error);
    if (hasSuccess) {
      await store.fetchLibrary();
    }

    // 给用户一点时间查看最终状态（比如 "Done" 或 "Error"）
    setTimeout(() => {
      resetUploadState();
    }, 1500);

  } finally {
    // 清空 input，以便可以再次选择相同的文件
    if (fileInput.value) fileInput.value.value = '';
  }
};

// 处理单个文件的上传逻辑
const uploadFile = async (upload) => {
  // 模拟进度条动画
  const progressInterval = setInterval(() => {
    if (upload.progress < 90) {
      const increment = (90 - upload.progress) / 10;
      upload.progress += Math.max(1, increment);
    }
  }, 200);

  try {
    // 调用 Store 的上传方法
    await store.uploadSong(upload.file);

    // 上传成功
    clearInterval(progressInterval);
    upload.progress = 100;

  } catch (error) {
    // 上传失败
    clearInterval(progressInterval);
    console.error(`Upload failed for ${upload.name}:`, error);
    upload.error = "Upload failed. Please try again.";
    // 可以将进度条设为100并变红，或者保持原样
    // upload.progress = 100;
  }
};

// 重置所有状态
const resetUploadState = () => {
  uploads.value = [];
};
</script>

<style scoped>
/* --- (原有样式基本保持不变, 只需添加一些新样式和微调) --- */
.upload-section {
  margin-top: 1rem;
  margin-bottom: 1rem;
  padding: 0 0.25rem;
}

.hidden-input {
  display: none;
}

.upload-btn {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  padding: 0.75rem;
  background-color: #282828;
  color: #fff;
  border: 1px dashed #535353;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.9rem;
  font-weight: 700;
  gap: 0.5rem;
  box-sizing: border-box;
}

.upload-btn:hover {
  background-color: #333;
  border-color: #1db954;
  transform: scale(1.01);
}

.upload-btn:active {
  background-color: #222;
  transform: scale(0.99);
}

/* 新增：上传列表的容器 */
.upload-status-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem; /* 每个进度条之间的间距 */
}

.upload-status-panel {
  width: 100%;
  padding: 0.75rem;
  background-color: #282828;
  border: 1px solid #282828;
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
  color: #1db954;
}

/* 新增：完成状态的标签样式 */
.status-done {
  color: #b3b3b3;
}

/* 新增：错误状态的标签样式 */
.status-error {
  color: #f44336; /* 红色表示错误 */
}

.progress-track {
  width: 100%;
  height: 4px;
  background-color: #404040;
  border-radius: 2px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background-color: #1db954;
  border-radius: 2px;
  transition: width 0.2s ease-out;
}

/* 新增：错误状态的进度条样式 */
.progress-bar.progress-error {
  background-color: #f44336;
}

/* 新增：错误信息样式 */
.error-message {
  font-size: 0.7rem;
  color: #f44336;
  margin-top: -0.25rem;
}
</style>
