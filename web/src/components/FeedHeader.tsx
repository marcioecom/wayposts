import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import type { TrafficState } from "../lib/metrics";
import { LivePill } from "./LivePill";
import { RefreshIcon } from "./icons";

interface FeedHeaderProps {
  tab: "home" | "profile";
  profileUser: User | null;
  loading: boolean;
  traffic: TrafficState;
  onSelectTab: (tab: "home" | "profile") => void;
  onRefresh: () => void;
  onOpenLive: () => void;
}

export function FeedHeader({
  tab,
  profileUser,
  loading,
  traffic,
  onSelectTab,
  onRefresh,
  onOpenLive,
}: FeedHeaderProps) {
  const { t } = useI18n();

  return (
    <header className="feed-header">
      <h1 className="sr-only feed-heading-mobile">
        {tab === "home" ? t("tabs.home") : t("tabs.profile")}
      </h1>
      <div className="feed-head-row">
        <div className="feed-title">
          <h1>{tab === "home" ? t("tabs.home") : t("tabs.profile")}</h1>
          <span className="feed-title-sub">
            {tab === "home"
              ? t("tabs.homeSub")
              : profileUser
                ? `@${profileUser.handle}`
                : t("tabs.profileSub")}
          </span>
        </div>

        <div className="feed-head-actions">
          <LivePill
            traffic={traffic}
            onOpenLive={onOpenLive}
            className="feed-live"
          />
          <button
            type="button"
            className={`icon-btn ${loading ? "is-spinning" : ""}`}
            onClick={onRefresh}
            aria-label={t("feed.refresh")}
            title={t("feed.refresh")}
          >
            <RefreshIcon size={18} />
          </button>
        </div>
      </div>

      <nav className="feed-tabs" aria-label={t("aria.timeline")}>
        <button
          type="button"
          className="feed-tab"
          aria-current={tab === "home" ? "page" : undefined}
          onClick={() => onSelectTab("home")}
        >
          {t("tabs.home")}
        </button>
        <button
          type="button"
          className="feed-tab"
          aria-current={tab === "profile" ? "page" : undefined}
          onClick={() => onSelectTab("profile")}
        >
          {t("tabs.profile")}
          {profileUser ? (
            <span className="feed-tab-meta mono">@{profileUser.handle}</span>
          ) : null}
        </button>
      </nav>
    </header>
  );
}
