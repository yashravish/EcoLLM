import type { ApiErrorResponse } from '@/types';

export class ApiError extends Error {
  constructor(
    public status: number,
    public message: string,
    public type: string,
    public traceId?: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

const TOKEN_KEY = 'ecollm_token';

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor() {
    this.baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem(TOKEN_KEY);
    }
  }

  setToken(token: string) {
    this.token = token;
    if (typeof window !== 'undefined') {
      localStorage.setItem(TOKEN_KEY, token);
    }
  }

  clearToken() {
    this.token = null;
    if (typeof window !== 'undefined') {
      localStorage.removeItem(TOKEN_KEY);
    }
  }

  getToken(): string | null {
    return this.token;
  }

  async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const merged = new Headers(options.headers);
    if (!merged.has('Content-Type')) merged.set('Content-Type', 'application/json');
    if (this.token) merged.set('Authorization', `Bearer ${this.token}`);
    const headers: Record<string, string> = {};
    merged.forEach((v, k) => { headers[k] = v; });

    const res = await fetch(`${this.baseUrl}${path}`, { ...options, headers });

    if (!res.ok) {
      const errBody = await res.json().catch(() => ({
        message: 'Request failed',
        type: 'unknown_error',
      })) as ApiErrorResponse;
      throw new ApiError(res.status, errBody.message, errBody.type, errBody.trace_id);
    }

    // 204 No Content (and a handful of related codes) have an empty body —
    // calling res.json() on them throws a SyntaxError that react-query
    // surfaces as a failed mutation, so callers see "Something went wrong"
    // even though the request actually succeeded.
    if (res.status === 204 || res.status === 205 || res.headers.get('content-length') === '0') {
      return undefined as T;
    }
    return res.json() as Promise<T>;
  }

  get<T>(path: string, params?: Record<string, string | number | boolean | undefined>): Promise<T> {
    if (params) {
      const filtered: Record<string, string> = Object.fromEntries(
        Object.entries(params)
          .filter((entry): entry is [string, string | number | boolean] => entry[1] !== undefined && entry[1] !== null)
          .map(([k, v]) => [k, String(v)]),
      );
      const qs = new URLSearchParams(filtered).toString();
      if (qs) return this.request<T>(`${path}?${qs}`);
    }
    return this.request<T>(path);
  }

  post<T>(path: string, body: object): Promise<T> {
    return this.request<T>(path, { method: 'POST', body: JSON.stringify(body) });
  }

  patch<T>(path: string, body: object): Promise<T> {
    return this.request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'DELETE' });
  }
}

export const api = new ApiClient();
