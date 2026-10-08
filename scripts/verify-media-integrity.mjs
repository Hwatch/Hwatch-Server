// Frontend check for the media integrity / VDF reconciliation UI.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const style = read("../src/style.css");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

check(app.includes('id: "integrity"'), "missing integrity nav item");
check(app.includes('integrity: "/integrity"'), "missing integrity route");
check(app.includes("/api/media-integrity/scan"), "missing scan call");
check(app.includes("/api/media-integrity/import"), "missing import call");
check(app.includes("/api/media-integrity/apply"), "missing apply call");
check(app.includes("确认执行清理"), "missing confirm execute button");
check(app.includes("Dry Run"), "dry run must be explicit");
check(app.includes("cleanupStatusLabel"), "missing per-status labels");
check(app.includes("cleanupStatusLabels"), "missing status label map");
check(style.includes(".integrity-badge"), "missing integrity badge style");

if (failures.length) {
  console.error("media integrity check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("media integrity check: OK");
