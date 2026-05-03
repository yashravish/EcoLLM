import { cn } from '@/lib/utils';
import { formatCost } from '@/lib/utils';

function getCostColor(usd: number): string {
  if (usd < 0.001) return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400';
  if (usd < 0.01) return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400';
  return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
}

export function CostBadge({ costUsd }: { costUsd: number }) {
  const colorClass = getCostColor(costUsd);
  const label = formatCost(costUsd);

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        colorClass,
      )}
      aria-label={`Cost: ${label}`}
    >
      {label}
    </span>
  );
}
