import { usePlayerStore } from '@/stores/player';

let socket = null;
const WS_URL = '/ws';

export const websocketService = {
  connect() {
    // 防止重复连接
    if (socket && socket.readyState === WebSocket.OPEN) {
      return;
    }

    socket = new WebSocket(WS_URL);
    const playerStore = usePlayerStore();

    socket.onopen = () => {
      console.log('WebSocket connected');
    };

    socket.onmessage = (event) => {
      const state = JSON.parse(event.data);
      // 将收到的完整状态交给 Pinia store 处理
      playerStore.setGlobalState(state);
    };

    socket.onclose = () => {
      console.log('WebSocket disconnected. Attempting to reconnect...');
      // 实现简单的重连机制
      setTimeout(() => {
        this.connect();
      }, 3000);
    };

    socket.onerror = (error) => {
      console.error('WebSocket error:', error);
      socket.close();
    };
  },

  disconnect() {
    if (socket) {
      socket.close();
      socket = null;
    }
  },
};
