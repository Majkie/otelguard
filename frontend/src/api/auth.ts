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
  expiresAt: number;
  user: User;
  csrfToken?: string;
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

// API functions
export async function refreshToken(): Promise<{ expiresAt: number }> {
  return api.post('/v1/auth/refresh');
}

// Hooks
export function useMe() {
  return useQuery({
    queryKey: authKeys.me(),
    queryFn: () => api.get<User>('/v1/me'),
    // We'll check authentication status differently since we can't read cookies directly
    enabled: true, // Let the API call determine if we're authenticated
    retry: (failureCount, error: any) => {
      // Don't retry on auth errors
      if (error?.status === 401) {
        return false;
      }
      return failureCount < 3;
    },
  });
}

export function useLogin() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: LoginInput) =>
      api.post<AuthResponse>('/v1/auth/login', input),
    onSuccess: (data) => {
      // Cookies are set automatically by the backend
      // Store user data in cache
      queryClient.setQueryData(authKeys.me(), data.user);
      // Also invalidate and refetch to ensure fresh data
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useRegister() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: RegisterInput) =>
      api.post<AuthResponse>('/v1/auth/register', input),
    onSuccess: (data) => {
      // Cookies are set automatically by the backend
      // Store user data in cache
      queryClient.setQueryData(authKeys.me(), data.user);
      // Also invalidate and refetch to ensure fresh data
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useRefreshToken() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: refreshToken,
    onSuccess: () => {
      // Invalidate user data to refetch with new token
      queryClient.invalidateQueries({ queryKey: authKeys.me() });
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.post('/v1/auth/logout'),
    onSuccess: () => {
      // Cookies are cleared by the backend
      queryClient.clear();
    },
  });
}
