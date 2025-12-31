<template>
  <div class="jukebox-layout">
    <header>
      <!-- 1. 将图标和标题分组 -->
      <div class="header-brand">
        <img src="@/assets/icon.png" alt="SyncJukebox Icon" class="icon" />
        <div class="title">
          SyncJukebox
        </div>
      </div>

      <!-- 2. 添加退出登录按钮 -->
      <button @click="handleLogout" class="logout-button">Logout</button>
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
import { ref, watch } from 'vue';
import { useRouter } from 'vue-router'; // <-- 导入 useRouter
import MediaLibrary from '../components/MediaLibrary.vue';
import Playlist from '../components/Playlist.vue';
import PlayerControls from '../components/PlayerControls.vue';
import PlaybackPermissionModal from '../components/PlaybackPermissionModal.vue';
import { usePlayerStore } from '@/stores/player';

const store = usePlayerStore();
const router = useRouter(); // <-- 获取 router 实例
const showPermissionModal = ref(false);

// 3. 创建 handleLogout 方法
const handleLogout = () => {
  store.logout();
  // 虽然路由守卫会自动处理重定向，但显式跳转是更好的用户体验
  router.push({ name: 'login' });
};

// 处理弹窗点击确认
const handlePermissionConfirm = async () => {
  try {
    await store.play();
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
  /* 4. 修改布局以将内容推向两边 */
  justify-content: space-between;
  padding: 1rem 1.5rem; /* 左右也增加一些内边距 */
  background-color: #181818;
  font-size: 1.2rem;
  font-weight: bold;
  flex-shrink: 0;
}

/* 5. 为新添加的元素添加样式 */
.header-brand {
  display: flex;
  align-items: center;
}

.logout-button {
  padding: 0.5rem 1rem;
  font-size: 0.9rem;
  font-weight: bold;
  color: #fff;
  background-color: transparent;
  border: 1px solid #535353;
  border-radius: 20px;
  cursor: pointer;
  transition: background-color 0.2s, color 0.2s;
}

.logout-button:hover {
  background-color: #fff;
  color: #121212;
}


main {
  display: flex;
  flex: 1;
  overflow: hidden;
  padding: 1.5rem;
  gap: 1.5rem;
}

.left-panel,
.right-panel {
  flex: 1;
  background-color: #181818;
  border-radius: 8px;
  padding: 1.5rem;
  min-width: 0;
}

footer {
  background-color: #181818;
  flex-shrink: 0;
}
</style>
