// Frontend check for the desktop library card layout regression.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const read = (rel) => readFileSync(fileURLToPath(new URL(rel, import.meta.url)), "utf8");
const app = read("../src/App.vue");
const style = read("../src/style.css");
const privacy = read("../src/privacy.css");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// Card uses dedicated classes, not the leaked .library-actions grid.
check(app.includes('class="library-card-actions"'), "library card must use .library-card-actions");
check(app.includes('class="library-card__play"'), "play button must use .library-card__play");
check(!app.includes('class="library-actions"'), "library card must not use .library-actions");

// No full-width only-child CTA leak anywhere.
check(!privacy.includes("button:only-child"), "must not force only-child button full width");

// Cover ratio, title clamp, single-line metadata.
check(/\.poster[^{]*\{[^}]*aspect-ratio:\s*16\s*\/\s*9/.test(style), "cover must use aspect-ratio 16/9");

// Full-width but compact action (not a giant CTA).
const base = privacy.match(/\.library-card__play\s*\{([^}]*)\}/);
check(!!base, "missing .library-card__play rule");
check(!!base && /width:\s*100%/.test(base[1]), "play button must be full width");
const height = base && base[1].match(/height:\s*(\d+)px/);
check(!!height && Number(height[1]) >= 38 && Number(height[1]) <= 42, "play button height must be 38-42px");
check(!!base && /border-radius/.test(base[1]), "play button needs a rounded corner");

check(/\.meta b\{[^}]*line-clamp:\s*2/.test(style), "title must clamp to 2 lines");
check(/\.meta b\{[^}]*min-height/.test(style), "title area needs a min-height for alignment");
check(/\.meta small\{[^}]*white-space:\s*nowrap/.test(style), "metadata subtitle must be single line");

if (failures.length) {
  console.error("library card check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("library card check: OK");
