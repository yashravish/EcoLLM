'use client';

import { useState } from 'react';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { RequestLogTable } from '@/components/dashboard/request-log-table';
import { EmptyState } from '@/components/shared/empty-state';
import { useRequests } from '@/lib/hooks/use-requests';

const MODELS = ['phi-3-mini', 'mistral-7b', 'llama-13b', 'llama-70b'];
const TASK_TYPES = ['simple', 'medium', 'hard', 'specialized'];
const STATUSES = ['completed', 'failed', 'cached'];
const PAGE_SIZE = 20;

export default function RequestsPage() {
  const [model, setModel] = useState('');
  const [taskType, setTaskType] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);

  const offset = (page - 1) * PAGE_SIZE;
  const { data, isLoading } = useRequests({
    limit: PAGE_SIZE,
    offset,
    model: model || undefined,
    task_type: taskType || undefined,
    status: status || undefined,
  });

  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Requests</h1>
        {total > 0 && (
          <span className="text-sm text-gray-500">{total.toLocaleString()} total</span>
        )}
      </div>

      {/* Filters */}
      <Card className="p-3">
        <div className="flex flex-wrap gap-3">
          <select
            value={model}
            onChange={(e) => { setModel(e.target.value); setPage(1); }}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
            aria-label="Filter by model"
          >
            <option value="">All models</option>
            {MODELS.map((m) => <option key={m} value={m}>{m}</option>)}
          </select>

          <select
            value={taskType}
            onChange={(e) => { setTaskType(e.target.value); setPage(1); }}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
            aria-label="Filter by task type"
          >
            <option value="">All tasks</option>
            {TASK_TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>

          <select
            value={status}
            onChange={(e) => { setStatus(e.target.value); setPage(1); }}
            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
            aria-label="Filter by status"
          >
            <option value="">All statuses</option>
            {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>

          {(model || taskType || status) && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => { setModel(''); setTaskType(''); setStatus(''); setPage(1); }}
            >
              Clear filters
            </Button>
          )}
        </div>
      </Card>

      {/* Table */}
      <Card className="overflow-hidden p-0">
        {!isLoading && data?.requests?.length === 0 ? (
          <EmptyState
            title="No requests found"
            description="Requests will appear here once your API key is used for inference."
          />
        ) : (
          <RequestLogTable requests={data?.requests ?? []} loading={isLoading} />
        )}
      </Card>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            disabled={page === 1}
          >
            Previous
          </Button>
          <span className="text-gray-500">
            Page {page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
            disabled={page === totalPages}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  );
}
