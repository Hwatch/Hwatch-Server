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

// The image must not participate in layout: absolute, cover, so vertical
// posters cannot stretch the card.
check(/\.poster\s*>\s*img[^{]*\{[^}]*position:\s*absolute/.test(style), "poster image must be absolutely positioned");
check(/\.poster\s*>\s*img[^{]*\{[^}]*width:\s*100%/.test(style), "poster image must be width 100%");
check(/\.poster\s*>\s*img[^{]*\{[^}]*height:\s*100%/.test(style), "poster image must be height 100%");
check(/\.poster\s*>\s*img[^{]*\{[^}]*object-fit:\s*cover/.test(style), "poster image must use object-fit cover");

// Loading skeleton + fade-in and reduced-motion fallback.
check(/\.poster-skeleton/.test(style), "missing poster skeleton");
check(/poster-shimmer/.test(style), "missing skeleton shimmer keyframes");
check(/\.poster>img[^{]*\{[^}]*transition:\s*opacity/.test(style), "cover must fade in");
check(/prefers-reduced-motion/.test(style), "must respect prefers-reduced-motion");

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

// Cover load errors must fall back to a placeholder, never a broken image.
check(/@error="onCoverError\(\$event/.test(app), "library cover must handle load errors with the video id");
check(app.includes("function onCoverError"), "missing cover error fallback");

if (failures.length) {
  console.error("library card check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("library card check: OK");
