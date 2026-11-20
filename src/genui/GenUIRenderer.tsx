import { type Spec, type StateModel, validateSpec } from "@json-render/core";
import {
  ActionProvider,
  type ComponentRenderer,
  createStateStore,
  Renderer,
  StateProvider,
  type StateStore,
  VisibilityProvider,
} from "@json-render/react";
import {
  Component,
  type ErrorInfo,
  memo,
  type ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useMessages } from "../i18n";
import { genUICatalog } from "./catalog";
import { createGenUIRegistry } from "./registry";
import { loadUIState, saveUIState } from "./storage";

type GenUIRendererProps = {
  spec: Spec;
  loading?: boolean;
  complete?: boolean;
  storageKey?: string;
  onAskAgent?: (message: string) => void;
};

function GenUIRendererComponent({
  spec,
  loading = false,
  complete = true,
  storageKey = "",
  onAskAgent,
}: GenUIRendererProps) {
  const t = useMessages();
  const validation = useMemo(() => validateGenUI(spec, complete, t), [spec, complete, t]);
  const [store] = useState<StateStore>(() =>
    createStateStore(loadUIState(storageKey, spec.state ?? {})),
  );
  const registry = useMemo(
    () => createGenUIRegistry((message) => onAskAgent?.(message)),
    [onAskAgent],
  );
  const handlers = useMemo(
    () =>
      registry.handlers(
        () => undefined,
        () => store.getSnapshot(),
      ),
    [registry, store],
  );

  useSpecState(store, spec.state);
  usePersistedState(store, storageKey);

  if (!validation.valid) {
    return (
      <div className="genui-failure" role="status">
        {t.genuiRenderUnavailable(validation.message)}
      </div>
    );
  }

  return (
    <GenUIErrorBoundary messages={t}>
      <div className="genui" data-loading={loading ? "true" : undefined}>
        <StateProvider store={store}>
          <VisibilityProvider>
            <ActionProvider handlers={handlers}>
              <Renderer
                spec={spec}
                registry={registry.registry}
                loading={loading}
                fallback={UnknownComponent}
              />
            </ActionProvider>
          </VisibilityProvider>
        </StateProvider>
      </div>
    </GenUIErrorBoundary>
  );
}

export const GenUIRenderer = memo(GenUIRendererComponent, (previous, next) => {
  return (
    previous.spec === next.spec &&
    previous.loading === next.loading &&
    previous.complete === next.complete &&
    previous.storageKey === next.storageKey &&
    previous.onAskAgent === next.onAskAgent
  );
});

function useSpecState(store: StateStore, state: StateModel | undefined) {
  const previous = useRef<StateModel>(state ?? {});
  useEffect(() => {
    const next = state ?? {};
    const updates: Record<string, unknown> = {};
    for (const key of new Set([...Object.keys(previous.current), ...Object.keys(next)])) {
      if (Object.is(previous.current[key], next[key])) continue;
      updates[`/${key.replaceAll("~", "~0").replaceAll("/", "~1")}`] = next[key];
    }
    previous.current = next;
    if (Object.keys(updates).length > 0) store.update(updates);
  }, [store, state]);
}

function usePersistedState(store: StateStore, storageKey: string) {
  useEffect(() => {
    if (!storageKey) return;
    let timer: number | undefined;
    const persist = () => saveUIState(storageKey, store.getSnapshot());
    const unsubscribe = store.subscribe(() => {
      window.clearTimeout(timer);
      timer = window.setTimeout(persist, 180);
    });
    return () => {
      unsubscribe();
      window.clearTimeout(timer);
      persist();
    };
  }, [store, storageKey]);
}

function validateGenUI(spec: Spec, complete: boolean, t: ReturnType<typeof useMessages>) {
  if (!complete) return { valid: true, message: "" };
  const structural = validateSpec(spec, { checkOrphans: true });
  if (!structural.valid) {
    const issue = structural.issues.find((item) => item.severity === "error");
    return { valid: false, message: issue?.message ?? t.genuiInvalidStructure() };
  }
  const catalog = genUICatalog.validate(spec);
  if (!catalog.success) {
    const issue = catalog.error?.issues[0];
    return { valid: false, message: issue?.message ?? t.genuiInvalidProps() };
  }
  return { valid: true, message: "" };
}

const UnknownComponent: ComponentRenderer = ({ element }) => {
  const t = useMessages();
  return <div className="genui-unknown">{t.genuiUnknownComponent(element.type)}</div>;
};

type BoundaryProps = {
  children: ReactNode;
  messages: ReturnType<typeof useMessages>;
};

type BoundaryState = {
  error: Error | null;
};

class GenUIErrorBoundary extends Component<BoundaryProps, BoundaryState> {
  state: BoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): BoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    if (import.meta.env.DEV) {
      console.error("Intelligent UI render failed", error, info.componentStack);
    }
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <div className="genui-failure" role="status">
        {this.props.messages.genuiRenderFailed()}
      </div>
    );
  }
}

export default GenUIRenderer;
