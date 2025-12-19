import axios from 'axios';

const apiClient = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

export default {
  validateToken(token) {
    return apiClient.get(`/validate-token?token=${token}`);
  },
  getLibrary() {
    return apiClient.get('/library');
  },
  uploadSong(formData) {
    return apiClient.post('/library/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },
  removeSong(songId) {
    return apiClient.post('/library/remove', { songId });
  },
  addToPlaylist(songId) {
    return apiClient.post('/playlist/add', { songId });
  },
  removeFromPlaylist(songId) {
    return apiClient.post('/playlist/remove', { songId });
  },
  // --- 播放列表操作 ---
  movePlaylistItem(songId, newIndex) {
    return apiClient.post('/playlist/move', { songId, newIndex });
  },
  shufflePlaylist() {
    return apiClient.post('/playlist/shuffle');
  },
  // 播放器控制
  play() {
    return apiClient.post('/player/play');
  },
  // --- 播放指定曲目 ---
  playSpecific(songId) {
    return apiClient.post('/player/play-specific', { songId });
  },
  pause() {
    return apiClient.post('/player/pause');
  },
  next() {
    return apiClient.post('/player/next');
  },
  prev() {
    return apiClient.post('/player/prev');
  },
  seek(positionMs) {
    return apiClient.post('/player/seek', { positionMs });
  },
};
