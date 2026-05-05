import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { cn } from '@/lib/utils';

const stripClass: Record<string, string> = {
  default: 'accent-strip',
  green:   'accent-strip',
  amber:   'accent-strip-amber',
  blue:    'accent-strip-blue',
  red:     'accent-strip-red',
};

const valueClass: Record<string, string> = {
  default: 'text-eco-50',
  green:   'text-accent',
  amber:   'text-amber-400',
  blue:    'text-blue-400',
  red:     'text-red-400',
};

interface TrendProps {
  direction: 'up' | 'down' | 'flat';
  value: string;
}

interface MetricCardProps {
  label: string;
  value: string;
  subtext?: string;
  trend?: TrendProps;
  icon?: React.ReactNode;
  color?: 'default' | 'green' | 'amber' | 'blue' | 'red';
  loading?: boolean;
}

export function MetricCard({
  label,
  value,
  subtext,
  trend,
  icon,
  color = 'default',
  loading = false,
}: MetricCardProps) {
  const ariaLabel = [label, value, trend ? `${trend.direction} ${trend.value}` : '']
    .filter(Boolean)
    .join(', ');

  if (loading) {
    return <LoadingSkeleton />;
  }

  return (
    <div
      aria-label={ariaLabel}
      className="relative overflow-hidden rounded-lg border border-eco-600 bg-eco-800 p-5 transition-all duration-200 hover:border-eco-500 hover:bg-eco-700"
    >
      {/* Accent strip */}
      <div className={cn('absolute top-0 left-0 right-0 h-[2px]', stripClass[color])} aria-hidden="true" />

      <div className="mt-1 flex items-start justify-between">
        <div className="flex-1 min-w-0">
          <p className="text-[10px] font-semibold uppercase tracking-[0.15em] text-eco-400">
            {label}
          </p>
          <p className={cn('mt-2 font-mono text-2xl font-bold tabular-nums leading-none', valueClass[color])}>
            {value}
          </p>
          {(subtext || trend) && (
            <div className="mt-2 flex items-center gap-1.5">
              {trend && <TrendIndicator direction={trend.direction} value={trend.value} />}
              {subtext && (
                <span className="font-mono text-[10px] text-eco-400">{subtext}</span>
              )}
            </div>
          )}
        </div>

        {icon && (
          <div
            className="flex h-8 w-8 items-center justify-center rounded border border-eco-500 bg-eco-700 shrink-0"
            aria-hidden="true"
          >
            {icon}
          </div>
        )}
      </div>
    </div>
  );
}

function TrendIndicator({ direction, value }: TrendProps) {
  if (direction === 'up') {
    return (
      <span className="flex items-center gap-0.5 font-mono text-[10px] font-medium text-accent">
        <TrendingUp className="h-3 w-3" aria-hidden="true" />
        {value}
      </span>
    );
  }
  if (direction === 'down') {
    return (
      <span className="flex items-center gap-0.5 font-mono text-[10px] font-medium text-red-400">
        <TrendingDown className="h-3 w-3" aria-hidden="true" />
        {value}
      </span>
    );
  }
  return (
    <span className="flex items-center gap-0.5 font-mono text-[10px] font-medium text-eco-400">
      <Minus className="h-3 w-3" aria-hidden="true" />
      {value}
    </span>
  );
}

function LoadingSkeleton() {
  return (
    <div
      aria-busy="true"
      aria-label="Loading metric"
      className="relative overflow-hidden rounded-lg border border-eco-600 bg-eco-800 p-5"
    >
      <div className="absolute top-0 left-0 right-0 h-[2px] bg-eco-600" />
      <div className="mt-1 space-y-3">
        <div className="h-2.5 w-20 rounded bg-eco-700 animate-pulse" />
        <div className="h-7 w-28 rounded bg-eco-700 animate-pulse" />
        <div className="h-2 w-14 rounded bg-eco-700 animate-pulse" />
      </div>
    </div>
  );
}
