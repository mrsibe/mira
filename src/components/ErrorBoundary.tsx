import { Component, type ErrorInfo, type ReactNode } from "react";
import { useMessages } from "../i18n";
import { BrandMark } from "./BrandMark";

type ErrorBoundaryProps = {
  children: ReactNode;
};

type ErrorBoundaryState = {
  error: Error | null;
};

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    reportRenderError(error, info);
  }

  render() {
    const { error } = this.state;
    if (!error) return this.props.children;

    return <ErrorScreen error={error} />;
  }
}

function ErrorScreen({ error }: { error: Error }) {
  const t = useMessages();
  return (
    <div className="boot-screen error" role="alert">
      <BrandMark className="boot-mark" />
      <h1>{t.errorBoundaryTitle()}</h1>
      <p>{error.message || t.errorBoundaryMessage()}</p>
      <button type="button" className="button primary" onClick={reloadApp}>
        {t.errorBoundaryReload()}
      </button>
    </div>
  );
}

function reloadApp() {
  window.location.reload();
}

function reportRenderError(error: Error, info: ErrorInfo) {
  if (import.meta.env.DEV) {
    console.error("Uncaught render error", error, info.componentStack);
  }
}
