// Lightweight frontend check for the video-deletion confirm modal.
// No test framework is configured in this repo, so this asserts on the source.
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

const appPath = fileURLToPath(new URL("../src/App.vue", import.meta.url));
const src = readFileSync(appPath, "utf8");

const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

const start = src.indexOf('v-if="deleteTarget"');
check(start !== -1, "delete modal block not found");
const block = start === -1 ? "" : src.slice(start, src.indexOf("</template>", start));

check((block.match(/delete-option__radio/g) || []).length === 2, "expected two delete-option radios");
check(block.includes('class="delete-option"'), "missing .delete-option class");
check(block.includes('class="delete-option__radio"'), "missing .delete-option__radio class");
check(block.includes('class="delete-option__content"'), "missing .delete-option__content class");
check(block.includes('value="delete_and_exclude"'), "missing delete_and_exclude option");
check(block.includes('value="delete_only"'), "missing delete_only option");
check(!block.includes("setting-check"), "delete modal must not reuse .setting-check");
check(src.includes('deleteMode.value = "delete_and_exclude"'), "default mode must be delete_and_exclude");

if (failures.length) {
  console.error("delete modal check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("delete modal check: OK");
