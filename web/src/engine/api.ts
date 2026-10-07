export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly data: Record<string, unknown> = {},
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

export type FetchFn = typeof fetch;

export async function getJSON<T>(fetchFn: FetchFn, path: string, timeoutMs = 8000): Promise<T> {
  return request<T>(fetchFn, "GET", path, undefined, timeoutMs);
}

export async function upload<T>(fetchFn: FetchFn, path: string, form: FormData, timeoutMs = 120000): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetchFn(path, { method: "POST", headers: { Accept: "application/json" }, credentials: "same-origin", signal: controller.signal, body: form });
    const parsed: unknown = await res.json().catch(() => null);
    if (!res.ok) {
      const err = (parsed as { error?: { code?: string; message?: string; data?: Record<string, unknown> } } | null)?.error;
      throw new ApiError(res.status, err?.code ?? "http_error", err?.message ?? `HTTP ${res.status}`, err?.data ?? {});
    }
    return parsed as T;
  } catch (e) {
    if (e instanceof ApiError) throw e;
    const aborted = e instanceof DOMException && e.name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "request timed out" : "network unreachable");
  } finally {
    clearTimeout(timer);
  }
}

export async function request<T>(fetchFn: FetchFn, method: Method, path: string, body?: unknown, timeoutMs = 15000): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const headers: Record<string, string> = { Accept: "application/json" };
    if (body !== undefined) headers["Content-Type"] = "application/json";
    const res = await fetchFn(path, {
      method,
      headers,
      credentials: "same-origin",
      signal: controller.signal,
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    });
    if (res.status === 204) return undefined as T;
    const parsed: unknown = await res.json().catch(() => null);
    if (!res.ok) {
      const err = (parsed as { error?: { code?: string; message?: string; data?: Record<string, unknown> } } | null)?.error;
      throw new ApiError(res.status, err?.code ?? "http_error", err?.message ?? `HTTP ${res.status}`, err?.data ?? {});
    }
    return parsed as T;
  } catch (e) {
    if (e instanceof ApiError) throw e;
    const aborted = e instanceof DOMException && e.name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "request timed out" : "network unreachable");
  } finally {
    clearTimeout(timer);
  }
}
