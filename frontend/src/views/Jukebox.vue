<template>
  <div class="jukebox-layout">
    <header>
      <img src="@/assets/icon.png" alt="SyncJukebox Icon" class="icon" />
      <div class="title">
        SyncJukebox
      </div>
    </header>
    <main>
      <div class="left-panel">
        <MediaLibrary />
      </div>
      <div class="right-panel">
        <Playlist />
      </div>
    </main>
    <footer>
      <PlayerControls />
    </footer>

    <!-- 播放权限弹窗 -->
    <PlaybackPermissionModal :visible="showPermissionModal" @confirm="handlePermissionConfirm" />
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue';
import MediaLibrary from '../components/MediaLibrary.vue';
import Playlist from '../components/Playlist.vue';
import PlayerControls from '../components/PlayerControls.vue';
import PlaybackPermissionModal from '../components/PlaybackPermissionModal.vue';
import { usePlayerStore } from '@/stores/player';

const store = usePlayerStore();
const showPermissionModal = ref(false);

// 处理弹窗点击确认
const handlePermissionConfirm = async () => {
  // 用户点击了按钮，产生了交互，现在可以安全地调用播放了
  try {
    await store.play();
    // 或者如果是初始化Context: store.audioContext.resume();
    showPermissionModal.value = false;
  } catch (e) {
    console.error("依然无法播放:", e);
  }
};

// 监听 Store 中的错误状态
watch(() => store.playbackError, (newError) => {
  if (newError && newError.name === 'NotAllowedError') {
    showPermissionModal.value = true;
  }
});
</script>

<style scoped>
.jukebox-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.icon {
  width: 32px;
  height: 32px;
  margin-right: 0.5rem;
}

header {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background-color: #181818;
  text-align: center;
  font-size: 1.2rem;
  font-weight: bold;
  flex-shrink: 0;
  /* 防止header被压缩 */
}

main {
  display: flex;
  flex: 1;
  overflow: hidden;
  padding: 1.5rem;
  /* --- 给整个主内容区一个外边距 --- */
  gap: 1.5rem;
  /* --- 在左右面板之间创建间距 --- */
}

.left-panel,
.right-panel {
  flex: 1;
  background-color: #181818;
  /* 给面板一个背景色，视觉上更清晰 */
  border-radius: 8px;
  /* 添加圆角，更美观 */
  padding: 1.5rem;
  /* --- 给面板内部内容增加边距 --- */
  min-width: 0;
}

footer {
  background-color: #181818;
  /* 与PlayerControls的背景色统一 */
  flex-shrink: 0;
  /* 防止footer被压缩 */
}
</style>
