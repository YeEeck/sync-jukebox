<template>
  <div class="search-bar">
    <input
        type="text"
        v-model="model"
        placeholder="Search by title or artist (a: or @)"
    />
  </div>
</template>

<script setup>
const model = defineModel();
</script>

<style scoped>
.search-bar {
  /* --- MODIFIED: 减小了下边距 --- */
  /* 修改前: margin: 1rem 0 0.5rem; */
  margin: 0.5rem 0 0.5rem; /* 将下边距从 0.5rem 减小到 0.25rem */
  padding: 0 0.25rem;
}

.search-bar input {
  width: 100%;
  padding: 0.5rem;
  background-color: #282828;
  border: 1px solid #535353;
  color: #fff;
  border-radius: 4px;
  box-sizing: border-box;
}

.search-bar input::placeholder {
  color: #888;
}
</style>
