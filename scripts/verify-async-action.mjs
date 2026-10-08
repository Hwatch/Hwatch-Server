// Behavioural test of the shared async-action composable (delay + failure).
import { createAsyncActions } from "../src/asyncAction.js";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const failures = [];
const check = (ok, message) => {
  if (!ok) failures.push(message);
};

// A: pending is set synchronously before the first await.
{
  const a = createAsyncActions();
  let sawPendingBeforeAwait = false;
  const run = a.run("a", async () => {
    sawPendingBeforeAwait = a.pending("a");
    await sleep(50);
    return 1;
  });
  check(a.pending("a") === true, "A: pending must be true immediately after run()");
  check(sawPendingBeforeAwait, "A: pending must be set before the first await");
  const r = await run;
  check(r.ok && r.value === 1, "A: success result");
  check(a.pending("a") === false, "A: pending cleared after completion");
}

// C: duplicate submissions of the same key run only once.
{
  const a = createAsyncActions();
  let calls = 0;
  const first = a.run("dup", async () => {
    calls++;
    await sleep(60);
    return "first";
  });
  const second = await a.run("dup", async () => {
    calls++;
    return "second";
  });
  check(second.duplicate === true, "C: second submit while pending is a duplicate");
  check(calls === 1, "C: fn must run once, got " + calls);
  await first;
  check(calls === 1, "C: no extra call after first completes");
}

// E: business/failure errors are captured with the message.
{
  const a = createAsyncActions();
  const r = await a.run("err", async () => {
    throw new Error("业务错误：源已变化");
  });
  check(r.ok === false && r.error === "业务错误：源已变化", "E: error captured");
  check(a.error("err") === "业务错误：源已变化", "E: error() exposes message");
  check(a.pending("err") === false, "E: pending restored after failure");
}

// K: independent keys do not affect each other.
{
  const a = createAsyncActions();
  const slow = a.run("slow", async () => {
    await sleep(80);
    return "slow";
  });
  const fast = a.run("fast", async () => "fast");
  const fastResult = await fast;
  check(fastResult.ok && fastResult.value === "fast", "K: fast action succeeds");
  check(a.pending("slow") === true, "K: slow action still pending");
  check(a.pending("fast") === false, "K: fast action finished");
  await slow;
  check(a.pending("slow") === false, "K: slow action finished");
}

// reset() clears state and error.
{
  const a = createAsyncActions();
  await a.run("r", async () => {
    throw new Error("x");
  });
  a.reset("r");
  check(a.status("r") === "idle" && a.error("r") === "", "reset clears state");
}

if (failures.length) {
  console.error("async action check failed:");
  for (const f of failures) console.error(" - " + f);
  process.exit(1);
}
console.log("async action check: OK");
