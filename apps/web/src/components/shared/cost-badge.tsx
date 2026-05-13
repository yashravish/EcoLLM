import { cn, formatCost } from '@/lib/utils';

function getCostColor(usd: number): string {
  if (usd < 0.001) return 'bg-accent/15 text-accent';
  if (usd < 0.01) return 'bg-amber-500/15 text-amber-400';
  return 'bg-eco-700 text-eco-300';
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
