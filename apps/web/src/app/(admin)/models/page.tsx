'use client';

import { useState } from 'react';
import { Plus } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { EmptyState } from '@/components/shared/empty-state';
import { useModels } from '@/lib/hooks/use-models';
import { api } from '@/lib/api';
import type { Model } from '@/types';

const modelSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  tasks: z.string().min(1, 'At least one task required'),
  max_context: z.coerce.number().min(512),
  quality_benchmark: z.coerce.number().min(0).max(1),
  latency_p95_ms: z.coerce.number().min(1),
  energy_per_request_kwh: z.coerce.number().min(0),
});

type ModelForm = z.infer<typeof modelSchema>;

function CreateModelDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (data: ModelForm) =>
      api.post('/admin/models', {
        ...data,
        tasks: data.tasks.split(',').map((t) => t.trim()),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['models'] });
      onClose();
    },
  });

  const { register, handleSubmit, formState: { errors }, reset } = useForm<ModelForm>({
    resolver: zodResolver(modelSchema),
  });

  const handleClose = () => { reset(); onClose(); };

  return (
    <Dialog open={open} onClose={handleClose}>
      <DialogContent onClose={handleClose} aria-labelledby="create-model-title">
        <DialogTitle id="create-model-title">Register Model</DialogTitle>
        <DialogDescription>Add a new model to the routing registry.</DialogDescription>
        <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate className="mt-4 space-y-4">
          <Input label="Model name" error={errors.name?.message} {...register('name')} />
          <Input label="Tasks (comma-separated)" placeholder="chat,summarise,code" error={errors.tasks?.message} {...register('tasks')} />
          <div className="grid grid-cols-2 gap-4">
            <Input label="Max context" type="number" error={errors.max_context?.message} {...register('max_context')} />
            <Input label="Quality (0–1)" type="number" step="0.01" error={errors.quality_benchmark?.message} {...register('quality_benchmark')} />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <Input label="Latency p95 (ms)" type="number" error={errors.latency_p95_ms?.message} {...register('latency_p95_ms')} />
            <Input label="Energy/req (kWh)" type="number" step="0.000001" error={errors.energy_per_request_kwh?.message} {...register('energy_per_request_kwh')} />
          </div>
          <DialogFooter>
            <Button type="button" variant="secondary" onClick={handleClose}>Cancel</Button>
            <Button type="submit" loading={mutation.isPending}>Register</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default function AdminModelsPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { data, isLoading } = useModels();
  const queryClient = useQueryClient();

  const toggleStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`/admin/models/${id}`, { status: status === 'active' ? 'inactive' : 'active' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['models'] }),
  });

  const models = data?.models ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Model Registry</h1>
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="h-4 w-4" aria-hidden="true" />
          Register Model
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Registered Models</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading && <LoadingSkeleton variant="table" lines={4} />}
          {!isLoading && models.length === 0 && (
            <EmptyState title="No models" description="Register your first model to enable routing." />
          )}
          {models.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="Model registry admin">
                <thead>
                  <tr className="border-b border-gray-100 dark:border-gray-800">
                    {['Name', 'Status', 'Tasks', 'Quality', 'Latency p95', 'Actions'].map((h) => (
                      <th key={h} className="py-3 px-4 text-left text-xs font-medium uppercase tracking-wide text-gray-500 last:text-right">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {models.map((m: Model) => (
                    <tr key={m.id} className="border-b border-gray-100 last:border-0 dark:border-gray-800">
                      <td className="py-3 px-4 font-medium text-gray-900 dark:text-white">{m.name}</td>
                      <td className="py-3 px-4">
                        <Badge variant={m.status === 'active' ? 'default' : 'secondary'}>{m.status}</Badge>
                      </td>
                      <td className="py-3 px-4 text-gray-600 dark:text-gray-400">{m.tasks.join(', ')}</td>
                      <td className="py-3 px-4 text-gray-600 dark:text-gray-400">{(m.quality_benchmark * 100).toFixed(0)}%</td>
                      <td className="py-3 px-4 text-gray-600 dark:text-gray-400">{m.latency_p95_ms}ms</td>
                      <td className="py-3 px-4 text-right">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => toggleStatus.mutate({ id: m.id, status: m.status })}
                          disabled={toggleStatus.isPending}
                        >
                          {m.status === 'active' ? 'Deactivate' : 'Activate'}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <CreateModelDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </div>
  );
}
