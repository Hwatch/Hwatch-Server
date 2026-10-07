// Browser layout regression: a vertical cover must not stretch the card.
// Skips gracefully when no Chromium/Chrome is available.
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const stylePath = fileURLToPath(new URL("../src/style.css", import.meta.url));

const candidates = [
  process.env.CHROME,
  "chromium",
  "chromium-browser",
  "google-chrome",
  "google-chrome-stable",
].filter(Boolean);
const chrome = candidates.find((c) => c.includes("/") ? existsSync(c) : which(c));
if (!chrome) {
  console.log("poster layout check: skipped (no chromium)");
  process.exit(0);
}

function which(bin) {
  return spawnSync("which", [bin], { stdio: "ignore" }).status === 0;
}

const vertical = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='270' height='480'%3E%3Crect width='100%25' height='100%25' fill='%23335'/%3E%3C/svg%3E";
const horizontal = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='1920' height='1080'%3E%3Crect width='100%25' height='100%25' fill='%23753'/%3E%3C/svg%3E";

const html = `<!doctype html><html><head><meta charset="utf-8">
<link rel="stylesheet" href="file://${stylePath}"></head><body>
<div class="library">
<article><button class="poster" id="pv"><span class="poster-skeleton"></span><img id="v" src="${vertical}" alt="v"><span class="poster-duration">1:30</span></button><div class="meta"><b>Vertical</b><small>x</small><p>0</p></div></article>
<article><button class="poster" id="ph"><span class="poster-skeleton"></span><img id="h" src="${horizontal}" alt="h"><span class="poster-duration">0:20</span></button><div class="meta"><b>Horizontal</b><small>x</small><p>0</p></div></article>
</div>
<script>
window.addEventListener('load', () => {
  const pv = document.getElementById('pv').getBoundingClientRect();
  const ph = document.getElementById('ph').getBoundingClientRect();
  const cs = getComputedStyle(document.getElementById('v'));
  document.body.setAttribute('data-result', JSON.stringify({
    posterVerticalH: pv.height, posterHorizontalH: ph.height,
    imgPosition: cs.position, imgObjectFit: cs.objectFit,
    imgWidth: cs.width, imgHeight: cs.height,
  }));
});
</script></body></html>`;

const dir = mkdtempSync(join(tmpdir(), "hwatch-poster-"));
const file = join(dir, "poster.html");
writeFileSync(file, html);
const out = spawnSync(
  chrome,
  ["--headless=new", "--no-sandbox", "--disable-gpu", "--virtual-time-budget=3000", "--dump-dom", `file://${file}`],
  { encoding: "utf8", maxBuffer: 16 * 1024 * 1024 },
).stdout || "";
const dom = out;

const failures = [];
const match = dom.match(/data-result="([^"]*)"/);
if (!match) {
  console.log("poster layout check: skipped (no result from browser)");
  process.exit(0);
}
const decode = (s) => s.replaceAll("&quot;", '"').replaceAll("&amp;", "&");
const result = JSON.parse(decode(match[1]));
const check = (ok, msg) => { if (!ok) failures.push(msg); };

check(Math.abs(result.posterVerticalH - result.posterHorizontalH) <= 1,
  `vertical poster height ${result.posterVerticalH} != horizontal ${result.posterHorizontalH}`);
check(result.imgPosition === "absolute", `poster img must be absolute, got ${result.imgPosition}`);
check(result.imgObjectFit === "cover", `poster img must be object-fit cover, got ${result.imgObjectFit}`);

if (failures.length) {
  console.error("poster layout check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log(`poster layout check: OK (poster ${Math.round(result.posterVerticalH)}px, vertical == horizontal)`);
