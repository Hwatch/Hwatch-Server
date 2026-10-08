<script setup lang="ts">
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
      <section v-else-if="view === 'integrity'">
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
          <MediaCard
            v-for="v in videos"
            :key="v.video_id"
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
            :downloadable="view === 'undownloaded'"
            :pending="actionPending(videoActionKey(v.video_id, 'download'))"
            @image="(el) => registerCoverImg(v.video_id, el)"
            @loaded="markCoverLoaded(v.video_id)"
            @error="onCoverError($event, v.video_id)"
            @play="selected = v"
            @download="videoAction(v, 'download')"
          />
          <StatePanel
            v-if="!videos.length"
            :title="
              view === 'library' ? '影视库暂无已下载视频' : '没有未下载视频'
            "
            description="可调整筛选条件，或在订阅管理中添加来源。"
          />
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
            ><label
              class="upload"
              :class="{ disabled: translationImportPending }"
              >回导翻译 JSON<input
                type="file"
                accept="application/json,.json"
                :disabled="translationImportPending"
                @change="importTranslation"
            /></label>
          </div>
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
          <small
            >WebAI 只需填写 translations，并保留 key、source_hash 和
            fields；原文变化的条目会自动跳过。</small
          >
        </div>
        <SettingsView
          v-else
          :settings="settings"
          :api="api"
          :busy="settingsBusy"
          :settings-busy="settingsBusy"
          :backups-busy="backupsBusy"
          :import-busy="importBusy"
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
              !['home', 'subscriptions', 'downloads', 'library'].includes(
                view,
              )),
        }"
        @click="
          item.id === 'more' ? setSidebarCollapsed(false) : changeView(item.id)
        "
      >
        <NavIcon :name="item.id" /><span>{{ item.label }}</span
        ><i
          v-if="
            item.id === 'downloads' && data.counts.active + data.counts.waiting
          "
          >{{ data.counts.active + data.counts.waiting }}</i
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
        <button class="danger" @click="openDelete(selected)">删除</button>
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
