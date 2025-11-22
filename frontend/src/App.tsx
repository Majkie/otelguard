import { Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from '@/components/ui/toaster';
import { AuthProvider } from '@/hooks/use-auth';
import { ThemeProvider } from '@/hooks/use-theme';
import { ProtectedRoute } from '@/components/features/auth/protected-route';

// Pages
import { LoginPage } from '@/pages/auth/login';
import { RegisterPage } from '@/pages/auth/register';
import { DashboardPage } from '@/pages/dashboard';
import { TracesPage } from '@/pages/traces';
import { TraceDetailPage } from '@/pages/traces/detail';
import { TraceComparePage } from '@/pages/traces/compare';
import { PromptsPage } from '@/pages/prompts';
import { GuardrailsPage } from '@/pages/guardrails';
import { SettingsPage } from '@/pages/settings';

// Layout
import { DashboardLayout } from '@/components/features/layout/dashboard-layout';

function App() {
  return (
    <ThemeProvider defaultTheme="system" storageKey="otelguard-theme">
      <AuthProvider>
        <Routes>
        {/* Public routes */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />

        {/* Protected routes */}
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <DashboardLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="traces" element={<TracesPage />} />
          <Route path="traces/compare" element={<TraceComparePage />} />
          <Route path="traces/:id" element={<TraceDetailPage />} />
          <Route path="prompts" element={<PromptsPage />} />
          <Route path="guardrails" element={<GuardrailsPage />} />
          <Route path="settings" element={<SettingsPage />} />
        </Route>

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
        <Toaster />
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
