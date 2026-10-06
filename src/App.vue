<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import NavIcon from "./NavIcon.vue";
import SettingsView from "./SettingsView.vue";
import { requestJSON } from "./api-client";
type Link = { Name?: string; URL?: string; Image?: string };
type Creator = {
  platform: string;
  source_id: string;
  name: string;
  url: string;
  avatar_url: string;
  video_count: number;
};
type Sub = {
  id: number;
  slug: string;
  platform: string;
  subscription_type: string;
  source_id: string;
  name: string;
  url: string;
  avatar_url: string;
  avatar_proxy_url: string;
  external_links: Link[];
  enabled: boolean;
  last_checked_at: string;
  last_error: string;
  progress: {
    total: number;
    completed: number;
    active: number;
    waiting: number;
    failed: number;
    bytes_downloaded: number;
    total_bytes: number;
  };
};
type Video = {
  video_id: string;
  subscription_slug: string;
  title: string;
  description: string;
  page_url: string;
  published_at: string;
  discovered_at: string;
  thumbnail_url: string;
  views: number;
  likes: number;
  like_rate: number;
  duration_seconds: number;
  creator_name: string;
  uploader_name: string;
  categories: Link[];
  tags: Link[];
  status: string;
  aria_gid: string;
  plex_path: string;
  bytes_downloaded: number;
  total_bytes: number;
  download_speed: number;
  connections: number;
  video_codec: string;
  audio_codec: string;
  container: string;
  width: number;
  height: number;
  frame_rate: number;
  bit_rate: number;
  bit_depth: number;
  media_probed_at: string;
  transcode_status: string;
  plex_season: number;
  plex_episode: number;
  error: string;
  error_code: string;
  error_retryable: boolean;
  downloaded_at: string;
  translated_title: string;
  translated_tags: string[];
  translation_language: string;
};
type DownloaderHealth = {
  ID: string;
  Name: string;
  Version: string;
  Error: string;
  Online: boolean;
  SupportsMulti: boolean;
  Active: number;
  Connections: number;
  DownloadSpeed: number;
  UptimeSeconds: number;
};
type HealthWarning = {
  code: string;
  severity: "warning" | "error";
  message: string;
};
type Notification = {
  id: number;
  code: string;
  severity: string;
  message: string;
  created_at: string;
  read_at: string;
};
type Duplicate = {
  id: number;
  video_id: string;
  video_title: string;
  video_thumbnail: string;
  match_video_id: string;
  match_title: string;
  match_thumbnail: string;
  match_type: string;
  classification: string;
  backend: string;
  algorithm_version: string;
  metrics_json: string;
  confidence: number;
  duration_delta: number;
  status: string;
  created_at: string;
  reviewed_at: string;
  estimated_bytes: number;
};
type Dashboard = {
  subscriptions: Sub[];
  videos: Video[];
  recently_played: Video[];
  counts: {
    videos: number;
    active: number;
    waiting: number;
    completed: number;
    discovered: number;
  };
  traffic_bytes: number;
  library_bytes: number;
  current_download_speed: number;
  current_connections: number;
  downloads_paused: boolean;
  disk_total: number;
  disk_free: number;
  aria2_online: boolean;
  downloader_mode: string;
  downloaders: DownloaderHealth[];
  health_status: string;
  health_warnings: HealthWarning[];
  unread_notifications: number;
  timezone: string;
};
type Settings = {
  http_proxy: string;
  download_dir: string;
  aria_rpc_url: string;
  aria_secret: string;
  downloader_mode: string;
  check_time: string;
  timezone: string;
  max_concurrent: number;
  connections_per_download: number;
  download_limit_kbps: number;
  min_free_bytes: number;
  require_creator_match: boolean;
  exclude_title_keywords: string;
  exclude_uploaders: string;
  exclude_categories: string;
  exclude_tags: string;
  max_duration_seconds: number;
  backup_dir: string;
  backup_time: string;
  backup_retention: number;
  backup_last_success: string;
  backup_last_error: string;
};
type Backup = {
  name: string;
  size: number;
  created_at: string;
  sha256: string;
  valid: boolean;
};
type Facet = { value: string; count: number };
type AuthState = {
  configured: boolean;
  authenticated: boolean;
  username?: string;
  csrf_token?: string;
};
const auth = ref<AuthState>({ configured: true, authenticated: false }),
  authReady = ref(false),
  authUsername = ref(""),
  authPassword = ref(""),
  authPasswordConfirm = ref(""),
  authError = ref("");
const empty: Dashboard = {
  subscriptions: [],
  videos: [],
  recently_played: [],
  counts: { videos: 0, active: 0, waiting: 0, completed: 0, discovered: 0 },
  traffic_bytes: 0,
  library_bytes: 0,
  current_download_speed: 0,
  current_connections: 0,
  downloads_paused: false,
  disk_total: 0,
  disk_free: 0,
  aria2_online: false,
  downloader_mode: "aria2",
  downloaders: [],
  health_status: "ok",
  health_warnings: [],
  unread_notifications: 0,
  timezone: "Asia/Shanghai",
};
const data = ref<Dashboard>(empty),
  videos = ref<Video[]>([]),
  total = ref(0),
  view = ref("home"),
  query = ref(""),
  status = ref(""),
  page = ref(0),
  notice = ref(""),
  busy = ref(false),
  modelURL = ref(""),
  selected = ref<Video>(),
  importHTML = ref(""),
  translationLanguage = ref(
    localStorage.getItem("r34watch-language") || "zh-CN",
  ),
  creatorQuery = ref(""),
  creatorResults = ref<Creator[]>([]),
  selectedCreator = ref<Creator>(),
  searchingCreators = ref(false),
  downloadExisting = ref(false);
const viewRoutes: Record<string, string> = {
  home: "/",
  subscriptions: "/subscriptions",
  downloads: "/downloads",
  undownloaded: "/undownloaded",
  library: "/library",
  translations: "/translations",
  duplicates: "/duplicates",
  nodes: "/nodes",
  notifications: "/notifications",
  settings: "/settings",
};
const viewTitles: Record<string, string> = {
  home: "概览",
  subscriptions: "订阅管理",
  downloads: "下载中心",
  undownloaded: "未下载",
  library: "影视库",
  translations: "翻译管理",
  duplicates: "重复审查",
  notifications: "通知",
  settings: "设置",
};
const viewFromPath = () => {
  const path = window.location.pathname.replace(/\/+$/, "") || "/";
  return Object.entries(viewRoutes).find(([, route]) => route === path)?.[0] || "home";
};
const demoMode = ref(
  Math.min(
    2,
    Math.max(0, Number(localStorage.getItem("r34watch-demo-mode") || 0)),
  ),
);
const storedSidebarState = localStorage.getItem("hwatch-sidebar-collapsed");
const sidebarCollapsed = ref(
  storedSidebarState === null
    ? window.matchMedia("(max-width: 680px)").matches
    : storedSidebarState === "1",
);
const sidebarItems = ref({
  language: localStorage.getItem("hwatch-sidebar-language") !== "0",
  original: localStorage.getItem("hwatch-sidebar-original") !== "0",
  demo: localStorage.getItem("hwatch-sidebar-demo") !== "0",
  aria: localStorage.getItem("hwatch-sidebar-aria") !== "0",
});
const navGroups = [
  { id: "overview", label: "", items: [{ id: "home", label: "概览" }] },
  {
    id: "content",
    label: "内容获取",
    items: [
      { id: "subscriptions", label: "订阅管理" },
      { id: "downloads", label: "下载中心" },
      { id: "undownloaded", label: "未下载" },
    ],
  },
  {
    id: "media",
    label: "媒体库",
    items: [
      { id: "library", label: "影视库" },
      { id: "duplicates", label: "重复审查" },
    ],
  },
  {
    id: "system",
    label: "系统",
    items: [
      { id: "translations", label: "翻译管理" },
      { id: "notifications", label: "通知" },
    ],
  },
];
function loadOpenGroups(): Record<string, boolean> {
  try {
    const raw = localStorage.getItem("hwatch-nav-groups");
    if (raw) return JSON.parse(raw);
  } catch {}
  return { overview: true, content: true, media: true, system: true };
}
const openGroups = ref<Record<string, boolean>>(loadOpenGroups());
function persistOpenGroups() {
  try {
    localStorage.setItem("hwatch-nav-groups", JSON.stringify(openGroups.value));
  } catch {}
}
function toggleGroup(id: string) {
  openGroups.value = { ...openGroups.value, [id]: openGroups.value[id] === false };
  persistOpenGroups();
}
function revealGroupFor(viewID: string) {
  const group = navGroups.find((g) => g.items.some((i) => i.id === viewID));
  if (group && group.label && openGroups.value[group.id] === false) {
    openGroups.value = { ...openGroups.value, [group.id]: true };
    persistOpenGroups();
  }
}
const showOriginal = ref(
  localStorage.getItem("r34watch-show-original") === "1",
);
const settings = ref<Settings>({
  http_proxy: "",
  download_dir: "media",
  aria_rpc_url: "http://127.0.0.1:6800/jsonrpc",
  aria_secret: "",
  downloader_mode: "aria2",
  check_time: "04:00",
  timezone: "Asia/Shanghai",
  max_concurrent: 2,
  connections_per_download: 4,
  download_limit_kbps: 0,
  min_free_bytes: 5368709120,
  require_creator_match: true,
  exclude_title_keywords: "compilation,collection,合集,集锦",
  exclude_uploaders: "",
  exclude_categories: "",
  exclude_tags: "",
  max_duration_seconds: 0,
  backup_dir: "",
  backup_time: "03:30",
  backup_retention: 7,
  backup_last_success: "",
  backup_last_error: "",
});
const backups = ref<Backup[]>([]),
  backupDirectory = ref(""),
  backupSensitive = ref(false);
const notifications = ref<Notification[]>([]);
const duplicates = ref<Duplicate[]>([]);
const selectedDuplicates = ref<number[]>([]);
const exclusionPreview = ref<any[]>([]);
const exclusionTotal = ref(0);
const facets = ref<{ creators: Facet[]; tags: Facet[]; codecs: Facet[] }>({
    creators: [],
    tags: [],
    codecs: [],
  }),
  showFilters = ref(false),
  searchScope = ref("all"),
  tagMode = ref("all"),
  selectedTags = ref<string[]>([]),
  excludedTags = ref<string[]>([]),
  creatorFilter = ref(""),
  codecFilter = ref(""),
  minHeight = ref(0),
  sortBy = ref("downloaded_at"),
  sortOrder = ref("desc");
const pages = computed(() => Math.max(1, Math.ceil(total.value / 48))),
  downloadTasks = computed(() => videos.value),
  unfinishedDownloadTasks = computed(() =>
    downloadTasks.value.filter((video) => video.status !== "completed"),
  ),
  completedDownloadTasks = computed(() =>
    downloadTasks.value.filter((video) => video.status === "completed"),
  ),
  subscriptionGroups = computed(() => {
    const groups = new Map<string, { key: string; name: string; sources: Sub[]; avatar?: Sub }>();
    for (const sub of data.value.subscriptions) {
      const key = sub.name.trim().toLocaleLowerCase();
      const group = groups.get(key) || { key, name: sub.name, sources: [] };
      group.sources.push(sub);
      if (!group.avatar && sub.avatar_url) group.avatar = sub;
      groups.set(key, group);
    }
    return [...groups.values()].sort((a, b) => a.name.localeCompare(b.name));
  });
const showCompletedTasks = ref(false);
const subscriptionTypeLabel = (type = "") =>
  type === "playlist" ? "番剧订阅" : "作者订阅";
const size = (n = 0) =>
  n >= 1099511627776
    ? (n / 1099511627776).toFixed(2) + " TB"
    : n >= 1073741824
      ? (n / 1073741824).toFixed(2) + " GB"
      : n >= 1048576
        ? (n / 1048576).toFixed(1) + " MB"
        : (n / 1024).toFixed(1) + " KB";
const duration = (n = 0) =>
  n ? Math.floor(n / 60) + ":" + String(n % 60).padStart(2, "0") : "--:--";
const localTime = (raw = "") => {
  if (!raw) return "时间未知";
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) return raw;
  try {
    return new Intl.DateTimeFormat("zh-CN", {
      timeZone: data.value.timezone || "Asia/Shanghai",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    })
      .format(date)
      .replaceAll("/", "-");
  } catch {
    return date.toLocaleString("zh-CN", { hour12: false });
  }
};
const resolution = (v: Video) =>
  v.height ? (v.height >= 2160 ? "2160p / 4K" : v.height + "p") : "";
const mediaSummary = (v: Video) =>
  [resolution(v), v.video_codec, v.bit_depth ? v.bit_depth + "-bit" : ""]
    .filter(Boolean)
    .join(" · ");
const pct = (v: Video) =>
  v.total_bytes
    ? Math.min(100, Math.round((v.bytes_downloaded / v.total_bytes) * 100))
    : 0;
async function api(path: string, init?: RequestInit) {
  return requestJSON<any>(path, init, auth.value.csrf_token, () => {
    auth.value = { configured: true, authenticated: false };
    events?.close();
  });
}
async function loadAuth() {
  try {
    auth.value = await api("/api/auth/status");
    authUsername.value = auth.value.username || "";
  } finally {
    authReady.value = true;
  }
}
async function submitAuth() {
  authError.value = "";
  if (
    !auth.value.configured &&
    authPassword.value !== authPasswordConfirm.value
  ) {
    authError.value = "两次输入的密码不一致";
    return;
  }
  busy.value = true;
  try {
    auth.value = await api(
      auth.value.configured ? "/api/auth/login" : "/api/auth/setup",
      {
        method: "POST",
        body: JSON.stringify({
          username: authUsername.value,
          password: authPassword.value,
        }),
      },
    );
    authPassword.value = "";
    authPasswordConfirm.value = "";
    await startApplication();
    await changeView(viewFromPath(), false);
  } catch (e) {
    authError.value = String(e).replace(/^Error:\s*/, "");
  } finally {
    busy.value = false;
  }
}
async function logout() {
  try {
    await api("/api/auth/logout", { method: "POST" });
  } finally {
    events?.close();
    auth.value = { configured: true, authenticated: false };
    authPassword.value = "";
    data.value = empty;
  }
}
async function refreshDashboard() {
  try {
    data.value = await api(
      "/api/dashboard?language=" +
        encodeURIComponent(translationLanguage.value),
    );
  } catch (e) {
    notice.value = String(e);
  }
}
async function loadNotifications() {
  const result = await api("/api/notifications");
  notifications.value = result.items;
  data.value.unread_notifications = result.unread;
}
async function markNotificationsRead() {
  await api("/api/notifications/read-all", { method: "POST" });
  data.value.unread_notifications = 0;
  await loadNotifications();
}
async function loadDuplicates() {
  const result = await api("/api/duplicates");
  duplicates.value = result.items;
  selectedDuplicates.value = selectedDuplicates.value.filter((id) =>
    duplicates.value.some((item) => item.id === id && item.status === "pending"),
  );
}
async function reviewDuplicate(item: Duplicate, action: "confirm" | "dismiss") {
  await act(async () => {
    await api("/api/duplicates/" + item.id + "/" + action, { method: "POST" });
    notice.value =
      action === "confirm"
        ? "已确认重复，保留现有文件"
        : "已忽略匹配并重新排队";
    await loadDuplicates();
    await refreshDashboard();
  });
}
const classificationLabel = (value: string) =>
  ({
    "whole-video": "完整视频重复",
    "trimmed-video": "疑似裁切版本",
    "possible-match": "可能重复",
    different: "内容不同",
  } as Record<string, string>)[value] || value;
const duplicateReason = (item: Duplicate) => {
  const external: Record<string, string> = {
    external_exact: "外部分析：文件内容一致",
    external_reencoded: "外部分析：同一内容的重新编码版本",
    external_contained: "外部分析：该视频已完整包含于另一视频",
    external_partial_overlap: "外部分析：两个视频存在部分重叠",
    external_compilation_contains: "外部分析：合集包含匹配视频",
  };
  if (external[item.match_type]) return external[item.match_type];
  if (item.match_type === "media_duplicate") return `视频内容去重 · ${classificationLabel(item.classification)}`;
  if (item.match_type === "sha256") return "文件 SHA-256 完全一致";
  if (item.match_type === "page_url") return "详情页地址一致";
  return `封面相似 ${Math.round(item.confidence * 100)}% · 时长差 ${item.duration_delta} 秒`;
};
const duplicateMetrics = (item: Duplicate) => {
  try {
    return JSON.parse(item.metrics_json || "{}");
  } catch {
    return {};
  }
};
async function reviewDuplicateBatch(action: "confirm" | "dismiss") {
  if (!selectedDuplicates.value.length) return;
  await act(async () => {
    const result = await api("/api/duplicates/batch", {
      method: "POST",
      body: JSON.stringify({ ids: selectedDuplicates.value, action }),
    });
    notice.value = `已处理 ${result.reviewed} 项，预计涉及 ${size(result.estimated_bytes)}`;
    selectedDuplicates.value = [];
    await loadDuplicates();
    await refreshDashboard();
  });
}
async function previewExclusions() {
  await act(async () => {
    const result = await api("/api/exclusions/preview", { method: "POST", body: JSON.stringify(settings.value) });
    exclusionPreview.value = result.items;
    exclusionTotal.value = result.total;
    notice.value = `当前规则会排除 ${result.total} 项`;
  });
}
async function loadVideos() {
	const requestID = ++videoRequestID;
	videoAbort?.abort();
	videoAbort = new AbortController();
  const p = new URLSearchParams({
    limit: "48",
    offset: String(page.value * 48),
    language: translationLanguage.value,
  });
  if (query.value) p.set("q", query.value);
  if (status.value) p.set("status", status.value);
  if (view.value === "library") {
    p.set("scope", searchScope.value);
    p.set("tag_mode", tagMode.value);
    p.set("sort", sortBy.value);
    p.set("order", sortOrder.value);
    if (selectedTags.value.length) p.set("tags", selectedTags.value.join(","));
    if (excludedTags.value.length)
      p.set("exclude_tags", excludedTags.value.join(","));
    if (creatorFilter.value) p.set("creator", creatorFilter.value);
    if (codecFilter.value) p.set("codec", codecFilter.value);
    if (minHeight.value) p.set("min_height", String(minHeight.value));
  }
	try {
		const result = await api(
			(view.value === "downloads" ? "/api/tasks?" : "/api/videos?") + p,
			{ signal: videoAbort.signal },
		);
		if (requestID !== videoRequestID) return;
		videos.value = result.items;
		total.value = result.total;
	} catch (error) {
		if (error instanceof DOMException && error.name === "AbortError") return;
		throw error;
	}
}
async function loadFacets() {
  if (!facets.value.tags.length) facets.value = await api("/api/videos/facets");
}
async function loadSettings() {
  settings.value = await api("/api/settings");
  await loadBackups();
}
async function loadBackups() {
  const result = await api("/api/backups");
  backups.value = result.items;
  backupDirectory.value = result.directory;
  backupSensitive.value = result.sensitive;
}
async function createBackup() {
  await act(async () => {
    await api("/api/backups", { method: "POST" });
    notice.value = "数据库备份已创建";
    await loadSettings();
  });
}
async function restoreBackup(name: string) {
  if (
    !confirm("确认使用 " + name + " 恢复数据库？下载会暂停，服务随后自动重启。")
  )
    return;
  await act(async () => {
    await api("/api/backups/" + encodeURIComponent(name) + "/restore", {
      method: "POST",
      body: JSON.stringify({ confirm: true }),
    });
    notice.value = "正在恢复数据库并重启服务…";
  });
}
async function changeView(next: string, updateHistory = true) {
  if (!viewRoutes[next]) next = "home";
  view.value = next;
  revealGroupFor(next);
  document.title = (viewTitles[next] || "HWatch") + " · HWatch";
  if (updateHistory && window.location.pathname !== viewRoutes[next])
    window.history.pushState({ view: next }, "", viewRoutes[next]);
  if (window.matchMedia("(max-width: 680px)").matches)
    setSidebarCollapsed(true);
  page.value = 0;
  query.value = "";
  if (next === "library") {
    status.value = "completed";
    await loadFacets();
  } else if (next === "undownloaded") status.value = "discovered";
  else if (next === "downloads") {
    status.value = "";
    showCompletedTasks.value = false;
  }
  if (["library", "undownloaded", "downloads"].includes(next))
    await loadVideos();
  if (next === "settings") await loadSettings();
  if (next === "notifications") await loadNotifications();
  if (next === "duplicates") await loadDuplicates();
}
async function act(run: () => Promise<void>) {
  busy.value = true;
  try {
    await run();
  } catch (e) {
    notice.value = String(e);
  } finally {
    busy.value = false;
  }
}
async function subscribe() {
  await act(async () => {
    const r = await api("/api/subscriptions", {
      method: "POST",
      body: JSON.stringify({
        url: modelURL.value,
        download_existing: downloadExisting.value,
      }),
    });
    notice.value =
      "已建立 " +
      r.slug +
      " 基线，共 " +
      r.baselined +
      " 个视频" +
      (r.queued ? "，已排队 " + r.queued + " 个" : "");
    modelURL.value = "";
    creatorQuery.value = "";
    creatorResults.value = [];
    selectedCreator.value = undefined;
    downloadExisting.value = false;
    await refreshDashboard();
  });
}
async function searchCreators() {
  if (creatorTimer) clearTimeout(creatorTimer);
  creatorTimer = undefined;
  const searchQuery = creatorQuery.value.trim();
  if ([...searchQuery].length < 2) {
    creatorResults.value = [];
    searchingCreators.value = false;
    return;
  }
  const searchID = ++creatorSearchID;
  searchingCreators.value = true;
  try {
    const result = await api(
      "/api/creators/search?platform=rule34video&q=" +
        encodeURIComponent(searchQuery),
    );
    if (searchID === creatorSearchID) creatorResults.value = result.items;
  } catch (e) {
    if (searchID === creatorSearchID) notice.value = String(e);
  } finally {
    if (searchID === creatorSearchID) searchingCreators.value = false;
  }
}
function chooseCreator(creator: Creator) {
  selectedCreator.value = creator;
  modelURL.value = creator.url;
}
async function check(slug = "") {
  await act(async () => {
    await api("/api/check" + (slug ? "/" + slug : ""), { method: "POST" });
    notice.value = "检查任务已开始";
    await changeView("downloads");
  });
}
async function toggle(sub: Sub) {
  await api("/api/subscriptions/" + sub.slug, {
    method: "PATCH",
    body: JSON.stringify({ enabled: !sub.enabled }),
  });
  await refreshDashboard();
}
async function removeSub(sub: Sub) {
  if (
    !confirm(
      "删除 " +
        sub.platform +
        " 的 “" +
        sub.name +
        "” 订阅及其数据库记录？进行中的下载会停止；已下载文件仍保留在磁盘，但不会继续显示在影视库。此操作无法撤销。",
    )
  )
    return;
  await api("/api/subscriptions/" + sub.slug, { method: "DELETE" });
  await refreshDashboard();
}
async function videoAction(v: Video, action: string) {
  if (
    action === "delete" &&
    !confirm(
      v.status === "completed"
        ? "确认删除“" + displayTitle(v) + "”的媒体文件、NFO、海报和下载记录？此操作无法撤销。"
        : "确认删除“" + displayTitle(v) + "”的下载任务与临时文件？此操作无法撤销。",
    )
  )
    return;
  await act(async () => {
    await api("/api/videos/" + v.video_id + "/" + action, { method: "POST" });
    notice.value = "操作已提交";
    await loadVideos();
  });
}
async function downloadAll() {
  if (
    !confirm(
      "确认下载全部 " +
        total.value +
        " 个未下载视频？任务会在后台逐条加入下载队列。",
    )
  )
    return;
  await act(async () => {
    const r = await api("/api/videos/download-all", { method: "POST" });
    notice.value = "正在后台建立 " + r.accepted + " 个下载任务";
    await changeView("downloads");
  });
}
async function toggleDownloads() {
  const paused = !data.value.downloads_paused;
  await act(async () => {
    await api("/api/downloads/toggle", {
      method: "POST",
      body: JSON.stringify({ paused }),
    });
    data.value.downloads_paused = paused;
    notice.value = paused ? "已暂停全部下载" : "已继续全部下载";
    await loadVideos();
  });
}
async function saveSettings() {
  await act(async () => {
    settings.value = await api("/api/settings", {
      method: "PUT",
      body: JSON.stringify(settings.value),
    });
    notice.value = "设置已保存";
    await refreshDashboard();
  });
}
async function importPage() {
  await act(async () => {
    await api("/api/import/html", {
      method: "POST",
      body: JSON.stringify({ html: importHTML.value }),
    });
    notice.value = "HTML 数据已导入";
    importHTML.value = "";
    await loadVideos();
  });
}
function exportTranslation() {
  window.location.href =
    "/api/translations/export?language=" +
    encodeURIComponent(translationLanguage.value);
}
async function importTranslation(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  await act(async () => {
    const result = await api("/api/translations/import", {
      method: "POST",
      body: await file.text(),
    });
    notice.value =
      "已导入 " +
      result.imported_fields +
      " 个翻译字段，跳过 " +
      result.skipped_items +
      " 条";
    (event.target as HTMLInputElement).value = "";
  });
}
async function refreshNFO() {
  if (
    !confirm(
      "使用 " +
        translationLanguage.value +
        " 译文刷新全部已下载视频的 Plex NFO？",
    )
  )
    return;
  await act(async () => {
    const result = await api("/api/plex/refresh-nfo", {
      method: "POST",
      body: JSON.stringify({ language: translationLanguage.value }),
    });
    notice.value = "正在后台刷新 " + result.accepted + " 个 NFO 文件";
  });
}
function privateID(v: Video) {
  return "媒体项目 " + v.video_id.split(":").pop()!.slice(-4).toUpperCase();
}
function displayDuplicateTitle(title: string, id: string) {
  if (demoMode.value !== 2) return title;
  return "媒体项目 " + id.split(":").pop()!.slice(-4).toUpperCase();
}
function displayTitle(v: Video) {
  if (demoMode.value === 2) return privateID(v);
  return !showOriginal.value && v.translated_title
    ? v.translated_title
    : v.title;
}
function displaySubtitle(v: Video, fallback = "本地媒体") {
  return demoMode.value === 2
    ? v.status === "completed"
      ? "已下载 · 本地媒体"
      : v.status === "active"
        ? "下载中 · " + v.connections + " 个连接"
        : v.status === "queued" || v.status === "waiting"
          ? "等待下载 · 已加入队列"
          : v.status === "submitting"
            ? "正在提交 · 已持久化任务"
          : fallback
    : fallback;
}
function displayTags(v: Video) {
  if (demoMode.value === 2) return [];
  return !showOriginal.value && v.translated_tags?.length
    ? v.translated_tags
    : v.tags.map((t) => t.Name || "").filter(Boolean);
}
function toggleOriginal() {
  showOriginal.value = !showOriginal.value;
  localStorage.setItem(
    "r34watch-show-original",
    showOriginal.value ? "1" : "0",
  );
}
function toggleDemo() {
  demoMode.value = (demoMode.value + 1) % 3;
  localStorage.setItem("r34watch-demo-mode", String(demoMode.value));
}
function toggleSidebar() {
  setSidebarCollapsed(!sidebarCollapsed.value);
}
function setSidebarCollapsed(collapsed: boolean) {
  sidebarCollapsed.value = collapsed;
  localStorage.setItem(
    "hwatch-sidebar-collapsed",
    collapsed ? "1" : "0",
  );
}
function updateSidebarItem(
  item: keyof typeof sidebarItems.value,
  value: boolean,
) {
  sidebarItems.value[item] = value;
  localStorage.setItem("hwatch-sidebar-" + item, value ? "1" : "0");
}
function mergeTasks(payload: {
  tasks: Video[];
  download_speed: number;
  connections: number;
}) {
  for (const task of payload.tasks) {
    const i = videos.value.findIndex((v) => v.video_id === task.video_id);
    if (i >= 0)
      videos.value[i] = {
        ...task,
        translated_title: videos.value[i].translated_title,
        translated_tags: videos.value[i].translated_tags,
        translation_language: videos.value[i].translation_language,
      };
  }
  data.value.counts.active = payload.tasks.filter(
    (v) => v.status === "active",
  ).length;
  data.value.counts.waiting = payload.tasks.filter(
    (v) => v.status === "waiting",
  ).length;
  data.value.current_download_speed = payload.download_speed || 0;
  data.value.current_connections = payload.connections || 0;
}
let refreshTimer: number | undefined;
let events: EventSource | undefined;
let creatorTimer: number | undefined;
let creatorSearchID = 0;
let videoLoadTimer: number | undefined;
let videoAbort: AbortController | undefined;
let videoRequestID = 0;
const handlePopState = () => {
  if (auth.value.authenticated) void changeView(viewFromPath(), false);
};
const handleEscape = (event: KeyboardEvent) => {
  if (event.key !== "Escape") return;
  if (selected.value) selected.value = undefined;
  else if (!sidebarCollapsed.value) setSidebarCollapsed(true);
};
const handleVisibility = () => {
  if (!document.hidden && auth.value.authenticated) void refreshDashboard();
};
async function startApplication() {
  await refreshDashboard();
  events?.close();
  events = new EventSource("/api/events");
  events.onmessage = (e) => {
    try {
      mergeTasks(JSON.parse(e.data));
    } catch {}
  };
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = window.setInterval(() => {
    if (!document.hidden) void refreshDashboard();
  }, 60000);
}
onMounted(async () => {
  window.addEventListener("popstate", handlePopState);
  window.addEventListener("keydown", handleEscape);
  document.addEventListener("visibilitychange", handleVisibility);
  await loadAuth();
  if (auth.value.authenticated) {
    await startApplication();
    await changeView(viewFromPath(), false);
  }
});
onBeforeUnmount(() => {
  window.removeEventListener("popstate", handlePopState);
  window.removeEventListener("keydown", handleEscape);
  document.removeEventListener("visibilitychange", handleVisibility);
  events?.close();
  if (refreshTimer) clearInterval(refreshTimer);
  if (creatorTimer) clearTimeout(creatorTimer);
  if (videoLoadTimer) clearTimeout(videoLoadTimer);
  videoAbort?.abort();
});
watch(creatorQuery, () => {
  if (creatorTimer) clearTimeout(creatorTimer);
  creatorSearchID++;
  selectedCreator.value = undefined;
  modelURL.value = "";
  const searchable = [...creatorQuery.value.trim()].length >= 2;
  searchingCreators.value = searchable;
  if (!searchable) {
    creatorResults.value = [];
    return;
  }
  creatorTimer = window.setTimeout(searchCreators, 450);
});
watch(
  [
    query,
    status,
    translationLanguage,
    searchScope,
    tagMode,
    selectedTags,
    excludedTags,
    creatorFilter,
    codecFilter,
    minHeight,
    sortBy,
    sortOrder,
  ],
  () => {
    page.value = 0;
    localStorage.setItem("r34watch-language", translationLanguage.value);
    if (["library", "undownloaded", "downloads"].includes(view.value)) {
      if (videoLoadTimer) clearTimeout(videoLoadTimer);
      videoLoadTimer = window.setTimeout(() => void loadVideos(), 300);
    }
    if (view.value === "home") refreshDashboard();
  },
);
</script>
<template>
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
    <aside>
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
            ><span class="nav-group-caret" :class="{ open: openGroups[group.id] !== false }">⌄</span>
          </button>
          <template v-if="openGroups[group.id] !== false">
            <button
              v-for="item in group.items"
              :key="item.id"
              :class="{ active: view === item.id }"
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
              <i v-if="item.id === 'notifications' && data.unread_notifications">{{
                data.unread_notifications
              }}</i>
            </button>
          </template>
        </template>
      </nav>
      <div class="bottom">
        <label v-if="sidebarItems.language" class="sidebar-language"
          ><NavIcon name="language" /><select
            class="language-select"
            v-model="translationLanguage"
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
          v-on:click="changeView('settings')"
        >
          <NavIcon name="settings" /><span>设置</span></button
        ><small v-if="sidebarItems.aria" :class="{ online: data.aria2_online }"
          >● aria2 {{ data.aria2_online ? "在线" : "离线" }}</small
        >
        <div class="sidebar-account">
          <span><small>当前账户</small><b>{{ auth.username }}</b></span>
          <button class="sidebar-logout" aria-label="退出登录" title="退出登录" @click="logout">
            <NavIcon name="logout" /><span>退出登录</span>
          </button>
        </div>
      </div>
    </aside>
    <main>
      <header>
        <div>
          <p class="eyebrow">{{ view.toUpperCase() }}</p>
          <h1>
            {{ viewTitles[view] || "HWatch" }}
          </h1>
        </div>
        <button
          v-if="view === 'undownloaded'"
          class="primary"
          :disabled="busy || !total"
          @click="downloadAll"
        >
          下载全部（{{ total }}）</button
        ><button
          v-else-if="view === 'downloads'"
          class="download-master"
          :class="{ paused: data.downloads_paused }"
          :disabled="busy"
          @click="toggleDownloads"
        >
          {{ data.downloads_paused ? "继续全部" : "暂停全部" }}</button
        ><button
          v-else-if="
            ![
              'settings',
              'translations',
              'notifications',
              'duplicates',
            ].includes(view)
          "
          class="primary"
          :disabled="busy"
          @click="check()"
        >
          检查更新
        </button>
      </header>
      <button v-if="notice" class="notice" role="status" @click="notice = ''">{{ notice }}</button>
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
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
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
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
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
              :disabled="busy || !modelURL.trim()"
              @click="subscribe"
            >
              订阅
            </button>
          </div>
          <label class="download-existing"
            ><input v-model="downloadExisting" type="checkbox" />
            下载全部现有视频</label
          >
          <small>支持 Rule34Video 作者页，以及 Hanime1 用户上传页和播放清单。</small>
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
            ><button class="primary" :disabled="busy" v-on:click="subscribe">
              订阅所选作者
            </button>
          </div>
          <small>{{
            downloadExisting
              ? "订阅后会把全部现有视频写入持久下载队列。"
              : "仅建立现有内容基线，之后发现的新视频会自动下载。"
          }}</small>
        </div>
        <div class="rows subscription-groups">
          <article v-for="group in subscriptionGroups" :key="group.key">
            <img
              v-if="group.avatar"
              :src="group.avatar.avatar_proxy_url"
              :alt="group.name"
              loading="lazy"
            />
            <div v-else class="avatar">
              {{ group.name.slice(0, 1).toUpperCase() }}
            </div>
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
                    :class="sub.subscription_type === 'playlist' ? 'series' : 'creator'"
                    >{{ subscriptionTypeLabel(sub.subscription_type) }}</span
                  ><span class="platform-badge">{{ sub.platform }}</span
                  ><a :href="sub.url" target="_blank" rel="noreferrer">查看来源</a>
                  <p :class="{ error: sub.last_error }">
                    {{
                      sub.last_error ||
                      "上次检查 " +
                        (sub.last_checked_at ? localTime(sub.last_checked_at) : "尚未")
                    }}
                  </p>
                  <div class="sub-progress">
                    <progress
                      :value="sub.progress.completed"
                      :max="sub.progress.total || 1"
                    ></progress
                    ><small
                      >已完成 {{ sub.progress.completed }} / {{ sub.progress.total }} ·
                      下载中 {{ sub.progress.active }} · 等待 {{ sub.progress.waiting
                      }}<span v-if="sub.progress.failed"> · 失败 {{ sub.progress.failed }}</span></small
                    >
                  </div>
                </div>
                <div class="actions">
                  <button @click="check(sub.slug)">检查</button
                  ><button @click="toggle(sub)">{{ sub.enabled ? "暂停" : "启用" }}</button
                  ><button class="danger" @click="removeSub(sub)">删除</button>
                </div>
              </div>
            </div>
          </article>
          <div v-if="!data.subscriptions.length" class="empty">还没有订阅</div>
        </div>
      </section>
      <section v-else-if="view === 'duplicates'">
        <div class="toolbar">
          <span>已选 {{ selectedDuplicates.length }} 项 · 预计可释放 {{ size(duplicates.filter((x) => selectedDuplicates.includes(x.id)).reduce((sum, x) => sum + (x.estimated_bytes || 0), 0)) }}</span>
          <button :disabled="!selectedDuplicates.length" @click="reviewDuplicateBatch('dismiss')">批量忽略</button>
          <button class="primary" :disabled="!selectedDuplicates.length" @click="reviewDuplicateBatch('confirm')">批量确认</button>
        </div>
        <div class="duplicate-grid">
          <article
            v-for="item in duplicates"
            :key="item.id"
            class="duplicate-card"
          >
            <label v-if="item.status === 'pending'" class="duplicate-select"><input v-model="selectedDuplicates" type="checkbox" :value="item.id" /> 选择</label>
            <div class="duplicate-pair">
              <div>
                <img :src="'/thumb/' + item.video_id" alt="重复候选封面" /><b>{{
                  displayDuplicateTitle(item.video_title, item.video_id)
                }}</b>
              </div>
              <span>≈</span>
              <div>
                <img :src="'/thumb/' + item.match_video_id" alt="匹配媒体封面" /><b>{{
                  displayDuplicateTitle(item.match_title, item.match_video_id)
                }}</b>
              </div>
            </div>
            <small
              >{{ duplicateReason(item) }}
              · {{ item.status }}</small
            >
            <details v-if="item.match_type === 'media_duplicate'" class="duplicate-detail">
              <summary>技术详情</summary>
              <small>算法 {{ item.algorithm_version || '未知' }} · 后端 {{ item.backend || '未知' }}</small>
              <small v-if="duplicateMetrics(item).sample_count != null">
                采样 {{ duplicateMetrics(item).matched_samples ?? '?' }} / {{ duplicateMetrics(item).sample_count }}
                <template v-if="duplicateMetrics(item).median_hamming_distance != null"> · 中位汉明距离 {{ duplicateMetrics(item).median_hamming_distance }}</template>
              </small>
              <small v-if="duplicateMetrics(item).duration_ratio != null">时长比 {{ duplicateMetrics(item).duration_ratio }}</small>
            </details>
            <small v-if="item.estimated_bytes">预计可释放 {{ size(item.estimated_bytes) }}</small>
            <div v-if="item.status === 'pending'" class="actions">
              <button @click="reviewDuplicate(item, 'dismiss')">
                不是重复，继续下载</button
              ><button
                class="primary"
                @click="reviewDuplicate(item, 'confirm')"
              >
                确认重复
              </button>
            </div>
          </article>
          <div v-if="!duplicates.length" class="empty">暂无重复候选</div>
        </div>
      </section>
      <section v-else-if="view === 'notifications'">
        <div class="toolbar">
          <button
            :disabled="!data.unread_notifications"
            @click="markNotificationsRead"
          >
            全部标为已读</button
          ><span>{{ data.unread_notifications }} 条未读</span>
        </div>
        <div class="rows notification-list">
          <article
            v-for="item in notifications"
            :key="item.id"
            :class="{ unread: !item.read_at }"
          >
            <div class="notification-dot" :class="item.severity"></div>
            <div>
              <b>{{ item.message }}</b
              ><small>{{ localTime(item.created_at) }} · {{ item.code }}</small>
            </div>
          </article>
          <div v-if="!notifications.length" class="empty">暂无系统通知</div>
        </div>
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
            ><b
              >{{ engine.Online ? "在线" : "离线" }} ·
              {{ engine.Active || 0 }} 个任务</b
            ><span
              >{{ size(engine.DownloadSpeed || 0) }}/s ·
              {{ engine.Connections || 0 }} 个连接</span
            ><span v-if="engine.Error" class="error">{{ engine.Error }}</span>
          </div>
          <div>
            <small>总下载速度</small
            ><b>{{ size(data.current_download_speed) }}/s</b>
          </div>
          <div>
            <small>总连接数</small><b>{{ data.current_connections }}</b>
          </div>
        </div>
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
        <div class="queue">
          <article v-for="v in unfinishedDownloadTasks" :key="v.video_id">
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b
                ><span>{{ pct(v) }}%</span>
              </div>
              <small
                >{{ size(v.bytes_downloaded) }} /
                {{ v.total_bytes ? size(v.total_bytes) : "等待获取大小" }} ·
                {{ size(v.download_speed) }}/s ·
                {{ v.connections }} 个连接</small
              ><progress
                :value="v.bytes_downloaded"
                :max="v.total_bytes || 1"
              />
              <p v-if="v.error" class="error">{{ v.error }}</p>
            </div>
            <div class="actions">
              <button
                v-if="v.status === 'active' || v.status === 'waiting'"
                @click="videoAction(v, 'pause')"
              >
                暂停</button
              ><button
                v-if="v.status === 'paused'"
                @click="videoAction(v, 'resume')"
              >
                继续</button
              ><button
                v-if="v.status === 'failed' || v.status === 'excluded'"
                @click="videoAction(v, 'retry')"
              >
                {{ v.status === "excluded" ? "重新判断" : "重试" }}</button
              ><button class="danger" @click="videoAction(v, 'delete')">
                删除
              </button>
            </div>
          </article>
          <div
            v-if="!unfinishedDownloadTasks.length && !completedDownloadTasks.length"
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
          <span><b>已完成任务</b><small>{{ completedDownloadTasks.length }} 项</small></span>
          <span>{{ showCompletedTasks ? "收起" : "展开" }} {{ showCompletedTasks ? "⌃" : "⌄" }}</span>
        </button>
        <div v-if="showCompletedTasks" class="queue completed-task-list">
          <article v-for="v in completedDownloadTasks" :key="v.video_id">
            <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" />
            <div>
              <div class="download-title">
                <b>{{ displayTitle(v) }}</b><span>100%</span>
              </div>
              <small>{{ v.total_bytes ? size(v.total_bytes) : "已完成" }}</small>
              <progress :value="1" :max="1" />
            </div>
            <div class="actions">
              <button class="danger" @click="videoAction(v, 'delete')">删除</button>
            </div>
          </article>
        </div>
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
          ><span
            >{{ total }} 个{{
              view === "library" ? "已下载" : "未下载"
            }}视频</span
          >
        </div>
        <div
          v-if="view === 'library' && showFilters"
          class="panel filter-panel"
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
        <div class="library">
          <article v-for="v in videos" :key="v.video_id">
            <button class="poster" @click="selected = v">
              <img :src="'/thumb/' + v.video_id" :alt="displayTitle(v)" loading="lazy" /><span>{{
                duration(v.duration_seconds)
              }}</span>
            </button>
            <div class="meta">
              <b>{{ displayTitle(v) }}</b
              ><small>{{
                displaySubtitle(
                  v,
                  (v.creator_name || v.subscription_slug) +
                    " · " +
                    (v.published_at ? localTime(v.published_at) : "日期未知"),
                )
              }}</small>
              <p>
                {{ mediaSummary(v) || v.views.toLocaleString() + " 次观看" }}
              </p>
            </div>
            <div class="library-actions">
              <button v-if="view === 'library'" class="primary" @click="selected = v">播放</button>
              <button v-else class="primary" @click="videoAction(v, 'download')">下载</button>
            </div>
          </article>
          <div v-if="!videos.length" class="empty">
            {{ view === "library" ? "影视库暂无已下载视频" : "没有未下载视频" }}
          </div>
        </div>
        <div class="pagination">
          <button
            :disabled="page === 0"
            @click="
              page--;
              loadVideos();
            "
          >
            上一页</button
          ><span>{{ page + 1 }} / {{ pages }}</span
          ><button
            :disabled="page + 1 >= pages"
            @click="
              page++;
              loadVideos();
            "
          >
            下一页
          </button>
        </div>
      </section>
      <section v-else>
        <div v-if="view === 'translations'" class="panel">
          <h2>WebAI 翻译交换</h2>
          <p>
            导出只包含视频标识、标题、标签和原文哈希，不包含代理密码、Cookie、下载地址或本地路径。
          </p>
          <div class="inline">
            <select v-model="translationLanguage">
              <option value="zh-CN">简体中文 zh-CN</option>
              <option value="zh-TW">繁体中文 zh-TW</option>
              <option value="en">英语 en</option>
              <option value="ja">日语 ja</option></select
            ><button class="primary" @click="exportTranslation">
              导出翻译 JSON</button
            ><button v-on:click="refreshNFO">刷新全部 Plex NFO</button
            ><label class="upload"
              >回导翻译 JSON<input
                type="file"
                accept="application/json,.json"
                @change="importTranslation"
            /></label>
          </div>
          <small
            >WebAI 只需填写 translations，并保留 key、source_hash 和
            fields；原文变化的条目会自动跳过。</small
          >
        </div>
        <SettingsView
          v-else
          :settings="settings"
          :busy="busy"
          :import-html="importHTML"
          :demo-mode="demoMode"
          :sidebar-items="sidebarItems"
          :backups="backups"
          :backup-directory="backupDirectory"
          :backup-sensitive="backupSensitive"
          :exclusion-preview="exclusionPreview"
          :exclusion-total="exclusionTotal"
          @preview-exclusions="previewExclusions"
          v-on:toggle-demo="toggleDemo"
          v-on:update-sidebar-item="updateSidebarItem"
          v-on:save="saveSettings"
          v-on:create-backup="createBackup"
          v-on:restore-backup="restoreBackup"
          v-on:import="importPage"
          v-on:update:import-html="importHTML = $event"
        />
      </section>
    </main>
    <nav class="mobile-tabbar" aria-label="主要导航">
      <button
        v-for="item in [
          { id: 'home', label: '概览' },
          { id: 'subscriptions', label: '订阅' },
          { id: 'downloads', label: '下载' },
          { id: 'library', label: '影视库' },
          { id: 'more', label: '更多' },
        ]"
        :key="item.id"
        :class="{
          active:
            view === item.id ||
            (item.id === 'more' &&
              !['home', 'subscriptions', 'downloads', 'library'].includes(view)),
        }"
        @click="item.id === 'more' ? setSidebarCollapsed(false) : changeView(item.id)"
      >
        <NavIcon :name="item.id" /><span>{{ item.label }}</span
        ><i
          v-if="item.id === 'downloads' && data.counts.active + data.counts.waiting"
          >{{ data.counts.active + data.counts.waiting }}</i
        >
      </button>
    </nav>
  </div>
  <div v-if="selected" class="modal" @click.self="selected = undefined">
    <div class="player" role="dialog" aria-modal="true" :aria-label="displayTitle(selected)">
      <button class="close" @click="selected = undefined">关闭</button>
      <h2>{{ displayTitle(selected) }}</h2>
      <video
        v-if="selected.status === 'completed'"
        :src="'/media/' + selected.video_id"
        controls
        playsinline
        preload="metadata"
      /><img v-else :src="'/thumb/' + selected.video_id" :alt="displayTitle(selected)" />
      <div class="detail">
        <span>时长 {{ duration(selected.duration_seconds) }}</span
        ><span>播放 {{ selected.views.toLocaleString() }}</span
        ><span v-if="selected.likes">点赞 {{ selected.likes.toLocaleString() }}</span
        ><span v-if="selected.like_rate">好评 {{ selected.like_rate.toFixed(0) }}%</span
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
      <p v-if="selected.description" class="video-description">{{ selected.description }}</p>
      <div class="chips">
        <span v-for="tag in displayTags(selected)" :key="tag">{{ tag }}</span>
      </div>
      <a :href="selected.page_url" target="_blank">打开详情页</a>
    </div>
  </div>
</template>
