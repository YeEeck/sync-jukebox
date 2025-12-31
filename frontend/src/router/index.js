// src/router/index.js
import { createRouter, createWebHistory } from 'vue-router';
import { usePlayerStore } from '@/stores/player';
// 导入你的视图组件
import LoginAuth from '../views/LoginAuth.vue';
import Jukebox from '../views/Jukebox.vue';
const routes = [
  {
    path: '/login',
    name: 'login',
    component: LoginAuth,
    // 元信息：这个路由只对未登录的用户开放
    meta: { requiresGuest: true }
  },
  {
    path: '/',
    name: 'jukebox',
    component: Jukebox,
    // 元信息：这个路由需要用户进行认证
    meta: { requiresAuth: true }
  },
];
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});
/**
 * 全局路由守卫
 * 这个函数会在每一次路由导航之前被调用
 */
router.beforeEach((to, from, next) => {
  // 在守卫内部获取 store 实例
  const playerStore = usePlayerStore();
  const isAuthenticated = playerStore.isAuthenticated;
  // 1. 检查目标路由是否需要认证
  if (to.meta.requiresAuth && !isAuthenticated) {
    // 如果用户未认证，则重定向到登录页面
    console.log('Access denied. User not authenticated. Redirecting to /login.');
    next({ name: 'login' });
  }
  // 2. 检查目标路由是否只对访客开放（例如登录页）
  else if (to.meta.requiresGuest && isAuthenticated) {
    // 如果用户已认证，则重定向到主页，防止重复登录
    console.log('User already authenticated. Redirecting to home page.');
    next({ name: 'jukebox' });
  }
  // 3. 在所有其他情况下，允许导航
  else {
    next();
  }
});
export default router;