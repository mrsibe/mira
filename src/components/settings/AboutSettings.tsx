import { useEffect, useState } from "react";
import { useLocale, useMessages } from "../../i18n";
import { MiraUpdate, type UpdateStatus } from "../../mygo";
import { Button, Notice, Switch } from "../../ui";

export function AboutSettings() {
  const t = useMessages();
  const locale = useLocale();
  const [status, setStatus] = useState<UpdateStatus | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;

    async function refresh() {
      try {
        const next = await MiraUpdate.getStatus();
        if (active) {
          setStatus(next);
          setError("");
        }
      } catch (refreshError) {
        if (active) {
          setError(
            refreshError instanceof Error ? refreshError.message : t.aboutUpdateStatusFailed(),
          );
        }
      }
    }

    void refresh();
    const timer = window.setInterval(() => void refresh(), 2000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [t.aboutUpdateStatusFailed]);

  async function toggleAutomaticChecks() {
    if (!status || busy) return;
    setBusy(true);
    setError("");
    try {
      setStatus(await MiraUpdate.setAutomaticChecks(!status.automaticChecks));
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : t.aboutUpdateSettingsFailed());
    } finally {
      setBusy(false);
    }
  }

  async function checkForUpdates() {
    setError("");
    try {
      await MiraUpdate.checkForUpdates();
    } catch (checkError) {
      setError(checkError instanceof Error ? checkError.message : t.aboutCheckFailed());
    }
  }

  const version = status?.version || "…";
  const availability = updateAvailability(status, t, locale);

  return (
    <section className="settings-group">
      <div className="about-heading">
        <strong>Mira</strong>
        <span>{t.aboutVersion(version)}</span>
      </div>
      <Switch
        checked={status?.automaticChecks ?? false}
        disabled={!status?.enabled || busy}
        title={t.aboutAutoCheck()}
        description={t.aboutAutoCheckDescription()}
        onChange={() => void toggleAutomaticChecks()}
      />

      <div className="update-row">
        <div className="update-copy">
          <strong>{availability.title}</strong>
          <span>{availability.detail}</span>
        </div>
        <Button variant="quiet" onClick={() => void checkForUpdates()} disabled={!status}>
          {status?.availableVersion
            ? t.aboutUpdate()
            : status?.enabled
              ? t.aboutCheck()
              : t.aboutViewStatus()}
        </Button>
      </div>

      {status && !status.enabled && <Notice>{t.aboutUnsupported()}</Notice>}
      {error && <Notice tone="error">{error}</Notice>}
    </section>
  );
}

function updateAvailability(
  status: UpdateStatus | null,
  t: ReturnType<typeof useMessages>,
  locale: string,
) {
  if (!status) {
    return { title: t.aboutReading(), detail: "" };
  }
  if (status.availableVersion) {
    return {
      title: t.aboutAvailable(status.availableVersion),
      detail: status.lastCheck ? t.aboutCheckedAt(formatCheckTime(status.lastCheck, locale)) : "",
    };
  }
  if (status.lastCheck) {
    return {
      title: t.aboutUpToDate(),
      detail: t.aboutCheckedAt(formatCheckTime(status.lastCheck, locale)),
    };
  }
  return {
    title: t.aboutNotChecked(),
    detail: status.enabled ? t.aboutAutoCheckOnStartup() : "",
  };
}

function formatCheckTime(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(locale, {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
