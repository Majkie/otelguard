import { useMemo } from 'react';
import { Activity, DollarSign, Zap, AlertTriangle, Users, MessageSquare, Building2 } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { useOverviewMetrics } from '@/api/analytics';
import { useUsers } from '@/api/users';
import { useProjectContext } from '@/contexts/project-context';
import { formatCost } from '@/lib/utils';

export function DashboardPage() {
  const { selectedProject, hasProjects } = useProjectContext();

  // Memoize date ranges to prevent infinite re-renders
  const dateRange = useMemo(() => {
    const now = new Date();
    const twentyFourHoursAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000);

    // Format dates for ClickHouse compatibility (YYYY-MM-DD HH:mm:ss format)
    const formatDate = (date: Date) => {
      return date.toISOString().slice(0, 19).replace('T', ' '); // Remove milliseconds and Z, replace T with space
    };

    return {
      startTime: formatDate(twentyFourHoursAgo),
      endTime: formatDate(now),
    };
  }, []); // Empty dependency array means this only calculates once

  // Get overview metrics from analytics API
  const { data: overviewData, isLoading: overviewLoading } = useOverviewMetrics(dateRange);

  // Get users count (keeping for reference, not currently used in UI)
  const { isLoading: usersLoading } = useUsers({
    limit: 1000,
  });

  const stats = useMemo(() => {
    if (!overviewData) {
      return null;
    }

    const errorRate = overviewData.errorRate * 100; // Convert from decimal to percentage

    return [
      {
        name: 'Total Traces',
        value: overviewData.totalTraces.toString(),
        icon: Activity,
        description: 'Last 24 hours',
      },
      {
        name: 'Total Cost',
        value: formatCost(overviewData.totalCost),
        icon: DollarSign,
        description: 'Last 24 hours',
      },
      {
        name: 'Avg Latency',
        value: `${Math.round(overviewData.avgLatencyMs)}ms`,
        icon: Zap,
        description: 'Last 24 hours',
      },
      {
        name: 'Error Rate',
        value: `${errorRate.toFixed(1)}%`,
        icon: AlertTriangle,
        description: 'Last 24 hours',
      },
    ];
  }, [overviewData]);

  const isLoading = overviewLoading || usersLoading;

  if (!hasProjects) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-muted-foreground">
            Overview of your LLM application performance
          </p>
        </div>

        {/* Getting started */}
        <Card>
          <CardHeader>
            <CardTitle>Welcome to OTelGuard</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-muted-foreground">
              To get started, create your first project to begin collecting traces from your LLM applications.
            </p>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Building2 className="h-4 w-4" />
              <span>Go to Settings → Projects to create a project</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!selectedProject) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-muted-foreground">
            Overview of your LLM application performance
          </p>
        </div>

        {/* No project selected */}
        <Card>
          <CardHeader>
            <CardTitle>Select a Project</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-muted-foreground">
              Please select a project from the project selector above to view dashboard data.
            </p>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Building2 className="h-4 w-4" />
              <span>Use the project selector in the header to choose a project</span>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">
          Overview of your LLM application performance for {selectedProject.name}
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {isLoading ? (
          Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-4" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-8 w-16 mb-2" />
                <Skeleton className="h-3 w-20" />
              </CardContent>
            </Card>
          ))
        ) : stats ? (
          stats.map((stat) => (
            <Card key={stat.name}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{stat.name}</CardTitle>
                <stat.icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stat.value}</div>
                <p className="text-xs text-muted-foreground">
                  {stat.description}
                </p>
              </CardContent>
            </Card>
          ))
        ) : (
          Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">No Data</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">-</div>
                <p className="text-xs text-muted-foreground">
                  No traces found
                </p>
              </CardContent>
            </Card>
          ))
        )}
      </div>

      {/* Additional stats */}
      {stats && !isLoading && (
        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Users</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {overviewData?.uniqueUsers || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                Last 24 hours
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Sessions</CardTitle>
              <MessageSquare className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {overviewData?.uniqueSessions || 0}
              </div>
              <p className="text-xs text-muted-foreground">
                Last 24 hours
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
              <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {overviewData ? ((overviewData.successCount / Math.max(overviewData.totalTraces, 1)) * 100).toFixed(1) : '0.0'}%
              </div>
              <p className="text-xs text-muted-foreground">
                Last 24 hours
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Getting started when no data */}
      {(!overviewData || overviewData.totalTraces === 0) && !isLoading && (
        <Card>
          <CardHeader>
            <CardTitle>Get Started</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <p className="text-muted-foreground">
              Welcome to OTelGuard! To start collecting traces from your LLM
              application:
            </p>
            <ol className="list-decimal list-inside space-y-2 text-sm">
              <li>Install the OTelGuard SDK in your project</li>
              <li>Configure the SDK with your API key</li>
              <li>Instrument your LLM calls</li>
              <li>Start seeing traces in the dashboard</li>
            </ol>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <p className="text-muted-foreground"># Python</p>
              <p>pip install otelguard-sdk</p>
              <br />
              <p className="text-muted-foreground"># JavaScript</p>
              <p>npm install @otelguard/sdk</p>
              <br />
              <p className="text-muted-foreground"># PHP</p>
              <p>composer install otelguard-sdk</p>
              <br />
              <p className="text-muted-foreground"># Go</p>
              <p>go mod download github.com/otelguard/otelguard-go</p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
