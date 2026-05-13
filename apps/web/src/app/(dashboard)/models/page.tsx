'use client';

import { Cpu } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { EmptyState } from '@/components/shared/empty-state';
import { useModels } from '@/lib/hooks/use-models';
import type { Model } from '@/types';

function ModelRow({ model }: { model: Model }) {
  const statusColor = model.status === 'active' ? 'bg-accent/15 text-accent' : 'bg-eco-700 text-eco-400';

  return (
    <tr className="border-b border-eco-700 last:border-0 hover:bg-eco-800/40 transition-colors">
      <td className="py-3 px-4">
        <div className="flex items-center gap-2">
          <Cpu className="h-4 w-4 text-eco-500" aria-hidden="true" />
          <span className="font-mono text-xs font-medium text-eco-100">{model.name}</span>
        </div>
      </td>
      <td className="py-3 px-4">
        <span className={`inline-flex items-center rounded-full px-2 py-0.5 font-mono text-xs font-medium ${statusColor}`}>
          {model.status}
        </span>
      </td>
      <td className="py-3 px-4 font-mono text-xs text-eco-400">
        {model.tasks.join(', ')}
      </td>
      <td className="py-3 px-4 text-right font-mono text-xs text-eco-300">
        {(model.quality_benchmark * 100).toFixed(0)}%
      </td>
      <td className="py-3 px-4 text-right font-mono text-xs text-eco-300">
        {model.latency_p95_ms}ms
      </td>
      <td className="py-3 px-4 text-right font-mono text-xs text-eco-300">
        {(model.energy_per_request_kwh * 1000).toFixed(2)} Wh
      </td>
      <td className="py-3 px-4 text-right font-mono text-xs text-eco-300">
        {model.max_context.toLocaleString()}
      </td>
    </tr>
  );
}

export default function ModelsPage() {
  const { data, isLoading } = useModels();

  const hasModels = !isLoading && (data?.models?.length ?? 0) > 0;

  return (
    <div className="space-y-6">
      <h1 className="font-mono text-base font-semibold uppercase tracking-widest text-eco-300">Model Registry</h1>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Available Models</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading && <LoadingSkeleton variant="table" lines={4} />}

          {!isLoading && !hasModels && (
            <EmptyState
              title="No models registered"
              description="Models are added via the admin panel or model registry seeder."
            />
          )}

          {hasModels && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="Model registry">
                <thead>
                  <tr className="border-b border-eco-700">
                    <th className="py-3 px-4 text-left font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Model</th>
                    <th className="py-3 px-4 text-left font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Status</th>
                    <th className="py-3 px-4 text-left font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Tasks</th>
                    <th className="py-3 px-4 text-right font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Quality</th>
                    <th className="py-3 px-4 text-right font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Latency p95</th>
                    <th className="py-3 px-4 text-right font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Energy/req</th>
                    <th className="py-3 px-4 text-right font-mono text-[10px] font-medium uppercase tracking-widest text-eco-500">Max ctx</th>
                  </tr>
                </thead>
                <tbody>
                  {data!.models.map((m) => (
                    <ModelRow key={m.id} model={m} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Routing Priority</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            {[
              { label: 'Energy', weight: '40%', color: 'text-accent' },
              { label: 'Cost', weight: '30%', color: 'text-amber-400' },
              { label: 'Quality', weight: '20%', color: 'text-blue-400' },
              { label: 'Latency', weight: '10%', color: 'text-eco-300' },
            ].map(({ label, weight, color }) => (
              <div key={label} className="rounded-lg border border-eco-700 bg-eco-900/50 p-4 text-center">
                <p className={`font-mono text-2xl font-bold ${color}`}>{weight}</p>
                <p className="mt-1 font-mono text-xs text-eco-500">{label}</p>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
