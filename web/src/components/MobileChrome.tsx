import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import type { TrafficState } from "../lib/metrics";
import { avatarTone, initials } from "../lib/initials";
import { BrandMark } from "./Brand";
import { ActivityIcon, HomeIcon, RefreshIcon, UserIcon } from "./icons";
import { LivePill } from "./LivePill";

interface TopbarProps {
  currentUserId: number;
  currentUser: User | null;
  loading: boolean;
  traffic: TrafficState;
  onOpenLive: () => void;
  onOpenYou: () => void;
  onRefresh: () => void;
}

export function Topbar({
  currentUserId,
  currentUser,
  loading,
  traffic,
  onOpenLive,
  onOpenYou,
  onRefresh,
}: TopbarProps) {
  const { t } = useI18n();

  return (
    <header className="topbar">
      <div className="topbar-brand">
        <BrandMark size={30} />
        <strong>Wayposts</strong>
        <span className="topbar-lab">{t("brand.lab")}</span>
      </div>

      <div className="topbar-actions">
        <button
          type="button"
          className={`icon-btn topbar-refresh ${loading ? "is-spinning" : ""}`}
          onClick={onRefresh}
          aria-label={t("feed.refresh")}
          title={t("feed.refresh")}
        >
          <RefreshIcon size={17} />
        </button>
        <LivePill traffic={traffic} onOpenLive={onOpenLive} />

        <button
          type="button"
          className="avatar-button"
          onClick={onOpenYou}
          aria-haspopup="dialog"
          aria-busy={loading}
          aria-label={t("nav.openYou")}
          title={currentUser ? `@${currentUser.handle}` : `user ${currentUserId}`}
        >
          <span className={`avatar avatar-sm tone-${avatarTone(currentUserId)}`} aria-hidden="true">
            {initials(currentUser, currentUserId)}
          </span>
        </button>
      </div>
    </header>
  );
}

interface BottomNavProps {
  tab: "home" | "profile";
  currentUserId: number;
  currentUser: User | null;
  onSelectTab: (tab: "home" | "profile") => void;
  onOpenLive: () => void;
  onOpenYou: () => void;
}

export function BottomNav({
  tab,
  currentUserId,
  currentUser,
  onSelectTab,
  onOpenLive,
  onOpenYou,
}: BottomNavProps) {
  const { t } = useI18n();

  return (
    <nav className="bottom-nav" aria-label={t("nav.primary")}>
      <button
        type="button"
        className="bottom-nav-item"
        aria-current={tab === "home" ? "page" : undefined}
        onClick={() => onSelectTab("home")}
      >
        <HomeIcon size={22} />
        <span>{t("nav.home")}</span>
      </button>
      <button
        type="button"
        className="bottom-nav-item"
        aria-current={tab === "profile" ? "page" : undefined}
        onClick={() => onSelectTab("profile")}
      >
        <UserIcon size={22} />
        <span>{t("nav.profile")}</span>
      </button>
      <button
        type="button"
        className="bottom-nav-item"
        aria-haspopup="dialog"
        onClick={onOpenLive}
      >
        <ActivityIcon size={22} />
        <span>{t("nav.live")}</span>
      </button>
      <button
        type="button"
        className="bottom-nav-item"
        aria-haspopup="dialog"
        onClick={onOpenYou}
      >
        <span className={`avatar avatar-nav tone-${avatarTone(currentUserId)}`} aria-hidden="true">
          {initials(currentUser, currentUserId)}
        </span>
        <span>{t("nav.you")}</span>
      </button>
    </nav>
  );
}
