const BASE_URL = import.meta.env.VITE_API_URL ?? "";

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

interface Envelope<T> {
  data?: T;
  error?: { code: string; message: string };
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });

  let body: Envelope<T> | undefined;
  try {
    body = (await res.json()) as Envelope<T>;
  } catch {
    body = undefined;
  }

  if (!res.ok) {
    throw new ApiError(
      body?.error?.message ?? `Request failed with status ${res.status}`,
      res.status,
      body?.error?.code ?? "unknown",
    );
  }
  return body?.data as T;
}

export interface Subscriber {
  id: string;
  email: string;
  name: string;
  createdAt: string;
}

export interface JoinWaitlistInput {
  email: string;
  name: string;
}

export function joinWaitlist(input: JoinWaitlistInput): Promise<Subscriber> {
  return request<Subscriber>("/api/v1/waitlist", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export interface Health {
  status: string;
}

export function getHealth(): Promise<Health> {
  return request<Health>("/api/v1/healthz");
}

export type UserRole = "customer" | "viewer" | "staff" | "admin";

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  /** Capability strings (e.g. "orders:write") granted by the role. */
  permissions: string[];
  /** True for bootstrap super-admins (ADMIN_EMAILS) — role can't be changed. */
  isSuperAdmin: boolean;
}

export interface RequestLoginLinkInput {
  email: string;
  name?: string;
}

export function requestLoginLink(input: RequestLoginLinkInput): Promise<{ status: string }> {
  return request<{ status: string }>("/api/v1/auth/request-link", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export function verifyLogin(token: string): Promise<{ user: User }> {
  return request<{ user: User }>("/api/v1/auth/verify", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export async function getMe(): Promise<User | null> {
  try {
    const { user } = await request<{ user: User }>("/api/v1/auth/me");
    return user;
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) return null;
    throw error;
  }
}

export function logout(): Promise<void> {
  return request<void>("/api/v1/auth/logout", { method: "POST" });
}
