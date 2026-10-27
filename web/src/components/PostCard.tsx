import { useState } from "react";
import type { Post, User } from "../types";
import { useI18n } from "../lib/i18n";
import { avatarTone, initials } from "../lib/initials";
import { fullTime, relativeTime } from "../lib/time";
import { HeartIcon, TrashIcon, UserIcon } from "./icons";

interface PostCardProps {
  post: Post;
  currentUserId: number;
  author: User | undefined;
  highlighted?: boolean;
  onLike: (post: Post) => Promise<void>;
  onDelete: (post: Post) => Promise<void>;
  onOpenProfile: (id: number) => void;
}

function httpMedia(media: string[]): string[] {
  return media.filter((value) => /^https?:\/\//i.test(value));
}

export function PostCard({
  post,
  currentUserId,
  author,
  highlighted = false,
  onLike,
  onDelete,
  onOpenProfile,
}: PostCardProps) {
  const { t, lang } = useI18n();
  const [liking, setLiking] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const own = post.author_id === currentUserId;
  const media = httpMedia(post.media);
  const handle = author?.handle ?? String(post.author_id);
  const name = author?.display_name ?? `User ${post.author_id}`;

  async function like() {
    setLiking(true);
    try {
      await onLike(post);
    } finally {
      setLiking(false);
    }
  }

  async function remove() {
    setDeleting(true);
    try {
      await onDelete(post);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <article className={`post ${highlighted ? "is-highlighted" : ""}`}>
      <header className="post-head">
        <button
          type="button"
          className="post-author"
          onClick={() => onOpenProfile(post.author_id)}
          title={t("post.openProfile", { id: post.author_id })}
        >
          <span className={`avatar tone-${avatarTone(post.author_id)}`} aria-hidden="true">
            {initials(author, post.author_id)}
          </span>
          <span className="post-author-copy">
            <strong>{name}</strong>
            <span className="post-sub">
              @{handle}
              <span aria-hidden="true"> · </span>
              <time
                dateTime={post.created_at}
                title={fullTime(post.created_at, lang)}
              >
                {relativeTime(post.created_at, lang)}
              </time>
            </span>
          </span>
        </button>
      </header>

      <p className="post-body">{post.body}</p>

      {media.length > 0 ? (
        <div className={`post-media ${media.length > 1 ? "is-grid" : ""}`}>
          {media.map((src) => (
            <img
              key={src}
              src={src}
              alt={t("post.attachment")}
              loading="lazy"
            />
          ))}
        </div>
      ) : null}

      <footer className="post-actions">
        <button
          type="button"
          className={`action ${post.viewer_liked ? "is-liked" : ""}`}
          aria-pressed={post.viewer_liked}
          aria-label={post.viewer_liked ? t("post.unlike") : t("post.like")}
          disabled={liking}
          onClick={() => void like()}
          title={post.viewer_liked ? t("post.unlike") : t("post.like")}
        >
          <HeartIcon size={17} filled={post.viewer_liked} />
          <span className="action-count">{post.like_count}</span>
        </button>

        <button
          type="button"
          className="action"
          onClick={() => onOpenProfile(post.author_id)}
          aria-label={t("post.profile")}
        >
          <UserIcon size={16} />
          <span className="action-label">{t("post.profile")}</span>
        </button>

        {own ? (
          <button
            type="button"
            className="action action-danger"
            disabled={deleting}
            onClick={() => void remove()}
            aria-label={t("post.delete")}
          >
            <TrashIcon size={16} />
            <span className="action-label">
              {deleting ? t("post.deleting") : t("post.delete")}
            </span>
          </button>
        ) : null}

        <span className="post-id mono" title={t("post.id", { id: post.id })}>
          #{post.id}
        </span>
      </footer>
    </article>
  );
}
