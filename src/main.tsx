import { createRoot } from "react-dom/client";
import App from "./App";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { applyStoredAppearance } from "./lib/appearance";
import { applyStoredSidebarWidth } from "./lib/sidebarWidth";

applyStoredAppearance();
applyStoredSidebarWidth();

const root = document.getElementById("root");
if (!root) {
  throw new Error("Missing #root");
}

createRoot(root).render(
  <ErrorBoundary>
    <App />
  </ErrorBoundary>,
);
