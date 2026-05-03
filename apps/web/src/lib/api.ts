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

class ApiClient {
  private baseUrl: string;
  private token: string | null = null;

  constructor() {
    this.baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  }

  setToken(token: string) {
    this.token = token;
  }

  clearToken() {
    this.token = null;
  }

  async request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options.headers as Record<string, string>) || {}),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(`${this.baseUrl}${path}`, { ...options, headers });

    if (!res.ok) {
      const errBody = await res.json().catch(() => ({
        message: 'Request failed',
        type: 'unknown_error',
      })) as ApiErrorResponse;
      throw new ApiError(res.status, errBody.message, errBody.type, errBody.trace_id);
    }

    return res.json() as Promise<T>;
  }

  get<T>(path: string, params?: Record<string, string | number | boolean | undefined>): Promise<T> {
    if (params) {
      const filtered = Object.fromEntries(
        Object.entries(params).filter(([, v]) => v !== undefined && v !== null),
      ) as Record<string, string>;
      const qs = new URLSearchParams(filtered as Record<string, string>).toString();
      if (qs) return this.request<T>(`${path}?${qs}`);
    }
    return this.request<T>(path);
  }

  post<T>(path: string, body: Record<string, unknown>): Promise<T> {
    return this.request<T>(path, { method: 'POST', body: JSON.stringify(body) });
  }

  patch<T>(path: string, body: Record<string, unknown>): Promise<T> {
    return this.request<T>(path, { method: 'PATCH', body: JSON.stringify(body) });
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>(path, { method: 'DELETE' });
  }
}

export const api = new ApiClient();
