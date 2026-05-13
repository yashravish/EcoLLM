'use client';

import { useQuery } from '@tanstack/react-query';
import { Activity, Cpu, Zap, Leaf } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { api } from '@/lib/api';
import { formatCO2, formatLatency } from '@/lib/utils';
import type { SystemMetrics } from '@/types';

function MetricTile({
  label,
  value,
  icon,
  loading,
}: {
  label: string;
  value: string;
  icon: React.ReactNode;
  loading: boolean;
}) {
  return (
    <Card>
      <CardContent className="pt-6">
        {loading ? (
          <LoadingSkeleton variant="card" />
        ) : (
          <div className="flex items-start gap-3">
            <div className="mt-0.5">{icon}</div>
            <div>
              <p className="font-mono text-2xl font-bold text-eco-50">{value}</p>
              <p className="mt-1 font-mono text-xs text-eco-400">{label}</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function AdminMetricsPage() {
  const { data, isLoading } = useQuery<SystemMetrics>({
    queryKey: ['admin', 'metrics'],
    queryFn: () => api.get('/admin/metrics'),
    refetchInterval: 30 * 1000,
    staleTime: 15 * 1000,
  });

  const gpuEntries = Object.entries(data?.gpu_utilization ?? {});

  return (
    <div className="space-y-6">
      <h1 className="font-mono text-base font-semibold uppercase tracking-widest text-eco-300">System Metrics</h1>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricTile
          label="Requests today"
          value={data?.total_requests_today.toLocaleString() ?? '—'}
          icon={<Activity className="h-5 w-5 text-blue-500" />}
          loading={isLoading}
        />
        <MetricTile
          label="Active models"
          value={data?.active_models.toString() ?? '—'}
          icon={<Cpu className="h-5 w-5 text-eco-300" />}
          loading={isLoading}
        />
        <MetricTile
          label="Avg latency"
          value={data ? formatLatency(data.avg_latency_ms) : '—'}
          icon={<Zap className="h-5 w-5 text-amber-500" />}
          loading={isLoading}
        />
        <MetricTile
          label="CO2e today"
          value={data ? formatCO2(data.total_co2e_today_grams) : '—'}
          icon={<Leaf className="h-5 w-5 text-accent" />}
          loading={isLoading}
        />
      </div>

      {gpuEntries.length > 0 && (
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">GPU Utilization</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {gpuEntries.map(([model, pct]) => (
                <div key={model}>
                  <div className="mb-1 flex items-center justify-between text-sm">
                    <span className="font-mono text-xs text-eco-400">{model}</span>
                    <span className="font-mono text-xs font-medium text-eco-200">{pct.toFixed(1)}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-eco-700">
                    <div
                      className="h-full rounded-full bg-accent transition-all"
                      style={{ width: `${Math.min(pct, 100)}%` }}
                      role="progressbar"
                      aria-valuenow={pct}
                      aria-valuemin={0}
                      aria-valuemax={100}
                      aria-label={`${model} GPU utilization`}
                    />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Cache Performance</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <LoadingSkeleton variant="text" lines={2} />
          ) : (
            <div className="flex items-center gap-4">
              <div className="font-mono text-3xl font-bold text-eco-50">
                {data ? `${(data.cache_hit_rate * 100).toFixed(1)}%` : '—'}
              </div>
              <div className="font-mono text-xs text-eco-500">
                Response cache hit rate
                <br />
                <span className="text-eco-600">Target: &gt;15%</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
