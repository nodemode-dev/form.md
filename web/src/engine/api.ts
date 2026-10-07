export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export type FetchFn = typeof fetch;

export async function getJSON<T>(fetchFn: FetchFn, path: string, timeoutMs = 8000): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetchFn(path, {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
      signal: controller.signal,
    });
    const body: unknown = await res.json().catch(() => null);
    if (!res.ok) {
      const err = (body as { error?: { code?: string; message?: string } } | null)?.error;
      throw new ApiError(res.status, err?.code ?? "http_error", err?.message ?? `HTTP ${res.status}`);
    }
    return body as T;
  } catch (e) {
    if (e instanceof ApiError) throw e;
    const aborted = e instanceof DOMException && e.name === "AbortError";
    throw new ApiError(0, aborted ? "timeout" : "network", aborted ? "request timed out" : "network unreachable");
  } finally {
    clearTimeout(timer);
  }
}
