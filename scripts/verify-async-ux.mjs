// Static checks that App.vue wires the shared async feedback correctly.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const style = read("../src/style.css");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// Shared composable, no new state-management dependency.
check(app.includes('from "./asyncAction"'), "must import the shared async composable");
check(app.includes("createAsyncActions()"), "must create async actions");
check(!app.includes("pinia") && !app.includes("vuex"), "no new state-management dependency");

// A/B: subscribe shows loading immediately and keeps it while pending.
check(app.includes("subscribePending") && app.includes("正在添加"), "subscribe button must show loading");
check(app.includes('subscribePending || !modelURL.trim()') || /subscribePending[^>]*disabled/.test(app), "subscribe must be disabled while pending");
check(app.includes('role="alert"') && app.includes("subscribeError"), "subscribe error must be visible");

// F: non-idempotent timeout guidance (no blind resubmit).
check(app.includes("请先刷新订阅列表确认是否已创建"), "subscribe must warn to verify before retrying");

// G/H/I/J: translation import feedback, no duplicate submit, input reset, real counts.
check(app.includes("translationFileName") && app.includes("正在导入"), "import must show file + progress");
check(app.includes(":disabled=\"translationImportPending\""), "import input must be disabled while pending");
check(app.includes('input.value = ""'), "file input must reset for re-selecting the same file");
check(app.includes("imported_fields") && app.includes("skipped_items"), "import result must use real server counts");
check(!/正在导入.*\d+%/.test(app), "must not fake percentage progress");

// L: stale feedback cleared on view change.
check(app.includes("subscribeError.value = \"\"") && app.includes("integrityNotice.value = \"\""), "view change must clear stale feedback");

// M: VDF conflict still blocks execution (no UX bypass).
check(app.includes("vdfPreviewData.has_conflicts"), "VDF execute must stay disabled on conflicts");
check(app.includes("二次确认并执行 VDF 清理"), "VDF double-confirm retained");
check(style.includes(".upload.disabled"), "disabled upload style");
check(style.includes(".error"), "error style retained");

if (failures.length) {
  console.error("async UX check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("async UX check: OK");
