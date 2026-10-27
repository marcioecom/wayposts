import type { WsStatus } from "../types";
import { useI18n } from "../lib/i18n";
import { ArrowUpRightIcon } from "./icons";

interface SystemDetailsProps {
  apiBase: string;
  wsStatus: WsStatus;
  postCount: number;
  nextCursor: number;
  hasMore: boolean;
}

function hostOf(value: string): string {
  try {
    return new URL(value).host;
  } catch {
    return value;
  }
}

export function SystemDetails({
  apiBase,
  wsStatus,
  postCount,
  nextCursor,
  hasMore,
}: SystemDetailsProps) {
  const { t } = useI18n();

  return (
    <details className="rail-details">
      <summary>
        <span>{t("system.details")}</span>
        <span className="details-chevron" aria-hidden="true" />
      </summary>
      <dl className="kv">
        <div>
          <dt>{t("system.api")}</dt>
          <dd className="mono" title={apiBase}>{hostOf(apiBase)}</dd>
        </div>
        <div>
          <dt>{t("system.ws")}</dt>
          <dd className="mono">{t(`status.${wsStatus}`)}</dd>
        </div>
        <div>
          <dt>{t("system.posts")}</dt>
          <dd className="mono">{postCount}</dd>
        </div>
        <div>
          <dt>{t("system.cursor")}</dt>
          <dd className="mono" title={String(nextCursor)}>
            {nextCursor || t("system.none")}
          </dd>
        </div>
        <div>
          <dt>{t("system.more")}</dt>
          <dd className="mono">{hasMore ? t("system.yes") : t("system.no")}</dd>
        </div>
      </dl>

      <div className="rail-links">
        <a href={`${apiBase}/healthz`} target="_blank" rel="noreferrer">
          {t("system.health")} <ArrowUpRightIcon size={13} />
        </a>
        <a href={`${apiBase}/metrics`} target="_blank" rel="noreferrer">
          {t("system.metrics")} <ArrowUpRightIcon size={13} />
        </a>
      </div>
    </details>
  );
}

export function ArchitectureNotes() {
  const { t } = useI18n();

  return (
    <details className="rail-details">
      <summary>
        <span>{t("notes.summary")}</span>
        <span className="details-chevron" aria-hidden="true" />
      </summary>
      <ul className="rail-notes">
        <li>{t("notes.one")}</li>
        <li>{t("notes.two")}</li>
        <li>{t("notes.three")}</li>
        <li>{t("notes.four")}</li>
      </ul>
    </details>
  );
}
