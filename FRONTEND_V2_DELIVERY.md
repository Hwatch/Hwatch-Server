# Frontend 2.0 交付记录

## Git 与边界

- 本地 main 保持 `6aafd7f`；远端 main 核对为 `db225b4`。
- 功能基线 `3b06eab`，来自 `origin/refactor/transrunner-integration`，保留六个近期 UX/VDF 提交。
- Frontend 分支 `feature/frontend-v2`，工作树 `/root/hwatch-frontend-v2`。
- 原生产开发分支引用不变。没有 Go 源码、API 权限或协议修改，没有 tag、合并或生产部署。后续经用户授权将体验分支推送远端，仍不合并主分支。
- SSH host key 改变，生产部署版本仍 NOT VERIFIED。该限制不影响隔离开发，但正式合并前需要维护者核对。

阶段提交：`0d96a54` 基线审计/E2E；`74d6929` tokens/主题；`f46ec3f` 页面与响应式；`8d42f2e` 导入/分页/通知；`da7ec07` 媒体展示与可访问性；`ac68cfc` 状态与页面组合拆分；最后的验收提交记录测试与文档。提交完成后工作树干净（构建资源和截图按 gitignore 保留在本地）。

## 架构

## 体验反馈调整

- 中性炭黑与金色强调；侧栏导航独立滚动，底部固定。导航分组不再折叠，显示偏好与账户菜单可展开。
- 下载指标统一高度与数字对齐，下载器错误详情单独折叠；媒体卡和任务行统一间距、圆角与按钮样式。
- 完整性页面增加区域间距；翻译交换拆为导出、回导、刷新 NFO 三步，窄屏纵向排列。
- 外观主题移至设置的“显示与隐私”，保留浏览器持久化与跟随系统。
- 本轮使用隔离预览验证，未进行生产部署；真实 iPhone/Safari 与真实媒体验收仍待完成。

## 架构说明

App 保留页面组合与模板绑定，原业务状态、API 调用、SSE、乱序保护和生命周期机械提取到 `src/useHWatchApp.ts`。没有建立第二套状态管理。复用 `asyncAction.js`、`toast.js`、API Client。

新增 SidebarView、PageHeader、ThemeControl、MediaCard、StatusBadge、StatePanel、ToastContainer、PaginationBar、JSONImport；媒体卡片只接收展示字段，通过事件交由业务层操作。新增弹窗焦点指令，提供 Tab 循环和关闭后焦点恢复。

设计系统使用 `frontend-v2.css` 的 CSS Variables：背景/表面/文字/边框/语义颜色、圆角、间距和阴影，支持暗色/亮色/系统主题与持久化。使用系统中文字体，无专有字体、UI 框架或额外运行时依赖。

新增开发依赖 `@playwright/test`。兼容版本修复 brace-expansion 与 source-map-js，npm audit 为 0。

## 页面完成情况

| 页面 | 改进 | 保留与验收 |
|---|---|---|
| 登录 | 统一主题、连接失败反馈、重复提交 guard | 首次管理员、登录、退出；真实临时 SQLite 联调 |
| 概览 | 统一指标卡、存储条、页头与响应式 | 真实字段、最近入库/播放、检查更新；四视口截图 |
| 订阅 | 名称/平台/链接和状态筛选，单来源按钮禁用与文案 | 作者/番剧、多平台、基线选择、搜索与检查；延迟提交 E2E |
| 下载 | 状态徽标、异常提示、分页、全局控制 | 原引擎与任务语义、限流提示、SSE；CSRF 与操作 E2E |
| 未下载 | 共享媒体卡、下载提交防重复 | 现有筛选、下载全部；路由与四视口 |
| 影视库 | 展示组件、图片懒加载、加载失败重试、统一按钮 | 译文优先、字段/多标签搜索、排序、分页、播放器 |
| 已排除 | 搜索标题/原因/ID，恢复前确认 | 恢复允许下载不直接建队；页面截图 |
| 媒体完整性 | JSON 文件读入、组折叠、状态筛选、每页25组、编辑后使预览失效 | Dry Run、has_conflicts、跨组 Keep/Remove、already_done、计划与隔离区；冲突 E2E与原 Go 测试 |
| 翻译 | 统一表单、文件状态、结果反馈 | 原导出/回导覆盖合并语义、同文件可重选、NFO请求级文案 |
| 通知 | Toast 组件、导航避让、级别/未读筛选 | 持久系统提醒、服务端操作记录、自动关闭；请求受理不等于完成 |
| 设置 | 主题统一、手机表单、保存失败保留输入、令牌操作防重复 | 原设置字段/默认值、TransRunner；真实设置保存和 Mock 失败 |
| 备份恢复 | 一致按钮与反馈、响应式列表 | 原二次确认、不确定状态和禁止自动重试；Mock取消/受理 |

## 运行与浏览器验收

```sh
npm ci
npx playwright install chromium
npm test
npm run build
npm run test:e2e
```

最后一项默认运行16个合成交互场景，真实后端场景按需启用。以下方式运行独立的临时 Go / SQLite / 媒体实例，Vite API 指向独立后端，不会触达生产：

```sh
npm run preview:v2
# 另一个终端
HWATCH_LIVE_TEST=1 npm run test:e2e
```

预览地址 `http://127.0.0.1:5189`，后端 `127.0.0.1:18787`；首次打开创建测试管理员。每次启动生成新的 `/tmp/hwatch-v2-*` 数据目录。测试专用账号仅在临时实例中。Ctrl+C 停止两个进程。当前预览已启动，可直接人工体验。

验收结果：17项 Chromium E2E通过，四视口（390/768/1280/1920）无横向溢出；主要页面无 pageerror。npm test、Vue TypeScript/Vite build、Go tests/vet/build、diff check通过。旧竖版封面回归亦使用安装的 Chromium验证。

截图保存在 `acceptance/before`（40张）、`acceptance/after`（暗色四视口、亮色两视口、手机弹窗）；对比入口 `acceptance/index.html`，交互报告 `acceptance/report/index.html`。截图为可再生成的验收产物，不提交 Git。

## 性能

基线：JS 164.51 kB / gzip 56.93 kB；CSS 30.39 kB / gzip 7.23 kB。
新版：JS 184.39 kB / gzip 64.20 kB；CSS 40.47 kB / gzip 9.19 kB。
新增主题、交互与提取结构使压缩 JS 增加约7.27kB、CSS增加约1.96kB，没有大型组件框架。

沿用媒体48项分页、封面 lazy/async；下载增加分页入口；VDF改为25组分页、按需展开；没有增加周期轮询。减少动效设置覆盖主题和骨架动画。尚未做真实数千媒体性能基准，不能承诺具体首屏耗时。

## 已知限制与 NOT TESTED

- 真实生产媒体播放、NFS故障、真实恢复重启与VDF隔离区执行未在生产验收；使用Mock与现有后端测试验证安全分支。
- 部分复杂页面模板仍在App中；业务状态集中提取后可继续拆页，未重写状态机。
- 主题使用末层tokens覆盖原CSS，旧样式仍保留，后续可按页收敛重复规则。
- 没有扩展API提供尚不存在的订阅完整编辑、外部实时遥测、全库统计或高级字幕能力。
- 未在真实iPhone/Safari、电视、屏幕阅读器执行验收；当前验证为Chromium手机视口与键盘。
- 未进行生产首次加载速度与大规模真实数据压力测试。

## 结论

新版可独立运行，与TransRunner集成分支现有API兼容，可进入人工体验。自动化核心回归通过；合并主线前仍需核对生产基线、评审分支差异并人工验证真实媒体、Safari和大数据量。独立分支与Worktree保留。
