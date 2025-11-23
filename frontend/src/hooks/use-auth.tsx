import {
  createContext,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from 'react';
import { useNavigate } from 'react-router-dom';
import { useMe, useLogout, type User } from '@/api/auth';

interface AuthContextType {
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const [isInitialized, setIsInitialized] = useState(false);

  const { data: user, isLoading, error, isError } = useMe();
  const logoutMutation = useLogout();

  useEffect(() => {
    if (!isLoading) {
      setIsInitialized(true);
    }
  }, [isLoading]);

  useEffect(() => {
    // If we get a 401 error, the user is not authenticated
    if (isError && (error as any)?.status === 401) {
      // Clear any cached auth state
      setIsInitialized(true);
    }
  }, [isError, error]);

  const logout = () => {
    logoutMutation.mutate(undefined, {
      onSuccess: () => {
        navigate('/login');
      },
      onError: () => {
        // Even if logout fails, clear local state
        navigate('/login');
      },
    });
  };

  const value: AuthContextType = {
    user: user ?? null,
    isLoading: isLoading || !isInitialized,
    isAuthenticated: !!user && !isError,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
