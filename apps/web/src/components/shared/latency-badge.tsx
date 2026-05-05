import { cn } from '@/lib/utils';
import { formatLatency } from '@/lib/utils';

function getLatencyColor(ms: number): string {
  if (ms < 500) return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
  if (ms < 1500) return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
  return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400';
}

export function LatencyBadge({ latencyMs }: { latencyMs: number | undefined | null }) {
  if (latencyMs == null || isNaN(latencyMs)) {
    return <span className="text-xs text-gray-400">—</span>;
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
