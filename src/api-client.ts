import type { paths } from "./api.generated";

export type APIPath = keyof paths;

export async function requestJSON<T>(
  path: string,
  init: RequestInit | undefined,
  csrfToken: string | undefined,
  onUnauthorized: () => void,
): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !(init.body instanceof FormData))
    headers.set("Content-Type", "application/json");
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken);
  const response = await fetch(path, { ...init, headers });
  const text = await response.text();
  let body: any = {};
  try { body = JSON.parse(text); } catch { body = { error: text }; }
  if (response.status === 401 && !path.startsWith("/api/auth/")) onUnauthorized();
  if (!response.ok) throw Error(body.error || body.code || "请求失败");
  return body as T;
}
