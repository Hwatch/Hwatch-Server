<script setup lang="ts">
defineProps<{name:string}>()
</script>

<template>
  <svg class="nav-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <template v-if="name==='home'"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></template>
    <template v-else-if="name==='subscriptions'"><circle cx="12" cy="8" r="3"/><path d="M5.5 20a6.5 6.5 0 0 1 13 0"/><path d="M19 6v4M17 8h4"/></template>
    <template v-else-if="name==='downloads'"><path d="M12 3v12m0 0 5-5m-5 5-5-5"/><path d="M4 19h16"/></template>
    <template v-else-if="name==='undownloaded'"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="m10 9 5 3-5 3z"/><path d="M18 2v6m-3-3 3 3 3-3"/></template>
    <template v-else-if="name==='library'"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="m10 9 5 3-5 3z"/></template>
    <template v-else-if="name==='translations'"><path d="M4 5h9M8.5 3v2c0 5-2 8-5 10"/><path d="M5 10c1.5 2.5 3.5 4 6 5"/><path d="m14 20 4-10 4 10m-6.5-4h5"/></template>
  </svg>
</template>
