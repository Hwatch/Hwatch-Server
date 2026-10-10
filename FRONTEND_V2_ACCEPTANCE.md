# Frontend 2.0 Integration & Quality Acceptance

## 结论与范围

具备隔离环境人工长期试用条件，不等同于生产发布批准。未合并 main、未推送本轮提交、未部署生产、未修改 NAS/生产 SQLite/TransRunner。

本轮只修复请求调度问题，不继续重做视觉。真实 iPhone Safari、真实 NAS 性能、真实媒体解码/拖动播放均 **NOT TESTED**。播放器入口和弹窗已测试，但 Mock 媒体不是实际播放成功。

## 功能对照

| 对照版本 | 证据与边界 |
|---|---|
| 本地 main `6aafd7f` | 已比较；较早的 compute-node API 与 TransRunner 基线不同，该差异先于 V2，不是本轮移除功能 |
| 已获取 origin/main `db225b4` | 已核对引用；未修改、未合并。不是本轮重新查询生产部署的证明 |
| refactor/transrunner-integration `3b06eab` | 与 UX-3 参考提交一致 |
| 用户指定部署版本 `3b06eab` | 作为 Classic 对照；未重新 SSH 确认生产运行二进制 |
| feature/frontend-v2 `0ea3618` → 本轮 | `cmd/`、`internal/`、`openapi.yaml`、`API_V1.md` 相对 `3b06eab` 无差异 |

| 能力 | 结果与验证 |
|---|---|
| UX-1～3 Loading、防重复、乱序保护 | 原 asyncAction 保留；静态验证与延迟提交浏览器测试通过 |
| Toast 自动消失 | 浏览器测试通过 |
| 通知操作记录 | 保留 accepted 请求级语义，不冒充任务完成；浏览器测试通过 |
| VDF Keep/Remove 跨组冲突 | 整计划执行按钮被阻止；修改 JSON 会清除旧预览；两引擎测试通过 |
| VDF already_done | 此前已处理状态显示、无可执行组时禁止确认执行；两引擎专项测试通过 |
| Service API | 后端/契约无改动；原 Go Service Token/TransRunner 测试通过，不调用实际 TransRunner |
| 已排除恢复 | 取消不请求、确认仅恢复允许下载，不直接建下载；专项浏览器测试通过 |
| 下载恢复/暂停 | 原 API 与持久状态机未改；单任务控制/CSRF 测试与 Go 测试通过；真实 aria2 大任务未测试 |
| 翻译 | 回导、重选文件、JSON 错误、结果反馈测试通过；导出下载与目标语言另有专项测试。编辑仅指修改导入 JSON，不新增在线翻译编辑器 |
| 设置/备份 | 真实隔离设置保存与 CSRF；失败保留输入；恢复取消/确认与 202 语义测试通过。浏览器测试不执行真实备份恢复重启，后端原恢复测试通过 |

## 隔离环境与复现

- V2：`http://127.0.0.1:5189`，`npm run preview:v2` 创建临时 SQLite/media、仅监听 loopback，后端 `127.0.0.1:18787`。
- Classic：`http://127.0.0.1:5187`，`npm run preview:classic` 从 `3b06eab` 只读归档到临时目录，代理仅指向同一隔离后端，不覆盖 Classic 仓库/构建。
- 自动化 V2：`127.0.0.1:5188`，受控 `/api/**` Mock；未命中的开发代理仍指向隔离后端。
- 使用专用测试管理员；凭据不属于生产账号。真实联调只登录、保存隔离设置、验证 SSE 响应头、退出自己的会话。
- `npx playwright install chromium webkit`；先开两个预览，再 `HWATCH_LIVE_TEST=1 npm run test:e2e`。
- SSE Mock 明确不测试真实传输：短暂 route.fulfill 流引发 WebKit 导航访问控制噪声，改为确定性 EventSource 替身。live 测试使用真实 SSE 200/content-type；断网重连和事件负载端到端仍未专门覆盖。

## 页面验收与截图

Chromium / WebKit 覆盖 390、768、1280、1920px 的十页：概览、订阅、下载、未下载、影视库、已排除、完整性、翻译、通知、设置；断言无横向溢出、页面脚本/控制台错误，保存截图。另覆盖手机导航、筛选、分页、播放器入口、删除弹窗、Tab/Esc 焦点、强演示模式、主题持久化与跟随系统。

- 功能回归与真实联调：44/44 通过（每引擎 22 项，45.2 秒）；规模测试 2/2 通过（21.8 秒），共 46 项。
- `npm test` 通过，CHROME 指定 Playwright Chromium，poster-layout 未跳过。
- `go test ./...`、`go vet ./...`、`go build`、`npm run build`、`git diff --check` 通过。
- Desktop：`acceptance/after/chromium/1280-library.png`；Mobile：`acceptance/after/chromium/390-library.png`。
- WebKit 同路径前缀 `acceptance/after/webkit/`；延迟图片大列表截图在 `acceptance/scale/<engine>/`。
- 截图/HTML 报告保留本地，不提交大量二进制图片；合成性能 JSON 提交在 `acceptance/metrics/`。

## 性能方法和实际数据

相同受控数据：3400 个媒体记录、3400 个活动任务，长原文与长译文标题；列表响应固定延迟 50ms，封面固定延迟 150ms。分页是真正按 offset/limit 返回，搜索按 q 过滤；不调用生产数据库。每版本/宽度单次测量，开发模式、同机顺序执行，非统计显著基准。

单位 ms。Ready 从 goto 到 48 张卡片出现（不等全部封面）；分页/搜索包含 Playwright 等待与应用 debounce；切页指从 goto 下载页到 48 任务出现。“列表请求”只统计此流程的 `/api/videos`、`/api/tasks`；“全部 API 请求”包含该流程的全部 `/api/*`（不含图片/媒体），不代表长期轮询总量。滚动操作耗时在原始 JSON 中，不等于 FPS 或卡顿率。

| 引擎 | 版本 | 宽度 | Ready | 下一页 | 搜索 | 切页 | 列表请求 | 全部 API 请求 |
|---|---|---:|---:|---:|---:|---:|---:|---:|
| chromium | classic | 390 | 212 | 138 | 818 | 66 | 4 | 9 |
| chromium | classic | 768 | 135 | 143 | 808 | 67 | 4 | 9 |
| chromium | classic | 1280 | 118 | 147 | 809 | 55 | 4 | 9 |
| chromium | classic | 1920 | 131 | 219 | 793 | 81 | 5 | 10 |
| chromium | v2 | 390 | 275 | 289 | 864 | 122 | 4 | 9 |
| chromium | v2 | 768 | 141 | 110 | 369 | 88 | 4 | 9 |
| chromium | v2 | 1280 | 154 | 291 | 944 | 68 | 4 | 9 |
| chromium | v2 | 1920 | 151 | 327 | 856 | 70 | 4 | 9 |
| webkit | classic | 390 | 258 | 224 | 808 | 109 | 5 | 10 |
| webkit | classic | 768 | 153 | 146 | 796 | 70 | 4 | 9 |
| webkit | classic | 1280 | 135 | 150 | 797 | 81 | 4 | 9 |
| webkit | classic | 1920 | 146 | 154 | 788 | 83 | 5 | 10 |
| webkit | v2 | 390 | 408 | 262 | 838 | 142 | 4 | 9 |
| webkit | v2 | 768 | 144 | 148 | 849 | 97 | 4 | 9 |
| webkit | v2 | 1280 | 143 | 159 | 374 | 95 | 4 | 9 |
| webkit | v2 | 1920 | 163 | 186 | 383 | 91 | 4 | 9 |

两版本每页均 48 个媒体/任务 DOM 卡片。懒加载宽屏可能一次加载全部 48 个，窄屏则分批；不宣称只加载可见图片。V2 个别分页/首次加载比 Classic 慢，需真实设备、多轮生产构建测量后再决定优化，不能据单次结果声称全面提速或百分比改善。

## 本轮修复

`5e3b42e`：取消导航重置字段产生的重复延迟列表请求；取消离开列表后仍待执行的搜索；显式加载清除 debounce。新增测试断言初次列表仅请求一次、离开后不发延迟请求。保留请求 ID、AbortController 和每页 48 条，不修改 API/业务语义。

`6ed6cbb`：WebKit 的受控 attachment 导出测试未产生下载事件，将 location 导航导出改为通过原 API 客户端取 JSON、Blob 下载；文件名为 `hwatch-translations-<language>.json`，不离开当前页面。新增 Loading/重复提交保护与可见错误反馈。Chromium/WebKit 专项测试通过；该失败来自 Mock 场景，不据此断言所有真实 Safari 导出均失败。

测试与工具修复：WebKit Mock SSE 隔离；截图按引擎分目录防覆盖；真实退出测试适配账户菜单；补齐 already_done 合成响应契约字段；固定测试代理指向隔离后端。未把测试夹具错误当应用故障。

## 未解决/未测试

- 真实 iPhone Safari **NOT TESTED**（桌面 WebKit 不可冒充）。
- 生产部署版本未重新确认；NAS RTT、SQLite 查询性能、真实缩略图缓存、真实 aria2/wget 任务与大媒体播放解码不在 Mock 结论内。
- 3400 条是受控 Mock 记录，不是对 3400 条 SQLite 查询的性能保证。
- 无滚动帧率/内存长期泄漏/长时间后台恢复指标；SSE 重连负载恢复需后续网络故障专项。
- 本轮能供隔离人工试用，不自动批准生产切换；明确许可后再考虑合并/部署。

## Git 交付

- `5e3b42e` 列表重复/延迟请求修复。
- `6ed6cbb` 不导航的 JSON 导出、Loading 与错误反馈。
- `087303d` 双浏览器、隔离 Classic、功能与规模验收测试。
- 最后的文档提交保存本报告与合成指标 JSON；本轮仅本地提交，不自动推送或合并。
