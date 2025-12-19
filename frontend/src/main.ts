// src/main.js

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

// 在挂载应用之前，尝试自动登录
const playerStore = usePlayerStore(pinia) // 注意：在 setup 外使用 store 需要传入 pinia 实例
const token = localStorage.getItem('jukebox_token')

if (token) {
  // 使用已有的 action 尝试自动连接和获取状态
  // 这个 action 已经是 async 的，但我们这里可以“即发即忘”，让它在后台执行
  playerStore.validateTokenAndConnect(token).then(success => {
    if (!success) {
      // 如果 token 失效，清空它并强制跳转到登录页
      localStorage.removeItem('jukebox_token')
      router.push('/login')
    }
  })
}

app.mount('#app')
