'use client';

import { DollarSign } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { EmptyState } from '@/components/shared/empty-state';
import { useBilling } from '@/lib/hooks/use-billing';
import { formatCost, formatCO2, dateRange } from '@/lib/utils';
import type { BillingEvent } from '@/types';

function BillingRow({ event }: { event: BillingEvent }) {
  const discountLabel =
    event.discount_percent > 0 ? `${event.discount_percent}% volume` : '—';

  return (
    <tr className="border-b border-gray-100 last:border-0 dark:border-gray-800">
      <td className="py-3 px-4 text-sm text-gray-700 dark:text-gray-300">
        {event.period_start.slice(0, 10)} – {event.period_end.slice(0, 10)}
      </td>
      <td className="py-3 px-4 text-sm text-gray-700 dark:text-gray-300">
        {event.total_requests.toLocaleString()}
      </td>
      <td className="py-3 px-4 text-sm text-gray-700 dark:text-gray-300">
        {formatCO2(event.total_co2e_grams)}
      </td>
      <td className="py-3 px-4 text-right text-sm text-gray-700 dark:text-gray-300">
        {formatCost(event.base_cost_usd)}
      </td>
      <td className="py-3 px-4 text-right text-sm text-gray-500 dark:text-gray-400">
        {discountLabel}
      </td>
      <td className="py-3 px-4 text-right text-sm font-semibold text-gray-900 dark:text-white">
        {formatCost(event.final_cost_usd)}
      </td>
    </tr>
  );
}

export default function BillingPage() {
  const { from, to } = dateRange(90);

  const { data, isLoading } = useBilling(from, to);

  const events = data?.events ?? [];
  const hasEvents = !isLoading && events.length > 0;
  const noData = !isLoading && events.length === 0;

  const totalFinal = events.reduce((s, e) => s + e.final_cost_usd, 0);

  return (
    <div className="space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Billing</h1>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="pt-6">
            {isLoading ? (
              <LoadingSkeleton variant="stat" />
            ) : (
              <>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {formatCost(totalFinal)}
                </p>
                <p className="mt-1 text-sm text-gray-500">Total billed (90d)</p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6">
            {isLoading ? (
              <LoadingSkeleton variant="stat" />
            ) : (
              <>
                <p className="text-2xl font-bold text-gray-900 dark:text-white">
                  {events.length}
                </p>
                <p className="mt-1 text-sm text-gray-500">Billing periods</p>
              </>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-6 flex items-center gap-3">
            <DollarSign className="h-8 w-8 text-amber-500" aria-hidden="true" />
            <div>
              <p className="text-xs text-gray-500">Formula</p>
              <p className="mt-0.5 text-xs font-mono text-gray-700 dark:text-gray-300">
                CO₂e × $0.001 + reqs × $0.0001
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Billing Events</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading && <LoadingSkeleton variant="table" lines={4} />}

          {noData && (
            <EmptyState
              title="No billing events yet"
              description="Billing events are generated daily once you have usage."
            />
          )}

          {hasEvents && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm" aria-label="Billing events">
                <thead>
                  <tr className="border-b border-gray-100 dark:border-gray-800">
                    {['Period', 'Requests', 'CO2e', 'Base cost', 'Discount', 'Final cost'].map((h) => (
                      <th
                        key={h}
                        className="py-3 px-4 text-left text-xs font-medium uppercase tracking-wide text-gray-500 last:text-right [&:nth-child(4)]:text-right [&:nth-child(5)]:text-right"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {events.map((e) => (
                    <BillingRow key={e.id} event={e} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
