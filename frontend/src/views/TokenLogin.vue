<template>
  <div class="login-container">
    <h1>SyncJukebox v2.0</h1>
    <input v-model="token" placeholder="Enter your access token" @keyup.enter="submitToken" />
    <button @click="submitToken">Enter</button>
    <p v-if="error" class="error-message">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { usePlayerStore } from '@/stores/player';

const token = ref('');
const error = ref('');
const router = useRouter();
const playerStore = usePlayerStore();

const submitToken = async () => {
  if (!token.value) {
    error.value = 'Token cannot be empty.';
    return;
  }
  const success = await playerStore.validateTokenAndConnect(token.value);
  if (success) {
    router.push('/');
  } else {
    error.value = 'Invalid or expired token.';
  }
};
</script>

<style scoped>
.login-container { text-align: center; margin-top: 20vh; }
/* ... 其他样式 */
</style>
