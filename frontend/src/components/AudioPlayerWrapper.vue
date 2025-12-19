<template>
  <audio ref="audioPlayer" style="display: none;" @canplay="handleCanPlay"></audio>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue';
import { usePlayerStore } from '@/stores/player';

const audioPlayer = ref(null);
const store = usePlayerStore();
// 增加一个状态，标记当前歌曲是否已准备好播放
let isReadyToPlay = false;

// 监听要播放的歌曲 URL 变化
watch(() => store.currentSongUrl, (newUrl, oldUrl) => {
  if (newUrl && newUrl !== oldUrl) {
    console.log('Audio source changed to:', newUrl);
    isReadyToPlay = false; // 新歌曲需要重新准备
    audioPlayer.value.src = newUrl;
    // 设置src后，浏览器会自动开始加载
  } else if (!newUrl) {
    audioPlayer.value.src = '';
  }
});

// 当音频可以播放时触发
const handleCanPlay = () => {
  isReadyToPlay = true;
  // 如果此时store的状态是播放，就立即开始播放
  if (store.isPlaying) {
    playAudio();
  }
};

// 封装一个带错误处理的播放函数
const playAudio = () => {
  const playPromise = audioPlayer.value.play();
  if (playPromise !== undefined) {
    playPromise.catch(error => {
      // 这里是关键！捕获并打印自动播放失败的错误
      console.error("Audio play failed:", error);
      // 可以在这里给用户一个提示，例如“请点击页面以开始播放”
      store.playbackError = error;
    });
  }
};

// 监听播放状态的变化
watch(() => store.isPlaying, (isPlaying) => {
  // 只有在音频准备好之后才响应播放/暂停指令
  if (isReadyToPlay) {
    if (isPlaying) {
      playAudio();
    } else {
      audioPlayer.value.pause();
    }
  }
});

// 监听并校准播放进度 (这个逻辑保持不变)
watch(() => store.progressMs, (newProgress) => {
  const player = audioPlayer.value;
  if (!player || !isReadyToPlay) return;
  
  const timeDifference = Math.abs(newProgress - player.currentTime * 1000);

  if (timeDifference > 2000 && !player.seeking) {
    console.log(`Syncing time: server=${newProgress/1000}s, client=${player.currentTime}s. Correcting...`);
    player.currentTime = newProgress / 1000;
  }
});

// 监听本地音量变化 (这个逻辑保持不变)
watch(() => store.localVolume, (newVolume) => {
  if (audioPlayer.value) {
    audioPlayer.value.volume = newVolume;
  }
});

onMounted(() => {
  if (audioPlayer.value) {
    // 直接从 store 读取当前的音量值，并设置给 audio 元素
    audioPlayer.value.volume = store.localVolume;
  }
});

</script>
