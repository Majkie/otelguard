import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface User {
  id: string;
  email: string;
  name: string;
  avatarUrl?: string;
  createdAt: string;
}

export interface AuthResponse {
  token: string;
  refreshToken: string;
  expiresAt: number;
  user: User;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  email: string;
  password: string;
  name: string;
}

// Query keys
export const authKeys = {
  all: ['auth'] as const,
  me: () => [...authKeys.all, 'me'] as const,
};

// Hooks
export function useMe() {
  return useQuery({
    queryKey: authKeys.me(),
    queryFn: () => api.get<User>('/v1/me'),
    enabled: !!localStorage.getItem('token'),
  });
}

export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: LoginInput) =>
      api.post<AuthResponse>('/v1/auth/login', input),
    onSuccess: (data) => {
      localStorage.setItem('token', data.token);
      localStorage.setItem('refreshToken', data.refreshToken);
      queryClient.setQueryData(authKeys.me(), data.user);
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: RegisterInput) =>
      api.post<AuthResponse>('/v1/auth/register', input),
    onSuccess: (data) => {
      localStorage.setItem('token', data.token);
      localStorage.setItem('refreshToken', data.refreshToken);
      queryClient.setQueryData(authKeys.me(), data.user);
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      localStorage.removeItem('token');
      localStorage.removeItem('refreshToken');
    },
    onSuccess: () => {
      queryClient.clear();
    },
  });
}
