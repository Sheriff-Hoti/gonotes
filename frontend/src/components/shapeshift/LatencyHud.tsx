
import { AnimatedNumber } from "@/components/intents/shared";
import { cn } from "@/lib/utils";

export function LatencyHud({
  latency,
  questions,
  model,
  cached,
}: {
  latency: number | null;
  questions: number;
  model: string;
  cached: boolean;
}) {
  if (latency === null) return null;
  return (
    <div
      className={cn(
        "pointer-events-none fixed end-[max(1rem,env(safe-area-inset-right))] bottom-[max(1rem,env(safe-area-inset-bottom))] font-mono text-[12px] leading-4 text-muted-foreground tabular-nums",
      )}
      aria-hidden
    >
      {cached ? "cached" : <AnimatedNumber value={latency} format={(n) => `${Math.round(n)}ms`} />} · {questions}q · {model}
    </div>
  );
}
