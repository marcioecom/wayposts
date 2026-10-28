export interface AuthorSummary {
  id: number;
  handle: string;
  display_name: string;
}

export interface Post {
  id: number;
  author_id: number;
  body: string;
  media: string[];
  created_at: string;
  deleted: boolean;
  like_count: number;
  viewer_liked: boolean;
  // Preenchido pela timeline. Pode faltar se o autor não puder ser resolvido.
  author?: AuthorSummary;
}

export interface User {
  id: number;
  handle: string;
  display_name: string;
  follower_count: number;
}

export interface TimelinePage {
  items: Post[];
  next_cursor: number;
}

export type WsStatus = "connecting" | "open" | "closed";

export type ToastKind = "success" | "error";

export interface ToastMessage {
  id: number;
  kind: ToastKind;
  message: string;
}
