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
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-lg font-semibold text-eco-50">Overview</h1>
        <p className="mt-0.5 text-xs text-eco-400 font-mono">Carbon-aware LLM inference — current month</p>
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          label="Total Requests"
          value={summary?.total_requests != null ? summary.total_requests.toLocaleString() : '—'}
          icon={<Activity className="h-4 w-4 text-eco-300" />}
          loading={usageLoading}
        />
        <MetricCard
          label="Avg Latency"
          value={summary?.avg_latency_ms != null ? formatLatency(summary.avg_latency_ms) : '—'}
          icon={<Clock className="h-4 w-4 text-blue-400" />}
          color="blue"
          loading={usageLoading}
        />
        <MetricCard
          label="CO₂e Saved"
          value={summary?.total_co2e_grams != null ? formatCO2(summary.total_co2e_grams) : '—'}
          subtext="vs GPT-4 equivalent"
          icon={<Leaf className="h-4 w-4 text-accent" />}
          color="green"
          loading={usageLoading}
        />
        <MetricCard
          label="Total Cost"
          value={summary?.total_cost_usd != null ? formatCost(summary.total_cost_usd) : '—'}
          subtext={summary?.cache_hit_rate != null ? `Cache hit ${(summary.cache_hit_rate * 100).toFixed(0)}%` : undefined}
          icon={<DollarSign className="h-4 w-4 text-amber-400" />}
          color="amber"
          loading={usageLoading}
        />
      </div>

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
