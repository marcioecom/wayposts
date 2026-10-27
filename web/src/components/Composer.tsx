import { useState } from "react";
import type { RefObject } from "react";
import type { User } from "../types";
import { useI18n } from "../lib/i18n";
import { avatarTone, initials } from "../lib/initials";
import { SendIcon } from "./icons";

interface ComposerProps {
  currentUserId: number;
  currentUser: User | null;
  creating: boolean;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  onCreate: (body: string) => Promise<boolean>;
}

const maxLength = 2000;

export function Composer({
  currentUserId,
  currentUser,
  creating,
  textareaRef,
  onCreate,
}: ComposerProps) {
  const { t } = useI18n();
  const [body, setBody] = useState("");
  const [focused, setFocused] = useState(false);
  const length = Array.from(body).length;
  const canPublish = body.trim().length > 0 && length <= maxLength && !creating;

  function resize(element: HTMLTextAreaElement) {
    element.style.height = "auto";
    element.style.height = `${Math.min(Math.max(element.scrollHeight, 88), 320)}px`;
  }

  async function submit() {
    if (!canPublish) return;
    const published = await onCreate(body.trim());
    if (published) {
      setBody("");
      const element = textareaRef.current;
      if (element) {
        element.style.height = "";
        element.focus();
      }
    }
  }

  return (
    <section className={`composer ${focused ? "is-focused" : ""}`}>
      <div className="composer-head">
        <span className={`avatar tone-${avatarTone(currentUserId)}`} aria-hidden="true">
          {initials(currentUser, currentUserId)}
        </span>
        <div className="composer-copy">
          <strong>
            {t("composer.as")}{" "}
            <span className="composer-handle">
              @{currentUser?.handle ?? currentUserId}
            </span>
          </strong>
          <span className="muted">
            {currentUser?.display_name ?? `User ${currentUserId}`}
          </span>
        </div>
      </div>

      <label className="sr-only" htmlFor="composer-input">
        {t("composer.placeholder")}
      </label>
      <textarea
        id="composer-input"
        ref={textareaRef}
        className="composer-input"
        placeholder={t("composer.placeholder")}
        value={body}
        rows={3}
        maxLength={maxLength}
        onChange={(event) => {
          setBody(event.target.value);
          resize(event.currentTarget);
        }}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onKeyDown={(event) => {
          if ((event.metaKey || event.ctrlKey) && event.key === "Enter") {
            event.preventDefault();
            void submit();
          }
        }}
      />

      <div className="composer-foot">
        <div className="composer-meta">
          <span className={`counter mono ${length > maxLength ? "is-over" : ""}`}>
            {t("composer.limit", { count: length, max: maxLength })}
          </span>
          <span className="composer-shortcut">{t("composer.shortcut")}</span>
        </div>
        <button
          type="button"
          className="btn btn-accent"
          disabled={!canPublish}
          onClick={() => void submit()}
        >
          <SendIcon size={15} />
          {creating ? t("composer.publishing") : t("composer.publish")}
        </button>
      </div>

      <p className="composer-hint">{t("composer.hint")}</p>
    </section>
  );
}
