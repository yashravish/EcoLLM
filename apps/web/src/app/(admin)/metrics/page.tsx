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
          <LoadingSkeleton variant="stat" />
        ) : (
          <div className="flex items-start gap-3">
            <div className="mt-0.5">{icon}</div>
            <div>
              <p className="text-2xl font-bold text-gray-900 dark:text-white">{value}</p>
              <p className="mt-1 text-sm text-gray-500">{label}</p>
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
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">System Metrics</h1>

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
          icon={<Cpu className="h-5 w-5 text-purple-500" />}
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
          icon={<Leaf className="h-5 w-5 text-green-600" />}
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
                    <span className="text-gray-700 dark:text-gray-300">{model}</span>
                    <span className="font-medium text-gray-900 dark:text-white">{pct.toFixed(1)}%</span>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-gray-800">
                    <div
                      className="h-full rounded-full bg-green-500 transition-all"
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
            <LoadingSkeleton lines={2} />
          ) : (
            <div className="flex items-center gap-4">
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {data ? `${(data.cache_hit_rate * 100).toFixed(1)}%` : '—'}
              </div>
              <div className="text-sm text-gray-500">
                Response cache hit rate
                <br />
                <span className="text-xs">Target: &gt;15%</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
