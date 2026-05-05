'use client';

import { ImpactSummary } from '@/components/carbon/impact-summary';
import { CarbonChart } from '@/components/carbon/carbon-chart';
import { CarbonComparison } from '@/components/carbon/carbon-comparison';
import { ModelEnergyBreakdown } from '@/components/carbon/model-energy-breakdown';
import { EmptyState } from '@/components/shared/empty-state';
import { useCarbon } from '@/lib/hooks/use-carbon';

export default function CarbonPage() {
  const { data, isLoading } = useCarbon('monthly');

  const hasData = !isLoading && data !== undefined;
  const noData = hasData && data.total_co2e_grams === 0;

  if (hasData && noData) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Carbon Impact</h1>
        <EmptyState
          title="No carbon data yet"
          description="Carbon metrics will appear here once you start sending inference requests."
        />
      </div>
    );
  }

  const totalRequests =
    data?.model_energy_breakdown?.reduce((sum, m) => sum + m.request_count, 0) ?? 0;

  const avgCO2eGrams =
    totalRequests > 0 ? (data?.total_co2e_grams ?? 0) / totalRequests : 0;

  const savedGrams = data
    ? data.gpt4_equivalent_co2e_grams - data.total_co2e_grams
    : 0;

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Carbon Impact</h1>

      {/* Hero: CO2 avoided */}
      <ImpactSummary
        totalCO2eSavedGrams={savedGrams > 0 ? savedGrams : 0}
        loading={isLoading}
      />

      {/* Dual-line chart */}
      <CarbonChart
        data={data?.daily_breakdown ?? []}
        loading={isLoading}
      />

      {/* Comparison + model breakdown */}
      <div className="grid gap-4 lg:grid-cols-2">
        <CarbonComparison
          avgCO2eGrams={avgCO2eGrams}
          gpt4EquivalentGrams={
            totalRequests > 0
              ? (data?.gpt4_equivalent_co2e_grams ?? 0) / totalRequests
              : 0
          }
          savingsPercent={data?.savings_percent ?? 0}
          gridRegion={data?.grid_region ?? 'US-EAST'}
          gridIntensity={data?.grid_carbon_intensity ?? 450}
          loading={isLoading}
        />
        <ModelEnergyBreakdown
          data={data?.model_energy_breakdown ?? []}
          loading={isLoading}
        />
      </div>
    </div>
  );
}
