import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { ModelListResponse } from '@/types';

export function useModels() {
  return useQuery<ModelListResponse>({
    queryKey: ['models'],
    queryFn: () => api.get('/v1/models'),
    staleTime: 5 * 60 * 1000,
  });
}
