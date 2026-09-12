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
    <template v-else-if="name==='notifications'"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></template>
    <template v-else-if="name==='menu'"><path d="M4 6h16M4 12h16M4 18h16"/></template>
    <template v-else-if="name==='settings'"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-2.83 2.83-.06-.06A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1v.1h-4V21a1.7 1.7 0 0 0-1.1-1.6 1.7 1.7 0 0 0-1.88.34l-.06.06-2.83-2.83.06-.06A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1-.4h-.1v-4H3A1.7 1.7 0 0 0 4.6 8.5a1.7 1.7 0 0 0-.34-1.88l-.06-.06 2.83-2.83.06.06A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1v-.1h4V3a1.7 1.7 0 0 0 1.1 1.6 1.7 1.7 0 0 0 1.88-.34l.06-.06 2.83 2.83-.06.06A1.7 1.7 0 0 0 19.4 9c.12.38.33.72.6 1 .27.27.62.48 1 .6h.1v4H21a1.7 1.7 0 0 0-1.6.4z"/></template>
    <template v-else-if="name==='language'"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/></template>
    <template v-else-if="name==='privacy'"><path d="M2 12s3.5-6 10-6 10 6 10 6-3.5 6-10 6S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></template>
    <template v-else-if="name==='demo'"><rect x="3" y="4" width="18" height="14" rx="2"/><path d="m9 9 5 3-5 3zM8 21h8"/></template>
    <template v-else-if="name==='network'"><circle cx="6" cy="12" r="2"/><circle cx="18" cy="6" r="2"/><circle cx="18" cy="18" r="2"/><path d="m8 11 8-4m-8 6 8 4"/></template>
    <template v-else-if="name==='exclude'"><circle cx="12" cy="12" r="9"/><path d="m6 6 12 12"/></template>
    <template v-else-if="name==='import'"><path d="M12 3v12m0 0-4-4m4 4 4-4M4 20h16"/></template>
  </svg>
</template>
