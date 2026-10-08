<script setup lang="ts">
import ThemeControl from './ThemeControl.vue';
defineProps<{title:string;description?:string}>();
</script>
<template><header class="page-header"><div><h1 tabindex="-1">{{ title }}</h1><p v-if="description" class="page-description">{{ description }}</p></div><div class="page-header-actions"><ThemeControl/><slot/></div></header></template>
