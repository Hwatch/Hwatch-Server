import { spawn } from "node:child_process";
import { mkdtempSync, mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
const root = mkdtempSync(join(tmpdir(), "hwatch-v2-"));
const media = join(root, "media");
mkdirSync(media);
const binary = join(root, "hwatch");
const build = spawn("go", ["build", "-o", binary, "./cmd/r34watch"], {
  stdio: "inherit",
});
await new Promise((resolve, reject) => {
  build.on("error", reject);
  build.on("exit", (code) =>
    code === 0 ? resolve() : reject(Error("Go build failed")),
  );
});
const backend = spawn(binary, [], {
  env: {
    ...process.env,
    R34WATCH_DB: join(root, "test.db"),
    R34WATCH_WEB: join(process.cwd(), "web/dist"),
    R34WATCH_MEDIA_ROOT: media,
    R34WATCH_ADDR: "127.0.0.1:18787",
    R34WATCH_ARIA_RPC: "http://127.0.0.1:16888/jsonrpc",
    R34WATCH_ARIA_SECRET: "",
  },
  stdio: "inherit",
});
const frontend = spawn(
  process.execPath,
  ["node_modules/vite/bin/vite.js", "--host", "127.0.0.1", "--port", "5189", "--strictPort"],
  {
    env: { ...process.env, HWATCH_DEV_API: "http://127.0.0.1:18787" },
    stdio: "inherit",
  },
);
let stopped = false;
function stop() {
  if (stopped) return;
  stopped = true;
  backend.kill("SIGTERM");
  frontend.kill("SIGTERM");
}
process.on("SIGINT", stop);
process.on("SIGTERM", stop);
frontend.on("exit", stop);
backend.on("exit", stop);
console.log(
  "Frontend 2.0: http://127.0.0.1:5189 · isolated backend http://127.0.0.1:18787",
);
console.log("Temporary SQLite and media directory:", root);
