type KeyStorageStatusProps = {
  storage: string;
  keychainAvailable: boolean;
  keychainError?: string;
};

export function KeyStorageStatus({
  storage,
  keychainAvailable,
  keychainError,
}: KeyStorageStatusProps) {
  if (storage === "none" && keychainAvailable) return null;

  const labels: Record<string, string> = {
    keychain: "已存入系统钥匙串",
    environment: "来自环境变量，只读",
    session: "仅保存在本次会话，重启后需要重新输入",
    none: "未保存",
  };

  return (
    <div className="key-storage-status">
      <p>{labels[storage] ?? "未保存"}</p>
      {!keychainAvailable && (
        <p className="settings-warning" title={keychainError}>
          系统钥匙串不可用；在这里输入的密钥只会保留到本次会话结束。
        </p>
      )}
    </div>
  );
}
