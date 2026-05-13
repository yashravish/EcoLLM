import { cn, formatLatency } from '@/lib/utils';

function getLatencyColor(ms: number): string {
  if (ms < 500) return 'bg-accent/15 text-accent';
  if (ms < 1500) return 'bg-amber-500/15 text-amber-400';
  return 'bg-red-500/15 text-red-400';
}

export function LatencyBadge({ latencyMs }: { latencyMs: number | undefined | null }) {
  if (latencyMs == null || isNaN(latencyMs)) {
    return <span className="font-mono text-xs text-eco-500">—</span>;
  }
  const colorClass = getLatencyColor(latencyMs);
  const label = formatLatency(latencyMs);

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        colorClass,
      )}
      aria-label={`Latency: ${label}`}
    >
      {label}
    </span>
  );
}
