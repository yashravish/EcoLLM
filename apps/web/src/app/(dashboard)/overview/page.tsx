'use client';

import { useState } from 'react';
import { Activity, Clock, Leaf, DollarSign } from 'lucide-react';
import { MetricCard } from '@/components/dashboard/metric-card';
import { UsageChart } from '@/components/dashboard/usage-chart';
import { ModelDistribution } from '@/components/dashboard/model-distribution';
import { RequestLogTable } from '@/components/dashboard/request-log-table';
import { EmptyState } from '@/components/shared/empty-state';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useUsage } from '@/lib/hooks/use-usage';
import { useRequests } from '@/lib/hooks/use-requests';
import { formatCO2, formatCost, formatLatency, dateRange } from '@/lib/utils';

export default function OverviewPage() {
  const [period, setPeriod] = useState<'7d' | '30d'>('7d');

  const days = period === '7d' ? 7 : 30;
  const { from, to } = dateRange(days);

  const { data: usage, isLoading: usageLoading } = useUsage('daily', from, to);
  const { data: requestList, isLoading: requestsLoading } = useRequests({
    limit: 10,
    sort: 'created_at:desc',
  });

  const summary = usage?.summary;
  const hasRequests = (requestList?.requests?.length ?? 0) > 0;

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Overview</h1>

      {/* Metric Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label="Total Requests"
          value={summary ? summary.total_requests.toLocaleString() : '—'}
          icon={<Activity className="h-5 w-5 text-green-600" />}
          loading={usageLoading}
        />
        <MetricCard
          label="Avg Latency"
          value={summary ? formatLatency(summary.avg_latency_ms) : '—'}
          icon={<Clock className="h-5 w-5 text-blue-500" />}
          loading={usageLoading}
        />
        <MetricCard
          label="CO2e Saved"
          value={
            summary
              ? formatCO2(summary.total_co2e_grams)
              : '—'
          }
          subtext="vs GPT-4 equivalent"
          icon={<Leaf className="h-5 w-5 text-green-600" />}
          color="green"
          loading={usageLoading}
        />
        <MetricCard
          label="Total Cost"
          value={summary ? formatCost(summary.total_cost_usd) : '—'}
          subtext={summary ? `Cache hit ${(summary.cache_hit_rate * 100).toFixed(0)}%` : undefined}
          icon={<DollarSign className="h-5 w-5 text-amber-500" />}
          loading={usageLoading}
        />
      </div>

      {/* Charts Row */}
      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <UsageChart
            data={usage?.daily_breakdown ?? []}
            period={period}
            onPeriodChange={setPeriod}
            loading={usageLoading}
          />
        </div>
        <ModelDistribution
          data={usage?.model_distribution ?? {}}
          loading={usageLoading}
        />
      </div>

      {/* Recent Requests */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Recent Requests</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {!requestsLoading && !hasRequests ? (
            <EmptyState
              title="No requests yet"
              description="Send your first request using the API."
            />
          ) : (
            <RequestLogTable
              requests={requestList?.requests ?? []}
              loading={requestsLoading}
            />
          )}
        </CardContent>
      </Card>
    </div>
  );
}
