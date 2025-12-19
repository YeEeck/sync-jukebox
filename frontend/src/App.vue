<template>
  <div id="app-container">
    <router-view />
    <!-- 将 AudioPlayerWrapper 放在顶层，使其在路由切换时也不中断 -->
    <AudioPlayerWrapper />
  </div>
</template>

<script setup>
import AudioPlayerWrapper from './components/AudioPlayerWrapper.vue';
</script>

<style>
/* 添加一些全局样式 */
#app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background-color: #121212;
  color: #fff;
}
</style>
