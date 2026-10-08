<script setup lang="ts">
defineProps<{ status: string }>();
const labels: Record<string, string> = {
  discovered: "待下载",
  queued: "建队中",
  submitting: "提交中",
  waiting: "等待中",
  active: "下载中",
  paused: "已暂停",
  failed: "失败",
  excluded: "已排除",
  completed: "已完成",
};
</script>
<template>
  <span class="status-badge" :class="status">{{
    labels[status] || status
  }}</span>
</template>
