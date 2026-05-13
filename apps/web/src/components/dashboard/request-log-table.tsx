import Link from 'next/link';
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ModelBadge } from '@/components/shared/model-badge';
import { CarbonBadge } from '@/components/shared/carbon-badge';
import { LatencyBadge } from '@/components/shared/latency-badge';
import { CostBadge } from '@/components/shared/cost-badge';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { formatRelativeTime, truncate } from '@/lib/utils';
import type { RequestRecord } from '@/types';

interface RequestLogTableProps {
  requests: RequestRecord[];
  loading?: boolean;
  onRowClick?: (id: string) => void;
}

export function RequestLogTable({ requests, loading = false }: RequestLogTableProps) {
  if (loading) {
    return <LoadingSkeleton variant="table" lines={5} />;
  }

  return (
    <Table>
      <TableCaption className="sr-only">Recent inference requests</TableCaption>
      <TableHeader>
        <TableRow>
          <TableHead scope="col">Time</TableHead>
          <TableHead scope="col">Prompt</TableHead>
          <TableHead scope="col">Model</TableHead>
          <TableHead scope="col">Latency</TableHead>
          <TableHead scope="col">CO2</TableHead>
          <TableHead scope="col">Cost</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {requests.map((req) => (
          <TableRow key={req.id}>
            <TableCell className="whitespace-nowrap font-mono text-xs text-eco-500">
              <Link
                href={`/requests/${req.id}`}
                className="block hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent rounded"
                aria-label={`View request from ${formatRelativeTime(req.created_at)}`}
              >
                {formatRelativeTime(req.created_at)}
              </Link>
            </TableCell>
            <TableCell className="max-w-xs">
              <Link
                href={`/requests/${req.id}`}
                className="block text-xs text-eco-200 hover:text-accent hover:underline focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent rounded transition-colors"
              >
                {truncate(req.prompt_original, 60)}
              </Link>
            </TableCell>
            <TableCell>
              <ModelBadge name={req.model_selected} />
            </TableCell>
            <TableCell>
              <LatencyBadge latencyMs={req.latency_ms} />
            </TableCell>
            <TableCell>
              {req.co2e_grams !== undefined ? (
                <CarbonBadge co2eGrams={req.co2e_grams} />
              ) : (
                <span className="font-mono text-xs text-eco-500">—</span>
              )}
            </TableCell>
            <TableCell>
              {req.cost_usd !== undefined ? (
                <CostBadge costUsd={req.cost_usd} />
              ) : (
                <span className="font-mono text-xs text-eco-500">—</span>
              )}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
