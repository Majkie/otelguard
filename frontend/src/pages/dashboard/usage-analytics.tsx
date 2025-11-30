import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { LineChart, BarChart, PieChart, MetricCard } from '@/components/charts';
import { Activity, Users, Zap, TrendingUp } from 'lucide-react';
import { format, subDays, subHours } from 'date-fns';
import {useProjectContext} from "@/contexts/project-context.tsx";

interface TimeRange {
  label: string;
  value: string;
  startTime: Date;
  endTime: Date;
}

const TIME_RANGES: TimeRange[] = [
  {
    label: 'Last 24 hours',
    value: '24h',
    startTime: subHours(new Date(), 24),
    endTime: new Date(),
  },
  {
    label: 'Last 7 days',
    value: '7d',
    startTime: subDays(new Date(), 7),
    endTime: new Date(),
  },
  {
    label: 'Last 30 days',
    value: '30d',
    startTime: subDays(new Date(), 30),
    endTime: new Date(),
  },
  {
    label: 'Last 90 days',
    value: '90d',
    startTime: subDays(new Date(), 90),
    endTime: new Date(),
  },
];

export function UsageAnalyticsPage() {
  const { selectedProject } = useProjectContext();
  const [timeRange, setTimeRange] = useState<TimeRange>(TIME_RANGES[1]);
  const projectId = selectedProject;

  // Fetch core metrics
  const { data: coreMetrics, isLoading: coreLoading } = useQuery({
    queryKey: ['metrics', 'core', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/metrics/core', {
        params: {
          projectId,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch time series data
  const { data: traceTimeSeries } = useQuery({
    queryKey: ['metrics', 'timeseries', 'traces', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/metrics/timeseries', {
        params: {
          projectId,
          metric: 'traces',
          interval: timeRange.value === '24h' ? 'hour' : 'day',
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  const { data: latencyTimeSeries } = useQuery({
    queryKey: ['metrics', 'timeseries', 'latency', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/metrics/timeseries', {
        params: {
          projectId,
          metric: 'latency',
          interval: timeRange.value === '24h' ? 'hour' : 'day',
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch model breakdown
  const { data: modelBreakdown } = useQuery({
    queryKey: ['metrics', 'models', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/metrics/models', {
        params: {
          projectId,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch user breakdown
  const { data: userBreakdown } = useQuery({
    queryKey: ['metrics', 'users', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/metrics/users', {
        params: {
          projectId,
          limit: 10,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    if (timeRange.value === '24h') {
      return format(date, 'HH:mm');
    }
    return format(date, 'MMM dd');
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Usage Analytics</h1>
          <p className="text-muted-foreground">
            Monitor usage patterns, active users, and system utilization
          </p>
        </div>
        <Select
          value={timeRange.value}
          onValueChange={(value) => {
            const range = TIME_RANGES.find((r) => r.value === value);
            if (range) setTimeRange(range);
          }}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {TIME_RANGES.map((range) => (
              <SelectItem key={range.value} value={range.value}>
                {range.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Overview Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          title="Total Requests"
          value={coreMetrics?.total_traces?.toLocaleString() || '0'}
          icon={<Activity className="h-4 w-4" />}
          trend={{
            value: coreMetrics?.trace_growth_percentage || 0,
            label: 'vs previous period',
          }}
          loading={coreLoading}
        />
        <MetricCard
          title="Active Users"
          value={coreMetrics?.unique_users?.toLocaleString() || '0'}
          icon={<Users className="h-4 w-4" />}
          trend={{
            value: coreMetrics?.user_growth_percentage || 0,
            label: 'vs previous period',
          }}
          loading={coreLoading}
        />
        <MetricCard
          title="Avg Response Time"
          value={`${coreMetrics?.avg_latency_ms?.toFixed(0) || '0'}ms`}
          icon={<Zap className="h-4 w-4" />}
          trend={{
            value: -(coreMetrics?.latency_change_percentage || 0),
            label: 'vs previous period',
            isPositiveGood: false,
          }}
          loading={coreLoading}
        />
        <MetricCard
          title="Requests/User"
          value={
            coreMetrics?.unique_users
              ? (coreMetrics.total_traces / coreMetrics.unique_users).toFixed(1)
              : '0'
          }
          icon={<TrendingUp className="h-4 w-4" />}
          loading={coreLoading}
        />
      </div>

      <Tabs defaultValue="requests" className="space-y-4">
        <TabsList>
          <TabsTrigger value="requests">Requests</TabsTrigger>
          <TabsTrigger value="models">Models</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
        </TabsList>

        <TabsContent value="requests" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Request Volume</CardTitle>
              <CardDescription>Number of requests over time</CardDescription>
            </CardHeader>
            <CardContent>
              {traceTimeSeries?.data ? (
                <LineChart
                  data={traceTimeSeries.data}
                  xKey="timestamp"
                  lines={[
                    {
                      dataKey: 'value',
                      color: 'hsl(var(--chart-1))',
                      label: 'Requests',
                    },
                  ]}
                  formatXAxis={formatTimestamp}
                  height={300}
                />
              ) : (
                <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                  Loading...
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="models" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Usage by Model</CardTitle>
                <CardDescription>Request distribution across models</CardDescription>
              </CardHeader>
              <CardContent>
                {modelBreakdown?.data ? (
                  <PieChart
                    data={modelBreakdown.data.map((m: any) => ({
                      name: m.model,
                      value: m.trace_count,
                    }))}
                    height={300}
                  />
                ) : (
                  <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                    Loading...
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Model Statistics</CardTitle>
                <CardDescription>Performance metrics by model</CardDescription>
              </CardHeader>
              <CardContent>
                {modelBreakdown?.data ? (
                  <BarChart
                    data={modelBreakdown.data.slice(0, 10)}
                    xKey="model"
                    bars={[
                      {
                        dataKey: 'avg_latency_ms',
                        color: 'hsl(var(--chart-2))',
                        label: 'Avg Latency (ms)',
                      },
                    ]}
                    height={300}
                    formatYAxis={(value) => `${value}ms`}
                  />
                ) : (
                  <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                    Loading...
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="users" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Top Users by Volume</CardTitle>
                <CardDescription>Users with most requests</CardDescription>
              </CardHeader>
              <CardContent>
                {userBreakdown?.data ? (
                  <BarChart
                    data={userBreakdown.data.slice(0, 10)}
                    xKey="user_id"
                    bars={[
                      {
                        dataKey: 'trace_count',
                        color: 'hsl(var(--chart-3))',
                        label: 'Requests',
                      },
                    ]}
                    height={300}
                    layout="horizontal"
                  />
                ) : (
                  <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                    Loading...
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>User Engagement</CardTitle>
                <CardDescription>Average tokens and cost per user</CardDescription>
              </CardHeader>
              <CardContent>
                {userBreakdown?.data ? (
                  <BarChart
                    data={userBreakdown.data.slice(0, 10)}
                    xKey="user_id"
                    bars={[
                      {
                        dataKey: 'avg_tokens',
                        color: 'hsl(var(--chart-4))',
                        label: 'Avg Tokens',
                      },
                    ]}
                    height={300}
                    layout="horizontal"
                  />
                ) : (
                  <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                    Loading...
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Response Time Trend</CardTitle>
              <CardDescription>Average latency over time</CardDescription>
            </CardHeader>
            <CardContent>
              {latencyTimeSeries?.data ? (
                <LineChart
                  data={latencyTimeSeries.data}
                  xKey="timestamp"
                  lines={[
                    {
                      dataKey: 'value',
                      color: 'hsl(var(--chart-2))',
                      label: 'Latency (ms)',
                    },
                  ]}
                  formatXAxis={formatTimestamp}
                  formatYAxis={(value) => `${value}ms`}
                  height={300}
                />
              ) : (
                <div className="h-[300px] flex items-center justify-center text-muted-foreground">
                  Loading...
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
