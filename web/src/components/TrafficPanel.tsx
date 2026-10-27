import { useEffect, useRef, useState } from "react";
import { useI18n } from "../lib/i18n";
import type { TrafficState } from "../lib/metrics";
import { CheckIcon, CopyIcon, RefreshIcon } from "./icons";
import { Sparkline } from "./Sparkline";

interface TrafficPanelProps {
  traffic: TrafficState & { retry: () => void };
  apiBase: string;
}

function useNow(interval = 1000): number {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), interval);
    return () => window.clearInterval(timer);
  }, [interval]);
  return now;
}

export function TrafficPanel({ traffic, apiBase }: TrafficPanelProps) {
  const { t, locale } = useI18n();
  const now = useNow();
  const [copied, setCopied] = useState(false);
  const copyTimer = useRef<number | undefined>(undefined);

  useEffect(() => {
    return () => {
      if (copyTimer.current) window.clearTimeout(copyTimer.current);
    };
  }, []);

  const sample = traffic.sample;
  const rate = sample?.reqPerSec ?? 0;
  const stale =
    traffic.lastSuccessAt !== null && now - traffic.lastSuccessAt > 15_000;
  const live =
    Boolean(sample) && traffic.status === "ok" && !stale && rate >= 5;
  const statusKind = live ? "live" : stale || traffic.status === "error" ? "stale" : "idle";

  const statusLabel = !sample
    ? traffic.status === "error"
      ? t("traffic.error")
      : t("traffic.loading")
    : statusKind === "live"
      ? t("traffic.live")
      : statusKind === "stale"
        ? t("traffic.stale")
        : t("traffic.idle");

  const number = (value: number, digits: number): string =>
    new Intl.NumberFormat(locale, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    }).format(value);

  const formatRate = (value: number): string =>
    number(value, value > 0 && value < 10 ? 1 : 0);

  const p95 =
    sample?.p95Ms == null
      ? "—"
      : `${number(sample.p95Ms, sample.p95Ms < 10 ? 1 : 0)} ms`;
  const errorPct =
    sample == null
      ? "—"
      : `${number(sample.errorPct, sample.errorPct > 0 && sample.errorPct < 10 ? 1 : 0)}%`;
  const age =
    traffic.lastSuccessAt == null
      ? "—"
      : Math.max(0, Math.round((now - traffic.lastSuccessAt) / 1000)) < 2
        ? t("traffic.justNow")
        : t("traffic.ago", {
            seconds: Math.round((now - traffic.lastSuccessAt) / 1000),
          });

  const command = t("traffic.command");

  async function copyCommand() {
    try {
      await navigator.clipboard.writeText(command);
      setCopied(true);
      if (copyTimer.current) window.clearTimeout(copyTimer.current);
      copyTimer.current = window.setTimeout(() => setCopied(false), 1800);
    } catch {
      // Clipboard can be unavailable outside a secure context.
    }
  }

  return (
    <section className="traffic" aria-label={t("aria.traffic")}>
      <header className="traffic-head">
        <span className="traffic-title">
          {t("traffic.eyebrow")}
          <span className="traffic-view">{t("traffic.view")}</span>
        </span>
        <span className={`live-state is-${statusKind}`}>
          <span className="dot" />
          {statusLabel}
        </span>
      </header>

      {sample ? (
        <>
          <div className="traffic-hero">
            <strong className="traffic-rate">{formatRate(rate)}</strong>
            <span className="traffic-unit">{t("traffic.req")}</span>
          </div>

          <Sparkline values={traffic.history} label={t("aria.sparkline")} />

          <div className="traffic-window">
            <span>{t("traffic.window")}</span>
            <span>{t("traffic.updated", { age })}</span>
          </div>

          <dl className="traffic-grid">
            <div>
              <dt>{t("traffic.p95")}</dt>
              <dd>{p95}</dd>
            </div>
            <div>
              <dt>{t("traffic.errors")}</dt>
              <dd className={sample.errorPct > 0 ? "is-bad" : ""}>
                {errorPct}
              </dd>
            </div>
            <div>
              <dt>{t("traffic.inflight")}</dt>
              <dd>{number(sample.inFlight, 0)}</dd>
            </div>
            <div>
              <dt>{t("traffic.timeline")}</dt>
              <dd>{formatRate(sample.timelinePerSec)}</dd>
            </div>
            <div>
              <dt>{t("traffic.writes")}</dt>
              <dd>{formatRate(sample.writesPerSec)}</dd>
            </div>
            <div>
              <dt>{t("traffic.likes")}</dt>
              <dd>{formatRate(sample.likesPerSec)}</dd>
            </div>
          </dl>

          {statusKind === "idle" ? (
            <div className="traffic-idle">
              <p>{t("traffic.idleHint")}</p>
              <div className="command-row">
                <code>{command}</code>
                <button
                  type="button"
                  className="command-copy"
                  onClick={() => void copyCommand()}
                  aria-label={t("traffic.copy")}
                  title={t("traffic.copy")}
                >
                  {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
                </button>
              </div>
              <span className="sr-only" aria-live="polite">
                {copied ? t("traffic.copied") : ""}
              </span>
            </div>
          ) : null}

          {statusKind === "stale" ? (
            <p className="traffic-stale">
              {t("traffic.errorHint", { path: "/metrics" })}{" "}
              <button type="button" className="link-btn" onClick={traffic.retry}>
                <RefreshIcon size={13} /> {t("traffic.retry")}
              </button>
            </p>
          ) : null}
        </>
      ) : traffic.status === "error" ? (
        <div className="traffic-empty">
          <strong>{t("traffic.error")}</strong>
          <p>{t("traffic.errorHint", { path: "/metrics" })}</p>
          <button type="button" className="link-btn" onClick={traffic.retry}>
            <RefreshIcon size={13} /> {t("traffic.retry")}
          </button>
        </div>
      ) : (
        <div className="traffic-loading" aria-busy="true">
          <span className="skeleton-line xl" />
          <span className="skeleton-line" />
          <span className="skeleton-line medium" />
        </div>
      )}

      <footer className="traffic-foot">
        <a href={`${apiBase}/metrics`} target="_blank" rel="noreferrer">
          {t("system.metrics")}
        </a>
        <p>{t("traffic.note")}</p>
      </footer>
    </section>
  );
}
