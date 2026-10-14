// Frontend check: every navigation entry (sidebar + Settings hub) must resolve
// to a real SVG icon implementation.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const icon = read("../src/NavIcon.vue");
const app = read("../src/App.vue") + read("../src/useHWatchApp.ts");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// "已排除" must render a real SVG child (not an empty template).
const excluded = icon.match(/name==='excluded'[^>]*>([\s\S]*?)<\/template>/);
check(!!excluded, "NavIcon has no 'excluded' template");
check(!!excluded && /<(path|circle|line|rect|polyline)\b/.test(excluded[1]), "'excluded' icon has no SVG child");

const idsFrom = (block) => [...block.matchAll(/id:\s*"([a-z0-9_-]+)"/g)].map((m) => m[1]);

// Sidebar primary nav groups.
const navBlock = app.slice(app.indexOf("const navGroups"), app.indexOf("function loadOpenGroups"));
// Settings -> 高级管理 entries.
const mgmtStart = app.indexOf("const managementEntries");
const mgmtBlock = mgmtStart >= 0 ? app.slice(mgmtStart, app.indexOf("];", mgmtStart)) : "";
check(mgmtBlock.length > 0, "managementEntries not found");

const navIds = [...navBlock.matchAll(/items:\s*\[([\s\S]*?)\]/g)].flatMap((m) =>
  [...m[1].matchAll(/id:\s*"([a-z0-9_-]+)"/g)].map((x) => x[1]),
);
const ids = [...new Set([...navIds, ...idsFrom(mgmtBlock)])];
check(ids.includes("excluded"), "managementEntries must include 'excluded'");
check(ids.includes("subscriptions") && ids.includes("downloads"), "managementEntries must include subscriptions/downloads");
for (const id of ids) {
  check(icon.includes(`name==='${id}'`), `NavIcon missing implementation for '${id}'`);
}

if (failures.length) {
  console.error("nav icon check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log(`nav icon check: OK (${ids.length} entries)`);
