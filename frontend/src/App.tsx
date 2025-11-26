import { Routes, Route, Navigate } from 'react-router-dom';
import { Toaster } from '@/components/ui/toaster';
import { AuthProvider } from '@/hooks/use-auth';
import { ThemeProvider } from '@/hooks/use-theme';
import { ProjectProvider } from '@/contexts/project-context';
import { ProtectedRoute } from '@/components/features/auth/protected-route';

// Pages
import { LoginPage } from '@/pages/auth/login';
import { RegisterPage } from '@/pages/auth/register';
import { DashboardPage } from '@/pages/dashboard';
import { TracesPage } from '@/pages/traces';
import { TraceDetailPage } from '@/pages/traces/detail';
import { TraceComparePage } from '@/pages/traces/compare';
import { SessionsPage } from '@/pages/sessions';
import { SessionDetailPage } from '@/pages/sessions/detail';
import { UsersPage } from '@/pages/users';
import { UserDetailPage } from '@/pages/users/detail';
import { PromptsPage } from '@/pages/prompts';
import { PromptDetailPage } from '@/pages/prompts/detail';
import { GuardrailsPage } from '@/pages/guardrails';
import { SettingsPage } from '@/pages/settings';
import ScoresPage from '@/pages/scores';
import ScoreDetailPage from '@/pages/scores/detail';
import ScoreAnalyticsPage from '@/pages/scores/analytics';
import AdvancedScoreAnalyticsPage from '@/pages/scores/advanced-analytics';
import FeedbackPage from '@/pages/feedback';
import FeedbackAnalyticsPage from '@/pages/feedback/analytics';
import AnnotationsPage from '@/pages/annotations';
import AnnotatePage from '@/pages/annotations/annotate';
import AgentsPage from '@/pages/agents';
import AgentGraphDetailPage from '@/pages/agents/detail';
import EvaluatorsPage from '@/pages/evaluators';
import EvaluatorDetailPage from '@/pages/evaluators/detail';
import EvaluationResultsPage from '@/pages/evaluators/results';

// Layout
import { DashboardLayout } from '@/components/features/layout/dashboard-layout-updated';

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
                <ProjectProvider>
                  <DashboardLayout />
                </ProjectProvider>
              </ProtectedRoute>
            }
          >
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="traces" element={<TracesPage />} />
            <Route path="traces/compare" element={<TraceComparePage />} />
            <Route path="traces/:id" element={<TraceDetailPage />} />
            <Route path="sessions" element={<SessionsPage />} />
            <Route path="sessions/:id" element={<SessionDetailPage />} />
            <Route path="users" element={<UsersPage />} />
            <Route path="users/:id" element={<UserDetailPage />} />
            <Route path="prompts" element={<PromptsPage />} />
            <Route path="prompts/:id" element={<PromptDetailPage />} />
            <Route path="scores" element={<ScoresPage />} />
            <Route path="scores/analytics" element={<ScoreAnalyticsPage />} />
            <Route path="scores/advanced-analytics" element={<AdvancedScoreAnalyticsPage />} />
            <Route path="scores/:scoreId" element={<ScoreDetailPage />} />
            <Route path="feedback" element={<FeedbackPage />} />
            <Route path="feedback/analytics" element={<FeedbackAnalyticsPage />} />
            <Route path="guardrails" element={<GuardrailsPage />} />
            <Route path="annotations" element={<AnnotationsPage />} />
            <Route path="annotations/annotate" element={<AnnotatePage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="agents/:traceId" element={<AgentGraphDetailPage />} />
            <Route path="evaluators" element={<EvaluatorsPage />} />
            <Route path="evaluators/results" element={<EvaluationResultsPage />} />
            <Route path="evaluators/:id" element={<EvaluatorDetailPage />} />
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
