// Frontend + backend contract check for the library "random discovery" panel.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const composable = read("../src/useHWatchApp.ts");
const css = read("../src/frontend-v2.css");
const store = read("../internal/store/store.go");
const main = read("../cmd/r34watch/main.go");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// Backend: read-only random query over formal library media.
check(main.includes('"GET /api/library/random"'), "missing GET /api/library/random route");
check(main.includes("func (a *App) libraryRandom"), "missing libraryRandom handler");
check(/func \(s \*Store\) RandomVideos\(/.test(store), "missing Store.RandomVideos");
check(/opts\.Status = "completed"/.test(store), "random query must force status=completed");
check(/file_path<>''/.test(store), "random query must require a non-empty media path");
check(/id NOT LIKE 'test:%'/.test(store), "random query must exclude test fixtures");
check(/ORDER BY random\(\)/.test(store), "random query must use SQL random() over the whole pool");

// Frontend: panel, loading/dedupe, scope switch, history de-dup.
check(app.includes("随机发现"), "missing random discovery panel title");
check(app.includes("换一批"), "missing 换一批 button");
check(app.includes('actionPending("library-random")'), "换一批 must use the library-random action key");
check(/class="library random-grid"/.test(app), "random results must reuse the library grid");
check(/@play="selected = v"/.test(app), "random cards must reuse the play entry point");
check(/randomScope/.test(app) && app.includes("全部媒体") && app.includes("当前筛选"), "missing scope switch");
check(/const RANDOM_LIMIT = 6/.test(composable), "default batch must be 6");
check(/function loadRandom\(/.test(composable), "missing loadRandom");
check(/actions\.run\("library-random"/.test(composable), "loadRandom must use the library-random action");
check(/randomHistory/.test(composable) && /p\.set\("exclude"/.test(composable), "must pass recent history as exclude");
check(/randomScope\.value === "current"/.test(composable), "random scope must honor current filters");
check(/if \(items\.length === 0 && randomHistory\.value\.length && allowReset\)/.test(composable), "must reset history when the pool is exhausted");
check(/if \(!result\.ok\)/.test(composable) && /randomError\.value = result\.error/.test(composable), "failure must keep previous batch and set an error");

// Layout.
check(/\.random-discovery/.test(css), "missing random-discovery styles");

if (failures.length) {
  console.error("library random check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("library random check: OK");
