import { useId, useState } from "react";
import { useI18n } from "../lib/i18n";
import { ArrowUpRightIcon } from "./icons";

interface FollowCardProps {
  currentUserId: number;
  currentHandle: string;
  onFollow: (followeeId: number, action: "follow" | "unfollow") => void;
  onOpenProfile: (id: number) => void;
}

export function FollowCard({
  currentUserId,
  currentHandle,
  onFollow,
  onOpenProfile,
}: FollowCardProps) {
  const { t } = useI18n();
  const inputId = useId();
  const [value, setValue] = useState("1");
  const followee = Number(value);
  const valid = Number.isInteger(followee) && followee > 0;
  const invalid = value.trim().length > 0 && !valid;

  return (
    <section className="rail-section follow">
      <header className="rail-section-head">
        <span className="rail-label">{t("follow.eyebrow")}</span>
        <span className="rail-meta">{t("follow.title")}</span>
      </header>

      <form
        className="follow-form"
        onSubmit={(event) => {
          event.preventDefault();
          if (valid) onFollow(followee, "follow");
        }}
      >
        <label className="field" htmlFor={inputId}>
          <span>{t("follow.label")}</span>
          <input
            id={inputId}
            className="input input-dark"
            inputMode="numeric"
            min={1}
            value={value}
            aria-invalid={invalid}
            onChange={(event) => setValue(event.target.value)}
          />
        </label>

        {invalid ? <p className="field-error">{t("error.followInvalid")}</p> : null}

        <div className="follow-actions">
          <button type="submit" className="btn btn-accent btn-sm" disabled={!valid}>
            {t("follow.follow")}
          </button>
          <button
            type="button"
            className="btn btn-rail btn-sm"
            disabled={!valid}
            onClick={() => valid && onFollow(followee, "unfollow")}
          >
            {t("follow.unfollow")}
          </button>
        </div>
      </form>

      <button
        type="button"
        className="rail-link"
        disabled={!valid}
        onClick={() => valid && onOpenProfile(followee)}
      >
        {t("follow.open", { id: valid ? followee : "…" })}
        <ArrowUpRightIcon size={13} />
      </button>

      <p className="rail-hint">{t("follow.hint")}</p>
      <p className="rail-meta mono">{t("follow.acting", { handle: currentHandle || currentUserId })}</p>
    </section>
  );
}
