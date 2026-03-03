<template>
  <div class="media-library-container">
    <h2>Media Library</h2>

    <!-- 文件上传 -->
    <MediaUpload />

    <!-- SearchBar 组件 -->
    <!-- 可以在此处通过 props 传递 placeholder 提示用户支持高级搜索，如果 SearchBar 支持的话 -->
    <SearchBar v-model="searchQuery" />

    <!-- 歌曲列表 (现在使用 filteredLibrary) -->
    <ul class="song-list">
      <li v-for="song in filteredLibrary" :key="song.id" class="song-item">
        <div class="song-details">
          <span class="song-title">{{ song.title }}</span>
          <span class="song-artist">{{ song.artist || 'Unknown Artist' }}</span>
        </div>
        <div class="song-actions">
          <!--
            修改后的“播放下一首”按钮:
            1. v-if="!playlistSongIds.has(song.id) && store.currentSongId" 被移除，按钮始终显示。
            2. 点击事件固定调用 addSongNextInPlaylist，由后端处理所有复杂逻辑。
            3. title 改为 "Play Next"。
           -->
          <button
              @click="store.addSongNextInPlaylist(song.id)"
              title="Play Next"
              class="play-next-btn"
          >
            +⤋
          </button>

          <!-- Add to playlist 按钮现在只在歌曲不在播放列表时显示，保持原有逻辑 -->
          <button v-if="!playlistSongIds.has(song.id)" @click="store.addToPlaylist(song.id)" title="Add to playlist (end of list)">+</button>

          <!-- 删除按钮 -->
          <button @click="confirmRemove(song)" class="delete-btn" title="Delete from library">×</button>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { usePlayerStore } from '@/stores/player';
import MediaUpload from '@/components/MediaUpload.vue';
import SearchBar from '@/components/SearchBar.vue'; // --- 新增: 导入 SearchBar 组件 ---

const store = usePlayerStore();
const pollingInterval = ref(null);
const isLoading = ref(false);

const searchQuery = ref(''); // 搜索状态保留在父组件中

const playlistSongIds = computed(() => {
  return new Set(store.playlist.map(item => item.song_id));
});

// 修改后的过滤逻辑：默认搜歌名，支持 tag 搜歌手
const filteredLibrary = computed(() => {
  const rawInput = searchQuery.value.trim().toLowerCase();

  // 如果没有输入，显示全部
  if (!rawInput) {
    return store.mediaLibrary;
  }

  // 定义歌手搜索的前缀 (支持 "artist:" 或 "@")
  const artistPrefixes = ['a:', '@'];
  const matchedPrefix = artistPrefixes.find(prefix => rawInput.startsWith(prefix));

  // 如果检测到歌手 tag
  if (matchedPrefix) {
    const term = rawInput.slice(matchedPrefix.length).trim();
    // 如果只有前缀没有内容，依然返回全部，或者可以返回空，这里选择返回全部体验较好
    if (!term) return store.mediaLibrary;

    return store.mediaLibrary.filter(song =>
        song.artist && song.artist.toLowerCase().includes(term)
    );
  }

  // 默认逻辑：仅搜索歌名 (Title)
  return store.mediaLibrary.filter(song => {
    return song.title.toLowerCase().includes(rawInput);
  });
});


const refreshLibrary = async () => {
  if (isLoading.value) return;
  isLoading.value = true;
  console.log('Refreshing media library...');
  try {
    await store.fetchLibrary();
  } catch (error) {
    console.error('Failed to refresh media library:', error);
  } finally {
    isLoading.value = false;
  }
};

onMounted(() => {
  refreshLibrary();
  pollingInterval.value = setInterval(refreshLibrary, 5000);
});

onUnmounted(() => {
  if (pollingInterval.value) {
    clearInterval(pollingInterval.value);
  }
});

const confirmRemove = (song) => {
  if (window.confirm(`Are you sure you want to permanently delete "${song.title}"? This action cannot be undone.`)) {
    store.removeSongFromLibrary(song.id);
  }
};
</script>

<style scoped>
.media-library-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

h2,
.upload-section {
  flex-shrink: 0;
}

.song-list {
  flex: 1;
  overflow-y: auto;
  list-style: none;
  padding-right: 0.1rem;
  margin: 0;
}

.song-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0.25rem;
  padding-right: 1rem;
  border-bottom: 1px solid #282828;
  transition: background-color 0.2s;
  gap: 0.5rem;
}

.song-item:hover {
  background-color: #2a2a2a;
}

.song-details {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.song-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.song-artist {
  font-size: 0.8rem;
  color: #b3b3b3;
}

.song-actions {
  display: flex;
  flex-shrink: 0;
  gap: 0.5rem;
}

.song-actions button {
  background: none;
  border: 1px solid #535353;
  color: #fff;
  border-radius: 50%;
  width: 28px;
  height: 28px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  font-size: 1.2rem;
  line-height: 1;
}

.song-actions .play-next-btn {
  border-color: #6a6a6a; /* 稍微突出一点 */
  font-size: 1rem;
  font-weight: bold;
}

.song-actions .delete-btn {
  border-color: #812828;
  color: #e04444;
}

.song-list::-webkit-scrollbar {
  width: 8px;
}

.song-list::-webkit-scrollbar-track {
  background: #181818;
  border-radius: 10px;
}

.song-list::-webkit-scrollbar-thumb {
  background-color: #535353;
  border-radius: 10px;
  border: 2px solid #181818;
}

.song-list::-webkit-scrollbar-thumb:hover {
  background-color: #777;
}
</style>