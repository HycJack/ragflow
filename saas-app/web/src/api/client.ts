// Minimal fetch wrapper. In a full RAGFlow-style app this is the
// `utils/next-request` layer; here we keep a small dependency-free client.

export interface ApiEnvelope<T = unknown> {
  code: number;
  message: string;
  data?: T;
}

// Session token is stored in localStorage for this demo slice.
const TOKEN_KEY = 'saas_token';

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string | null): void {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export class ApiError extends Error {
  status: number;
  code: number;

  constructor(message: string, status: number, code: number) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let res: Response;
  try {
    res = await fetch(`/api/v1${path}`, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError('Network error — is the backend running?', 0, 0);
  }

  const envelope = (await res.json().catch(() => ({}))) as ApiEnvelope<T>;
  if (!res.ok || (envelope.code !== undefined && envelope.code !== 0)) {
    throw new ApiError(envelope.message || res.statusText, res.status, envelope.code ?? res.status);
  }
  return envelope.data as T;
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  patch: <T>(path: string, body?: unknown) => request<T>('PATCH', path, body),
  delete: <T>(path: string) => request<T>('DELETE', path),
};