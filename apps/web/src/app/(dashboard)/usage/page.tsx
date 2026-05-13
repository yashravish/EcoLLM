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
      <h1 className="font-mono text-base font-semibold uppercase tracking-widest text-eco-300">Usage</h1>

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
                <LoadingSkeleton variant="card" />
              ) : (
                <>
                  <p className="font-mono text-2xl font-bold text-eco-50">{value}</p>
                  <p className="mt-1 font-mono text-xs text-eco-400">{label}</p>
                </>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {noData ? (
        <EmptyState
          title="No usage data yet"
          description="Usage metrics will appear once you start sending inference requests."
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

      {hasData && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Daily Breakdown</CardTitle>
          </CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="Daily usage breakdown">
                <thead>
                  <tr className="border-b border-eco-700">
                    {['Date', 'Requests', 'Energy', 'CO2e', 'Cost'].map((h) => (
                      <th
                        key={h}
                        className="py-3 px-4 text-left font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500 last:text-right"
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
                      className="border-b border-eco-700/50 last:border-0 hover:bg-eco-800/40 transition-colors"
                    >
                      <td className="py-2.5 px-4 font-mono text-xs text-eco-300">{row.date}</td>
                      <td className="py-2.5 px-4 font-mono text-xs text-eco-200">
                        {(row.requests ?? 0).toLocaleString()}
                      </td>
                      <td className="py-2.5 px-4 font-mono text-xs text-eco-200">
                        {((row.energy_kwh ?? 0) * 1000).toFixed(2)} Wh
                      </td>
                      <td className="py-2.5 px-4 font-mono text-xs text-eco-200">
                        {formatCO2(row.co2e_grams ?? 0)}
                      </td>
                      <td className="py-2.5 px-4 text-right font-mono text-xs text-eco-200">
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
