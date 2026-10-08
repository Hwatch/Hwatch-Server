// Behavioural test of the toast lifecycle composable.
import { createToasts } from "../src/toast.js";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// A: success toast auto-dismisses.
{
  const t = createToasts();
  t.push({ type: "success", message: "ok", duration: 25 });
  check(t.list.length === 1, "A: toast is shown");
  await sleep(60);
  check(t.list.length === 0, "A: success toast auto-dismissed");
}

// B: manual dismiss is immediate.
{
  const t = createToasts();
  const id = t.push({ type: "error", message: "boom", duration: 10000 });
  t.dismiss(id);
  check(t.list.length === 0, "B: manual dismiss removes immediately");
}

// C: pending toast is not auto-dismissed, then updates to a result.
{
  const t = createToasts();
  const id = t.push({ type: "pending", message: "处理中…" });
  await sleep(60);
  check(t.list.length === 1, "C: pending must not auto-dismiss");
  t.update(id, { type: "success", message: "完成", pending: false, duration: 25 });
  await sleep(60);
  check(t.list.length === 0, "C: pending resolved then dismissed");
}

// F: duplicate toasts for the same key merge/update.
{
  const t = createToasts();
  t.push({ key: "op", type: "info", message: "a", duration: 1000 });
  t.push({ key: "op", type: "success", message: "b", duration: 1000 });
  check(t.list.length === 1, "F: same key must not create a second toast");
  check(t.list[0].message === "b" && t.list[0].type === "success", "F: existing toast updated");
}

// G: queue policy caps visible toasts at 3.
{
  const t = createToasts({ max: 3 });
  for (let i = 0; i < 6; i++) t.push({ type: "info", message: "m" + i, duration: 1000 });
  check(t.list.length === 3, "G: at most 3 toasts, got " + t.list.length);
}

// D: clearAll removes everything (used on unmount).
{
  const t = createToasts();
  t.push({ type: "pending", message: "p" });
  t.push({ type: "info", message: "i", duration: 1000 });
  t.clearAll();
  check(t.list.length === 0, "D: clearAll removes all toasts");
}

if (failures.length) {
  console.error("toast check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("toast check: OK");
