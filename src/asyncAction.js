import { reactive } from "vue";

// Lightweight per-action async state machine. Each action key owns an
// independent status, so unrelated operations never overwrite each other.
//
// status: idle | pending | success | error
export function createAsyncActions() {
  const states = reactive({});
  const errors = reactive({});

  const pending = (key) => states[key] === "pending";
  const error = (key) => errors[key] || "";
  const status = (key) => states[key] || "idle";

  async function run(key, fn) {
    // Duplicate submissions of the same action are ignored while pending.
    if (states[key] === "pending") {
      return { ok: false, duplicate: true };
    }
    states[key] = "pending";
    errors[key] = "";
    try {
      const value = await fn();
      states[key] = "success";
      return { ok: true, value };
    } catch (e) {
      states[key] = "error";
      errors[key] = e && e.message ? e.message : String(e);
      return { ok: false, error: errors[key] };
    } finally {
      if (states[key] === "pending") states[key] = "idle";
    }
  }

  function reset(key) {
    states[key] = "idle";
    errors[key] = "";
  }

  return { states, errors, pending, error, status, run, reset };
}
