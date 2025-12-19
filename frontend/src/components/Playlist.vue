<template>
  <div class="playlist-container">
    <div class="playlist-header">
      <h2>Playlist</h2>
      <!-- 新增: 打乱按钮 -->
      <button 
        class="shuffle-btn" 
        @click="handleShuffle" 
        title="Shuffle Playlist"
        :disabled="!store.playlist || store.playlist.length < 2"
      >
        <IconShuffle />
      </button>
    </div>
    <ul v-if="store.playlist && store.playlist.length > 0" class="song-list">
      <!-- 拖放、双击播放功能
        1. 增加 draggable="true"
        2. 增加 @dblclick 处理双击播放
        3. 增加一系列 @drag* 事件处理拖拽
        4. 增加 :class 绑定，用于显示拖拽时的视觉效果
      -->
      <li v-for="(item, index) in store.playlist" :key="item.song_id" class="song-item" :class="{
        'is-playing': store.currentSongId === item.song_id,
        // 'drag-over': dragOverIndex === index,
        'is-dragging': draggedItemIndex === index,
        /* 根据 dragOverPosition 显示不同的类 */
        'drag-over-top': dragOverIndex === index && dragOverPosition === 'top',
        'drag-over-bottom': dragOverIndex === index && dragOverPosition === 'bottom'
      }" draggable="true" @dblclick="handleDoubleClick(item)" @dragstart="onDragStart($event, index, item)"
        @dragover.prevent="onDragOver($event, index)" @dragend="onDragEnd" @drop="onDrop($event, index)">
        <div class="song-details">
          <span class="song-title">{{ item.song.title }}</span>
          <span class="song-artist">{{ item.song.artist }}</span>
        </div>

        <button class="remove-btn" @click.stop="handleRemove(item)" title="Remove from playlist">
          <IconTrash></IconTrash>
        </button>
      </li>
    </ul>
    <p v-else class="empty-message">The playlist is currently empty.</p>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import IconTrash from '@/components/icons/IconTrash.vue';
import IconShuffle from '@/components/icons/IconShuffle.vue';
import { usePlayerStore } from '@/stores/player';
const store = usePlayerStore();

// --- 处理打乱点击 ---
const handleShuffle = () => {
  store.shufflePlaylist();
};

const handleRemove = (item) => {
  // 注意：根据后端 API 定义，我们需要传 songId。
  // item.song_id 通常是数据库中关联表的 ID 或歌曲 ID。
  // 假设后端需要的是歌曲本身的 UUID (item.song.id) 或者关联记录的 UUID (item.song_id)。
  // 如果 item.song_id 指的是这首个具体的歌曲资源 ID：
  store.removeSongFromPlaylist(item.song_id);
};

// --- 双击播放功能 ---
const handleDoubleClick = (item) => {
  // 调用 API 播放指定曲目
  store.playSpecificSong(item.song_id);
};

// --- 拖拽排序逻辑 ---
const draggedItemIndex = ref(null); // 记录当前被拖拽项的索引
const draggedItemData = ref(null);  // 记录当前被拖拽项的数据
const dragOverIndex = ref(null);    // 记录当前鼠标悬停的目标索引（用于 UI 高亮）
const dragOverPosition = ref(null); // 记录当前鼠标悬停的位置（'top' 或 'bottom'）

const onDragStart = (event, index, item) => {
  draggedItemIndex.value = index;
  draggedItemData.value = item;
  event.dataTransfer.effectAllowed = 'move';
  // 兼容 Firefox
  event.dataTransfer.setData('text/plain', index);

  // 不在这里直接操作 event.target.style.opacity
  // 通过上面的 :class="{'is-dragging': ...}" 控制样式
};

// 无论拖拽是否成功，结束后都会触发，用于清理状态
const onDragEnd = (event) => {
  draggedItemIndex.value = null;
  draggedItemData.value = null;
  dragOverIndex.value = null;
  dragOverPosition.value = null;
};

const onDragOver = (event, index) => {
  // 必须调用 preventDefault 才能触发 drop
  // 只有当拖拽的不是自己时，才记录 dragOverIndex
  if (draggedItemIndex.value !== null && draggedItemIndex.value !== index) {
    dragOverIndex.value = index;
    // --- 计算鼠标在元素内的位置 ---
    const targetRect = event.currentTarget.getBoundingClientRect();
    const hoverY = event.clientY - targetRect.top; // 鼠标在元素内的 Y 坐标
    const threshold = targetRect.height / 2; // 元素高度的一半
    if (hoverY < threshold) {
      dragOverPosition.value = 'top';
    } else {
      dragOverPosition.value = 'bottom';
    }
  }

};

const onDrop = (event, targetIndex) => {
  // 移除 event.target.style.opacity = '1'; 
  // 状态清理工作交给 onDragEnd 统一处理
  const sourceIndex = draggedItemIndex.value;
  const item = draggedItemData.value;
  if (sourceIndex === null || sourceIndex === targetIndex) {
    return;
  }

  // 计算基础的目标插入位置
  // 如果是在上半部松手，就插入到当前 targetIndex
  // 如果是在下半部松手，就插入到 targetIndex + 1
  let finalIndex = targetIndex;
  if (dragOverPosition.value === 'bottom') {
    finalIndex++;
  }
  // 处理“从上往下拖”导致的索引偏移
  // 如果源元素在目标元素的前面 (source < target)，
  // 后端在移除源元素后，目标元素及其后面的元素索引都会自动 -1。
  // 为了抵消这个影响，我们需要把计算出的 finalIndex 也减 1。
  if (sourceIndex < targetIndex) {
    finalIndex--;
  }

  store.movePlaylistItem(item.song_id, finalIndex);
};
</script>

<style scoped>
/* 修改 playlist-header 样式以容纳按钮 */
.playlist-header {
  display: flex;
  justify-content: space-between; /* 标题左对齐，按钮右对齐 */
  align-items: center;
  flex-shrink: 0;
  margin-bottom: 1rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid #282828;
}

.playlist-header h2 {
  /* 去掉原来 h2 的 margin 和 padding，因为移动到了 header 容器上 */
  margin: 0;
  padding: 0;
  border: none;
}

.shuffle-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #b3b3b3; /* 默认灰色 */
  width: 32px;    /* 固定宽高，确保是正圆 */
  height: 32px;
  padding: 0;     /* 清除 padding，配合 flex 居中 */
  border-radius: 50%;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}
/* 悬停时：图标变白，背景微亮 */
.shuffle-btn:hover {
  color: #fff;
  background-color: rgba(255, 255, 255, 0.1);
  transform: scale(1.05); /* 微微放大 */
}
/* 点击时：图标变绿，模拟 Spotify 激活状态 */
.shuffle-btn:active {
  color: #1db954; /* Spotify Green */
  transform: scale(0.95);
  background-color: rgba(255, 255, 255, 0.15);
}
/* 禁用状态 */
.shuffle-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
  transform: none;
  color: #b3b3b3;
}

.playlist-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.song-list,
.empty-message {
  flex: 1;
  overflow-y: auto;
}

.song-list {
  list-style: none;
  padding-right: 0.1rem;
  margin: 0;
}

.song-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0.5rem;
  padding-right: 1.0rem;

  border-top: 2px solid transparent;
  border-bottom: 2px solid transparent;
  box-shadow: 0 1px 0 #282828;
  /* 模拟原来的 1px border-bottom */
  margin-bottom: 1px;
  /* 补偿 shadow 占据的空间 */
  /* 预留位置，防止绿线出现时元素抖动 */
  transition: background-color 0.2s ease, border-top 0.1s;
  /* 增加 border 动画 */
  cursor: pointer;

  position: relative;

  /* 防止文字被选中干扰拖拽 */
  user-select: none;
}

/* 拖拽时的鼠标手势 */
.song-item[draggable="true"] {
  cursor: grab;
}

.song-item[draggable="true"]:active {
  cursor: grabbing;
}

/* 拖拽悬停时的视觉反馈：上方显示一条亮线，表示插入位置 */
.song-item.drag-over {
  border-top: 2px solid #1db954;
  /* Spotify Green */
  background-color: #333;
}

.song-item.drag-over-top {
  border-top-color: #1db954;
  background-color: #333;
}

.song-item.drag-over-bottom {
  border-bottom-color: #1db954;
  background-color: #333;
  box-shadow: none;
  /* 隐藏灰色的分割线，只显示绿线 */
}

/* 正在被拖拽的源元素样式 */
.song-item.is-dragging {
  opacity: 0.5;
  background-color: #2a2a2a;
  /* 可选：让它稍微亮一点或暗一点 */
}

/* 拖拽悬停时的视觉反馈 */
.song-item.drag-over {
  border-top: 2px solid #1db954;
}

.song-item:hover {
  background-color: #2a2a2a;
}

.song-item:hover .remove-btn {
  opacity: 1;
  transform: translateX(0);
}

.song-item:last-child {
  border-bottom: none;
}

/* 歌曲详情 */
.song-details {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;

  /* 确保文字不会盖住按钮，留出空间 */
  max-width: 85%;

  pointer-events: none;
  /* 禁止子元素响应鼠标事件，防止拖拽经过文字时闪烁 */
}

.song-title {
  font-weight: 500;
  color: #fff;
}

.song-artist {
  font-size: 0.8rem;
  color: #b3b3b3;
}

/* 高亮当前播放的歌曲 */
.song-item.is-playing .song-title {
  color: #1db954;
  /* Spotify 绿色 */
  font-weight: bold;
}

.song-item.is-playing .song-artist {
  color: #1db954;
}

/* --- 删除按钮样式 --- */
.remove-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 8px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;

  /* 默认状态：完全透明或半透明，稍微向右偏移 */
  opacity: 0;
  transform: translateX(10px);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  color: #b3b3b3;
  /* 默认灰色 */
}

/* 按钮悬停状态：变成红色 */
.remove-btn:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: #ff5555;
  /* 警告红 */
}

.remove-btn:active {
  transform: scale(0.95);
}

/* 播放列表为空时的提示信息 */
.empty-message {
  color: #b3b3b3;
  text-align: center;
  margin-top: 2rem;
  font-style: italic;
}

/* --- 自定义滚动条样式 --- */
.song-list::-webkit-scrollbar {
  width: 8px;
  /* 滚动条宽度 */
}

.song-list::-webkit-scrollbar-track {
  background: #181818;
  /* 滚动条轨道颜色，与面板背景色相同 */
  border-radius: 10px;
}

.song-list::-webkit-scrollbar-thumb {
  background-color: #535353;
  /* 滚动条滑块颜色 */
  border-radius: 10px;
  border: 2px solid #181818;
  /* 创建一个围绕滑块的边框效果 */
}

.song-list::-webkit-scrollbar-thumb:hover {
  background-color: #777;
  /* 鼠标悬停时滑块的颜色 */
}
</style>
