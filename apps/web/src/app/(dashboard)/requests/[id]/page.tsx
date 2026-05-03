'use client';

import { useParams, useRouter } from 'next/navigation';
import { ArrowLeft, CheckCircle2, XCircle, Zap, Leaf, DollarSign, Clock, GitFork, AlertTriangle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { useRequest } from '@/lib/hooks/use-requests';
import { cn } from '@/lib/utils';

// ── RouteTraceTimeline ────────────────────────────────────────────────────────

interface TraceStep {
  label: string;
  value: string | null;
  icon: React.ReactNode;
  color: string;
}

function RouteTraceTimeline({ record }: { record: NonNullable<ReturnType<typeof useRequest>['data']> }) {
  const steps: TraceStep[] = [
    {
      label: 'Task Classified',
      value: `${record.task_type} (complexity ${record.complexity}/10)`,
      icon: <GitFork className="h-3.5 w-3.5" />,
      color: 'text-blue-600 bg-blue-50 dark:bg-blue-900/20 dark:text-blue-400',
    },
    {
      label: 'Model Selected',
      value: record.model_selected,
      icon: <Zap className="h-3.5 w-3.5" />,
      color: 'text-green-600 bg-green-50 dark:bg-green-900/20 dark:text-green-400',
    },
    ...(record.used_fallback && record.model_fallback
      ? [{
          label: 'Fallback Used',
          value: record.model_fallback,
          icon: <AlertTriangle className="h-3.5 w-3.5" />,
          color: 'text-amber-600 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400',
        }]
      : []),
    {
      label: 'Routing Score',
      value: `${(record.routing_score * 100).toFixed(1)}% confidence`,
      icon: record.routing_score >= 0.7
        ? <CheckCircle2 className="h-3.5 w-3.5" />
        : <AlertTriangle className="h-3.5 w-3.5" />,
      color: record.routing_score >= 0.7
        ? 'text-green-600 bg-green-50 dark:bg-green-900/20 dark:text-green-400'
        : 'text-amber-600 bg-amber-50 dark:bg-amber-900/20 dark:text-amber-400',
    },
    {
      label: 'Cache',
      value: record.cache_hit ? 'Hit — served from cache' : 'Miss — full inference',
      icon: record.cache_hit ? <CheckCircle2 className="h-3.5 w-3.5" /> : <XCircle className="h-3.5 w-3.5" />,
      color: record.cache_hit
        ? 'text-green-600 bg-green-50 dark:bg-green-900/20 dark:text-green-400'
        : 'text-gray-500 bg-gray-50 dark:bg-gray-800 dark:text-gray-400',
    },
  ];

  return (
    <div className="space-y-0">
      {steps.map((step, i) => (
        <div key={step.label} className="flex gap-3">
          {/* Spine */}
          <div className="flex flex-col items-center">
            <div className={cn('flex h-7 w-7 items-center justify-center rounded-full flex-shrink-0', step.color)}>
              {step.icon}
            </div>
            {i < steps.length - 1 && (
              <div className="w-px flex-1 bg-gray-200 dark:bg-gray-700 my-1" />
            )}
          </div>
          {/* Content */}
          <div className="pb-4 pt-0.5 min-w-0">
            <p className="text-xs font-medium text-gray-500 dark:text-gray-400">{step.label}</p>
            <p className="text-sm text-gray-900 dark:text-white truncate">{step.value}</p>
          </div>
        </div>
      ))}
    </div>
  );
}

// ── Metric tile ───────────────────────────────────────────────────────────────

function MetricTile({ icon, label, value, sub }: {
  icon: React.ReactNode;
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="rounded-lg border border-gray-100 bg-gray-50 p-3 dark:border-gray-700 dark:bg-gray-800">
      <div className="mb-1 flex items-center gap-1.5 text-xs text-gray-500">{icon}{label}</div>
      <p className="text-base font-semibold text-gray-900 dark:text-white">{value}</p>
      {sub && <p className="text-xs text-gray-400">{sub}</p>}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function RequestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { data: record, isLoading, isError } = useRequest(id);

  if (isLoading) {
    return (
      <div className="max-w-5xl mx-auto space-y-4">
        <Skeleton className="h-8 w-48" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <Skeleton className="h-64 lg:col-span-2" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  if (isError || !record) {
    return (
      <div className="max-w-5xl mx-auto text-center py-16">
        <XCircle className="mx-auto h-10 w-10 text-red-400 mb-3" />
        <p className="text-gray-500">Request not found or you don&apos;t have access.</p>
        <Button variant="ghost" className="mt-4" onClick={() => router.back()}>Go back</Button>
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto space-y-5">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => router.back()} aria-label="Back">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="min-w-0">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-white truncate">
            Request <span className="font-mono text-base">{record.request_id || record.id}</span>
          </h1>
          <p className="text-xs text-gray-400">{new Date(record.created_at).toLocaleString()}</p>
        </div>
        <Badge variant={record.status === 'completed' ? 'default' : 'danger'} className="ml-auto flex-shrink-0">
          {record.status}
        </Badge>
      </div>

      {/* Metrics row */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <MetricTile
          icon={<Clock className="h-3 w-3" />}
          label="Latency"
          value={`${record.latency_ms} ms`}
        />
        <MetricTile
          icon={<Leaf className="h-3 w-3 text-green-500" />}
          label="CO₂e"
          value={record.co2e_grams != null ? `${record.co2e_grams.toFixed(4)} g` : '—'}
        />
        <MetricTile
          icon={<Zap className="h-3 w-3 text-blue-500" />}
          label="Energy"
          value={record.energy_kwh != null ? `${(record.energy_kwh * 1000).toFixed(4)} Wh` : '—'}
        />
        <MetricTile
          icon={<DollarSign className="h-3 w-3 text-purple-500" />}
          label="Cost"
          value={record.cost_usd != null ? `$${record.cost_usd.toFixed(6)}` : '—'}
          sub={`${record.total_tokens} tokens`}
        />
      </div>

      {/* Main grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {/* Prompt + Response */}
        <div className="lg:col-span-2 space-y-4">
          <Card className="p-4">
            <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">Prompt</h2>
            {record.prompt_optimized && record.prompt_optimized !== record.prompt_original && (
              <details className="mb-2 text-xs text-gray-400">
                <summary className="cursor-pointer hover:text-gray-600">Show original</summary>
                <pre className="mt-1 whitespace-pre-wrap text-gray-500 text-xs bg-gray-50 rounded p-2">
                  {record.prompt_original}
                </pre>
              </details>
            )}
            <pre className="whitespace-pre-wrap text-sm text-gray-800 dark:text-gray-200">
              {record.prompt_optimized || record.prompt_original}
            </pre>
          </Card>

          {record.response_text && (
            <Card className="p-4">
              <h2 className="mb-2 text-xs font-semibold uppercase tracking-wider text-gray-500">Response</h2>
              <pre className="whitespace-pre-wrap text-sm text-gray-800 dark:text-gray-200">
                {record.response_text}
              </pre>
            </Card>
          )}

          {/* Token breakdown */}
          <Card className="p-4">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wider text-gray-500">Tokens</h2>
            <div className="flex gap-6 text-sm">
              <div>
                <p className="text-gray-500 text-xs">Prompt</p>
                <p className="font-semibold">{record.prompt_tokens.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-gray-500 text-xs">Completion</p>
                <p className="font-semibold">{record.completion_tokens.toLocaleString()}</p>
              </div>
              <div>
                <p className="text-gray-500 text-xs">Total</p>
                <p className="font-semibold">{record.total_tokens.toLocaleString()}</p>
              </div>
            </div>
          </Card>
        </div>

        {/* Route trace */}
        <div>
          <Card className="p-4">
            <h2 className="mb-4 text-xs font-semibold uppercase tracking-wider text-gray-500">Route Trace</h2>
            <RouteTraceTimeline record={record} />
          </Card>
        </div>
      </div>
    </div>
  );
}
