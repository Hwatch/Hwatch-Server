<script setup lang="ts">
import NavIcon from "../NavIcon.vue";
defineProps<{
  navGroups: {
    id: string;
    label: string;
    items: { id: string; label: string }[];
  }[];
  openGroups: Record<string, boolean>;
  view: string;
  data: { counts: { active:number; waiting:number }; unread_notifications:number; aria2_online:boolean };
  sidebarItems: {
    language: boolean;
    original: boolean;
    demo: boolean;
    aria: boolean;
  };
  translationLanguage: string;
  showOriginal: boolean;
  demoMode: number;
  auth: { username?:string };
  toggleGroup: (id: string) => void;
  changeView: (id: string) => void;
  toggleOriginal: () => void;
  toggleDemo: () => void;
  logout: () => void;
  setLanguage: (value: string) => void;
}>();
</script>
<template>
  <aside aria-label="应用导航">
    <div class="brand"><span>H</span><b>HWatch</b></div>
    <nav>
      <template v-for="group in navGroups" :key="group.id">
        <button
          v-if="group.label"
          type="button"
          class="nav-group"
          :aria-expanded="openGroups[group.id] !== false"
          @click="toggleGroup(group.id)"
        >
          <span>{{ group.label }}</span
          ><span
            class="nav-group-caret"
            :class="{ open: openGroups[group.id] !== false }"
            >⌄</span
          >
        </button>
        <template v-if="openGroups[group.id] !== false">
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
      </template>
    </nav>
    <div class="bottom">
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
        }}</span></button
      ><button
        :class="{ active: view === 'settings' }"
        :aria-current="view === 'settings' ? 'page' : undefined"
        v-on:click="changeView('settings')"
      >
        <NavIcon name="settings" /><span>设置</span></button
      ><small v-if="sidebarItems.aria" :class="{ online: data.aria2_online }"
        >● aria2 {{ data.aria2_online ? "在线" : "离线" }}</small
      >
      <div class="sidebar-account">
        <span
          ><small>当前账户</small><b>{{ auth.username }}</b></span
        >
        <button
          class="sidebar-logout"
          aria-label="退出登录"
          title="退出登录"
          @click="logout"
        >
          <NavIcon name="logout" /><span>退出登录</span>
        </button>
      </div>
    </div>
  </aside>
</template>
