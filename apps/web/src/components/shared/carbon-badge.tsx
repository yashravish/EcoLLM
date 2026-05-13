import { cn, formatCO2 } from '@/lib/utils';

function getCO2Color(grams: number): string {
  if (grams < 5) return 'bg-accent/15 text-accent';
  if (grams < 20) return 'bg-amber-500/15 text-amber-400';
  return 'bg-red-500/15 text-red-400';
}

export function CarbonBadge({ co2eGrams }: { co2eGrams: number }) {
  const colorClass = getCO2Color(co2eGrams);
  const label = formatCO2(co2eGrams);

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
        colorClass,
      )}
      aria-label={`${label} carbon emissions`}
    >
      {label}
    </span>
  );
}
