<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from "vue";
type Theme = "dark" | "light" | "system";
const saved = localStorage.getItem("hwatch-theme");
const theme = ref<Theme>(
  saved === "light" || saved === "system" ? saved : "dark",
);
const media = matchMedia("(prefers-color-scheme: dark)");
function apply() {
  document.documentElement.dataset.theme =
    theme.value === "system" ? (media.matches ? "dark" : "light") : theme.value;
}
watch(theme, () => {
  localStorage.setItem("hwatch-theme", theme.value);
  apply();
});
apply();
media.addEventListener("change", apply);
onBeforeUnmount(() => media.removeEventListener("change", apply));
</script>
<template>
  <label class="theme-control"
    >外观<select v-model="theme" aria-label="外观主题">
      <option value="dark">暗色</option>
      <option value="light">亮色</option>
      <option value="system">跟随系统</option>
    </select></label
  >
</template>
