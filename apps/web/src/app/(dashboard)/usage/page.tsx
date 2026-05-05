'use client';

import { useState } from 'react';
import { BarChart3 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { EmptyState } from '@/components/shared/empty-state';
import { UsageChart } from '@/components/dashboard/usage-chart';
import { ModelDistribution } from '@/components/dashboard/model-distribution';
import { useUsage } from '@/lib/hooks/use-usage';
import { formatCO2, formatCost, formatLatency, dateRange } from '@/lib/utils';

export default function UsagePage() {
  const [period, setPeriod] = useState<'7d' | '30d'>('30d');

  const days = period === '7d' ? 7 : 30;
  const { from, to } = dateRange(days);

  const { data, isLoading } = useUsage('daily', from, to);

  const summary = data?.summary;
  const totalRequests = summary?.total_requests ?? 0;
  const hasData = !isLoading && totalRequests > 0;
  const noData = !isLoading && data !== undefined && totalRequests === 0;

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Usage</h1>

      {/* Summary tiles */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          {
            label: 'Total Requests',
            value: isLoading ? '—' : (summary?.total_requests?.toLocaleString() ?? '—'),
          },
          {
            label: 'Total Tokens',
            value: isLoading ? '—' : (summary?.total_tokens?.toLocaleString() ?? '—'),
          },
          {
            label: 'Avg Latency',
            value: isLoading ? '—' : formatLatency(summary?.avg_latency_ms),
          },
          {
            label: 'Cache Hit Rate',
            value: isLoading || summary == null
              ? '—'
              : `${((summary.cache_hit_rate ?? 0) * 100).toFixed(1)}%`,
          },
        ].map(({ label, value }) => (
          <Card key={label}>
            <CardContent className="pt-6">
              {isLoading ? (
                <LoadingSkeleton variant="stat" />
              ) : (
                <>
                  <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
                  <p className="mt-1 text-sm text-gray-500">{label}</p>
                </>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Charts */}
      {noData ? (
        <EmptyState
          title="No usage data yet"
          description="Usage metrics will appear once you start sending inference requests."
          icon={<BarChart3 className="h-8 w-8 text-gray-400" />}
        />
      ) : (
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <UsageChart
              data={data?.daily_breakdown ?? []}
              period={period}
              onPeriodChange={setPeriod}
              loading={isLoading}
            />
          </div>
          <ModelDistribution
            data={data?.model_distribution ?? {}}
            loading={isLoading}
          />
        </div>
      )}

      {/* Daily breakdown table */}
      {hasData && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Daily Breakdown</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="Daily usage breakdown">
                <thead>
                  <tr className="border-b border-gray-100 dark:border-gray-800">
                    {['Date', 'Requests', 'Energy', 'CO2e', 'Cost'].map((h) => (
                      <th
                        key={h}
                        className="py-3 px-4 text-left text-xs font-medium uppercase tracking-wide text-gray-500 last:text-right"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(data?.daily_breakdown ?? []).map((row) => (
                    <tr
                      key={row.date}
                      className="border-b border-gray-50 last:border-0 dark:border-gray-800/50"
                    >
                      <td className="py-2.5 px-4 text-gray-700 dark:text-gray-300">{row.date}</td>
                      <td className="py-2.5 px-4 text-gray-700 dark:text-gray-300">
                        {(row.requests ?? 0).toLocaleString()}
                      </td>
                      <td className="py-2.5 px-4 text-gray-700 dark:text-gray-300">
                        {((row.energy_kwh ?? 0) * 1000).toFixed(2)} Wh
                      </td>
                      <td className="py-2.5 px-4 text-gray-700 dark:text-gray-300">
                        {formatCO2(row.co2e_grams ?? 0)}
                      </td>
                      <td className="py-2.5 px-4 text-right text-gray-700 dark:text-gray-300">
                        {formatCost(row.cost_usd ?? 0)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
