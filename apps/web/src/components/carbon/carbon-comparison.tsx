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
            <h3 className="font-mono text-xs font-semibold uppercase tracking-widest text-eco-500">
              Your Emissions
            </h3>
            <p className="mt-2 font-mono text-2xl font-bold text-eco-50">
              {formatCO2(avgCO2eGrams)}
            </p>
            <p className="font-mono text-xs text-eco-500">avg per request</p>
            <div className="mt-3 space-y-1 font-mono text-xs text-eco-400">
              <p>
                <span className="font-medium text-eco-300">Grid:</span> {gridRegion}
              </p>
              <p>
                <span className="font-medium text-eco-300">Intensity:</span>{' '}
                {gridIntensity.toFixed(0)} gCO₂/kWh
              </p>
            </div>
          </div>

          {/* vs GPT-4 */}
          <div className="border-l border-eco-700 pl-6">
            <h3 className="font-mono text-xs font-semibold uppercase tracking-widest text-eco-500">
              vs GPT-4
            </h3>
            <p className="mt-2 font-mono text-2xl font-bold text-eco-400">
              {formatCO2(gpt4EquivalentGrams)}
            </p>
            <p className="font-mono text-xs text-eco-500">GPT-4 avg per request</p>
            <p
              className="mt-3 font-mono text-3xl font-extrabold text-accent"
              aria-label={`${savingsPercent.toFixed(0)}% lower emissions than GPT-4`}
            >
              {savingsPercent.toFixed(0)}% less
            </p>
            <p className="font-mono text-xs text-eco-500">carbon savings vs GPT-4</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
