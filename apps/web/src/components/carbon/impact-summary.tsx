import { TreePine } from 'lucide-react';
import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCO2, co2ToTrees } from '@/lib/utils';

interface ImpactSummaryProps {
  totalCO2eSavedGrams: number;
  loading?: boolean;
}

export function ImpactSummary({ totalCO2eSavedGrams, loading = false }: ImpactSummaryProps) {
  const trees = co2ToTrees(totalCO2eSavedGrams);
  const label = formatCO2(totalCO2eSavedGrams);
  const ariaLabel = `${label} CO2e avoided this period. Equivalent to ${trees} tree-days of carbon absorption.`;

  if (loading) {
    return (
      <Card>
        <CardContent className="pt-8 pb-8">
          <div
            aria-busy="true"
            aria-label="Loading carbon impact"
            className="flex flex-col items-center gap-3"
          >
            <Skeleton className="h-6 w-40" />
            <Skeleton className="h-12 w-48" />
            <Skeleton className="h-5 w-56" />
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card aria-label={ariaLabel}>
      <CardContent className="pt-8 pb-8">
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
            <TreePine className="h-6 w-6" aria-hidden="true" />
            <span className="text-sm font-medium uppercase tracking-wide">Carbon Avoided</span>
          </div>
          <p className="text-4xl font-bold tabular-nums text-green-700 dark:text-green-400">
            {label}
          </p>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Equivalent to{' '}
            <span className="font-semibold text-gray-900 dark:text-white">
              {trees.toLocaleString()} {trees === 1 ? 'tree' : 'trees'}
            </span>{' '}
            absorbing CO₂ for a day
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
