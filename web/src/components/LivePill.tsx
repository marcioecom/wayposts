import { useI18n } from "../lib/i18n";
import type { TrafficState } from "../lib/metrics";

interface LivePillProps {
  traffic: TrafficState;
  onOpenLive: () => void;
  className?: string;
}

export function LivePill({ traffic, onOpenLive, className = "" }: LivePillProps) {
  const { t, locale } = useI18n();
  const sample = traffic.sample;
  const stale =
    traffic.lastSuccessAt !== null && Date.now() - traffic.lastSuccessAt > 15_000;
  const live =
    Boolean(sample) &&
    traffic.status === "ok" &&
    !stale &&
    (sample?.reqPerSec ?? 0) >= 5;
  const rate =
    sample == null
      ? "n/a"
      : new Intl.NumberFormat(locale, {
          maximumFractionDigits:
            sample.reqPerSec > 0 && sample.reqPerSec < 10 ? 1 : 0,
        }).format(sample.reqPerSec);

  return (
    <button
      type="button"
      className={`live-pill ${live ? "is-live" : ""} ${className}`.trim()}
      onClick={onOpenLive}
      aria-haspopup="dialog"
      aria-label={t("nav.openLive")}
    >
      <span className="dot" />
      <span className="mono">{t("feed.liveRate", { rate })}</span>
    </button>
  );
}
