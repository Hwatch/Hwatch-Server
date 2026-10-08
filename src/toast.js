import { reactive } from "vue";

const DEFAULT_DURATION = {
  success: 4000,
  info: 4000,
  warning: 6000,
  error: 10000,
  pending: 0,
};

// Toast lifecycle: idle -> showing -> dismissed, with pending -> result -> dismissed
// for long operations. Each toast owns a timer that is always cleaned up.
export function createToasts(options = {}) {
  const max = options.max || 3;
  const list = reactive([]);
  const timers = new Map();
  let seq = 0;

  function clearTimer(id) {
    const timer = timers.get(id);
    if (timer) {
      clearTimeout(timer);
      timers.delete(id);
    }
  }

  function remove(id) {
    clearTimer(id);
    const index = list.findIndex((item) => item.id === id);
    if (index >= 0) list.splice(index, 1);
  }

  function schedule(item) {
    clearTimer(item.id);
    const duration =
      item.duration != null
        ? item.duration
        : DEFAULT_DURATION[item.type] ?? 4000;
    if (duration > 0) {
      timers.set(
        item.id,
        setTimeout(() => remove(item.id), duration),
      );
    }
  }

  function enforceLimit() {
    while (list.length > max) {
      // Prefer dropping the oldest non-pending toast.
      const pendingIndex = list.findIndex((item) => !item.pending);
      const victim = pendingIndex >= 0 ? list[pendingIndex] : list[0];
      remove(victim.id);
    }
  }

  function push(input) {
    const key = input.key;
    if (key) {
      const existing = list.find((item) => item.key === key);
      if (existing) {
        existing.type = input.type || existing.type;
        existing.message = input.message || existing.message;
        existing.pending = !!input.pending;
        if (input.duration != null) existing.duration = input.duration;
        schedule(existing);
        return existing.id;
      }
    }
    const id = ++seq;
    const item = {
      id,
      key,
      type: input.type || "info",
      message: input.message || "",
      pending: !!input.pending,
      duration: input.duration,
      createdAt: Date.now(),
    };
    list.push(item);
    enforceLimit();
    schedule(item);
    return id;
  }

  function update(id, patch) {
    const item = list.find((entry) => entry.id === id);
    if (!item) return;
    Object.assign(item, patch);
    schedule(item);
  }

  function clearAll() {
    for (const item of [...list]) remove(item.id);
  }

  return { list, push, update, dismiss: remove, clearAll };
}
