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

export type StreamEvent = { event: string; data: unknown };

export async function stream(fetchFn: FetchFn, path: string, body: unknown, onEvent: (e: StreamEvent) => void, cancel?: AbortSignal): Promise<void> {
  let res: Response;
  try {
    res = await fetchFn(path, { method: "POST", headers: { Accept: "text/event-stream", "Content-Type": "application/json" }, credentials: "same-origin", signal: cancel ?? null, body: JSON.stringify(body) });
  } catch {
    if (cancel?.aborted) throw new ApiError(0, "cancelled", "request cancelled");
    throw new ApiError(0, "network", "network unreachable");
  }
  if (!res.ok || !res.body) {
    const parsed = (await res.json().catch(() => null)) as { error?: { code?: string; message?: string } } | null;
    throw new ApiError(res.status, parsed?.error?.code ?? "http_error", parsed?.error?.message ?? `HTTP ${res.status}`);
  }
  const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
  let buf = "";
  try {
    for (;;) {
      const { value, done } = await reader.read();
      if (done) break;
      buf += value;
      let cut: number;
      while ((cut = buf.indexOf("\n\n")) >= 0) {
        const chunk = buf.slice(0, cut);
        buf = buf.slice(cut + 2);
        let event = "message";
        const data: string[] = [];
        for (const line of chunk.split("\n")) {
          if (line.startsWith("event:")) event = line.slice(6).trim();
          else if (line.startsWith("data:")) data.push(line.slice(5).replace(/^ /, ""));
        }
        let parsed: unknown = data.join("\n");
        try {
          parsed = JSON.parse(parsed as string);
        } catch {
          continue;
        }
        if (event === "error") {
          const e = parsed as { code?: string; message?: string };
          throw new ApiError(502, e.code ?? "ai_failed", e.message ?? "the AI request failed");
        }
        onEvent({ event, data: parsed });
      }
    }
  } catch (e) {
    if (e instanceof ApiError) throw e;
    if (cancel?.aborted) throw new ApiError(0, "cancelled", "request cancelled");
    throw new ApiError(0, "network", "the connection broke");
  }
}

export async function request<T>(fetchFn: FetchFn, method: Method, path: string, body?: unknown, timeoutMs = 15000, cancel?: AbortSignal): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  if (cancel?.aborted) controller.abort();
  cancel?.addEventListener("abort", () => controller.abort(), { once: true });
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
    if (cancel?.aborted) throw new ApiError(0, "cancelled", "request cancelled");
    const aborted = e instanceof DOMException && e.name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "request timed out" : "network unreachable");
  } finally {
    clearTimeout(timer);
  }
}
