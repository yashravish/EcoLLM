import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { CarbonResponse } from '@/types';

export function useCarbon(period: 'daily' | 'monthly') {
  return useQuery<CarbonResponse>({
    queryKey: ['carbon', period],
    queryFn: () => api.get('/v1/carbon', { period }),
    staleTime: 60 * 1000,
    refetchInterval: 60 * 1000,
  });
}
