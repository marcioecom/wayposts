import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import { avatarTone, initials } from "../lib/initials";
import { RefreshIcon, UserIcon } from "./icons";

export interface ProfileError {
  message: string;
  notFound: boolean;
}

interface ProfileHeaderProps {
  userId: number;
  user: User | null;
  loading: boolean;
  error: ProfileError | null;
  currentUserId: number;
  currentUser: User | null;
  onFollow: (followeeId: number, action: "follow" | "unfollow") => void;
  onOpenIdentity: () => void;
  onRefresh: () => void;
}

export function ProfileHeader({
  userId,
  user,
  loading,
  error,
  currentUserId,
  currentUser,
  onFollow,
  onOpenIdentity,
  onRefresh,
}: ProfileHeaderProps) {
  const { t } = useI18n();

  if (!user) {
    if (!loading) {
      if (error && !error.notFound) {
        return (
          <section className="profile-header">
            <div className="empty empty-inline">
              <strong>{t("profile.errorTitle")}</strong>
              <p>
                {error.message.startsWith("Cannot reach the API")
                  ? t("error.network")
                  : error.message}
              </p>
              <button type="button" className="btn" onClick={onRefresh}>
                {t("error.retry")}
              </button>
            </div>
          </section>
        );
      }
      return (
        <section className="profile-header">
          <div className="empty empty-inline">
            <span className="empty-icon" aria-hidden="true">
              <UserIcon size={24} />
            </span>
            <strong>{t("profile.notFound", { id: userId })}</strong>
            <p>{t("profile.notFoundCopy")}</p>
            <button type="button" className="btn" onClick={onRefresh}>
              {t("error.retry")}
            </button>
          </div>
        </section>
      );
    }
    return (
      <section className="profile-header" aria-busy="true">
        <div className="profile-top">
          <span className="skeleton-avatar xl" />
          <div className="grow">
            <span className="skeleton-line medium" />
            <span className="skeleton-line short" />
            <span className="skeleton-line tiny" />
          </div>
        </div>
      </section>
    );
  }

  const self = user.id === currentUserId;
  const viewerHandle = currentUser?.handle ?? String(currentUserId);

  return (
    <section className="profile-header">
      <div className="profile-top">
        <span className={`avatar avatar-xl tone-${avatarTone(user.id)}`} aria-hidden="true">
          {initials(user, user.id)}
        </span>
        <div className="profile-identity">
          <div className="profile-name-row">
            <h2>{user.display_name}</h2>
            <span className="profile-id mono">user {user.id}</span>
          </div>
          <p className="profile-handle">@{user.handle}</p>
          <p className="profile-followers">
            {t("identity.followers", { count: user.follower_count })}
          </p>
        </div>
        <button
          type="button"
          className="icon-btn"
          onClick={onRefresh}
          title={t("profile.refresh")}
          aria-label={t("profile.refresh")}
        >
          <RefreshIcon size={18} />
        </button>
      </div>

      <div className="profile-actions">
        {self ? (
          <button type="button" className="btn btn-accent" onClick={onOpenIdentity}>
            {t("identity.switch")}
          </button>
        ) : (
          <>
            <button
              type="button"
              className="btn btn-accent"
              onClick={() => onFollow(user.id, "follow")}
            >
              {t("profile.follow")}
            </button>
            <button
              type="button"
              className="btn"
              onClick={() => onFollow(user.id, "unfollow")}
            >
              {t("profile.unfollow")}
            </button>
          </>
        )}
      </div>

      <p className="profile-hint">
        {self ? `${t("profile.current")} ${t("profile.currentHint")}` : t("profile.hint")}
      </p>

      <p className="profile-meta mono">
        {t("profile.viewing", { handle: viewerHandle })} · profile {userId}
      </p>
    </section>
  );
}
