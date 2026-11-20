import { direction, locale, onLocaleChange } from "mygo-runtime";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import { messagesIn } from "../locales";
import { Mira } from "../mygo";

const fallbackLocale = "zh-Hans";

function preferredLocale() {
  try {
    const tag = locale();
    if (tag) return tag;
  } catch {
    // The page may run outside a MyGo window, as in tests or `vite dev:web`.
  }
  if (typeof navigator !== "undefined") {
    return navigator.languages?.[0] ?? navigator.language ?? fallbackLocale;
  }
  return fallbackLocale;
}

function preferredDirection() {
  try {
    return direction();
  } catch {
    return "ltr";
  }
}

type I18nValue = {
  locale: string;
  messages: ReturnType<typeof messagesIn>;
  setLocale: (tag: string) => Promise<void>;
};

const I18nContext = createContext<I18nValue | null>(null);
const fallbackMessages = messagesIn(preferredLocale());

type I18nProviderProps = {
  children: ReactNode;
  initialLocale?: string;
};

export function I18nProvider({ children, initialLocale }: I18nProviderProps) {
  const [current, setCurrent] = useState(() => initialLocale ?? preferredLocale());

  useEffect(() => {
    document.documentElement.lang = current;
  }, [current]);

  useEffect(() => {
    document.documentElement.dir = preferredDirection();
    try {
      return onLocaleChange((next, dir) => {
        setCurrent(next);
        document.documentElement.dir = dir;
      });
    } catch {
      return undefined;
    }
  }, []);

  const setLocale = useCallback(async (tag: string) => {
    await Mira.setLanguage(tag);
  }, []);

  const value = useMemo(
    () => ({ locale: current, messages: messagesIn(current), setLocale }),
    [current, setLocale],
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n() {
  const value = useContext(I18nContext);
  return value ?? { locale: preferredLocale(), messages: fallbackMessages };
}

export function useMessages() {
  return useI18n().messages;
}

export function useLocale() {
  return useI18n().locale;
}

export { fallbackLocale };
