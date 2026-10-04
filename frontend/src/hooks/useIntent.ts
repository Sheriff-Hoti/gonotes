import { useEffect, useRef, useState } from "react";
import { mockClassifyAsync, MOCK_MODEL, MOCK_QUESTION_COUNT } from "@/lib/jev/mock";
import { type IntentResult, noneResult } from "@/lib/jev/types";
import { LRU, normalizeKey } from "@/lib/lru";

export type IntentStatus = "idle" | "thinking" | "ready";

export type IntentState = {
  result: IntentResult;
  /** The text `result` was computed for. */
  resultText: string;
  status: IntentStatus;
  /** Last real classification, for the latency HUD. */
  hud: { latency: number | null; questions: number; model: string; cached: boolean };
};

const clientCache = new LRU<string, IntentResult>(300);

/**
 * Debounced, abortable, stale-safe intent classification (offline only).
 * Only the latest request id may update state.
 */
export function useIntent(text: string, { debounceMs = 120 }: { debounceMs?: number } = {}): IntentState {
  const [state, setState] = useState<IntentState>({
    result: noneResult(),
    resultText: "",
    status: "idle",
    hud: { latency: null, questions: 0, model: MOCK_MODEL, cached: false },
  });
  const reqId = useRef(0);
  const controller = useRef<AbortController | null>(null);

  useEffect(() => {
    const id = ++reqId.current;
    controller.current?.abort();
    const key = normalizeKey(text);

    if (key.length < 2) {
      // Nothing to ask; resolve synchronously on the next frame.
      const raf = requestAnimationFrame(() => {
        if (id === reqId.current) setState((s) => ({ ...s, result: noneResult(), resultText: text, status: "idle" }));
      });
      return () => cancelAnimationFrame(raf);
    }

    const cached = clientCache.get(key);
    if (cached) {
      const raf = requestAnimationFrame(() => {
        if (id === reqId.current)
          setState((s) => ({
            ...s,
            result: { ...cached, cached: true },
            resultText: text,
            status: "ready",
            hud: { latency: 0, questions: MOCK_QUESTION_COUNT, model: MOCK_MODEL, cached: true },
          }));
      });
      return () => cancelAnimationFrame(raf);
    }

    const timer = setTimeout(async () => {
      const ctrl = new AbortController();
      controller.current = ctrl;
      setState((s) => ({ ...s, status: "thinking" }));
      try {
        const started = performance.now();
        const result = await mockClassifyAsync(text, ctrl.signal);
        if (id !== reqId.current) return; // stale
        if (!result.error) clientCache.set(key, result);
        setState({
          result,
          resultText: text,
          status: "ready",
          hud: {
            latency: result.cached ? 0 : performance.now() - started,
            questions: MOCK_QUESTION_COUNT,
            model: MOCK_MODEL,
            cached: Boolean(result.cached),
          },
        });
      } catch {
        // aborted — a newer request owns the state
      }
    }, debounceMs);

    return () => clearTimeout(timer);
  }, [text, debounceMs]);

  useEffect(() => () => controller.current?.abort(), []);

  return state;
}
