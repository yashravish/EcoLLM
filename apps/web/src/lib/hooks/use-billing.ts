import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { BillingListResponse } from '@/types';

export function useBilling(from?: string, to?: string) {
  return useQuery<BillingListResponse>({
    queryKey: ['billing', from, to],
    queryFn: () => api.get('/v1/billing', { from, to }),
    staleTime: 5 * 60 * 1000,
  });
}
