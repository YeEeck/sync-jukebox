<template>
  <div class="login-container">
    <h1>SyncJukebox v2.0</h1>

    <!-- 登录表单 -->
    <form v-if="!isRegistering" @submit.prevent="handleLogin">
      <h2>Login</h2>
      <input v-model="username" type="text" placeholder="Username" required autocomplete="username"/>
      <input v-model="password" type="password" placeholder="Password" required autocomplete="current-password"/>
      <button type="submit">Login</button>
      <p>
        Don't have an account? <br>
        <a href="#" @click.prevent="toggleForm">Register here</a>
      </p>
    </form>

    <!-- 注册表单 -->
    <form v-else @submit.prevent="handleRegister">
      <h2>Register</h2>
      <input v-model="username" type="text" placeholder="Username" required autocomplete="username"/>
      <input v-model="password" type="password" placeholder="Password" required autocomplete="new-password"/>
      <!-- 新增: 邀请密钥输入框 -->
      <input v-model="invitationKey" type="text" placeholder="Invitation Key" required />
      <button type="submit">Register</button>
      <p>
        Already have an account? <br>
        <a href="#" @click.prevent="toggleForm">Login here</a>
      </p>
    </form>

    <p v-if="message" :class="{ 'error-message': isError, 'success-message': !isError }">{{ message }}</p>
  </div>
</template>

<script setup>
import {ref} from 'vue';
import {useRouter} from 'vue-router';
import {usePlayerStore} from '@/stores/player';

// 响应式状态
const isRegistering = ref(false);
const username = ref('');
const password = ref('');
const invitationKey = ref(''); // 新增: 邀请密钥的状态
const message = ref('');
const isError = ref(false);

const router = useRouter();
const playerStore = usePlayerStore();

// 清理状态的辅助函数
const clearForm = () => {
  message.value = '';
  isError.value = false;
  username.value = '';
  password.value = '';
  invitationKey.value = ''; // 新增: 清理密钥
};

// 切换登录/注册表单
const toggleForm = () => {
  isRegistering.value = !isRegistering.value;
  clearForm();
};

// 处理注册
const handleRegister = async () => {
  // 修改: 检查邀请密钥
  if (!username.value || !password.value || !invitationKey.value) {
    message.value = 'Username, password and invitation key cannot be empty.';
    isError.value = true;
    return;
  }
  // 修改: 将密钥传递给 store action
  const result = await playerStore.register(username.value, password.value, invitationKey.value);
  if (result.success) {
    message.value = result.message || 'Registration successful! Please log in.';
    isError.value = false;
    isRegistering.value = false;
    // 清空密码和密钥，保留用户名以便登录
    password.value = '';
    invitationKey.value = '';
  } else {
    message.value = result.message || 'Registration failed.';
    isError.value = true;
  }
};

// 处理登录 (保持不变)
const handleLogin = async () => {
  if (!username.value || !password.value) {
    message.value = 'Username and password cannot be empty.';
    isError.value = true;
    return;
  }
  const success = await playerStore.loginAndConnect(username.value, password.value);
  if (success) {
    router.push('/');
  } else {
    message.value = playerStore.authError || 'Invalid username or password.';
    isError.value = true;
  }
};
</script>

<style scoped>
/* 样式保持不变 */
.login-container {
  text-align: center;
  margin-top: 15vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}
form {
  display: flex;
  flex-direction: column;
  gap: 0.8rem;
  width: 300px;
  padding: 2rem;
  border: 1px solid #333;
  border-radius: 8px;
  background-color: #181818;
}
input {
  padding: 0.8rem;
  border-radius: 4px;
  border: 1px solid #444;
  background-color: #222;
  color: white;
  font-size: 1rem;
}
button {
  padding: 0.8rem;
  border-radius: 4px;
  border: none;
  background-color: #1DB954;
  color: white;
  cursor: pointer;
  font-weight: bold;
  font-size: 1rem;
  margin-top: 0.5rem;
}
button:hover {
  background-color: #1ED760;
}
p {
  margin-top: 1rem;
}
a {
  color: #1DB954;
  text-decoration: none;
  cursor: pointer;
}
a:hover {
  text-decoration: underline;
}
.error-message {
  color: #f44336;
}
.success-message {
  color: #4CAF50;
}
</style>
