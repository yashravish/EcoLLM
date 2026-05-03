import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { UsageResponse } from '@/types';

export function useUsage(period: 'daily' | 'monthly', from?: string, to?: string) {
  return useQuery<UsageResponse>({
    queryKey: ['usage', period, from, to],
    queryFn: () => api.get('/v1/usage', { period, from, to }),
    staleTime: 60 * 1000,
    refetchInterval: 60 * 1000,
  });
}
