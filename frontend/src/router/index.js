import { createRouter, createWebHistory } from 'vue-router';
import TokenLogin from '../views/TokenLogin.vue';
import Jukebox from '../views/Jukebox.vue';

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: TokenLogin,
    },
    {
      path: '/',
      name: 'jukebox',
      component: Jukebox,
      // 路由守卫：未验证Token则跳转到登录页
      beforeEnter: (to, from, next) => {
        const token = localStorage.getItem('jukebox_token');
        if (token) {
          // 实际应用中可以在这里加一个快速的token验证
          next();
        } else {
          next('/login');
        }
      },
    },
  ],
});

export default router;
