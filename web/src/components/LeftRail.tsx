import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import { avatarTone, initials } from "../lib/initials";
import { BrandMark } from "./Brand";
import { LanguageToggle } from "./LanguageToggle";
import {
  ActivityIcon,
  ChevronRightIcon,
  HomeIcon,
  PenIcon,
  UserIcon,
} from "./icons";

interface LeftRailProps {
  tab: "home" | "profile";
  currentUserId: number;
  currentUser: User | null;
  loading: boolean;
  onSelectTab: (tab: "home" | "profile") => void;
  onNewPost: () => void;
  onOpenIdentity: () => void;
  onOpenLive: () => void;
}

export function LeftRail({
  tab,
  currentUserId,
  currentUser,
  loading,
  onSelectTab,
  onNewPost,
  onOpenIdentity,
  onOpenLive,
}: LeftRailProps) {
  const { t } = useI18n();

  return (
    <aside className="rail-left">
      <div className="rail-left-inner">
        <div className="brand">
          <BrandMark size={38} />
          <div className="brand-copy">
            <strong>Wayposts</strong>
            <span>{t("brand.tagline")}</span>
          </div>
        </div>

        <nav className="primary-nav" aria-label={t("nav.primary")}>
          <button
            type="button"
            className="nav-item"
            aria-current={tab === "home" ? "page" : undefined}
            onClick={() => onSelectTab("home")}
          >
            <HomeIcon size={20} />
            <span>{t("nav.home")}</span>
          </button>
          <button
            type="button"
            className="nav-item"
            aria-current={tab === "profile" ? "page" : undefined}
            onClick={() => onSelectTab("profile")}
          >
            <UserIcon size={20} />
            <span>{t("nav.profile")}</span>
          </button>
          <button
            type="button"
            className="nav-item nav-live"
            aria-haspopup="dialog"
            onClick={onOpenLive}
          >
            <ActivityIcon size={20} />
            <span>{t("nav.live")}</span>
          </button>
        </nav>

        <button type="button" className="btn btn-accent compose-btn" onClick={onNewPost}>
          <PenIcon size={17} />
          {t("nav.newPost")}
        </button>

        <div className="rail-spacer" />

        <button
          type="button"
          className="identity-block"
          onClick={onOpenIdentity}
          aria-haspopup="dialog"
          aria-label={t("identity.switch")}
          title={t("identity.switch")}
        >
          <span className={`avatar tone-${avatarTone(currentUserId)}`} aria-hidden="true">
            {initials(currentUser, currentUserId)}
          </span>
          <span className="identity-block-copy">
            <strong>
              {loading
                ? t("identity.loading")
                : currentUser?.display_name ?? `User ${currentUserId}`}
            </strong>
            <span>
              @{currentUser?.handle ?? currentUserId}
            </span>
          </span>
          <ChevronRightIcon size={16} />
        </button>

        <div className="rail-left-foot">
          <LanguageToggle />
          <span className="rail-version mono">{t("nav.version")}</span>
        </div>
      </div>
    </aside>
  );
}
