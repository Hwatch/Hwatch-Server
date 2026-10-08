import type { Directive } from "vue";
const handlers = new WeakMap<
  HTMLElement,
  { keydown: (e: KeyboardEvent) => void; previous: HTMLElement | null }
>();
export const dialogFocus: Directive<HTMLElement> = {
  mounted(el) {
    const previous = document.activeElement as HTMLElement | null;
    const focusables = () =>
      Array.from(
        el.querySelectorAll<HTMLElement>(
          'button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),a[href],[tabindex="0"]',
        ),
      ).filter((n) => n.getClientRects().length);
    el.tabIndex = -1;
    const keydown = (e: KeyboardEvent) => {
      if (e.key !== "Tab") return;
      const nodes = focusables();
      if (!nodes.length) {
        e.preventDefault();
        el.focus();
        return;
      }
      const first = nodes[0],
        last = nodes[nodes.length - 1];
      if (
        e.shiftKey &&
        (document.activeElement === first || document.activeElement === el)
      ) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    };
    el.addEventListener("keydown", keydown);
    handlers.set(el, { keydown, previous });
    queueMicrotask(() => {
      (focusables()[0] || el).focus();
    });
  },
  unmounted(el) {
    const state = handlers.get(el);
    if (state) {
      el.removeEventListener("keydown", state.keydown);
      if (state.previous?.isConnected) state.previous.focus();
      handlers.delete(el);
    }
  },
};
