import type { ToastMessage } from "../types";
import { useI18n } from "../lib/i18n";

interface ToastProps {
  toast: ToastMessage;
  onClose: () => void;
}

export function Toast({ toast, onClose }: ToastProps) {
  const { t } = useI18n();

  return (
    <div
      className={`toast toast-${toast.kind}`}
      role={toast.kind === "error" ? "alert" : "status"}
    >
      <span className="dot" aria-hidden="true" />
      <span className="toast-message">{toast.message}</span>
      <button
        type="button"
        className="toast-close"
        onClick={onClose}
        aria-label={t("toast.dismiss")}
      >
        ×
      </button>
    </div>
  );
}
