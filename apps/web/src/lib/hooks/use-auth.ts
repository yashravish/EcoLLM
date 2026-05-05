import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import type { LoginRequest, LoginResponse, MeResponse, RegisterRequest, RegisterResponse } from '@/types';

export function useLogin() {
  const queryClient = useQueryClient();
  return useMutation<LoginResponse, Error, LoginRequest>({
    mutationFn: (data) => api.post('/auth/login', data),
    onSuccess: (data) => {
      api.setToken(data.token);
      queryClient.invalidateQueries({ queryKey: ['me'] });
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () => api.post('/auth/logout', {}),
    onSuccess: () => {
      api.clearToken();
      queryClient.clear();
    },
    onError: () => {
      api.clearToken();
      queryClient.clear();
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();
  return useMutation<RegisterResponse, Error, RegisterRequest>({
    mutationFn: (data) => api.post('/auth/register', data),
    onSuccess: (data) => {
      api.setToken(data.token);
      queryClient.setQueryData(['me'], { user: data.user, org: data.org });
    },
  });
}

export function useMe() {
  return useQuery<MeResponse>({
    queryKey: ['me'],
    queryFn: () => api.get('/me'),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}
