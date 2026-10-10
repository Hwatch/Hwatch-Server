// Frontend check for subscription-check auto-refresh (no forced navigation).
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/useHWatchApp.ts");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// The check action polls the bounded-check status endpoint.
check(app.includes("/api/check/status"), "check() must poll /api/check/status");
check(/function watchCheckCompletion\(/.test(app), "missing watchCheckCompletion");
check(/status\.running/.test(app), "watcher must read the running count");
// Completion refreshes subscription/dashboard state.
check(/notify\("订阅检查已完成，状态已刷新"/.test(app), "missing completion toast");
check(/await refreshDashboard\(\)/.test(app), "completion must refresh dashboard/subscriptions");
// It must not force-navigate away from the current view any more.
const checkFn = app.slice(app.indexOf("async function check("), app.indexOf("async function toggle("));
check(!!checkFn, "missing check()");
check(!/changeView\("downloads"\)/.test(checkFn), "check() must not force-navigate to downloads");

if (failures.length) {
  console.error("check refresh check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("check refresh check: OK");
