import { Component, type ErrorInfo, type ReactNode } from "react";
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

    return (
      <div className="boot-screen error" role="alert">
        <BrandMark className="boot-mark" />
        <h1>界面出现错误</h1>
        <p>{error.message || "发生了未预期的渲染错误。"}</p>
        <button type="button" className="button primary" onClick={reloadApp}>
          重新加载
        </button>
      </div>
    );
  }
}

function reloadApp() {
  window.location.reload();
}

function reportRenderError(error: Error, info: ErrorInfo) {
  if (import.meta.env.DEV) {
    console.error("Uncaught render error", error, info.componentStack);
  }
}
