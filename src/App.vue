<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import NavIcon from "./NavIcon.vue";
import SettingsView from "./SettingsView.vue";
import PageHeader from "./components/PageHeader.vue";
import MediaCard from "./components/MediaCard.vue";
import ToastContainer from "./components/ToastContainer.vue";
import SidebarView from "./components/SidebarView.vue";
import JSONImport from "./components/JSONImport.vue";
import PaginationBar from "./components/PaginationBar.vue";
import StatePanel from "./components/StatePanel.vue";
import StatusBadge from "./components/StatusBadge.vue";
import { dialogFocus as vDialogFocus } from "./dialogFocus";
import { useHWatchApp } from "./useHWatchApp";
const {
  auth,
  authReady,
  authUsername,
  authPassword,
  authPasswordConfirm,
  authError,
  empty,
  data,
  videos,
  total,
  view,
  query,
  status,
  page,
  notice,
  busy,
  subscribeError,
  translationFileName,
  translationFileSize,
  translationStage,
  modelURL,
  selected,
  importHTML,
  translationLanguage,
  creatorQuery,
  creatorResults,
  selectedCreator,
  searchingCreators,
  downloadExisting,
  actions,
  toasts,
  subscribePending,
  subscribeStatus,
  translationImportPending,
  translationImportError,
  vdfExecutePending,
  downloadedStat,
  viewTitles,
  pageDescriptions,
  subscriptionQuery,
  subscriptionState,
  exclusionQuery,
  pageLoading,
  pageError,
  videosLoading,
  videosError,
  notificationSeverity,
  visibleNotifications,
  filteredSubscriptionGroups,
  filteredExclusions,
  vdfFilter,
  vdfPage,
  expandedVdfGroups,
  filteredVdfGroups,
  visibleVdfGroups,
  demoMode,
  sidebarCollapsed,
  sidebarItems,
  navGroups,
  openGroups,
  toggleGroup,
  showOriginal,
  settings,
  backups,
  backupDirectory,
  backupSensitive,
  notifications,
  notificationTab,
  auditLogs,
  auditResultLabel,
  switchNotificationTab,
  exclusions,
  mediaExclusionTotal,
  mediaScan,
  cleanupPreview,
  vdfPreviewData,
  deletionListText,
  integrityNotice,
  integrityBusy,
  integrityImportPending,
  cleanupStatusLabel,
  planText,
  planPreviewData,
  planResult,
  deletionPlans,
  planNotice,
  planBusy,
  planPreviewPending,
  planExecutePending,
  actionPending,
  videoActionKey,
  settingsBusy,
  backupsBusy,
  importBusy,
  planStatusLabel,
  vdfGroupStatusLabel,
  vdfGroupBadgeClass,
  vdfItemStatusLabel,
  deleteTarget,
  deleteMode,
  deleting,
  exclusionPreview,
  exclusionTotal,
  facets,
  showFilters,
  searchScope,
  tagMode,
  selectedTags,
  excludedTags,
  creatorFilter,
  codecFilter,
  minHeight,
  sortBy,
  sortOrder,
  randomItems,
  randomError,
  randomMode,
  loadRandom,
  pages,
  unfinishedDownloadTasks,
  completedDownloadTasks,
  showCompletedTasks,
  rlLabel,
  activeRateLimits,
  fmtCountdown,
  sourceLabel,
  rateLimitedReason,
  subscriptionTypeLabel,
  size,
  duration,
  localTime,
  resolution,
  mediaSummary,
  pct,
  api,
  submitAuth,
  logout,
  onCoverError,
  coverState,
  markCoverLoaded,
  registerCoverImg,
  markNotificationsRead,
  restoreExclusion,
  loadIntegrity,
  logs,
  logsTotal,
  logsLevel,
  logsError,
  loadLogs,
  isAdmin,
  shorts,
  shortsError,
  shortsBusy,
  loadShorts,
  shortsWatchedDays,
  setShortsWatchedDays,
  users,
  usersError,
  loadUsers,
  refreshDashboard,
  createUser,
  updateUserRole,
  deleteUser,
  resetUserPassword,
  changeOwnPassword,
  uploadAvatar,
  clearAvatar,
  timeline,
  timelineTotal,
  timelineError,
  loadTimeline,
  previewTransRunnerPlan,
  executeTransRunnerPlan,
  recoverPlan,
  importDeletionList,
  vdfExecute,
  applyDeletionList,
  openDelete,
  confirmDelete,
  previewExclusions,
  loadVideos,
  createBackup,
  restoreBackup,
  changeView,
  subscribe,
  searchCreators,
  chooseCreator,
  check,
  toggle,
  removeSub,
  videoAction,
  downloadAll,
  toggleDownloads,
  saveSettings,
  importPage,
  exportTranslation,
  importTranslation,
  refreshNFO,
  displayTitle,
  displaySubtitle,
  displayTags,
  toggleOriginal,
  toggleDemo,
  toggleSidebar,
  setSidebarCollapsed,
  updateSidebarItem,
} = useHWatchApp();

// Advanced management pages, shown in the sidebar instead of the main nav.
const managementEntries: { id: string; label: string; desc: string }[] = [
  { id: "subscriptions", label: "订阅管理", desc: "创作者 / 来源订阅与检查" },
  { id: "downloads", label: "下载中心", desc: "进行中与历史下载任务" },
  { id: "undownloaded", label: "未下载", desc: "已发现但尚未下载的视频" },
  { id: "excluded", label: "已排除", desc: "不再自动下载的媒体" },
  { id: "integrity", label: "媒体完整性", desc: "扫描、删除清单核对与 VDF" },
  { id: "translations", label: "翻译管理", desc: "WebAI 翻译导出 / 回导" },
  { id: "logs", label: "日志中心", desc: "订阅检查与任务日志" },
  { id: "users", label: "用户管理", desc: "账号与角色（管理员 / 仅观看）" },
];
const ADVANCED_VIEWS = ["settings", ...managementEntries.map((m) => m.id)];
const advancedMode = ref(false);
const advanced = computed(() => advancedMode.value);
function openAdvanced() {
  advancedMode.value = true;
  changeView("settings");
}
const timelineGroups = computed(() => {
  const map: Record<string, any[]> = {};
  for (const it of timeline.value) {
    const day = localTime(it.watched_at).slice(0, 10);
    (map[day] ||= []).push(it);
  }
  return Object.keys(map).map((date) => ({ date, items: map[date] }));
});
const TIMELINE_PAGE_SIZE = 24;
const TIMELINE_DAYS_PAGE = 20;
const timelineDaysShown = ref(TIMELINE_DAYS_PAGE);
const visibleTimelineGroups = computed(() =>
  timelineGroups.value.slice(0, timelineDaysShown.value),
);
watch(timeline, () => {
  timelineDaysShown.value = TIMELINE_DAYS_PAGE;
  nextTick(updateTimelineCols);
});
watch(view, (v) => {
  if (v === "timeline") nextTick(updateTimelineCols);
});
const timelineDay = ref<string | null>(null);
const timelineDayPage = ref(0);
const timelineDayItems = computed(() =>
  timelineDay.value
    ? timelineGroups.value.find((g) => g.date === timelineDay.value)?.items || []
    : [],
);
const timelineDayPages = computed(() =>
  Math.max(1, Math.ceil(timelineDayItems.value.length / TIMELINE_PAGE_SIZE)),
);
const timelineDayPageItems = computed(() =>
  timelineDayItems.value.slice(
    timelineDayPage.value * TIMELINE_PAGE_SIZE,
    (timelineDayPage.value + 1) * TIMELINE_PAGE_SIZE,
  ),
);
function openTimelineDay(date: string) {
  timelineDay.value = date;
  timelineDayPage.value = 0;
}
function closeTimelineDay() {
  timelineDay.value = null;
}
// Remember the last normal page so the sidebar "back" returns there.
const lastNormalView = ref("home");
watch(view, (next) => {
  if (!ADVANCED_VIEWS.includes(next)) lastNormalView.value = next;
});
function exitAdvanced() {
  advancedMode.value = false;
  changeView(lastNormalView.value || "home");
}
const newUser = ref({ username: "", password: "", role: "viewer" });
const avatarInput = ref<HTMLInputElement | null>(null);
const avatarTarget = ref<{ kind: string; id: string } | null>(null);
function pickAvatar(target: { kind: string; id: string }) {
  avatarTarget.value = target;
  avatarInput.value?.click();
}
function bumpSelfAvatar() {
  if (auth.value)
    auth.value = {
      ...auth.value,
      avatar_url: (auth.value.avatar_url || "").split("?")[0] + "?t=" + Date.now(),
    };
}
async function onAvatarFile(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  const target = avatarTarget.value;
  if (!file || !target) return;
  const result = await uploadAvatar(target.kind, target.id, file);
  input.value = "";
  if (result.duplicate) return;
  if (result.ok) {
    notice.value = "头像已更新";
    if (target.kind === "user") await loadUsers();
    else if (target.kind === "sub") await refreshDashboard();
    else bumpSelfAvatar();
  } else {
    notice.value = "上传失败：" + (result.error || "");
  }
}
async function clearAvatarTarget(kind: string, id: string) {
  const result = await clearAvatar(kind, id);
  if (result.duplicate) return;
  if (result.ok) {
    notice.value = "头像已清除";
    if (kind === "user") await loadUsers();
    else if (kind === "sub") await refreshDashboard();
    else bumpSelfAvatar();
  } else {
    notice.value = "清除失败：" + (result.error || "");
  }
}
const changePasswordOpen = ref(false);
const changePw = ref({ current: "", next: "", confirm: "" });
async function submitCreateUser() {
  const result = await createUser({ ...newUser.value });
  if (result.duplicate) return;
  if (result.ok) {
    notice.value = "已创建用户：" + newUser.value.username;
    newUser.value = { username: "", password: "", role: "viewer" };
    await loadUsers();
  } else {
    notice.value = "创建失败：" + (result.error || "");
  }
}
async function toggleRole(u: any) {
  const role = u.role === "admin" ? "viewer" : "admin";
  const result = await updateUserRole(u.id, role);
  if (result.duplicate) return;
  if (result.ok) {
    await loadUsers();
    notice.value = "已更新角色";
  } else {
    notice.value = "更新失败：" + (result.error || "");
  }
}
async function promptResetPassword(u: any) {
  const pw = prompt("为 " + u.username + " 设置新密码（至少 10 位）：");
  if (!pw) return;
  const result = await resetUserPassword(u.id, pw);
  if (result.duplicate) return;
  notice.value = result.ok ? "密码已重置" : "重置失败：" + (result.error || "");
}
async function removeUser(u: any) {
  if (!confirm("删除用户 " + u.username + "？其登录会话将立即失效。")) return;
  const result = await deleteUser(u.id);
  if (result.duplicate) return;
  if (result.ok) {
    await loadUsers();
    notice.value = "已删除用户";
  } else {
    notice.value = "删除失败：" + (result.error || "");
  }
}
async function submitChangePassword() {
  if (changePw.value.next !== changePw.value.confirm) {
    notice.value = "两次输入的新密码不一致";
    return;
  }
  const result = await changeOwnPassword(changePw.value.current, changePw.value.next);
  if (result.duplicate) return;
  if (result.ok) {
    notice.value = "密码已修改";
    changePasswordOpen.value = false;
    changePw.value = { current: "", next: "", confirm: "" };
  } else {
    notice.value = "修改失败：" + (result.error || "");
  }
}
const shortsMuted = ref(true);
const shortsRotated = ref(false);
const shortsFeed = ref<HTMLElement | null>(null);
function toggleShortsRotate() {
  const feed = shortsFeed.value;
  let idx = activeShort.value;
  if (feed && idx < 0) idx = Math.round(feed.scrollTop / (feed.clientHeight || 1));
  shortsRotated.value = !shortsRotated.value;
  nextTick(() => {
    const el = shortsFeed.value;
    if (!el || idx < 0) return;
    el.scrollTop = idx * el.clientHeight;
  });
}
const shortLandscape = ref<Record<string, boolean>>({});
const isNarrow = ref(
  typeof window !== "undefined" && window.matchMedia("(max-width: 680px)").matches,
);
function syncNarrow() {
  isNarrow.value = window.matchMedia("(max-width: 680px)").matches;
  updateTimelineCols();
}
const timelineCols = ref(5);
function updateTimelineCols() {
  const el = document.querySelector(".timeline") as HTMLElement | null;
  const width = el ? el.clientWidth : window.innerWidth;
  timelineCols.value = Math.min(8, Math.max(2, Math.floor(width / 210)));
}
function onShortKey(e: KeyboardEvent) {
  if (e.key === "Escape") shortsRotated.value = false;
}
onMounted(() => {
  window.addEventListener("resize", syncNarrow);
  window.addEventListener("keydown", onShortKey);
  updateTimelineCols();
});
onUnmounted(() => {
  window.removeEventListener("resize", syncNarrow);
  window.removeEventListener("keydown", onShortKey);
});
function dropShort(v: any) {
  const i = shorts.value.findIndex((x) => x.video_id === v.video_id);
  if (i >= 0) shorts.value.splice(i, 1);
}
function onShortMeta(v: any, e: Event) {
  const el = e.target as HTMLVideoElement;
  if (el.videoWidth === 0) {
    // Audio track decoded but no picture (e.g. HEVC with hev1 tag on Safari).
    flashShort("该视频编码不受支持，已跳过");
    dropShort(v);
    return;
  }
  shortLandscape.value[v.video_id] = el.videoWidth > el.videoHeight;
}
function onShortPlaying(v: any, e: Event) {
  shortBuffering.value[v.video_id] = false;
  if ((e.target as HTMLVideoElement).videoWidth === 0) dropShort(v);
}
const activeShort = ref(-1);
const shortBuffering = ref<Record<string, boolean>>({});
function shortSrc(i: number, v: any): string | undefined {
  return Math.abs(i - activeShort.value) <= 1
    ? "/media/" + v.video_id
    : undefined;
}
function primeShort(idx: number) {
  if (idx < 0) return;
  const video = document.querySelector(`.short[data-index="${idx}"] video`) as HTMLVideoElement | null;
  if (!video) return;
  video.preload = "auto";
  if (video.readyState === 0) video.load();
}
// iOS limits the number of live media elements (~16); recycle slides behind
// the active one so the feed keeps only a small window.
function trimShorts() {
  const feed = shortsFeed.value;
  if (!feed) return;
  const removeCount = Math.min(activeShort.value - 2, shorts.value.length - 12);
  if (removeCount <= 0) return;
  const slideHeight = feed.clientHeight;
  shorts.value.splice(0, removeCount);
  activeShort.value -= removeCount;
  nextTick(() => {
    if (shortsFeed.value) shortsFeed.value.scrollTop -= removeCount * slideHeight;
  });
}
let shortsObserver: IntersectionObserver | null = null;
function ensureShortsObserver() {
  if (shortsObserver || typeof IntersectionObserver === "undefined") return;
  shortsObserver = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        const video = entry.target.querySelector("video") as HTMLVideoElement | null;
        if (!video) continue;
        if (entry.isIntersecting && entry.intersectionRatio >= 0.6) {
          const idx = Number((entry.target as HTMLElement).dataset.index || 0);
          activeShort.value = idx;
          nextTick(() => {
            const el = document.querySelector(
              `.short[data-index="${idx}"] video`,
            ) as HTMLVideoElement | null;
            if (!el) return;
            if (el.readyState === 0) el.load();
            el.play().catch(() => {});
            primeShort(idx + 1);
          });
          trimShorts();
          if (idx >= shorts.value.length - 2) {
            void loadShorts(true).then(() => nextTick(trimShorts));
          }
        } else {
          video.pause();
        }
      }
    },
    { threshold: [0, 0.6, 1] },
  );
}
function registerShort(el: unknown, i: number) {
  if (!el) return;
  (el as HTMLElement).dataset.index = String(i);
  ensureShortsObserver();
  shortsObserver?.observe(el as Element);
}
function onShortError(e: Event) {
  const el = e.currentTarget as HTMLVideoElement;
  if (!el.getAttribute("src")) return;
  const slide = el.closest(".short") as HTMLElement | null;
  if (!slide) return;
  const idx = Number(slide.dataset.index ?? -1);
  if (idx >= 0) shorts.value.splice(idx, 1);
}
const shortPos = ref<Record<string, number>>({});
const shortDur = ref<Record<string, number>>({});
function fmtClock(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) seconds = 0;
  const m = Math.floor(seconds / 60);
  const sec = Math.floor(seconds % 60);
  return m + ":" + String(sec).padStart(2, "0");
}
function onShortTime(v: any, e: Event) {
  const video = e.target as HTMLVideoElement;
  shortPos.value[v.video_id] = video.currentTime;
  shortDur.value[v.video_id] = video.duration || 0;
}
function seekShort(e: Event) {
  const input = e.target as HTMLInputElement;
  const video = input.closest(".short")?.querySelector("video") as HTMLVideoElement | null;
  if (video) video.currentTime = Number(input.value);
}
const SHORT_SEEK_SECONDS = 5;
const shortFlash = ref<{ main: string; sub?: string } | null>(null);
let shortFlashTimer: ReturnType<typeof setTimeout> | undefined;
function flashShort(main: string, sub?: string) {
  shortFlash.value = { main, sub };
  if (shortFlashTimer) clearTimeout(shortFlashTimer);
  shortFlashTimer = setTimeout(() => (shortFlash.value = null), 900);
}
function skipBy(video: HTMLVideoElement, delta: number) {
  const max = isFinite(video.duration) && video.duration > 0 ? video.duration : Infinity;
  video.currentTime = Math.min(Math.max(0, video.currentTime + delta), max);
}
function onShortTap(e: MouseEvent) {
  const video = e.currentTarget as HTMLVideoElement;
  const timeText = () => `${fmtClock(video.currentTime)} / ${fmtClock(video.duration || 0)}`;
  const toggle = () => {
    if (video.paused) video.play().catch(() => {});
    else video.pause();
    flashShort(video.paused ? "⏸ 暂停" : "▶ 播放", timeText());
  };
  if (!shortsRotated.value) {
    toggle();
    return;
  }
  const ratio = e.offsetX / (video.clientWidth || 1);
  if (ratio < 0.35) {
    skipBy(video, -SHORT_SEEK_SECONDS);
    flashShort(`⏪ ${SHORT_SEEK_SECONDS} 秒`, timeText());
  } else if (ratio > 0.65) {
    skipBy(video, SHORT_SEEK_SECONDS);
    flashShort(`${SHORT_SEEK_SECONDS} 秒 ⏩`, timeText());
  } else {
    toggle();
  }
}
watch(view, (next) => {
  if (next !== "shorts") {
    shortsObserver?.disconnect();
    shortsObserver = null;
    shortsRotated.value = false;
  }
});const logDetail = ref<any>(null);
function subscriptionName(slug: string): string {
  const sub = (data.value?.subscriptions || []).find((x: any) => x.slug === slug);
  return sub ? sub.name : "";
}
function logLines(log: any): string {
  const evs = log.events || [];
  if (!evs.length) return `${localTime(log.created_at)} 【完成】${log.message}`;
  return evs
    .map((e: any) => `${localTime(e.at)} 【${e.tag || "日志"}】${e.message}`)
    .join("\n");
}
async function copyLog(log: any) {
  try {
    await navigator.clipboard.writeText(logLines(log));
    notice.value = "日志已复制";
  } catch {
    notice.value = "复制失败，请手动选择";
  }
}
</script>
<template>
  <a v-if="auth.authenticated" href="#main-content" class="skip-link"
    >跳到主要内容</a
  >
  <div v-if="!authReady" class="auth-screen">
    <div class="auth-card">
      <div class="auth-logo">H</div>
      <p>正在载入 HWatch…</p>
    </div>
  </div>
  <div v-else-if="!auth.authenticated" class="auth-screen">
    <form class="auth-card" @submit.prevent="submitAuth">
      <div class="auth-logo">H</div>
      <h1>{{ auth.configured ? "登录 HWatch" : "创建管理员账号" }}</h1>
      <p>
        {{
          auth.configured
            ? "登录后管理订阅、下载与媒体库。"
            : "这是首次启动。该账号将拥有 HWatch 的管理权限。"
        }}
      </p>
      <label
        >用户名<input
          v-model.trim="authUsername"
          autocomplete="username"
          minlength="3"
          maxlength="64"
          required
          autofocus /></label
      ><label
        >密码<input
          v-model="authPassword"
          type="password"
          :autocomplete="auth.configured ? 'current-password' : 'new-password'"
          minlength="10"
          maxlength="256"
          required /></label
      ><label v-if="!auth.configured"
        >确认密码<input
          v-model="authPasswordConfirm"
          type="password"
          autocomplete="new-password"
          minlength="10"
          maxlength="256"
          required
      /></label>
      <p v-if="authError" class="auth-error">{{ authError }}</p>
      <button class="primary" :disabled="busy" type="submit">
        {{ busy ? "请稍候…" : auth.configured ? "登录" : "创建并进入" }}</button
      ><small v-if="!auth.configured"
        >密码至少 10 个字符。账号信息仅保存在本机 SQLite 数据库中。</small
      >
    </form>
  </div>
  <div
    v-if="auth.authenticated"
    class="shell"
    :class="{
      'demo-mode': demoMode > 0,
      'strong-demo': demoMode === 2,
      'sidebar-collapsed': sidebarCollapsed,
      immersive: view === 'shorts',
    }"
  >
    <button
      class="sidebar-toggle"
      :aria-label="sidebarCollapsed ? '展开侧栏' : '收起侧栏'"
      v-on:click="toggleSidebar"
    >
      <NavIcon name="menu" />
    </button>
    <button
      v-if="!sidebarCollapsed"
      class="sidebar-backdrop"
      aria-label="关闭侧栏"
      @click="setSidebarCollapsed(true)"
    ></button>
    <SidebarView
      :nav-groups="navGroups"
      :open-groups="openGroups"
      :view="view"
      :data="data"
      :sidebar-items="sidebarItems"
      :translation-language="translationLanguage"
      :show-original="showOriginal"
      :demo-mode="demoMode"
      :auth="auth"
      :advanced="advanced"
      :can-admin="isAdmin"
      :open-advanced="openAdvanced"
      :open-change-password="() => (changePasswordOpen = true)"
      :open-change-avatar="() => pickAvatar({ kind: 'self', id: '' })"
      :management-entries="managementEntries"
      :exit-advanced="exitAdvanced"
      :toggle-group="toggleGroup"
      :change-view="changeView"
      :toggle-original="toggleOriginal"
      :toggle-demo="toggleDemo"
      :logout="logout"
      :set-language="(value) => (translationLanguage = value)"
    />
    <main id="main-content" tabindex="-1">
      <PageHeader
        :title="viewTitles[view] || 'HWatch'"
        :description="pageDescriptions[view]"
      >
        <button
          v-if="view === 'undownloaded'"
          class="primary"
          :disabled="actionPending('download-all') || !total"
          :aria-busy="actionPending('download-all')"
          @click="downloadAll"
        >
          {{
            actionPending("download-all") ? "提交中…" : `下载全部（${total}）`
          }}</button
        ><button
          v-else-if="view === 'downloads'"
          class="download-master"
          :class="{ paused: data.downloads_paused }"
          :disabled="actionPending('downloads-toggle')"
          :aria-busy="actionPending('downloads-toggle')"
          @click="toggleDownloads"
        >
          {{
            actionPending("downloads-toggle")
              ? "处理中…"
              : data.downloads_paused
                ? "继续全部"
                : "暂停全部"
          }}</button
        ><button
          v-else-if="
            isAdmin &&
            ![
              'settings',
              'translations',
              'notifications',
              'excluded',
              'integrity',
            ].includes(view)
          "
          class="primary"
          :disabled="actionPending('check:all')"
          :aria-busy="actionPending('check:all')"
          @click="check()"
        >
          {{ actionPending("check:all") ? "检查中…" : "检查更新" }}
        </button>
      </PageHeader>
      <StatePanel v-if="pageLoading" kind="loading" title="正在加载页面…" />
      <StatePanel
        v-if="pageError"
        kind="error"
        title="页面加载失败"
        :description="pageError"
        ><button @click="changeView(view, false)">重试</button></StatePanel
      >
      <StatePanel
        v-if="
          videosError && ['library', 'undownloaded', 'downloads'].includes(view)
        "
        kind="error"
        title="列表加载失败"
        :description="videosError"
        ><button @click="loadVideos">重试加载</button></StatePanel
      >
      <ToastContainer :toasts="toasts" />
      <div v-if="data.health_warnings.length" class="health-alert">
        <b>系统运行提醒</b
        ><span
          v-for="warning in data.health_warnings"
          :key="warning.code"
          :class="warning.severity"
          >{{ warning.message }}</span
        >
      </div>
      <section v-if="view === 'home'">
        <div class="stats">
          <article>
            <small>订阅作者</small><b>{{ data.subscriptions.length }}</b
            ><span
              >启用
              {{ data.subscriptions.filter((s) => s.enabled).length }}</span
            >
          </article>
          <article>
            <small>下载任务</small
            ><b>{{ data.counts.active + data.counts.waiting }}</b
            ><span>{{ data.counts.active }} 个进行中</span>
          </article>
          <article>
            <small>{{ downloadedStat.label }}</small
            ><b>{{ data.downloaded_videos }}</b
            ><span>{{ downloadedStat.sub }}</span>
          </article>
          <article>
            <small>累计流量</small><b>{{ size(data.traffic_bytes) }}</b
            ><span>下载器接收流量</span>
          </article>
          <article>
            <small>影视库</small><b>{{ size(data.library_bytes) }}</b
            ><span>{{ data.counts.videos }} 个记录</span>
          </article>
        </div>
        <div class="storage">
          <div>
            <b>NAS 存储空间</b
            ><span
              >可用 {{ size(data.disk_free) }} / 总计
              {{ size(data.disk_total) }}</span
            >
          </div>
          <progress
            :value="data.disk_total - data.disk_free"
            :max="data.disk_total || 1"
          />
        </div>
        <h2>最近入库</h2>
        <div class="library compact">
          <article
            v-for="v in data.videos"
            :key="v.video_id"
            role="button"
            tabindex="0"
            v-on:click="selected = v"
            @keydown.enter="selected = v"
            @keydown.space.prevent="selected = v"
          >
            <img
              :src="'/thumb/' + v.video_id"
              :alt="displayTitle(v)"
              loading="lazy"
            />
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(v, v.subscription_slug + " · " + v.status)
              }}</small>
            </div>
          </article>
        </div>
        <h2 class="recent-heading">最近播放</h2>
        <div class="library compact">
          <article
            v-for="v in data.recently_played"
            :key="v.video_id"
            role="button"
            tabindex="0"
            v-on:click="selected = v"
            @keydown.enter="selected = v"
            @keydown.space.prevent="selected = v"
          >
            <img
              :src="'/thumb/' + v.video_id"
              :alt="displayTitle(v)"
              loading="lazy"
            />
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(v, v.creator_name || v.subscription_slug)
              }}</small>
            </div>
          </article>
          <div v-if="!data.recently_played.length" class="empty">
            还没有播放记录
          </div>
        </div>
      </section>
      <section v-else-if="view === 'subscriptions'">
        <div class="panel creator-search">
          <h2>通过链接订阅</h2>
          <div class="inline subscription-url">
            <input
              v-model="modelURL"
              type="url"
              autocomplete="off"
              placeholder="粘贴作者页或播放清单链接"
            /><button
              class="primary"
              :disabled="subscribePending || !modelURL.trim()"
              :aria-busy="subscribePending"
              @click="subscribe"
            >
              {{ subscribePending ? "正在添加…" : "订阅" }}
            </button>
          </div>
          <small v-if="subscribeStatus" role="status">{{
            subscribeStatus
          }}</small>
          <small v-if="subscribeError" class="error" role="alert">{{
            subscribeError
          }}</small>
          <label class="download-existing"
            ><input v-model="downloadExisting" type="checkbox" />
            下载全部现有视频</label
          >
          <small
            >支持 Rule34Video 作者页，以及 Hanime1 用户上传页和播放清单。</small
          >
          <h2 class="search-heading">搜索 Rule34Video 作者</h2>
          <div class="inline">
            <input
              v-model="creatorQuery"
              type="search"
              autocomplete="off"
              placeholder="输入作者名称，例如 Akt"
            /><button
              :disabled="searchingCreators || creatorQuery.trim().length < 2"
              @click="searchCreators"
            >
              {{ searchingCreators ? "搜索中…" : "搜索" }}
            </button>
          </div>
          <small v-if="creatorQuery.trim().length === 1"
            >至少输入 2 个字符</small
          ><small v-else-if="searchingCreators">正在搜索作者并加载头像…</small>
          <div v-else-if="creatorResults.length" class="creator-results">
            <button
              v-for="creator in creatorResults"
              :key="creator.source_id"
              :class="{
                selected: selectedCreator?.source_id === creator.source_id,
              }"
              @click="chooseCreator(creator)"
            >
              <img
                v-if="creator.avatar_url"
                :src="creator.avatar_url"
                :alt="creator.name"
                loading="lazy"
              /><span v-else class="avatar">{{
                creator.name.slice(0, 1).toUpperCase()
              }}</span
              ><span
                ><b>{{ creator.name }}</b
                ><small>{{ creator.video_count }} 个视频</small></span
              >
            </button>
          </div>
          <div
            v-else-if="creatorQuery.trim().length >= 2 && !searchingCreators"
            class="empty compact-empty"
          >
            没有找到匹配作者
          </div>
          <div v-if="selectedCreator" class="selected-creator">
            <span
              ><small>已选择</small><b>{{ selectedCreator.name }}</b></span
            ><button
              class="primary"
              :disabled="subscribePending"
              :aria-busy="subscribePending"
              v-on:click="subscribe"
            >
              {{ subscribePending ? "正在添加…" : "订阅所选作者" }}
            </button>
          </div>
          <small>{{
            downloadExisting
              ? "订阅后会把全部现有视频写入持久下载队列。"
              : "仅建立现有内容基线，之后发现的新视频会自动下载。"
          }}</small>
        </div>
        <div class="toolbar">
          <input
            v-model="subscriptionQuery"
            type="search"
            aria-label="筛选订阅"
            placeholder="筛选名称、平台或来源链接"
          /><select v-model="subscriptionState" aria-label="订阅状态">
            <option value="all">全部订阅</option>
            <option value="enabled">已启用</option>
            <option value="paused">已暂停</option>
            <option value="error">检查异常</option></select
          ><span class="muted"
            >{{ filteredSubscriptionGroups.length }} 个分组</span
          >
        </div>
        <div class="rows subscription-groups">
          <article v-for="group in filteredSubscriptionGroups" :key="group.key">
            <img
              v-if="group.avatar"
              class="avatar"
              :src="group.avatar.avatar_proxy_url"
              :alt="group.name"
              loading="lazy"
              role="button"
              tabindex="0"
              title="点击更换头像"
              @click="pickAvatar({ kind: 'sub', id: group.avatar.slug })"
              @keydown.enter="pickAvatar({ kind: 'sub', id: group.avatar.slug })"
            />
            <div
              v-else
              class="avatar"
              role="button"
              tabindex="0"
              title="点击更换头像"
              @click="pickAvatar({ kind: 'sub', id: group.sources[0].slug })"
              @keydown.enter="pickAvatar({ kind: 'sub', id: group.sources[0].slug })"
            >{{ group.name.slice(0, 1).toUpperCase() }}</div>
            <div class="subscription-group-body">
              <div class="subscription-group-title">
                <b>{{ group.name }}</b
                ><small v-if="group.sources.length > 1"
                  >{{ group.sources.length }} 个平台来源</small
                >
              </div>
              <div
                v-for="sub in group.sources"
                :key="sub.slug"
                class="subscription-source"
              >
                <div class="subscription-source-info">
                  <span
                    class="subscription-kind"
                    :class="
                      sub.subscription_type === 'playlist'
                        ? 'series'
                        : 'creator'
                    "
                    >{{ subscriptionTypeLabel(sub.subscription_type) }}</span
                  ><span class="platform-badge">{{ sub.platform }}</span
                  ><a :href="sub.url" target="_blank" rel="noreferrer"
                    >查看来源</a
                  >
                  <p :class="{ error: sub.last_error }">
                    {{
                      sub.last_error ||
                      "上次检查 " +
                        (sub.last_checked_at
                          ? localTime(sub.last_checked_at)
                          : "尚未")
                    }}
                  </p>
                  <div class="sub-progress">
                    <progress
                      :value="sub.progress.completed"
                      :max="sub.progress.total || 1"
                    ></progress
                    ><small
                      >已完成 {{ sub.progress.completed }} /
                      {{ sub.progress.total }} · 下载中
                      {{ sub.progress.active }} · 等待 {{ sub.progress.waiting
                      }}<span v-if="sub.progress.failed">
                        · 失败 {{ sub.progress.failed }}</span
                      ></small
                    >
                  </div>
                </div>
                <div class="actions">
                  <button
                    :disabled="actionPending('check:' + sub.slug)"
                    :aria-busy="actionPending('check:' + sub.slug)"
                    @click="check(sub.slug)"
                  >
                    {{
                      actionPending("check:" + sub.slug) ? "检查中…" : "检查"
                    }}</button
                  ><button
                    :disabled="actionPending('sub:' + sub.slug)"
                    @click="toggle(sub)"
                  >
                    {{
                      actionPending("sub:" + sub.slug)
                        ? "更新中…"
                        : sub.enabled
                          ? "暂停"
                          : "启用"
                    }}</button
                  ><button
                    class="danger"
                    :disabled="actionPending('sub-del:' + sub.slug)"
                    @click="removeSub(sub)"
                  >
                    删除
                  </button>
                </div>
              </div>
            </div>
          </article>
          <div v-if="!data.subscriptions.length" class="empty">还没有订阅</div>
          <StatePanel
            v-else-if="!filteredSubscriptionGroups.length"
            title="没有匹配的订阅"
            description="尝试调整名称或状态筛选。"
          />
        </div>
      </section>
      <section v-else-if="view === 'excluded'">
        <p class="muted">
          这些视频已被加入“已排除”，订阅或自动发现都不会重新下载。
        </p>
        <div class="toolbar">
          <input
            v-model="exclusionQuery"
            type="search"
            aria-label="搜索已排除"
            placeholder="搜索标题、标识或排除原因"
          />
        </div>
        <div class="rows">
          <article v-for="item in filteredExclusions" :key="item.video_id">
            <span class="avatar">{{
              (item.title_snapshot || item.video_id).slice(0, 1).toUpperCase()
            }}</span>
            <div>
              <b>{{ item.title_snapshot || item.video_id }}</b>
              <small
                >{{ item.source || "未知来源" }} · {{ item.video_id }} ·
                {{ localTime(item.created_at) }}</small
              >
              <small>{{ item.reason || "用户删除并排除" }}</small>
            </div>
            <div class="actions">
              <button
                :disabled="actionPending('exclusion-restore:' + item.video_id)"
                @click="restoreExclusion(item)"
              >
                恢复允许下载
              </button>
            </div>
          </article>
          <div v-if="!exclusions.length" class="empty">没有已排除的视频</div>
        </div>
        <p v-if="mediaExclusionTotal" class="muted">
          共 {{ mediaExclusionTotal }} 项
        </p>
      </section>
      <section v-else-if="view === 'integrity'" class="integrity-page">
        <p class="muted">
          扫描已完成媒体文件完整性。两种清理模式：<b>模式一</b>
          外部工具已删除、HWatch 核对后加入“已排除”； <b>模式二</b> 导入
          TransRunner 删除计划，由 HWatch
          将重复文件移入隔离区并更新数据库/排除。 全部默认 Dry
          Run，需二次确认；缺失但无计划确认的文件只展示、不自动排除。
        </p>
        <div class="toolbar">
          <button
            class="primary"
            :disabled="integrityBusy"
            @click="loadIntegrity"
          >
            扫描媒体库
          </button>
        </div>
        <div v-if="mediaScan" class="stats">
          <article>
            <small>总媒体数</small><b>{{ mediaScan.total }}</b
            ><span>已完成媒体</span>
          </article>
          <article>
            <small>正常文件</small><b>{{ mediaScan.present }}</b
            ><span>存在且可读</span>
          </article>
          <article>
            <small>确认缺失</small><b>{{ mediaScan.missing }}</b
            ><span>文件不存在</span>
          </article>
          <article>
            <small>存储状态</small
            ><b>{{ mediaScan.storage_available ? "正常" : "不可访问" }}</b
            ><span>{{
              mediaScan.storage_available ? "已挂载" : "已阻止缺失判定"
            }}</span>
          </article>
          <article>
            <small>疑似移动/权限</small
            ><b>{{ mediaScan.moved + mediaScan.permission_denied }}</b
            ><span>需人工确认</span>
          </article>
        </div>
        <div v-if="cleanupPreview" class="stats">
          <article>
            <small>清单条数</small><b>{{ cleanupPreview.deletions }}</b
            ><span>VDF deletions</span>
          </article>
          <article>
            <small>可加入排除</small><b>{{ cleanupPreview.candidates }}</b
            ><span>已验证缺失</span>
          </article>
          <article>
            <small>已排除</small><b>{{ cleanupPreview.already_excluded }}</b
            ><span>幂等跳过</span>
          </article>
          <article>
            <small>跳过</small><b>{{ cleanupPreview.skipped }}</b
            ><span>非候选</span>
          </article>
        </div>
        <div v-if="integrityNotice" class="notice" role="status">
          {{ integrityNotice }}
        </div>
        <div class="panel">
          <h2>导入 VDF 删除清单</h2>
          <JSONImport
            label="选择 VDF JSON 文件"
            :disabled="integrityBusy"
            @loaded="
              (text) => {
                deletionListText = text;
                vdfPreviewData = undefined;
                cleanupPreview = undefined;
              }
            "
          />
          <textarea
            v-model="deletionListText"
            :disabled="integrityBusy"
            aria-label="VDF 删除清单 JSON"
            rows="6"
            placeholder='{"version":1,"deletions":[{"video_id":"rule34video:...","original_path":"...","reason":"duplicate","kept_video_id":"..."}]}'
          ></textarea>
          <div class="inline" style="margin-top: 10px">
            <button
              :disabled="integrityBusy"
              :aria-busy="integrityImportPending"
              @click="importDeletionList"
            >
              {{ integrityImportPending ? "校验中…" : "预览" }}
            </button>
            <template v-if="vdfPreviewData">
              <button
                class="primary"
                :disabled="integrityBusy"
                :aria-busy="vdfExecutePending"
                @click="vdfExecute(false)"
              >
                {{ vdfExecutePending ? "处理中…" : "VDF Dry Run" }}
              </button>
              <button
                class="danger"
                :disabled="
                  integrityBusy ||
                  !vdfPreviewData.ready_groups ||
                  vdfPreviewData.has_conflicts
                "
                :aria-busy="vdfExecutePending"
                @click="vdfExecute(true)"
              >
                二次确认并执行 VDF 清理
              </button>
            </template>
            <template v-else>
              <button
                class="primary"
                :disabled="integrityBusy || !cleanupPreview"
                @click="applyDeletionList(false)"
              >
                Dry Run
              </button>
              <button
                class="danger"
                :disabled="
                  integrityBusy || !cleanupPreview || !cleanupPreview.candidates
                "
                @click="applyDeletionList(true)"
              >
                确认执行清理
              </button>
            </template>
          </div>
          <div v-if="vdfPreviewData" class="stats" style="margin-top: 12px">
            <article>
              <small>VDF 组</small><b>{{ vdfPreviewData.groups }}</b
              ><span>原生导出</span>
            </article>
            <article>
              <small>可执行组</small><b>{{ vdfPreviewData.ready_groups }}</b
              ><span>尚未执行</span>
            </article>
            <article>
              <small>此前已处理</small
              ><b>{{ vdfPreviewData.already_done_groups }}</b
              ><span>已在排除中</span>
            </article>
            <article>
              <small>冲突组</small><b>{{ vdfPreviewData.conflict_groups }}</b
              ><span>已阻止执行</span>
            </article>
            <article>
              <small>拒绝组</small><b>{{ vdfPreviewData.rejected_groups }}</b
              ><span>无法安全匹配</span>
            </article>
            <article>
              <small>预计释放</small
              ><b>{{ size(vdfPreviewData.estimated_freed) }}</b
              ><span>可回收空间</span>
            </article>
          </div>
          <div v-if="vdfPreviewData" class="stats" style="margin-top: 12px">
            <article>
              <small>删除项</small><b>{{ vdfPreviewData.remove_total }}</b>
              <span
                >已匹配 {{ vdfPreviewData.remove_matched }} · 已处理
                {{ vdfPreviewData.remove_already_done }} · 无法匹配
                {{ vdfPreviewData.remove_unmatched }}</span
              >
            </article>
            <article>
              <small>保留项</small><b>{{ vdfPreviewData.keep_total }}</b>
              <span
                >正常 {{ vdfPreviewData.keep_matched }} · 缺失/已删
                {{ vdfPreviewData.keep_missing }}</span
              >
            </article>
            <article>
              <small>跨组冲突</small
              ><b>{{ vdfPreviewData.cross_group_conflicts }}</b>
              <span>{{
                vdfPreviewData.has_conflicts ? "整个计划已阻止" : "无"
              }}</span>
            </article>
          </div>
          <div
            v-if="vdfPreviewData && vdfPreviewData.conflicts.length"
            class="panel error"
            style="margin-top: 12px"
          >
            <b>检测到跨组安全冲突，计划不可执行：</b>
            <p v-for="(cf, i) in vdfPreviewData.conflicts" :key="i">
              [{{ cf.kind }}] {{ cf.video_id || cf.path }} · 组
              {{ cf.groups.join(", ") }} — {{ cf.detail }}
            </p>
          </div>
          <div v-if="vdfPreviewData" class="toolbar" style="margin-top: 12px">
            <select v-model="vdfFilter" aria-label="VDF 分组状态">
              <option value="all">全部分组</option>
              <option value="ready">可执行</option>
              <option value="conflict">冲突</option>
              <option value="already_done">此前已处理</option>
              <option value="rejected">无法匹配</option></select
            ><span class="muted"
              >{{ filteredVdfGroups.length }} 组 · 每页 25 组</span
            >
          </div>
          <div v-if="vdfPreviewData" class="rows" style="margin-top: 12px">
            <article
              v-for="g in visibleVdfGroups"
              :key="g.group_id"
              class="vdf-group"
            >
              <button
                class="vdf-group-head"
                :aria-expanded="!!expandedVdfGroups[g.group_id]"
                @click="
                  expandedVdfGroups[g.group_id] = !expandedVdfGroups[g.group_id]
                "
              >
                <b>{{ g.group_id }}</b>
                <span
                  class="integrity-badge"
                  :class="'integrity-' + vdfGroupBadgeClass(g.status)"
                  >{{ vdfGroupStatusLabel(g.status) }}</span
                >
                <span class="muted"
                  >{{ g.subscription_slug || "?" }} · 释放
                  {{ size(g.estimated_freed) }}</span
                >
                <span>{{
                  expandedVdfGroups[g.group_id] ? "收起" : "查看保留 / 删除项"
                }}</span>
              </button>
              <div v-if="expandedVdfGroups[g.group_id]" class="vdf-items">
                <div>
                  <small class="error"
                    >删除 ({{ g.remove_items.length }})</small
                  >
                  <p v-for="it in g.remove_items" :key="it.path" class="muted">
                    {{
                      it.match_status === "matched" ||
                      it.match_status === "absent" ||
                      it.match_status === "already_done"
                        ? "✓"
                        : "✗"
                    }}
                    {{ it.path }}
                    <em v-if="it.video_id"
                      >→ {{ it.video_id }} ({{
                        vdfItemStatusLabel(it.match_status)
                      }})</em
                    >
                    <em v-else>({{ vdfItemStatusLabel(it.match_status) }})</em>
                  </p>
                </div>
                <div>
                  <small>保留 ({{ g.keep_items.length }})</small>
                  <p v-for="it in g.keep_items" :key="it.path" class="muted">
                    {{ it.match_status === "matched" ? "✓" : "✗" }}
                    {{ it.path }}
                    <em v-if="it.video_id">→ {{ it.video_id }}</em>
                  </p>
                </div>
              </div>
              <p v-if="g.problems.length" class="error">
                {{ g.problems.join("；") }}
              </p>
            </article>
          </div>
        </div>
        <div class="panel">
          <div v-if="filteredVdfGroups.length > 25" class="pagination">
            <button :disabled="vdfPage === 0" @click="vdfPage--">
              上一页分组</button
            ><span
              >{{ vdfPage + 1 }} /
              {{ Math.ceil(filteredVdfGroups.length / 25) }}</span
            ><button
              :disabled="(vdfPage + 1) * 25 >= filteredVdfGroups.length"
              @click="vdfPage++"
            >
              下一页分组
            </button>
          </div>
          <h2>模式二：导入 TransRunner 删除计划（HWatch 执行清理）</h2>
          <JSONImport
            label="选择删除计划 JSON 文件"
            :disabled="planBusy"
            @loaded="
              (text) => {
                planText = text;
                planPreviewData = undefined;
              }
            "
          />
          <textarea
            v-model="planText"
            :disabled="planBusy"
            aria-label="TransRunner 删除计划 JSON"
            rows="6"
            placeholder='{"kind":"transrunner-duplicate-deletion-plan","plan_version":1,"scan_id":"...","subscription_slug":"rule34video:...","executable":true,"stale":false,"user_confirmed":true,"items":[{"video_id":"...","kept_video_id":"...","original_path":"...","expected_size":123,"expected_sha256":"..."}]}'
          ></textarea>
          <div class="inline" style="margin-top: 10px">
            <button
              :disabled="planBusy"
              :aria-busy="planPreviewPending"
              @click="previewTransRunnerPlan"
            >
              {{ planPreviewPending ? "校验中…" : "校验预览" }}
            </button>
            <button
              class="primary"
              :disabled="planBusy || !planPreviewData"
              :aria-busy="planExecutePending"
              @click="executeTransRunnerPlan(false)"
            >
              Dry Run
            </button>
            <button
              class="danger"
              :disabled="planBusy || !planPreviewData || !planPreviewData.ready"
              :aria-busy="planExecutePending"
              @click="executeTransRunnerPlan(true)"
            >
              二次确认并执行清理
            </button>
          </div>
          <div
            v-if="planNotice"
            class="notice"
            role="status"
            style="margin-top: 10px"
          >
            {{ planNotice }}
          </div>
          <div v-if="planPreviewData" class="stats" style="margin-top: 12px">
            <article>
              <small>计划条数</small><b>{{ planPreviewData.total }}</b
              ><span>{{ planPreviewData.subscription_slug }}</span>
            </article>
            <article>
              <small>可清理</small><b>{{ planPreviewData.ready }}</b
              ><span>已验证一致</span>
            </article>
            <article>
              <small>已拒绝</small><b>{{ planPreviewData.rejected }}</b
              ><span>校验不通过</span>
            </article>
            <article>
              <small>需恢复</small><b>{{ planPreviewData.needs_recovery }}</b
              ><span>文件已不在</span>
            </article>
            <article>
              <small>预计释放</small
              ><b>{{ size(planPreviewData.estimated_freed) }}</b
              ><span>可回收空间</span>
            </article>
          </div>
          <div v-if="planResult" class="muted" style="margin-top: 8px">
            {{ planResult.dry_run ? "Dry Run" : "已执行" }} · 清理
            {{ planResult.executed }} · 跳过 {{ planResult.skipped }} · 失败
            {{ planResult.failed }} · 需恢复 {{ planResult.needs_recovery }}
          </div>
          <div v-if="planPreviewData" class="rows" style="margin-top: 12px">
            <article
              v-for="item in planPreviewData.items.slice(0, 200)"
              :key="item.video_id"
              class="integrity-row"
            >
              <span class="avatar">{{
                item.video_id.slice(-1).toUpperCase()
              }}</span>
              <div>
                <b>{{ item.video_id }}</b>
                <small>保留: {{ item.kept_video_id }}</small>
                <small>{{ item.original_path }}</small>
                <small v-if="item.reason" class="error">{{
                  item.reason
                }}</small>
              </div>
              <span
                class="integrity-badge"
                :class="'integrity-' + item.status"
                >{{ planStatusLabel(item.status) }}</span
              >
            </article>
          </div>
          <div
            v-if="planPreviewData && !planPreviewData.valid"
            class="error"
            style="margin-top: 8px"
          >
            计划不可执行：{{ planPreviewData.problems.join("；") }}
          </div>
        </div>
        <div v-if="deletionPlans.length" class="panel">
          <h2>删除计划执行记录</h2>
          <div class="rows">
            <article
              v-for="p in deletionPlans"
              :key="p.plan_id"
              class="integrity-row"
            >
              <span class="avatar">{{
                planStatusLabel(p.status).slice(0, 1)
              }}</span>
              <div>
                <b>{{ p.scan_id || p.plan_id }}</b>
                <small>{{ p.subscription_slug }}</small>
                <small>{{ localTime(p.created_at) }}</small>
              </div>
              <span class="actions">
                <button :disabled="planBusy" @click="recoverPlan(p.plan_id)">
                  恢复/核对
                </button>
                <span
                  class="integrity-badge"
                  :class="'integrity-' + p.status"
                  >{{ planStatusLabel(p.status) }}</span
                >
              </span>
            </article>
          </div>
        </div>
        <div v-if="mediaScan" class="panel">
          <h2>缺失/异常列表</h2>
          <div class="rows">
            <article
              v-for="item in mediaScan.items.slice(0, 200)"
              :key="item.video_id"
              class="integrity-row"
            >
              <span class="avatar">{{
                (item.title || item.video_id).slice(0, 1).toUpperCase()
              }}</span>
              <div>
                <b>{{ item.title || item.video_id }}</b>
                <small>{{ item.platform }} · {{ item.video_id }}</small>
                <small>{{ item.path || "（无路径）" }}</small>
                <small v-if="item.detail" class="error">{{
                  item.detail
                }}</small>
              </div>
              <span
                class="integrity-badge"
                :class="'integrity-' + item.status"
                >{{ cleanupStatusLabel(item.status) }}</span
              >
            </article>
            <div v-if="!mediaScan.items.length" class="empty">
              没有发现缺失或异常文件
            </div>
          </div>
        </div>
        <div v-if="cleanupPreview" class="panel">
          <h2>清单校验结果</h2>
          <div class="rows">
            <article
              v-for="item in cleanupPreview.items.slice(0, 200)"
              :key="item.video_id + item.status"
              class="integrity-row"
            >
              <span class="avatar">{{
                (item.title || item.video_id).slice(0, 1).toUpperCase()
              }}</span>
              <div>
                <b>{{ item.title || item.video_id }}</b>
                <small>{{ item.video_id }}</small>
                <small v-if="item.path">{{ item.path }}</small>
                <small v-if="item.detail" class="error">{{
                  item.detail
                }}</small>
              </div>
              <span
                class="integrity-badge"
                :class="'integrity-' + item.status"
                >{{ cleanupStatusLabel(item.status) }}</span
              >
            </article>
          </div>
        </div>
      </section>
      <section v-else-if="view === 'shorts'">
        <p v-if="shortsError" class="error" role="alert">{{ shortsError }}</p>
        <button
          v-if="shortsRotated"
          class="short-rotate-exit"
          aria-label="退出横屏"
          @click="toggleShortsRotate"
        >
          ✕ 退出横屏
        </button>
        <div ref="shortsFeed" class="shorts-feed" :class="{ rotated: shortsRotated }">
          <div
            v-for="(v, i) in shorts"
            :key="v._k"
            class="short"
            :ref="(el) => registerShort(el, i)"
          >
            <video
              :src="shortSrc(i, v)"
              :muted="shortsMuted"
              playsinline
              loop
              :preload="i === activeShort || i === activeShort + 1 ? 'auto' : 'none'"
              @error="onShortError"
              @loadedmetadata="onShortMeta(v, $event)"
              @waiting="shortBuffering[v.video_id] = true"
              @playing="onShortPlaying(v, $event)"
              @canplay="shortBuffering[v.video_id] = false"
              @timeupdate="onShortTime(v, $event)"
              @click="onShortTap"
            ></video>
            <div v-if="shortBuffering[v.video_id]" class="short-buffer">缓冲中…</div>
            <div v-if="shortFlash" class="short-flash">
              <b>{{ shortFlash.main }}</b>
              <small v-if="shortFlash.sub">{{ shortFlash.sub }}</small>
            </div>
            <div class="short-overlay">
              <div class="short-progress">
                <span class="short-time">{{ fmtClock(shortPos[v.video_id] || 0) }}</span>
                <input
                  type="range"
                  min="0"
                  :max="Math.max(1, Math.round(shortDur[v.video_id] || 1))"
                  :value="Math.round(shortPos[v.video_id] || 0)"
                  aria-label="播放进度"
                  @input="seekShort"
                />
                <span class="short-time">{{ fmtClock(shortDur[v.video_id] || 0) }}</span>
              </div>
              <div class="short-bottom">
                <div class="short-meta">
                  <b>{{ displayTitle(v) }}</b>
                  <small
                    >{{ v.creator_name || v.subscription_slug }} ·
                    {{ duration(v.duration_seconds) }}</small
                  >
                </div>
                <div class="short-buttons">
                  <button
                    v-if="isNarrow && shortLandscape[v.video_id]"
                    @click="toggleShortsRotate"
                  >
                    {{ shortsRotated ? "竖屏" : "横屏" }}
                  </button>
                  <button @click="shortsMuted = !shortsMuted">
                    {{ shortsMuted ? "🔇 静音" : "🔊 有声" }}
                  </button>
                  <button @click="selected = v">详情</button>
                </div>
              </div>
            </div>
          </div>
          <div v-if="shortsBusy" class="shorts-loading">加载中…</div>
        </div>
      </section>
      <section v-else-if="view === 'timeline'">
        <div class="toolbar">
          <span>{{ timelineTotal }} 条观看记录</span>
          <button
            :disabled="actionPending('timeline-load')"
            :aria-busy="actionPending('timeline-load')"
            @click="loadTimeline()"
          >
            {{ actionPending("timeline-load") ? "刷新中…" : "刷新" }}
          </button>
        </div>
        <p v-if="timelineError" class="error" role="alert">{{ timelineError }}</p>
        <div class="timeline">
          <div v-for="g in visibleTimelineGroups" :key="g.date" class="timeline-day">
            <div class="timeline-day-head">
              <h3 class="timeline-date">
                {{ g.date }} <small>{{ g.items.length }} 条</small>
              </h3>
              <button
                v-if="g.items.length > timelineCols"
                class="timeline-all"
                @click="openTimelineDay(g.date)"
              >
                查看全部（{{ g.items.length }}）
              </button>
            </div>
            <div class="library timeline-strip">
              <MediaCard
                v-for="v in g.items.slice(0, timelineCols)"
                :key="v.watched_at + v.video_id"
                :title="displayTitle(v)"
                :subtitle="
                  displaySubtitle(
                    v,
                    (v.creator_name || v.subscription_slug) +
                      ' · ' +
                      (v.watched_at ? localTime(v.watched_at) : ''),
                  )
                "
                :summary="mediaSummary(v) || v.views.toLocaleString() + ' 次观看'"
                :duration="duration(v.duration_seconds)"
                :src="'/thumb/' + v.video_id"
                :state="coverState(v.video_id)"
                :downloadable="false"
                :pending="false"
                @image="(el) => registerCoverImg(v.video_id, el)"
                @loaded="markCoverLoaded(v.video_id)"
                @error="onCoverError($event, v.video_id)"
                @play="selected = v"
              />
              <button
                v-if="g.items.length > timelineCols"
                class="timeline-more"
                @click="openTimelineDay(g.date)"
              >
                <span>+{{ g.items.length - timelineCols }}</span>
                查看全部
              </button>
            </div>
          </div>
          <button
            v-if="timelineGroups.length > visibleTimelineGroups.length"
            class="timeline-all timeline-load-more"
            @click="timelineDaysShown += TIMELINE_DAYS_PAGE"
          >
            加载更早的日期（还有
            {{ timelineGroups.length - visibleTimelineGroups.length }} 天）
          </button>
          <StatePanel
            v-if="!timeline.length && !timelineError"
            title="暂无观看记录"
            description="播放视频后会在这里按日期记录。"
          />
        </div>
        <div v-if="timelineDay" class="modal" @click.self="closeTimelineDay">
          <div
            class="log-detail timeline-day-detail"
            role="dialog"
            aria-modal="true"
            :aria-label="timelineDay + ' 观看记录'"
          >
            <div class="log-detail-head">
              <h2>{{ timelineDay }} · {{ timelineDayItems.length }} 条</h2>
              <button class="close" @click="closeTimelineDay">关闭</button>
            </div>
            <div class="library">
              <MediaCard
                v-for="v in timelineDayPageItems"
                :key="v.watched_at + v.video_id"
                :title="displayTitle(v)"
                :subtitle="
                  displaySubtitle(
                    v,
                    (v.creator_name || v.subscription_slug) +
                      ' · ' +
                      (v.watched_at ? localTime(v.watched_at) : ''),
                  )
                "
                :summary="mediaSummary(v) || v.views.toLocaleString() + ' 次观看'"
                :duration="duration(v.duration_seconds)"
                :src="'/thumb/' + v.video_id"
                :state="coverState(v.video_id)"
                :downloadable="false"
                :pending="false"
                @image="(el) => registerCoverImg(v.video_id, el)"
                @loaded="markCoverLoaded(v.video_id)"
                @error="onCoverError($event, v.video_id)"
                @play="selected = v"
              />
            </div>
            <div v-if="timelineDayPages > 1" class="timeline-pager">
              <button
                :disabled="timelineDayPage <= 0"
                @click="timelineDayPage--"
              >
                上一页
              </button>
              <span>第 {{ timelineDayPage + 1 }} / {{ timelineDayPages }} 页</span>
              <button
                :disabled="timelineDayPage >= timelineDayPages - 1"
                @click="timelineDayPage++"
              >
                下一页
              </button>
            </div>
          </div>
        </div>
      </section>
      <section v-else-if="view === 'users'">
        <div class="panel">
          <h2>创建用户</h2>
          <div class="setting-grid">
            <label>用户名<input v-model="newUser.username" autocomplete="off"></label>
            <label>初始密码<input v-model="newUser.password" type="password" autocomplete="new-password"><small>至少 10 位</small></label>
            <label>角色<select v-model="newUser.role"><option value="viewer">仅观看</option><option value="admin">管理员</option></select></label>
          </div>
          <div class="actions">
            <button class="primary" :disabled="actionPending('user-create') || !newUser.username || !newUser.password" @click="submitCreateUser">{{ actionPending("user-create") ? "创建中…" : "创建用户" }}</button>
            <button :disabled="actionPending('users-load')" @click="loadUsers()">{{ actionPending("users-load") ? "刷新中…" : "刷新" }}</button>
          </div>
          <p v-if="usersError" class="error">{{ usersError }}</p>
          <small>“仅观看”账号只能浏览、播放和查看时间线，无法修改任何设置或触发下载/删除。</small>
        </div>
        <div class="rows">
          <article v-for="u in users" :key="u.id">
            <img class="account-avatar user-avatar" :src="'/avatar/user/' + u.id" :alt="u.username" />
            <div>
              <b>{{ u.username }}</b>
              <small>{{ u.role === "admin" ? "管理员" : "仅观看" }} · 创建于 {{ localTime(u.created_at) }}</small>
            </div>
            <div class="actions">
              <button @click="pickAvatar({ kind: 'user', id: String(u.id) })">换头像</button>
              <button @click="clearAvatarTarget('user', String(u.id))">清除头像</button>
              <button :disabled="actionPending('user-role:' + u.id)" @click="toggleRole(u)">{{ u.role === "admin" ? "设为仅观看" : "设为管理员" }}</button>
              <button @click="promptResetPassword(u)">重置密码</button>
              <button class="danger" :disabled="actionPending('user-del:' + u.id)" @click="removeUser(u)">删除</button>
            </div>
          </article>
        </div>
      </section>
      <section v-else-if="view === 'logs'">
        <div class="toolbar log-toolbar">
          <select v-model="logsLevel" @change="loadLogs()">
            <option value="">全部级别</option>
            <option value="info">仅信息</option>
            <option value="error">仅错误</option>
          </select>
          <span>{{ logsTotal }} 条记录</span>
          <button
            :disabled="actionPending('logs-load')"
            :aria-busy="actionPending('logs-load')"
            @click="loadLogs()"
          >
            {{ actionPending("logs-load") ? "刷新中…" : "刷新" }}
          </button>
        </div>
        <p v-if="logsError" class="error" role="alert">{{ logsError }}</p>
        <div class="rows log-center">
          <article
            v-for="l in logs"
            :key="l.id"
            class="log-row"
            role="button"
            tabindex="0"
            @click="logDetail = l"
            @keydown.enter="logDetail = l"
            @keydown.space.prevent="logDetail = l"
          >
            <div class="log-time">{{ localTime(l.created_at) }}</div>
            <div class="log-body">
              <b>{{ l.platform }}</b> <small>{{ l.subscription_slug }}</small>
              <p :class="{ error: l.level === 'error' }">{{ l.message }}</p>
              <small class="log-meta"
                >耗时 {{ l.duration_ms }}ms · 发现 {{ l.found }} · 新增
                {{ l.created }} · 入队 {{ l.queued }}</small
              >
            </div>
            <span
              class="integrity-badge"
              :class="l.level === 'error' ? 'integrity-failed' : 'integrity-completed'"
              >{{ l.level }}</span
            >
          </article>
          <StatePanel
            v-if="!logs.length && !logsError"
            title="暂无日志"
            description="订阅检查完成后会在此记录结果。"
          />
        </div>
        <div v-if="logDetail" class="modal" @click.self="logDetail = null">
          <div class="log-detail" role="dialog" aria-modal="true" aria-label="检查日志">
            <div class="log-detail-head">
              <h2>检查日志</h2>
              <button class="close" @click="logDetail = null">关闭</button>
            </div>
            <dl>
              <div>
                <dt>时间</dt>
                <dd>{{ localTime(logDetail.created_at) }} <small>({{ logDetail.created_at }})</small></dd>
              </div>
              <div>
                <dt>级别</dt>
                <dd>
                  <span
                    class="integrity-badge"
                    :class="logDetail.level === 'error' ? 'integrity-failed' : 'integrity-completed'"
                    >{{ logDetail.level }}</span
                  >
                </dd>
              </div>
              <div><dt>平台</dt><dd>{{ logDetail.platform }}</dd></div>
              <div>
                <dt>订阅</dt>
                <dd>
                  {{ subscriptionName(logDetail.subscription_slug) || logDetail.subscription_slug }}
                  <small>({{ logDetail.subscription_slug }})</small>
                </dd>
              </div>
            </dl>
            <div class="log-stats">
              <span><small>耗时</small><b>{{ logDetail.duration_ms }} ms</b></span>
              <span><small>发现</small><b>{{ logDetail.found }}</b></span>
              <span><small>新增</small><b>{{ logDetail.created }}</b></span>
              <span><small>入队</small><b>{{ logDetail.queued }}</b></span>
            </div>
            <div class="terminal">
              <div class="terminal-bar">
                <span class="dot red"></span><span class="dot amber"></span
                ><span class="dot green"></span>
                <b>hwatch · {{ logDetail.subscription_slug }}</b>
              </div>
              <pre
                class="terminal-body"
                :class="{ error: logDetail.level === 'error' }"
              >{{ logLines(logDetail) }}</pre>
            </div>
            <div class="actions">
              <button @click="copyLog(logDetail)">复制日志</button>
            </div>
          </div>
        </div>
      </section>
      <section v-else-if="view === 'notifications'">
        <div class="tabs">
          <button
            :class="{ active: notificationTab === 'system' }"
            @click="switchNotificationTab('system')"
          >
            系统通知</button
          ><button
            :class="{ active: notificationTab === 'operations' }"
            @click="switchNotificationTab('operations')"
          >
            操作记录
          </button>
        </div>
        <template v-if="notificationTab === 'system'">
          <div class="toolbar">
            <select v-model="notificationSeverity" aria-label="通知筛选">
              <option value="all">全部通知</option>
              <option value="unread">未读</option>
              <option value="warning">警告</option>
              <option value="error">错误</option>
              <option value="info">信息</option>
            </select>
            <button
              :disabled="!data.unread_notifications"
              @click="markNotificationsRead"
            >
              全部标为已读</button
            ><span>{{ data.unread_notifications }} 条未读</span>
          </div>
          <div class="rows notification-list">
            <article
              v-for="item in visibleNotifications"
              :key="item.id"
              :class="{ unread: !item.read_at }"
            >
              <div class="notification-dot" :class="item.severity"></div>
              <div>
                <b>{{ item.message }}</b
                ><small
                  >{{ localTime(item.created_at) }} · {{ item.code }}</small
                >
              </div>
            </article>
            <div v-if="!notifications.length" class="empty">暂无系统通知</div>
          </div>
        </template>
        <template v-else>
          <p class="muted">
            以下为服务端请求审计：POST
            的“已受理”仅表示请求已被服务器接受，不代表后台任务已完成；任务最终状态请以媒体/任务状态为准。
          </p>
          <div class="rows notification-list">
            <article v-for="item in auditLogs" :key="item.id">
              <div
                class="notification-dot"
                :class="item.result === 'success' ? 'info' : 'error'"
              ></div>
              <div>
                <b>{{ item.action }}</b>
                <small>
                  {{ auditResultLabel(item.result, item.action) }}
                  <template v-if="item.target_type"
                    >· {{ item.target_type }}
                    <template v-if="item.target_id"
                      >#{{ item.target_id }}</template
                    ></template
                  >
                  · {{ localTime(item.created_at) }}
                </small>
              </div>
            </article>
            <div v-if="!auditLogs.length" class="empty">暂无操作记录</div>
          </div>
        </template>
      </section>
      <section v-else-if="view === 'downloads'">
        <div class="transfer-total">
          <div
            v-for="engine in data.downloaders"
            :key="engine.ID"
            class="downloader-total"
            :class="{ online: engine.Online }"
          >
            <small
              >{{ engine.Name
              }}{{
                engine.ID === data.downloader_mode ? " · 当前模式" : ""
              }}</small
            ><span
              class="engine-status"
              :class="{ available: engine.Online }"
              >{{ engine.Online ? "在线" : "离线" }}</span
            ><b>{{ engine.Active || 0 }} <em>个任务</em></b
            ><span
              >{{ size(engine.DownloadSpeed || 0) }}/s ·
              {{ engine.Connections || 0 }} 个连接</span
            >
          </div>
          <div>
            <small>总下载速度</small
            ><b>{{ size(data.current_download_speed) }}/s</b>
          </div>
          <div>
            <small>总连接数</small><b>{{ data.current_connections }}</b>
          </div>
        </div>
        <template
          v-for="engine in data.downloaders"
          :key="`error-${engine.ID}`"
        >
          <details v-if="engine.Error" class="engine-error-details">
            <summary>{{ engine.Name }} 连接异常 · 查看详情</summary>
            <p>{{ engine.Error }}</p>
          </details>
        </template>
        <div class="toolbar">
          <select v-model="status">
            <option value="">全部状态</option>
            <option value="queued">正在建立任务</option>
            <option value="submitting">正在提交下载器</option>
            <option value="waiting">等待中</option>
            <option value="active">下载中</option>
            <option value="paused">已暂停</option>
            <option value="failed">失败</option>
            <option value="excluded">已排除</option>
            <option value="completed">已完成</option></select
          ><button @click="loadVideos">刷新</button>
        </div>
        <div v-if="activeRateLimits.length" class="rate-limit-banner">
          <b>{{ rlLabel.title }}</b>
          <div
            v-for="r in activeRateLimits"
            :key="r.source"
            class="rate-limit-row"
          >
            <span class="rate-limit-source">{{ sourceLabel(r.source) }}</span>
            <span class="rate-limit-countdown">{{
              fmtCountdown(r.remaining)
            }}</span>
            <span>{{ rlLabel.waiting }}: {{ r.waiting }}</span>
            <span class="rate-limit-reason">{{
              r.reason === "network" ? rlLabel.taskRowNetwork : rlLabel.taskRow
            }}</span>
          </div>
          <small>{{ rlLabel.other }}</small>
        </div>
        <div class="queue">
          <article v-for="v in unfinishedDownloadTasks" :key="v.video_id">
            <img
              :src="'/thumb/' + v.video_id"
              :alt="displayTitle(v)"
              loading="lazy"
            />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b
                ><span>{{ pct(v) }}%</span>
              </div>
              <StatusBadge :status="v.status" />
              <small
                >{{ size(v.bytes_downloaded) }} /
                {{ v.total_bytes ? size(v.total_bytes) : "等待获取大小" }} ·
                {{ size(v.download_speed) }}/s ·
                {{ v.connections }} 个连接</small
              ><progress
                :value="v.bytes_downloaded"
                :max="v.total_bytes || 1"
              />
              <p
                v-if="v.status === 'queued' && rateLimitedReason(v.platform)"
                class="rate-limit-hint"
              >
                {{
                  rateLimitedReason(v.platform) === "network"
                    ? rlLabel.taskRowNetwork
                    : rlLabel.taskRow
                }}
              </p>
              <p v-else-if="v.status === 'failed' && v.error" class="error">
                {{ v.error }}
              </p>
              <p
                v-else-if="v.status === 'queued' && v.error"
                class="last-error"
              >
                {{ rlLabel.lastError }}: {{ v.error }}
              </p>
            </div>
            <div class="actions">
              <button
                v-if="v.status === 'active' || v.status === 'waiting'"
                :disabled="actionPending(videoActionKey(v.video_id, 'pause'))"
                @click="videoAction(v, 'pause')"
              >
                {{
                  actionPending(videoActionKey(v.video_id, "pause"))
                    ? "暂停中…"
                    : "暂停"
                }}</button
              ><button
                v-if="v.status === 'paused'"
                :disabled="actionPending(videoActionKey(v.video_id, 'resume'))"
                @click="videoAction(v, 'resume')"
              >
                {{
                  actionPending(videoActionKey(v.video_id, "resume"))
                    ? "恢复中…"
                    : "继续"
                }}</button
              ><button
                v-if="v.status === 'failed' || v.status === 'excluded'"
                :disabled="actionPending(videoActionKey(v.video_id, 'retry'))"
                @click="videoAction(v, 'retry')"
              >
                {{
                  actionPending(videoActionKey(v.video_id, "retry"))
                    ? "处理中…"
                    : v.status === "excluded"
                      ? "重新判断"
                      : "重试"
                }}</button
              ><button
                class="danger"
                :disabled="actionPending(videoActionKey(v.video_id, 'delete'))"
                @click="videoAction(v, 'delete')"
              >
                删除
              </button>
            </div>
          </article>
          <div
            v-if="
              !unfinishedDownloadTasks.length && !completedDownloadTasks.length
            "
            class="empty"
          >
            下载队列为空
          </div>
        </div>
        <button
          v-if="completedDownloadTasks.length"
          class="completed-tasks-toggle"
          :aria-expanded="showCompletedTasks"
          @click="showCompletedTasks = !showCompletedTasks"
        >
          <span
            ><b>已完成任务</b
            ><small>{{ completedDownloadTasks.length }} 项</small></span
          >
          <span
            >{{ showCompletedTasks ? "收起" : "展开" }}
            {{ showCompletedTasks ? "⌃" : "⌄" }}</span
          >
        </button>
        <div v-if="showCompletedTasks" class="queue completed-task-list">
          <article v-for="v in completedDownloadTasks" :key="v.video_id">
            <img
              :src="'/thumb/' + v.video_id"
              :alt="displayTitle(v)"
              loading="lazy"
            />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b
                ><span>100%</span>
              </div>
              <small>{{
                v.total_bytes ? size(v.total_bytes) : "已完成"
              }}</small>
              <progress :value="1" :max="1" />
            </div>
            <div class="actions">
              <button class="danger" @click="videoAction(v, 'delete')">
                删除
              </button>
            </div>
          </article>
        </div>
        <PaginationBar
          :page="page"
          :pages="pages"
          :total="total"
          :busy="videosLoading"
          @change="
            (next) => {
              page = next;
              loadVideos();
            }
          "
        />
      </section>
      <section v-else-if="view === 'library' || view === 'undownloaded'">
        <div class="toolbar library-toolbar">
          <input
            v-model="query"
            :type="demoMode === 2 ? 'password' : 'search'"
            placeholder="搜索标题、译文、作者或标签"
          /><button
            v-if="view === 'library'"
            :class="{ active: showFilters }"
            v-on:click="showFilters = !showFilters"
          >
            筛选</button
          ><select v-if="view === 'library'" v-model="sortBy">
            <option value="downloaded_at">最近下载</option>
            <option value="published_at">发布时间</option>
            <option value="views">播放量</option>
            <option value="duration">时长</option>
            <option value="size">文件大小</option>
            <option value="resolution">分辨率</option>
            <option value="bit_rate">码率</option>
            <option value="title">标题</option>
            <option value="recent_played">最近播放</option>
            <option value="play_count">播放次数</option></select
          ><select v-if="view === 'library'" v-model="sortOrder">
            <option value="desc">降序</option>
            <option value="asc">升序</option></select
          ><span v-if="randomMode && view === 'library'"
            >随机 {{ randomItems.length }} 条</span
          ><span v-else
            >{{ total }} 个{{
              view === "library" ? "已下载" : "未下载"
            }}视频</span
          >
        </div>
        <div
          v-if="view === 'library' && showFilters"
          class="panel filter-panel"
        >
          <label class="filter-toggle"
            ><input type="checkbox" v-model="randomMode" />随机发现<small
              >勾选后按当前筛选随机展示，翻页即换一批</small
            ></label
          >
          <label
            >搜索范围<select v-model="searchScope">
              <option value="all">原文和译文</option>
              <option value="translated">仅当前语言译文</option>
              <option value="original">仅原文</option>
            </select></label
          ><label
            >作者<select v-model="creatorFilter">
              <option value="">全部作者</option>
              <option
                v-for="item in facets.creators"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >编码<select v-model="codecFilter">
              <option value="">全部编码</option>
              <option
                v-for="item in facets.codecs"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >最低分辨率<select v-model.number="minHeight">
              <option :value="0">不限</option>
              <option :value="720">720p</option>
              <option :value="1080">1080p</option>
              <option :value="1440">1440p</option>
              <option :value="2160">2160p / 4K</option>
            </select></label
          ><label
            >包含标签<select v-model="selectedTags" multiple>
              <option
                v-for="item in facets.tags"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><label
            >标签关系<select v-model="tagMode">
              <option value="all">全部满足（AND）</option>
              <option value="any">任一满足（OR）</option>
            </select></label
          ><label
            >排除标签<select v-model="excludedTags" multiple>
              <option
                v-for="item in facets.tags"
                :key="item.value"
                :value="item.value"
              >
                {{ item.value }}（{{ item.count }}）
              </option>
            </select></label
          ><button
            v-on:click="
              selectedTags = [];
              excludedTags = [];
              creatorFilter = '';
              codecFilter = '';
              minHeight = 0;
              searchScope = 'all';
              tagMode = 'all';
            "
          >
            清除筛选
          </button>
        </div>
        <p v-if="randomMode && view === 'library' && randomError" class="error" role="status">
          {{ randomError }}
        </p>
        <div class="library">
          <MediaCard
            v-for="v in randomMode && view === 'library' ? randomItems : videos"
            :key="(randomMode && view === 'library' ? 'r-' : '') + v.video_id"
            :title="displayTitle(v)"
            :subtitle="
              displaySubtitle(
                v,
                (v.creator_name || v.subscription_slug) +
                  ' · ' +
                  (v.published_at ? localTime(v.published_at) : '日期未知'),
              )
            "
            :summary="mediaSummary(v) || v.views.toLocaleString() + ' 次观看'"
            :duration="duration(v.duration_seconds)"
            :src="'/thumb/' + v.video_id"
            :state="coverState(v.video_id)"
            :downloadable="!randomMode && view === 'undownloaded'"
            :pending="
              !randomMode && actionPending(videoActionKey(v.video_id, 'download'))
            "
            @image="(el) => registerCoverImg(v.video_id, el)"
            @loaded="markCoverLoaded(v.video_id)"
            @error="onCoverError($event, v.video_id)"
            @play="selected = v"
            @download="videoAction(v, 'download')"
          />
          <StatePanel
            v-if="
              randomMode &&
              view === 'library' &&
              !randomItems.length &&
              !actionPending('library-random')
            "
            title="暂无随机推荐"
            description="当前筛选范围内没有可随机发现的已下载媒体。"
          />
          <StatePanel
            v-else-if="!(randomMode && view === 'library') && !videos.length"
            :title="
              view === 'library' ? '影视库暂无已下载视频' : '没有未下载视频'
            "
            description="可调整筛选条件，或在订阅管理中添加来源。"
          />
        </div>
        <div class="pagination">
          <button
            :disabled="randomMode ? true : page === 0"
            @click="
              page--;
              loadVideos();
            "
          >
            上一页</button
          ><span v-if="randomMode && view === 'library'"
            >随机 · 第 {{ page + 1 }} 批</span
          ><span v-else>{{ page + 1 }} / {{ pages }}</span
          ><button
            :disabled="!randomMode && page + 1 >= pages"
            @click="
              randomMode && view === 'library'
                ? (page++, loadRandom(true))
                : (page++, loadVideos())
            "
          >
            {{ randomMode && view === "library" ? "换一批" : "下一页" }}
          </button>
        </div>
      </section>
      <section v-else>
        <div v-if="view === 'translations'" class="panel translation-workspace">
          <h2>WebAI 翻译交换</h2>
          <p>
            导出只包含视频标识、标题、标签和原文哈希，不包含代理密码、Cookie、下载地址或本地路径。
          </p>
          <div class="translation-steps">
          <div class="translation-step">
            <span class="step-number">01</span><h3>导出待翻译内容</h3>
            <p>选择目标语言，将 JSON 交给 WebAI 翻译。</p>
            <label class="translation-language">目标语言<select v-model="translationLanguage" aria-label="翻译目标语言">
              <option value="zh-CN">简体中文 zh-CN</option>
              <option value="zh-TW">繁体中文 zh-TW</option>
              <option value="en">英语 en</option>
              <option value="ja">日语 ja</option></select></label
            ><button class="primary" :disabled="actionPending('translation-export')" :aria-busy="actionPending('translation-export')" @click="exportTranslation">
              导出翻译 JSON</button
            ></div>
            <div class="translation-step">
            <span class="step-number">02</span><h3>回导翻译结果</h3>
            <p>保留 key、source_hash 和 fields，只填写 translations；原文变化的条目自动跳过。</p>
            <label
              class="upload"
              :class="{ disabled: translationImportPending }"
              >回导翻译 JSON<input
                type="file"
                accept="application/json,.json"
                :disabled="translationImportPending"
                @change="importTranslation"
            /></label>
          <small v-if="translationFileName" role="status">
            {{ translationFileName }}
            <template v-if="translationFileSize"
              >（{{ size(translationFileSize) }}）</template
            >
            · {{ translationImportPending ? "正在导入…" : translationStage }}
          </small>
          <small v-if="translationImportError" class="error" role="alert">{{
            translationImportError
          }}</small>
          </div>
          <div class="translation-step">
            <span class="step-number">03</span><h3>更新 Plex 元数据</h3>
            <p>回导完成后，使用现有译文批量刷新已下载视频的 NFO；不会重新下载视频。</p>
            <button v-on:click="refreshNFO">刷新全部 Plex NFO</button>
          </div>
          </div>
        </div>
        <SettingsView
          v-else
          :management-entries="managementEntries"
          :settings="settings"
          :api="api"
          :busy="settingsBusy"
          :settings-busy="settingsBusy"
          :backups-busy="backupsBusy"
          :import-busy="importBusy"
          :import-html="importHTML"
          :demo-mode="demoMode"
          :sidebar-items="sidebarItems"
          :shorts-watched-days="shortsWatchedDays"
          :backups="backups"
          :backup-directory="backupDirectory"
          :backup-sensitive="backupSensitive"
          :exclusion-preview="exclusionPreview"
          :exclusion-total="exclusionTotal"
          @preview-exclusions="previewExclusions"
          v-on:toggle-demo="toggleDemo"
          v-on:update-sidebar-item="updateSidebarItem"
          v-on:update-shorts-watched-days="setShortsWatchedDays"
          v-on:save="saveSettings"
          v-on:create-backup="createBackup"
          v-on:restore-backup="restoreBackup"
          v-on:import="importPage"
          v-on:update:import-html="importHTML = $event"
        />
      </section>
      <div v-if="changePasswordOpen" class="modal" @click.self="changePasswordOpen = false">
        <div class="log-detail" role="dialog" aria-modal="true" aria-label="修改密码">
          <div class="log-detail-head"><h2>修改密码</h2><button class="close" @click="changePasswordOpen = false">关闭</button></div>
          <label class="pw-field">当前密码<input v-model="changePw.current" type="password" autocomplete="current-password"></label>
          <label class="pw-field">新密码<input v-model="changePw.next" type="password" autocomplete="new-password"><small>至少 10 位</small></label>
          <label class="pw-field">确认新密码<input v-model="changePw.confirm" type="password" autocomplete="new-password"></label>
          <div class="actions"><button class="primary" :disabled="actionPending('change-password')" @click="submitChangePassword">{{ actionPending("change-password") ? "提交中…" : "提交" }}</button></div>
        </div>
      </div>
      <input ref="avatarInput" type="file" accept="image/*" class="hidden-file" @change="onAvatarFile" />
    </main>
    <nav class="mobile-tabbar" aria-label="主要导航">
      <button
        v-for="item in [
          { id: 'home', label: '概览' },
          { id: 'library', label: '影视库' },
          { id: 'notifications', label: '通知' },
          { id: 'more', label: '更多' },
        ]"
        :key="item.id"
        :class="{
          active:
            view === item.id ||
            (item.id === 'more' &&
              !['home', 'library', 'notifications'].includes(view)),
        }"
        @click="
          item.id === 'more' ? setSidebarCollapsed(false) : changeView(item.id)
        "
      >
        <NavIcon :name="item.id" /><span>{{ item.label }}</span
        ><i
          v-if="
            item.id === 'notifications' && data.unread_notifications
          "
          >{{ data.unread_notifications }}</i
        >
      </button>
    </nav>
  </div>
  <div v-if="selected" class="modal" @click.self="selected = undefined">
    <div
      v-dialog-focus
      class="player"
      role="dialog"
      aria-modal="true"
      :aria-label="displayTitle(selected)"
    >
      <div class="player-top-actions">
        <button v-if="isAdmin" class="danger" @click="openDelete(selected)">删除</button>
        <button class="close" @click="selected = undefined">关闭</button>
      </div>
      <h2>{{ displayTitle(selected) }}</h2>
      <video
        v-if="selected.status === 'completed'"
        :src="'/media/' + selected.video_id"
        controls
        playsinline
        preload="metadata"
      /><img
        v-else
        :src="'/thumb/' + selected.video_id"
        :alt="displayTitle(selected)"
      />
      <div class="detail">
        <span>时长 {{ duration(selected.duration_seconds) }}</span
        ><span>播放 {{ selected.views.toLocaleString() }}</span
        ><span v-if="selected.likes"
          >点赞 {{ selected.likes.toLocaleString() }}</span
        ><span v-if="selected.like_rate"
          >好评 {{ selected.like_rate.toFixed(0) }}%</span
        ><span>S{{ selected.plex_season }}E{{ selected.plex_episode }}</span
        ><span v-if="selected.video_codec">视频 {{ selected.video_codec }}</span
        ><span v-if="selected.width"
          >{{ selected.width }}×{{ selected.height }}</span
        ><span v-if="selected.frame_rate"
          >{{ selected.frame_rate.toFixed(2) }} FPS</span
        ><span v-if="selected.bit_depth">{{ selected.bit_depth }}-bit</span
        ><span v-if="selected.bit_rate"
          >码率 {{ size(selected.bit_rate / 8) }}/s</span
        ><span v-if="selected.audio_codec">音频 {{ selected.audio_codec }}</span
        ><span v-if="selected.container">容器 {{ selected.container }}</span>
      </div>
      <p v-if="selected.description" class="video-description">
        {{ selected.description }}
      </p>
      <div class="chips">
        <span v-for="tag in displayTags(selected)" :key="tag">{{ tag }}</span>
      </div>
      <a :href="selected.page_url" target="_blank">打开详情页</a>
    </div>
  </div>
  <div
    v-if="deleteTarget"
    class="modal"
    @click.self="!deleting && (deleteTarget = undefined)"
  >
    <div
      v-dialog-focus
      class="player"
      role="dialog"
      aria-modal="true"
      aria-label="删除这个视频"
      style="max-width: 520px"
    >
      <h2>删除这个视频？</h2>
      <p class="muted">{{ displayTitle(deleteTarget) }}</p>
      <div class="delete-options">
        <label
          class="delete-option"
          :class="{ selected: deleteMode === 'delete_and_exclude' }"
        >
          <input
            class="delete-option__radio"
            type="radio"
            name="delete-mode"
            value="delete_and_exclude"
            v-model="deleteMode"
          />
          <span class="delete-option__content"
            ><b>删除并加入已排除</b
            ><small
              >删除当前媒体文件，并阻止以后订阅或自动发现时再次下载。</small
            ></span
          >
        </label>
        <label
          class="delete-option"
          :class="{ selected: deleteMode === 'delete_only' }"
        >
          <input
            class="delete-option__radio"
            type="radio"
            name="delete-mode"
            value="delete_only"
            v-model="deleteMode"
          />
          <span class="delete-option__content"
            ><b>仅删除媒体</b
            ><small
              >删除当前媒体文件；以后如果订阅再次发现，仍允许重新下载。</small
            ></span
          >
        </label>
      </div>
      <div class="actions" style="justify-content: flex-end; margin-top: 14px">
        <button :disabled="deleting" @click="deleteTarget = undefined">
          取消
        </button>
        <button class="danger" :disabled="deleting" @click="confirmDelete">
          {{ deleting ? "正在删除…" : "确认删除" }}
        </button>
      </div>
    </div>
  </div>
</template>
