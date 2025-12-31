// src/main.ts

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import './assets/main.css'

// 导入 player store
import { usePlayerStore } from './stores/player'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)

// --- 新的自动登录逻辑 ---

// 在 setup 函数之外使用 store，需要将 pinia 实例作为参数传入
const playerStore = usePlayerStore(pinia)

// store 的 state 在创建时已经从 localStorage 中自动加载了 authHeader，
// 并设置了 isAuthenticated 的初始值。我们直接检查这个状态即可。
if (playerStore.isAuthenticated) {
  console.log('Found stored authentication credentials, attempting to auto-login...');

  // 调用新的 action 来验证凭证并重新建立连接
  playerStore.initializeAuthAndConnect().then(success => {
    if (!success) {
      // 如果凭证失效，initializeAuthAndConnect 内部会调用 logout() 来清理 localStorage。
      // 我们只需要将用户重定向到登录页面。
      console.log('Stored credentials invalid or session expired. Redirecting to login page.');
      router.push('/login');
    } else {
      console.log('Auto-login successful.');
    }
  });
}

app.mount('#app')

