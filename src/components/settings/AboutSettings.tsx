import { useEffect, useState } from "react";
import { MiraUpdate, type UpdateStatus } from "../../mygo";
import { Button, Notice, Switch } from "../../ui";
import { BrandMark } from "../BrandMark";

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
    <>
      <section className="settings-group">
        <div className="about-app">
          <span className="about-logo" aria-hidden="true">
            <BrandMark />
          </span>
          <div className="about-app-copy">
            <strong>Mira</strong>
            <span>版本 {version}</span>
          </div>
        </div>
        <p className="settings-description">
          本地优先的 Agent 桌面客户端。对话、设置和记忆都保存在这台电脑上。
        </p>
      </section>

      <section className="settings-group">
        <h3 className="settings-group-title">软件更新</h3>
        <Switch
          checked={status?.automaticChecks ?? false}
          disabled={!status?.enabled || busy}
          title="自动检查更新"
          description="启动后检查新版本，安装前会先征求你的同意。"
          onChange={() => void toggleAutomaticChecks()}
        />

        <div className="update-card">
          <div className="update-copy">
            <strong>{availability.title}</strong>
            <span>{availability.detail}</span>
          </div>
          <Button onClick={() => void checkForUpdates()} disabled={!status}>
            {status?.enabled ? "检查更新" : "查看状态"}
          </Button>
        </div>

        {status && !status.enabled && (
          <Notice>
            当前安装方式无法自动更新。开发模式、由包管理器安装或安装目录不可写时会这样；请从 GitHub
            Releases 下载新版本。
          </Notice>
        )}
        {error && <Notice tone="error">{error}</Notice>}
      </section>
    </>
  );
}

function updateAvailability(status: UpdateStatus | null) {
  if (!status) {
    return { title: "正在读取更新信息…", detail: "" };
  }
  if (status.availableVersion) {
    return {
      title: `发现新版本 ${status.availableVersion}`,
      detail: status.lastCheck
        ? `上次检查于 ${formatCheckTime(status.lastCheck)}，打开更新窗口即可安装。`
        : "打开更新窗口即可安装。",
    };
  }
  if (status.lastCheck) {
    return {
      title: "已是最新版本",
      detail: `上次检查于 ${formatCheckTime(status.lastCheck)}。`,
    };
  }
  return {
    title: "尚未检查更新",
    detail: status.enabled ? "检查会在后台自动进行，也可以现在手动检查。" : "",
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
