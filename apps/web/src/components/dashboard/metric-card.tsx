import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

const colorClasses = {
  default: 'text-gray-900 dark:text-white',
  green: 'text-green-700 dark:text-green-400',
  amber: 'text-amber-700 dark:text-amber-400',
  red: 'text-red-700 dark:text-red-400',
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
  color?: 'default' | 'green' | 'amber' | 'red';
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
    return (
      <Card>
        <CardContent className="pt-6">
          <LoadingSkeleton />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card aria-label={ariaLabel}>
      <CardContent className="pt-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{label}</p>
            <p className={cn('mt-1 text-2xl font-bold tabular-nums', colorClasses[color])}>
              {value}
            </p>
            {(subtext || trend) && (
              <div className="mt-1 flex items-center gap-1.5">
                {trend && <TrendIcon direction={trend.direction} value={trend.value} />}
                {subtext && (
                  <span className="text-xs text-gray-500 dark:text-gray-400">{subtext}</span>
                )}
              </div>
            )}
          </div>
          {icon && (
            <div
              className="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-800"
              aria-hidden="true"
            >
              {icon}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function TrendIcon({ direction, value }: TrendProps) {
  if (direction === 'up') {
    return (
      <span className="flex items-center gap-0.5 text-xs font-medium text-green-600 dark:text-green-400">
        <TrendingUp className="h-3 w-3" aria-hidden="true" />
        {value}
      </span>
    );
  }
  if (direction === 'down') {
    return (
      <span className="flex items-center gap-0.5 text-xs font-medium text-red-600 dark:text-red-400">
        <TrendingDown className="h-3 w-3" aria-hidden="true" />
        {value}
      </span>
    );
  }
  return (
    <span className="flex items-center gap-0.5 text-xs font-medium text-gray-500">
      <Minus className="h-3 w-3" aria-hidden="true" />
      {value}
    </span>
  );
}

function LoadingSkeleton() {
  return (
    <div aria-busy="true" aria-label="Loading metric" className="space-y-2">
      <Skeleton className="h-4 w-24" />
      <Skeleton className="h-8 w-32" />
      <Skeleton className="h-3 w-16" />
    </div>
  );
}
