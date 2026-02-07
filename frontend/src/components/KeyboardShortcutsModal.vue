<template>
  <transition name="fade">
    <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
      <div class="modal-content">
        <h2>键盘快捷键</h2>
        <ul class="shortcuts-list">
          <li><span>播放 / 暂停</span><kbd>P</kbd></li>
          <li><span>下一首</span><kbd>J</kbd></li>
          <li><span>上一首</span><kbd>K</kbd></li>
          <li><span>静音 / 取消静音</span><kbd>M</kbd></li>
          <li><span>切换播放模式</span><kbd>C</kbd></li>
          <li><span>显示此帮助</span><kbd>Shift</kbd> + <kbd>?</kbd></li>
        </ul>
        <button class="close-btn" @click="$emit('close')">&times;</button>
      </div>
    </div>
  </transition>
</template>

<script setup>
defineProps({
  visible: {
    type: Boolean,
    required: true,
  },
});

defineEmits(['close']);
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-content {
  position: relative;
  background-color: #282828;
  padding: 2rem 3rem;
  border-radius: 8px;
  color: #fff;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 5px 15px rgba(0, 0, 0, 0.5);
}

h2 {
  margin-top: 0;
  margin-bottom: 1.5rem;
  text-align: center;
  color: #fff;
}

.shortcuts-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.shortcuts-list li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid #404040;
  font-size: 0.9rem;
}

.shortcuts-list li:last-child {
  border-bottom: none;
}

kbd {
  background-color: #121212;
  border: 1px solid #535353;
  border-radius: 4px;
  padding: 0.2em 0.6em;
  font-family: monospace;
  font-size: 0.9em;
  margin-left: 0.5rem;
}

.close-btn {
  position: absolute;
  top: 10px;
  right: 15px;
  background: none;
  border: none;
  font-size: 2rem;
  color: #b3b3b3;
  cursor: pointer;
  line-height: 1;
}
.close-btn:hover {
  color: #fff;
}

/* Fade Transition */
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>
