export type AsyncState = "idle" | "pending" | "success" | "error";

export interface ActionResult<T> {
  ok: boolean;
  value?: T;
  error?: string;
  duplicate?: boolean;
}

export interface AsyncActions {
  states: Record<string, AsyncState>;
  errors: Record<string, string>;
  pending(key: string): boolean;
  error(key: string): string;
  status(key: string): AsyncState;
  run<T>(key: string, fn: () => Promise<T>): Promise<ActionResult<T>>;
  reset(key: string): void;
}

export function createAsyncActions(): AsyncActions;
