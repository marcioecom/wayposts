import { useI18n } from "../lib/i18n";
import type { User } from "../types";
import { Dialog } from "./Dialog";
import { IdentityPanel } from "./IdentityPanel";
import { LanguageToggle } from "./LanguageToggle";
import { LivePanels } from "./RightRail";
import type { LivePanelsProps } from "./RightRail";

interface LiveSheetProps extends LivePanelsProps {
  open: boolean;
  onClose: () => void;
}

export function LiveSheet({ open, onClose, ...panels }: LiveSheetProps) {
  const { t } = useI18n();

  return (
    <Dialog
      open={open}
      onClose={onClose}
      labelledBy="live-sheet-title"
      variant="sheet"
    >
      <div className="sheet-head">
        <h2 id="live-sheet-title">{t("sheet.liveTitle")}</h2>
        <button
          type="button"
          className="sheet-close"
          onClick={onClose}
          aria-label={t("sheet.close")}
        >
          ×
        </button>
      </div>
      <div className="sheet-body">
        <LivePanels {...panels} />
      </div>
    </Dialog>
  );
}

interface YouSheetProps {
  open: boolean;
  onClose: () => void;
  currentUserId: number;
  currentUser: User | null;
  currentUserLoading: boolean;
  userIdInput: string;
  onInput: (value: string) => void;
  onSwitch: (id: number) => void;
  onOpenProfile: (id: number) => void;
}

export function YouSheet({
  open,
  onClose,
  currentUserId,
  currentUser,
  currentUserLoading,
  userIdInput,
  onInput,
  onSwitch,
  onOpenProfile,
}: YouSheetProps) {
  const { t } = useI18n();

  return (
    <Dialog
      open={open}
      onClose={onClose}
      labelledBy="you-sheet-title"
      variant="sheet"
    >
      <div className="sheet-head">
        <h2 id="you-sheet-title">{t("sheet.youTitle")}</h2>
        <button
          type="button"
          className="sheet-close"
          onClick={onClose}
          aria-label={t("sheet.close")}
        >
          ×
        </button>
      </div>
      <div className="sheet-body">
        <IdentityPanel
          currentUserId={currentUserId}
          user={currentUser}
          loading={currentUserLoading}
          input={userIdInput}
          onInput={onInput}
          onSwitch={onSwitch}
          onOpenProfile={onOpenProfile}
        />
        <div className="sheet-language">
          <span className="rail-label">{t("aria.language")}</span>
          <LanguageToggle tone="dark" />
        </div>
      </div>
    </Dialog>
  );
}

interface IdentityDialogProps {
  open: boolean;
  onClose: () => void;
  currentUserId: number;
  currentUser: User | null;
  currentUserLoading: boolean;
  userIdInput: string;
  onInput: (value: string) => void;
  onSwitch: (id: number) => void;
  onOpenProfile: (id: number) => void;
}

export function IdentityDialog({
  open,
  onClose,
  currentUserId,
  currentUser,
  currentUserLoading,
  userIdInput,
  onInput,
  onSwitch,
  onOpenProfile,
}: IdentityDialogProps) {
  const { t } = useI18n();

  return (
    <Dialog
      open={open}
      onClose={onClose}
      labelledBy="identity-dialog-title"
      variant="center"
    >
      <div className="dialog-panel">
        <div className="dialog-head">
          <h2 id="identity-dialog-title">{t("identity.dialogTitle")}</h2>
          <button
            type="button"
            className="sheet-close"
            onClick={onClose}
            aria-label={t("sheet.close")}
          >
            ×
          </button>
        </div>
        <IdentityPanel
          currentUserId={currentUserId}
          user={currentUser}
          loading={currentUserLoading}
          input={userIdInput}
          onInput={onInput}
          onSwitch={onSwitch}
          onOpenProfile={onOpenProfile}
          onCancel={onClose}
          autoFocusInput
        />
      </div>
    </Dialog>
  );
}
