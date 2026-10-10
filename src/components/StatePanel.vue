<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
    kind?: "empty" | "error" | "loading";
  }>(),
  { kind: "empty" },
);
</script>
<template>
  <div
    class="state-panel"
    :class="kind"
    :role="kind === 'error' ? 'alert' : 'status'"
    :aria-busy="kind === 'loading'"
  >
    <span aria-hidden="true" class="state-symbol">{{
      kind === "error" ? "!" : kind === "loading" ? "…" : "—"
    }}</span
    ><b>{{ title }}</b>
    <p v-if="description">{{ description }}</p>
    <slot />
  </div>
</template>
