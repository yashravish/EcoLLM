'use client';

import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { api } from '@/lib/api';
import type { RouteConfig } from '@/types';

const routeSchema = z.object({
  energy_weight: z.coerce.number().min(0.35, 'Energy weight must be at least 0.35').max(1),
  cost_weight: z.coerce.number().min(0).max(1),
  quality_weight: z.coerce.number().min(0).max(1),
  latency_weight: z.coerce.number().min(0).max(1),
}).refine(
  (v) => Math.abs(v.energy_weight + v.cost_weight + v.quality_weight + v.latency_weight - 1) < 0.001,
  { message: 'Weights must sum to 1.0', path: ['energy_weight'] },
);

type RouteForm = z.infer<typeof routeSchema>;

export default function AdminRoutesPage() {
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery<RouteConfig>({
    queryKey: ['admin', 'routes'],
    queryFn: () => api.get('/admin/routes'),
    staleTime: 60 * 1000,
  });

  const mutation = useMutation({
    mutationFn: (config: RouteForm) => api.patch('/admin/routes', config),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['admin', 'routes'] }),
  });

  const { register, handleSubmit, reset, formState: { errors, isDirty } } = useForm<RouteForm>({
    resolver: zodResolver(routeSchema),
    defaultValues: { energy_weight: 0.4, cost_weight: 0.3, quality_weight: 0.2, latency_weight: 0.1 },
  });

  useEffect(() => {
    if (data) reset(data);
  }, [data, reset]);

  return (
    <div className="space-y-6">
      <h1 className="font-mono text-base font-semibold uppercase tracking-widest text-eco-300">Routing Configuration</h1>

      <Card className="max-w-lg">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Scoring Weights</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <LoadingSkeleton variant="text" lines={4} />
          ) : (
            <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate className="space-y-4">
              <p className="text-xs text-eco-500">
                Weights must sum to 1.0. Energy weight cannot go below 0.35 (architecture constraint).
              </p>

              {(
                [
                  { name: 'energy_weight', label: 'Energy weight (min 0.35)' },
                  { name: 'cost_weight', label: 'Cost weight' },
                  { name: 'quality_weight', label: 'Quality weight' },
                  { name: 'latency_weight', label: 'Latency weight' },
                ] as const
              ).map(({ name, label }) => (
                <Input
                  key={name}
                  label={label}
                  type="number"
                  step="0.05"
                  min="0"
                  max="1"
                  error={errors[name]?.message}
                  {...register(name)}
                />
              ))}

              {errors.energy_weight?.message?.includes('sum') && (
                <p role="alert" className="text-sm text-red-400">
                  {errors.energy_weight.message}
                </p>
              )}

              <div className="flex justify-end gap-3 pt-2">
                <Button type="button" variant="secondary" onClick={() => reset(data)} disabled={!isDirty}>
                  Reset
                </Button>
                <Button type="submit" loading={mutation.isPending} disabled={!isDirty}>
                  Save
                </Button>
              </div>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
