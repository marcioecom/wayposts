import type { WsStatus } from "../types";
import { useI18n } from "../lib/i18n";
import { RefreshIcon } from "./icons";

interface NudgeCardProps {
  status: WsStatus;
  count: number | null;
  head: number | null;
  onRefresh: () => void;
}

const labels: Record<WsStatus, "nudge.connecting" | "nudge.listening" | "nudge.closed"> = {
  connecting: "nudge.connecting",
  open: "nudge.listening",
  closed: "nudge.closed",
};

export function NudgeCard({ status, count, head, onRefresh }: NudgeCardProps) {
  const { t } = useI18n();

  return (
    <section className="rail-section nudge">
      <header className="rail-section-head">
        <span className="rail-label">{t("nudge.eyebrow")}</span>
        <span className={`live-state is-${status === "open" ? "live" : status === "connecting" ? "idle" : "stale"}`}>
          <span className="dot" />
          {t(`status.${status}`)}
        </span>
      </header>

      {count !== null ? (
        <div className="nudge-preview">
          <strong>{count}</strong>
          <span>{t("nudge.new", { count })}</span>
          {head ? <span className="mono">{t("nudge.head", { id: head })}</span> : null}
        </div>
      ) : (
        <p className="rail-copy">{t(labels[status])}</p>
      )}

      <p className="rail-hint">{t("nudge.hint")}</p>

      <button type="button" className="btn btn-rail btn-sm" onClick={onRefresh}>
        <RefreshIcon size={14} />
        {t("nudge.refreshNow")}
      </button>
    </section>
  );
}
