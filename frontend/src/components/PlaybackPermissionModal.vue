<template>
  <Transition name="fade">
    <div v-if="visible" class="modal-overlay">
      <div class="modal-content">
        <div class="icon-wrapper">
          <!-- 播放图标 -->
          <svg viewBox="0 0 24 24" fill="currentColor" class="play-icon">
            <path d="M8 5v14l11-7z" />
          </svg>
        </div>
        <h2>Continue Listening</h2>
        <p>Click below to enable audio playback for SyncJukebox.</p>
        <button class="resume-btn" @click="handleConfirm">
          Resume Playback
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['confirm']);

const handleConfirm = () => {
  emit('confirm');
};
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(5px); /* 磨砂玻璃效果 */
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-content {
  background-color: #282828;
  padding: 2.5rem;
  border-radius: 8px;
  text-align: center;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #fff;
}

.icon-wrapper {
  width: 64px;
  height: 64px;
  background-color: #121212;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1.5rem;
}

.play-icon {
  width: 32px;
  height: 32px;
  color: #1db954; /* Spotify Green */
  transform: translateX(2px); /* 视觉居中修正 */
}

h2 {
  font-size: 1.5rem;
  margin: 0 0 0.5rem 0;
  font-weight: 700;
}

p {
  color: #b3b3b3;
  margin-bottom: 2rem;
  font-size: 1rem;
  line-height: 1.5;
}

.resume-btn {
  background-color: #1db954;
  color: #000;
  border: none;
  border-radius: 500px; /* Pill shape */
  padding: 12px 32px;
  font-size: 1rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 1px;
  cursor: pointer;
  transition: transform 0.1s, background-color 0.2s, box-shadow 0.2s;
}

.resume-btn:hover {
  background-color: #1ed760;
  transform: scale(1.04);
}

.resume-btn:active {
  background-color: #1aa34a;
  transform: scale(1.0);
}

/* 动画效果 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>