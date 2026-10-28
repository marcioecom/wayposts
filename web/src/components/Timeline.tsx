import type { Post } from "../types";
import { useI18n } from "../lib/i18n";
import { InboxIcon } from "./icons";
import { PostCard } from "./PostCard";

interface TimelineProps {
  tab: "home" | "profile";
  items: Post[];
  currentUserId: number;
  loading: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  highlightedPostId: number | null;
  onLoadMore: () => void;
  onLike: (post: Post) => Promise<void>;
  onDelete: (post: Post) => Promise<void>;
  onOpenProfile: (id: number) => void;
  onEmptyAction: () => void;
}

function Skeleton() {
  return (
    <div className="post skeleton" aria-hidden="true">
      <div className="skeleton-row">
        <span className="skeleton-avatar" />
        <div className="grow">
          <span className="skeleton-line short" />
          <span className="skeleton-line tiny" />
        </div>
      </div>
      <span className="skeleton-line" />
      <span className="skeleton-line medium" />
    </div>
  );
}

export function Timeline({
  tab,
  items,
  currentUserId,
  loading,
  loadingMore,
  hasMore,
  highlightedPostId,
  onLoadMore,
  onLike,
  onDelete,
  onOpenProfile,
  onEmptyAction,
}: TimelineProps) {
  const { t } = useI18n();

  if (loading && items.length === 0) {
    return (
      <div className="timeline" aria-busy="true" aria-label={t("empty.loading")}>
        <Skeleton />
        <Skeleton />
        <Skeleton />
      </div>
    );
  }

  if (!loading && items.length === 0) {
    return (
      <div className="empty">
        <span className="empty-icon" aria-hidden="true">
          <InboxIcon size={26} />
        </span>
        <strong>
          {tab === "home" ? t("empty.homeTitle") : t("empty.profileTitle")}
        </strong>
        <p>
          {tab === "home" ? t("empty.homeCopy") : t("empty.profileCopy")}
        </p>
        <button type="button" className="btn" onClick={onEmptyAction}>
          {tab === "home" ? t("empty.homeCta") : t("empty.profileCta")}
        </button>
      </div>
    );
  }

  return (
    <div className="timeline">
      {items.map((post) => (
        <PostCard
          key={post.id}
          post={post}
          currentUserId={currentUserId}
          highlighted={post.id === highlightedPostId}
          onLike={onLike}
          onDelete={onDelete}
          onOpenProfile={onOpenProfile}
        />
      ))}

      {hasMore ? (
        <div className="timeline-more">
          <button
            type="button"
            className="btn btn-block"
            disabled={loadingMore}
            onClick={onLoadMore}
          >
            {loadingMore ? t("empty.loadingMore") : t("empty.loadMore")}
          </button>
        </div>
      ) : (
        <p className="timeline-end mono">{t("empty.end", { count: items.length })}</p>
      )}
    </div>
  );
}
