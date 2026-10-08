export type ToastType = "success" | "info" | "warning" | "error" | "pending";

export interface ToastItem {
  id: number;
  key?: string;
  type: ToastType;
  message: string;
  pending: boolean;
  duration?: number;
  createdAt: number;
}

export interface ToastInput {
  type?: ToastType;
  message?: string;
  key?: string;
  pending?: boolean;
  duration?: number;
}

export interface Toasts {
  list: ToastItem[];
  push(input: ToastInput): number;
  update(id: number, patch: Partial<ToastInput>): void;
  dismiss(id: number): void;
  clearAll(): void;
}

export function createToasts(options?: { max?: number }): Toasts;
