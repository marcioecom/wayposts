import type { Post, TimelinePage, User } from "./types";

const rawBase = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

export const apiBase = rawBase.replace(/\/+$/, "");

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(
  path: string,
  userId: number,
  init: RequestInit = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  headers.set("Authorization", `Bearer ${userId}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, { ...init, headers });
  } catch {
    throw new ApiError(
      0,
      `Cannot reach the API at ${apiBase}. Check the Cloudflare tunnel and CORS_ALLOWED_ORIGINS.`,
    );
  }

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const payload = (await response.json()) as { error?: string };
      if (payload.error) message = payload.error;
    } catch {
      // The body is not JSON. Keep the status text.
    }
    throw new ApiError(response.status, message);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

function query(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== 0) {
      search.set(key, String(value));
    }
  }
  const encoded = search.toString();
  return encoded ? `?${encoded}` : "";
}

export const api = {
  base: apiBase,

  getUser(viewerId: number, id: number) {
    return request<User>(`/v1/users/${id}`, viewerId);
  },

  home(userId: number, before = 0, limit = 20) {
    return request<TimelinePage>(
      `/v1/timeline/home${query({ before, limit })}`,
      userId,
    );
  },

  profile(viewerId: number, authorId: number, before = 0, limit = 20) {
    return request<TimelinePage>(
      `/v1/timeline/profile/${authorId}${query({ before, limit })}`,
      viewerId,
    );
  },

  createPost(userId: number, body: string) {
    return request<Post>("/v1/posts", userId, {
      method: "POST",
      body: JSON.stringify({ body }),
    });
  },

  deletePost(userId: number, postId: number) {
    return request<void>(`/v1/posts/${postId}`, userId, { method: "DELETE" });
  },

  like(userId: number, postId: number) {
    return request<{ post_id: number; like_count: number; liked: boolean }>(
      `/v1/posts/${postId}/likes`,
      userId,
      { method: "POST" },
    );
  },

  unlike(userId: number, postId: number) {
    return request<{ post_id: number; like_count: number; liked: boolean }>(
      `/v1/posts/${postId}/likes`,
      userId,
      { method: "DELETE" },
    );
  },

  follow(userId: number, followeeId: number) {
    return request<void>("/v1/follow", userId, {
      method: "POST",
      body: JSON.stringify({ followee_id: followeeId }),
    });
  },

  unfollow(userId: number, followeeId: number) {
    return request<void>(`/v1/follow/${followeeId}`, userId, {
      method: "DELETE",
    });
  },
};

export function nudgeUrl(userId: number, lastSeen: number): string {
  const url = new URL(apiBase);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  url.pathname = "/v1/ws";
  url.search = "";
  url.searchParams.set("token", String(userId));
  url.searchParams.set("last_seen", String(lastSeen));
  return url.toString();
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message;
  if (error instanceof Error) return error.message;
  return "Unexpected error.";
}
