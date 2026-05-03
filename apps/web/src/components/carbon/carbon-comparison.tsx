import { Card, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCO2 } from '@/lib/utils';

interface CarbonComparisonProps {
  avgCO2eGrams: number;
  gpt4EquivalentGrams: number;
  savingsPercent: number;
  gridRegion: string;
  gridIntensity: number;
  loading?: boolean;
}

export function CarbonComparison({
  avgCO2eGrams,
  gpt4EquivalentGrams,
  savingsPercent,
  gridRegion,
  gridIntensity,
  loading = false,
}: CarbonComparisonProps) {
  if (loading) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div aria-busy="true" aria-label="Loading carbon comparison" className="grid gap-6 sm:grid-cols-2">
            {[0, 1].map((i) => (
              <div key={i} className="space-y-3">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-8 w-24" />
                <Skeleton className="h-3 w-40" />
                <Skeleton className="h-3 w-36" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent className="pt-6">
        <div className="grid gap-6 sm:grid-cols-2">
          {/* Your emissions */}
          <div>
            <h3 className="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
              Your Emissions
            </h3>
            <p className="mt-2 text-2xl font-bold text-gray-900 dark:text-white">
              {formatCO2(avgCO2eGrams)}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">avg per request</p>
            <div className="mt-3 space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <p>
                <span className="font-medium">Grid:</span> {gridRegion}
              </p>
              <p>
                <span className="font-medium">Intensity:</span>{' '}
                {gridIntensity.toFixed(0)} gCO₂/kWh
              </p>
            </div>
          </div>

          {/* vs GPT-4 */}
          <div className="border-l border-gray-100 pl-6 dark:border-gray-700">
            <h3 className="text-sm font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
              vs GPT-4
            </h3>
            <p className="mt-2 text-2xl font-bold text-gray-400 dark:text-gray-500">
              {formatCO2(gpt4EquivalentGrams)}
            </p>
            <p className="text-sm text-gray-500 dark:text-gray-400">GPT-4 avg per request</p>
            <p
              className="mt-3 text-3xl font-extrabold text-green-600 dark:text-green-400"
              aria-label={`${savingsPercent.toFixed(0)}% lower emissions than GPT-4`}
            >
              {savingsPercent.toFixed(0)}% less
            </p>
            <p className="text-xs text-gray-500 dark:text-gray-400">carbon savings vs GPT-4</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
