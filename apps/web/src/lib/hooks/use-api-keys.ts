import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { ApiKey, CreateApiKeyRequest, CreateApiKeyResponse } from '@/types';

export function useApiKeys() {
  return useQuery<ApiKey[]>({
    queryKey: ['api-keys'],
    queryFn: () => api.get('/api-keys'),
    staleTime: 30 * 1000,
  });
}

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  return useMutation<CreateApiKeyResponse, Error, CreateApiKeyRequest>({
    mutationFn: (data) => api.post('/api-keys', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });
}

export function useRevokeApiKey() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (id) => api.delete(`/api-keys/${id}`),
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: ['api-keys'] });
      const previous = queryClient.getQueryData<ApiKey[]>(['api-keys']);
      queryClient.setQueryData<ApiKey[]>(['api-keys'], (old) =>
        old ? old.filter((k) => k.id !== id) : [],
      );
      return { previous };
    },
    onError: (_err, _id, context) => {
      const ctx = context as { previous?: ApiKey[] } | undefined;
      if (ctx?.previous) queryClient.setQueryData(['api-keys'], ctx.previous);
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });
}
