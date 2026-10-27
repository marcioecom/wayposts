import type { Lang } from "./i18n";

const formatters: Record<Lang, Intl.RelativeTimeFormat> = {
  en: new Intl.RelativeTimeFormat("en", { numeric: "auto" }),
  pt: new Intl.RelativeTimeFormat("pt-BR", { numeric: "auto" }),
};

const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
  ["year", 31_536_000_000],
  ["month", 2_592_000_000],
  ["week", 604_800_000],
  ["day", 86_400_000],
  ["hour", 3_600_000],
  ["minute", 60_000],
];

export function relativeTime(iso: string, lang: Lang): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return lang === "pt" ? "horário desconhecido" : "unknown time";
  }

  const diff = date.getTime() - Date.now();
  const absolute = Math.abs(diff);
  if (absolute < 60_000) return lang === "pt" ? "agora" : "just now";

  for (const [unit, milliseconds] of units) {
    if (absolute >= milliseconds) {
      return formatters[lang].format(Math.round(diff / milliseconds), unit);
    }
  }
  return lang === "pt" ? "agora" : "just now";
}

export function fullTime(iso: string, lang: Lang): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleString(lang === "pt" ? "pt-BR" : "en-US", {
    dateStyle: "medium",
    timeStyle: "medium",
  });
}
