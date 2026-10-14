// Frontend + backend contract check for library random browsing (filter toggle).
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

// Frontend: it is a checkbox inside the filter panel now (no standalone panel).
check(!/random-discovery/.test(app), "standalone random-discovery panel must be gone");
check(!/random-discovery/.test(css), "random-discovery styles must be removed");
check(/class="panel filter-panel"/.test(app), "library must keep the filter panel");
check(/type="checkbox" v-model="randomMode"/.test(app), "filter panel must have a randomMode checkbox");
check(/randomMode && view === 'library' \? randomItems : videos/.test(app), "grid must switch to randomItems in random mode");
check(/loadRandom\(true\)/.test(app) && app.includes("换一批"), "paging must fetch another random batch");
check(/@play="selected = v"/.test(app), "random cards must reuse the play entry point");
check(/const randomMode = ref\(false\)/.test(composable), "composable must define randomMode");
check(/const RANDOM_LIMIT = 24/.test(composable), "random batch size must be 24");
check(/function loadRandom\(/.test(composable), "missing loadRandom");
check(/actions\.run\("library-random"/.test(composable), "loadRandom must use the library-random action");
check(/randomHistory/.test(composable) && /p\.set\("exclude"/.test(composable), "must pass recent history as exclude");
check(/randomMode\.value/.test(composable), "random must honor the randomMode flag");
check(/if \(items\.length === 0 && randomHistory\.value\.length && allowReset\)/.test(composable), "must reset history when the pool is exhausted");
check(/if \(!result\.ok\)/.test(composable) && /randomError\.value = result\.error/.test(composable), "failure must keep previous batch and set an error");

// Layout.
check(/\.filter-panel \.filter-toggle/.test(css), "missing filter toggle styles");

if (failures.length) {
  console.error("library random check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("library random check: OK");
