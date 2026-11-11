import { useEffect, useState } from "react";
import { MiraUpdate, type UpdateStatus } from "../../mygo";
import { Button, Notice, Switch } from "../../ui";

export function AboutSettings() {
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
          setError(refreshError instanceof Error ? refreshError.message : "无法读取更新状态");
        }
      }
    }

    void refresh();
    const timer = window.setInterval(() => void refresh(), 2000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, []);

  async function toggleAutomaticChecks() {
    if (!status || busy) return;
    setBusy(true);
    setError("");
    try {
      setStatus(await MiraUpdate.setAutomaticChecks(!status.automaticChecks));
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : "无法修改更新设置");
    } finally {
      setBusy(false);
    }
  }

  async function checkForUpdates() {
    setError("");
    try {
      await MiraUpdate.checkForUpdates();
    } catch (checkError) {
      setError(checkError instanceof Error ? checkError.message : "无法检查更新");
    }
  }

  const version = status?.version || "…";
  const availability = updateAvailability(status);

  return (
    <section className="settings-group">
      <div className="about-heading">
        <strong>Mira</strong>
        <span>版本 {version}</span>
      </div>
      <Switch
        checked={status?.automaticChecks ?? false}
        disabled={!status?.enabled || busy}
        title="自动检查更新"
        description="启动时检查，安装前确认。"
        onChange={() => void toggleAutomaticChecks()}
      />

      <div className="update-row">
        <div className="update-copy">
          <strong>{availability.title}</strong>
          <span>{availability.detail}</span>
        </div>
        <Button variant="quiet" onClick={() => void checkForUpdates()} disabled={!status}>
          {status?.availableVersion ? "更新" : status?.enabled ? "检查" : "查看状态"}
        </Button>
      </div>

      {status && !status.enabled && (
        <Notice>当前安装方式不支持自动更新，请从 GitHub Releases 下载新版本。</Notice>
      )}
      {error && <Notice tone="error">{error}</Notice>}
    </section>
  );
}

function updateAvailability(status: UpdateStatus | null) {
  if (!status) {
    return { title: "正在读取更新信息…", detail: "" };
  }
  if (status.availableVersion) {
    return {
      title: `可更新到 ${status.availableVersion}`,
      detail: status.lastCheck ? `检查于 ${formatCheckTime(status.lastCheck)}` : "",
    };
  }
  if (status.lastCheck) {
    return {
      title: "已是最新版本",
      detail: `检查于 ${formatCheckTime(status.lastCheck)}`,
    };
  }
  return {
    title: "尚未检查",
    detail: status.enabled ? "启动时会自动检查" : "",
  };
}

function formatCheckTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}
