import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { RequestFilters, RequestListResponse, RequestRecord } from '@/types';

export function useRequests(filters: RequestFilters = {}) {
  return useQuery<RequestListResponse>({
    queryKey: ['requests', filters],
    queryFn: () =>
      api.get('/v1/requests', filters as Record<string, string | number | boolean | undefined>),
    staleTime: 30 * 1000,
    refetchInterval: 60 * 1000,
  });
}

export function useRequest(id: string) {
  return useQuery<RequestRecord>({
    queryKey: ['requests', id],
    queryFn: () => api.get(`/v1/requests/${id}`),
    staleTime: 5 * 60 * 1000,
    enabled: Boolean(id),
  });
}
