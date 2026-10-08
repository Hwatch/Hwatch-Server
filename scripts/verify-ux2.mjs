// Static checks that App.vue uses the toast lifecycle and operation history.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const style = read("../src/style.css");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// Toast lifecycle replaces the old sticky single notice.
check(app.includes('from "./toast"') && app.includes("createToasts"), "must use the toast composable");
check(!app.includes('<button v-if="notice" class="notice" role="status"'), "old sticky notice button must be gone");
check(app.includes('class="toasts"') && app.includes('aria-live="polite"'), "toast stack with aria-live");
check(app.includes("toasts.dismiss(t.id)") && app.includes("toast__close"), "toast needs a keyboard close button");
check(app.includes("toasts.clearAll()"), "toast timers cleared on unmount");
check(style.includes(".toast--error") && style.includes(".toast--success"), "toast type styles");

// Contextual messages instead of a bare "操作已提交".
check(!app.includes('notice.value = "操作已提交"'), "bare 操作已提交 must be contextualised");
check(app.includes("已提交下载任务"), "contextual action messages");

// Operation history reuses the server audit log (trusted, persisted).
check(app.includes("/api/audit"), "operation history must read the server audit log");
check(app.includes("操作记录") && app.includes("notificationTab") && app.includes('"system" | "operations"'), "notifications page must add an operations tab");
check(app.includes("auditResultLabel"), "operation history must show result status");
check(!app.includes("localStorage.setItem(\"hwatch-op-history"), "operation history must not be frontend-only");

if (failures.length) {
  console.error("UX-2 check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("UX-2 check: OK");
