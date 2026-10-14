<script setup lang="ts">
import { computed } from "vue";
import NavIcon from "../NavIcon.vue";
const props = defineProps<{
  navGroups: {
    id: string;
    label: string;
    items: { id: string; label: string }[];
  }[];
  openGroups: Record<string, boolean>;
  view: string;
  data: {
    counts: { active: number; waiting: number };
    unread_notifications: number;
    aria2_online: boolean;
  };
  sidebarItems: {
    language: boolean;
    original: boolean;
    demo: boolean;
    aria: boolean;
    [key: string]: boolean;
  };
  translationLanguage: string;
  showOriginal: boolean;
  demoMode: number;
  auth: { username?: string };
  advanced: boolean;
  canAdmin: boolean;
  openChangePassword: () => void;
  managementEntries: { id: string; label: string; desc: string }[];
  exitAdvanced: () => void;
  toggleGroup: (id: string) => void;
  changeView: (id: string) => void;
  toggleOriginal: () => void;
  toggleDemo: () => void;
  logout: () => void;
  setLanguage: (value: string) => void;
}>();

// Homepage quick links to advanced management pages (from Settings).
const quickItems = computed(() =>
  props.managementEntries.filter((m) => props.sidebarItems[m.id]),
);
</script>
<template>
  <aside aria-label="应用导航">
    <div class="brand"><span>H</span><b>HWatch</b></div>
    <nav v-if="advanced" aria-label="高级管理">
      <button
        :class="{ active: view === 'settings' }"
        :aria-current="view === 'settings' ? 'page' : undefined"
        @click="changeView('settings')"
      >
        <NavIcon name="settings" /><span class="nav-text">设置</span>
      </button>
      <button
        v-for="m in managementEntries"
        :key="m.id"
        :class="{ active: view === m.id }"
        :aria-current="view === m.id ? 'page' : undefined"
        @click="changeView(m.id)"
      >
        <NavIcon :name="m.id" /><span class="nav-text">{{ m.label }}</span>
      </button>
    </nav>
    <nav v-else>
      <template v-for="group in navGroups" :key="group.id">
        <div
          v-if="group.label"
          class="nav-group"
        >
          {{ group.label }}
        </div>
          <button
            v-for="item in group.items"
            :key="item.id"
            :class="{ active: view === item.id }"
            :aria-current="view === item.id ? 'page' : undefined"
            @click="changeView(item.id)"
          >
            <NavIcon :name="item.id" /><span class="nav-text">{{
              item.label
            }}</span
            ><i
              v-if="
                item.id === 'downloads' &&
                data.counts.active + data.counts.waiting
              "
              >{{ data.counts.active + data.counts.waiting }}</i
            >
            <i
              v-if="item.id === 'notifications' && data.unread_notifications"
              >{{ data.unread_notifications }}</i
            >
          </button>
      </template>
      <button
        v-for="m in advanced || !canAdmin ? [] : quickItems"
        :key="'quick-' + m.id"
        :class="{ active: view === m.id }"
        :aria-current="view === m.id ? 'page' : undefined"
        @click="changeView(m.id)"
      >
        <NavIcon :name="m.id" /><span class="nav-text">{{ m.label }}</span>
      </button>
    </nav>
    <div class="bottom">
      <details class="sidebar-shortcuts">
        <summary>
          <NavIcon name="language" /><span>显示偏好</span
          ><span class="shortcut-caret">⌄</span>
        </summary>
        <label v-if="sidebarItems.language" class="sidebar-language"
          ><NavIcon name="language" /><select
            class="language-select"
            :value="translationLanguage"
            @change="setLanguage(($event.target as HTMLSelectElement).value)"
            aria-label="内容语言"
          >
            <option value="zh-CN">简中</option>
            <option value="zh-TW">繁中</option>
            <option value="en">EN</option>
            <option value="ja">日本語</option>
          </select></label
        ><button v-if="sidebarItems.original" v-on:click="toggleOriginal">
          <NavIcon name="privacy" /><span>{{
            showOriginal ? "显示译文" : "显示原文"
          }}</span></button
        ><button
          v-if="sidebarItems.demo"
          :class="{ active: demoMode }"
          v-on:click="toggleDemo"
        >
          <NavIcon name="demo" /><span>{{
            demoMode === 0
              ? "开启演示模式"
              : demoMode === 1
                ? "进入强演示模式"
                : "关闭演示模式"
          }}</span>
        </button>
      </details>
      <button
        v-if="advanced"
        class="advanced-back"
        aria-label="返回正常页面"
        title="返回正常页面"
        @click="exitAdvanced()"
      >
        <svg
          class="nav-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.8"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="m15 18-6-6 6-6"></path>
        </svg>
        <span>返回</span>
      </button>
      <button
        v-else-if="canAdmin"
        :class="{ active: view === 'settings' }"
        :aria-current="view === 'settings' ? 'page' : undefined"
        v-on:click="changeView('settings')"
      >
        <NavIcon name="settings" /><span>设置</span></button
      ><small v-if="sidebarItems.aria" :class="{ online: data.aria2_online }"
        >● aria2 {{ data.aria2_online ? "在线" : "离线" }}</small
      >
      <details class="sidebar-account-menu">
        <summary class="sidebar-account">
          <span class="account-avatar">{{
            (auth.username || "H").slice(0, 1).toUpperCase()
          }}</span>
          <span
            ><small>当前账户</small><b>{{ auth.username }}</b></span
          ><span class="account-menu-caret">⌄</span>
        </summary>
        <button class="sidebar-logout" aria-label="修改密码" title="修改密码" @click="openChangePassword()">
          <NavIcon name="privacy" /><span>修改密码</span></button
        ><button
          class="sidebar-logout"
          aria-label="退出登录"
          title="退出登录"
          @click="logout"
        >
          <NavIcon name="logout" /><span>退出登录</span>
        </button>
      </details>
    </div>
  </aside>
</template>
