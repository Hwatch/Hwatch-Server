// Static checks that remaining button actions use the shared async feedback.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const settings = read("../src/SettingsView.vue");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// Download center: per-video keys, immediate loading, independent resources.
check(app.includes('return "video:" + videoId'), "video actions must key by video (conflict-safe)");
check(/actionPending\(videoActionKey\(v\.video_id/.test(app), "download rows must use per-video pending");
check(app.includes('actions.run("download-all"') && app.includes("已提交 "), "download-all must show request-level submission");
check(app.includes('actions.run("downloads-toggle"'), "global pause/resume must use its own action key");

// Settings save: failure keeps input; independent from backups.
check(app.includes('actions.run("settings-save"'), "settings save must use an action key");
check(/if \(result\.ok\)[\s\S]{0,200}settings\.value = result\.value/.test(app), "settings only refresh from server on success");
check(app.includes("保存设置失败："), "settings failure must surface the error");
check(app.includes("settingsBusy") && settings.includes("settingsBusy"), "settings busy is wired to SettingsView");
check(settings.includes('settingsBusy ? "正在保存…"'), "settings button shows saving state");

// Backups: distinguish create vs accepted restore, no premature completion, no auto-retry.
check(app.includes('actions.run("backup-create"'), "backup create action key");
check(app.includes('actions.run("backup-restore:"'), "restore action key per backup");
check(app.includes("已提交恢复请求") && !app.includes('notice.value = "正在恢复数据库并重启服务…"'), "restore must not claim completion");
check(app.includes("勿自动重试"), "restore timeout must not auto-retry");
check(settings.includes("backupsBusy") && settings.includes('backupsBusy ? "处理中…"'), "backup buttons show processing state");

// Subscriptions: independent toggle key + resync on failure.
check(app.includes('"sub:" + sub.slug'), "subscription toggle must key by slug");
check(/toggle\(sub[\s\S]{0,400}failed|更新订阅失败/.test(app) || app.includes("更新订阅失败："), "subscription toggle must surface failure");
check(app.includes('actions.run("sub-del:'), "subscription delete action key");

// Background tasks are not claimed complete.
check(!/refresh-nfo[\s\S]{0,200}完成/.test(app), "NFO refresh must not claim completion");
check(app.includes("已提交刷新"), "NFO refresh is request-level");

// Still the same shared mechanisms (no new framework).
check(app.includes('from "./asyncAction"') && app.includes('from "./toast"'), "must keep using asyncAction + toast");
check(!app.includes("pinia") && !app.includes("vuex"), "no new state framework");

if (failures.length) {
  console.error("UX-3 check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("UX-3 check: OK");
