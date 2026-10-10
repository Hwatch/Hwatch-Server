<script setup lang="ts">
import type { createToasts } from "../toast";
defineProps<{ toasts: ReturnType<typeof createToasts> }>();
</script>
<template>
  <div v-if="toasts.list.length" class="toasts" aria-live="polite">
    <div
      v-for="t in toasts.list"
      :key="t.id"
      class="toast"
      :class="'toast--' + t.type"
      :role="t.type === 'error' ? 'alert' : 'status'"
    >
      <span class="toast__message">{{ t.message }}</span>
      <button
        type="button"
        class="toast__close"
        aria-label="关闭通知"
        @click="toasts.dismiss(t.id)"
      >
        ×
      </button>
    </div>
  </div>
</template>
