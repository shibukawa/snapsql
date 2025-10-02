export interface RequestOptions extends RequestInit {
  query?: Record<string, string | number | boolean | undefined>;
}

export class ApiError extends Error {
  readonly status: number;
  readonly payload: unknown;

  constructor(message: string, status: number, payload: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

function shouldAttachJSONHeader(body: BodyInit | null | undefined): boolean {
  if (!body) return false;
  if (
    body instanceof FormData ||
    body instanceof Blob ||
    body instanceof ArrayBuffer ||
    ArrayBuffer.isView(body) ||
    body instanceof URLSearchParams
  ) {
    return false;
  }
  if (typeof body === 'string') return true;

  return true;
}

async function parseResponseBody(response: Response): Promise<unknown> {
  if (response.status === 204) {
    return undefined;
  }

  const contentType = response.headers.get('Content-Type') ?? '';

  if (contentType.includes('application/json')) {
    try {
      return await response.json();
    } catch (error) {
      // JSON parse failure should still fall back to text representation.
    }
  }

  const text = await response.text();

  return text.length ? text : undefined;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { query, headers: userHeaders, ...rest } = options;
  const headers = new Headers(userHeaders);

  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json');
  }

  if (!headers.has('Content-Type') && shouldAttachJSONHeader(rest.body)) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(path, { ...rest, headers });
  const payload = await parseResponseBody(response);

  if (!response.ok) {
    const message =
      (payload && typeof payload === 'object' && 'error' in payload && typeof (payload as any).error === 'string'
        ? (payload as any).error
        : `Request failed with status ${response.status}`);

    throw new ApiError(message, response.status, payload);
  }

  return payload as T;
}
