// Frontend check for the per-source 429 backoff banner and i18n.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue") + read("../src/useHWatchApp.ts");
const style = read("../src/style.css");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

check(app.includes("rate-limit-banner"), "missing rate limit banner markup");
check(app.includes("activeRateLimits"), "missing activeRateLimits computed");
check(app.includes("rateLimitedSource"), "missing per-task rate limited hint");
check(app.includes("fmtCountdown"), "missing countdown formatter");
check(/setInterval\(\(\) => \{\s*if \(view\.value === "downloads"\) nowMs\.value = Date\.now\(\)/.test(app),
  "countdown must tick locally (no per-second server request)");
check(app.includes("result.rate_limits"), "tasks response rate_limits not consumed");
check(style.includes(".rate-limit-banner"), "missing .rate-limit-banner style");

// i18n coverage for the new banner.
for (const label of ["来源请求受限", "來源請求受限", "Source temporarily unavailable", "取得元が一時的に制限中"]) {
  check(app.includes(label), `missing i18n label: ${label}`);
}
for (const label of ["等待来源限速解除", "Waiting for source rate limit", "等待来源连接恢复", "Waiting for source connection"]) {
  check(app.includes(label), `missing task-row i18n: ${label}`);
}
check(app.includes("rateLimitedReason"), "missing per-reason rate limit resolution");
check(app.includes("last-error"), "stale queued errors must be shown as historical, not current");

if (failures.length) {
  console.error("rate limit check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("rate limit check: OK");
