import { useI18n } from "../lib/i18n";

interface LanguageToggleProps {
  tone?: "light" | "dark";
}

export function LanguageToggle({ tone = "light" }: LanguageToggleProps) {
  const { lang, setLang, t } = useI18n();

  return (
    <div
      className={`lang-toggle lang-toggle-${tone}`}
      role="group"
      aria-label={t("aria.language")}
    >
      <button
        type="button"
        className={lang === "en" ? "is-active" : ""}
        aria-pressed={lang === "en"}
        onClick={() => setLang("en")}
      >
        EN
      </button>
      <button
        type="button"
        className={lang === "pt" ? "is-active" : ""}
        aria-pressed={lang === "pt"}
        onClick={() => setLang("pt")}
      >
        PT
      </button>
    </div>
  );
}
