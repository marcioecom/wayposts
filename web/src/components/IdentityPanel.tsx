import { useEffect, useRef } from "react";
import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import { avatarTone, initials } from "../lib/initials";
import { ArrowUpRightIcon } from "./icons";

interface IdentityPanelProps {
  currentUserId: number;
  user: User | null;
  loading: boolean;
  input: string;
  onInput: (value: string) => void;
  onSwitch: (id: number) => void;
  onOpenProfile: (id: number) => void;
  onCancel?: () => void;
  autoFocusInput?: boolean;
}

const quickUsers = [1, 5, 10, 11, 12];

export function IdentityPanel({
  currentUserId,
  user,
  loading,
  input,
  onInput,
  onSwitch,
  onOpenProfile,
  onCancel,
  autoFocusInput = false,
}: IdentityPanelProps) {
  const { t } = useI18n();
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!autoFocusInput) return;
    const timer = window.setTimeout(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    }, 40);
    return () => window.clearTimeout(timer);
  }, [autoFocusInput]);

  return (
    <div className="identity-panel">
      <div className="identity-summary">
        <span className={`avatar avatar-xl tone-${avatarTone(currentUserId)}`} aria-hidden="true">
          {initials(user, currentUserId)}
        </span>
        <div className="identity-copy">
          <strong>
            {loading
              ? t("identity.loading")
              : user?.display_name ?? t("identity.notFound")}
          </strong>
          <span className="muted">
            {user ? `@${user.handle}` : `user ${currentUserId}`}
          </span>
          <span className="identity-count mono">
            {t("identity.followers", { count: user?.follower_count ?? 0 })}
          </span>
        </div>
        <button
          type="button"
          className="rail-link identity-profile-link"
          onClick={() => onOpenProfile(user?.id ?? currentUserId)}
        >
          {t("identity.openProfile")}
          <ArrowUpRightIcon size={13} />
        </button>
      </div>

      <form
        className="identity-form"
        onSubmit={(event) => {
          event.preventDefault();
          onSwitch(Number(input));
        }}
      >
        <label className="field" htmlFor="identity-input">
          <span>{t("identity.label")}</span>
          <input
            id="identity-input"
            ref={inputRef}
            className="input"
            inputMode="numeric"
            min={1}
            value={input}
            onChange={(event) => onInput(event.target.value)}
          />
        </label>
        <div className="identity-form-actions">
          <button type="submit" className="btn btn-accent">
            {t("identity.use")}
          </button>
          {onCancel ? (
            <button type="button" className="btn" onClick={onCancel}>
              {t("identity.cancel")}
            </button>
          ) : null}
        </div>
      </form>

      <div className="quick-identities">
        <span className="field-label">{t("identity.quick")}</span>
        <div className="chip-row" aria-label={t("aria.quickIdentities")}>
          {quickUsers.map((id) => (
            <button
              key={id}
              type="button"
              className={`chip mono ${id === currentUserId ? "is-active" : ""}`}
              aria-pressed={id === currentUserId}
              onClick={() => onSwitch(id)}
            >
              {id === currentUserId ? `${id} · ${t("identity.current")}` : id}
            </button>
          ))}
        </div>
      </div>

      <p className="panel-hint">{t("identity.dialogCopy")}</p>
      <p className="panel-hint">{t("identity.hint")}</p>
    </div>
  );
}
