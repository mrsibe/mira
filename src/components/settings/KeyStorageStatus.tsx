import { useMessages } from "../../i18n";

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
  const t = useMessages();
  if (storage === "none" && keychainAvailable) return null;

  const labels: Record<string, string> = {
    keychain: t.keyStorageKeychain(),
    environment: t.keyStorageEnvironment(),
    session: t.keyStorageSession(),
    none: t.keyStorageNone(),
  };

  return (
    <div className="key-storage-status">
      <p>{labels[storage] ?? t.keyStorageNone()}</p>
      {!keychainAvailable && (
        <p className="settings-warning" title={keychainError}>
          {t.keyStorageKeychainUnavailable()}
        </p>
      )}
    </div>
  );
}
