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
  const treesLabel = trees > 0 ? `${trees.toLocaleString()} ${trees === 1 ? 'tree' : 'trees'}` : '< 1 tree';
  const label = formatCO2(totalCO2eSavedGrams);
  const ariaLabel = `${label} CO2e avoided this period. Equivalent to ${treesLabel} absorbing CO₂ for a day.`;

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
          <div className="flex items-center gap-2 text-accent">
            <TreePine className="h-6 w-6" aria-hidden="true" />
            <span className="font-mono text-xs font-medium uppercase tracking-widest">Carbon Avoided</span>
          </div>
          <p className="font-mono text-4xl font-bold tabular-nums text-accent">
            {label}
          </p>
          <p className="font-mono text-xs text-eco-500">
            Equivalent to{' '}
            <span className="font-semibold text-eco-200">
              {treesLabel}
            </span>{' '}
            absorbing CO₂ for a day
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
