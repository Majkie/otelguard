import React, { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  useCoreMetrics,
  useTimeSeriesMetrics,
  useModelBreakdown,
  type MetricsFilter,
} from '@/api/metrics';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Loader2, TrendingUp, TrendingDown, Activity, DollarSign, Zap, AlertTriangle } from 'lucide-react';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
} from 'recharts';

const COLORS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
];

export default function OverviewDashboard() {
  const [searchParams] = useSearchParams();
  const projectId = searchParams.get('projectId') || '';

  // Time range selection
  const [timeRange, setTimeRange] = useState('24h');

  // Calculate time range
  const getTimeRange = (range: string) => {
    const endTime = new Date();
    const startTime = new Date();

    switch (range) {
      case '1h':
        startTime.setHours(endTime.getHours() - 1);
        break;
      case '24h':
        startTime.setHours(endTime.getHours() - 24);
        break;
      case '7d':
        startTime.setDate(endTime.getDate() - 7);
        break;
      case '30d':
        startTime.setDate(endTime.getDate() - 30);
        break;
    }

    return {
      startTime: startTime.toISOString(),
      endTime: endTime.toISOString(),
    };
  };

  const { startTime, endTime } = getTimeRange(timeRange);

  const filter: MetricsFilter = {
    projectId,
    startTime,
    endTime,
  };

  // Fetch metrics
  const { data: coreMetrics, isLoading: isLoadingCore } = useCoreMetrics(filter);
  const { data: traceTimeSeries, isLoading: isLoadingTraces } = useTimeSeriesMetrics({
    ...filter,
    metric: 'traces',
    interval: timeRange === '1h' ? 'hour' : timeRange === '24h' ? 'hour' : 'day',
  });
  const { data: latencyTimeSeries, isLoading: isLoadingLatency } = useTimeSeriesMetrics({
    ...filter,
    metric: 'latency',
    interval: timeRange === '1h' ? 'hour' : timeRange === '24h' ? 'hour' : 'day',
  });
  const { data: costTimeSeries, isLoading: isLoadingCost } = useTimeSeriesMetrics({
    ...filter,
    metric: 'cost',
    interval: timeRange === '1h' ? 'hour' : timeRange === '24h' ? 'hour' : 'day',
  });
  const { data: modelBreakdown, isLoading: isLoadingModels } = useModelBreakdown(filter);

  if (!projectId) {
    return (
      <div className="container mx-auto py-8">
        <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900 dark:bg-yellow-950">
          <p className="text-sm">Please select a project to view metrics.</p>
        </div>
      </div>
    );
  }

  const isLoading = isLoadingCore || isLoadingTraces || isLoadingLatency || isLoadingCost || isLoadingModels;

  return (
    <div className="container mx-auto space-y-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Overview Dashboard</h1>
          <p className="mt-1 text-muted-foreground">
            Monitor your LLM application performance and usage
          </p>
        </div>
        <Select value={timeRange} onValueChange={setTimeRange}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Select time range" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="1h">Last 1 hour</SelectItem>
            <SelectItem value="24h">Last 24 hours</SelectItem>
            <SelectItem value="7d">Last 7 days</SelectItem>
            <SelectItem value="30d">Last 30 days</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading && (
        <div className="flex h-96 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {!isLoading && coreMetrics && (
        <>
          {/* Key Metrics Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <MetricCard
              title="Total Traces"
              value={coreMetrics.totalTraces.toLocaleString()}
              icon={<Activity className="h-4 w-4" />}
              description={`${coreMetrics.totalSpans.toLocaleString()} spans`}
            />
            <MetricCard
              title="Avg Latency"
              value={`${coreMetrics.avgLatencyMs.toFixed(0)}ms`}
              icon={<Zap className="h-4 w-4" />}
              description={`P95: ${coreMetrics.p95LatencyMs.toFixed(0)}ms`}
            />
            <MetricCard
              title="Total Cost"
              value={`$${coreMetrics.totalCost.toFixed(2)}`}
              icon={<DollarSign className="h-4 w-4" />}
              description={`Avg: $${coreMetrics.avgCost.toFixed(4)}/trace`}
            />
            <MetricCard
              title="Error Rate"
              value={`${(coreMetrics.errorRate * 100).toFixed(2)}%`}
              icon={<AlertTriangle className="h-4 w-4" />}
              description={`${coreMetrics.errorCount} errors`}
              variant={coreMetrics.errorRate > 0.05 ? 'error' : 'success'}
            />
          </div>

          {/* Charts */}
          <Tabs defaultValue="traces" className="w-full">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="traces">Traces</TabsTrigger>
              <TabsTrigger value="latency">Latency</TabsTrigger>
              <TabsTrigger value="cost">Cost</TabsTrigger>
              <TabsTrigger value="models">Models</TabsTrigger>
            </TabsList>

            {/* Traces Time Series */}
            <TabsContent value="traces" className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Trace Volume Over Time</CardTitle>
                  <CardDescription>
                    Number of traces processed in the selected time range
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {traceTimeSeries && (
                    <ResponsiveContainer width="100%" height={350}>
                      <LineChart data={traceTimeSeries.points}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="timestamp"
                          tickFormatter={(value) => new Date(value).toLocaleTimeString()}
                          className="text-xs"
                        />
                        <YAxis className="text-xs" />
                        <Tooltip
                          labelFormatter={(value) => new Date(value).toLocaleString()}
                          contentStyle={{
                            backgroundColor: 'hsl(var(--background))',
                            border: '1px solid hsl(var(--border))',
                            borderRadius: '6px',
                          }}
                        />
                        <Legend />
                        <Line
                          type="monotone"
                          dataKey="value"
                          name="Traces"
                          stroke={COLORS[0]}
                          strokeWidth={2}
                          dot={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            {/* Latency Time Series */}
            <TabsContent value="latency" className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Latency Over Time</CardTitle>
                  <CardDescription>
                    Average response latency in milliseconds
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {latencyTimeSeries && (
                    <ResponsiveContainer width="100%" height={350}>
                      <LineChart data={latencyTimeSeries.points}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="timestamp"
                          tickFormatter={(value) => new Date(value).toLocaleTimeString()}
                          className="text-xs"
                        />
                        <YAxis className="text-xs" label={{ value: 'ms', angle: -90, position: 'insideLeft' }} />
                        <Tooltip
                          labelFormatter={(value) => new Date(value).toLocaleString()}
                          formatter={(value: number) => `${value.toFixed(2)}ms`}
                          contentStyle={{
                            backgroundColor: 'hsl(var(--background))',
                            border: '1px solid hsl(var(--border))',
                            borderRadius: '6px',
                          }}
                        />
                        <Legend />
                        <Line
                          type="monotone"
                          dataKey="value"
                          name="Latency"
                          stroke={COLORS[1]}
                          strokeWidth={2}
                          dot={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            {/* Cost Time Series */}
            <TabsContent value="cost" className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>Cost Over Time</CardTitle>
                  <CardDescription>
                    Cumulative cost in the selected time range
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {costTimeSeries && (
                    <ResponsiveContainer width="100%" height={350}>
                      <LineChart data={costTimeSeries.points}>
                        <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                        <XAxis
                          dataKey="timestamp"
                          tickFormatter={(value) => new Date(value).toLocaleTimeString()}
                          className="text-xs"
                        />
                        <YAxis className="text-xs" label={{ value: '$', angle: -90, position: 'insideLeft' }} />
                        <Tooltip
                          labelFormatter={(value) => new Date(value).toLocaleString()}
                          formatter={(value: number) => `$${value.toFixed(4)}`}
                          contentStyle={{
                            backgroundColor: 'hsl(var(--background))',
                            border: '1px solid hsl(var(--border))',
                            borderRadius: '6px',
                          }}
                        />
                        <Legend />
                        <Line
                          type="monotone"
                          dataKey="value"
                          name="Cost"
                          stroke={COLORS[2]}
                          strokeWidth={2}
                          dot={false}
                        />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            {/* Model Breakdown */}
            <TabsContent value="models" className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <Card>
                  <CardHeader>
                    <CardTitle>Traces by Model</CardTitle>
                    <CardDescription>Distribution of trace volume</CardDescription>
                  </CardHeader>
                  <CardContent>
                    {modelBreakdown && (
                      <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                          <Pie
                            data={modelBreakdown}
                            dataKey="traceCount"
                            nameKey="model"
                            cx="50%"
                            cy="50%"
                            outerRadius={80}
                            label
                          >
                            {modelBreakdown.map((_, index) => (
                              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                            ))}
                          </Pie>
                          <Tooltip />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>Cost by Model</CardTitle>
                    <CardDescription>Total spending per model</CardDescription>
                  </CardHeader>
                  <CardContent>
                    {modelBreakdown && (
                      <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={modelBreakdown}>
                          <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                          <XAxis dataKey="model" className="text-xs" />
                          <YAxis className="text-xs" />
                          <Tooltip
                            formatter={(value: number) => `$${value.toFixed(4)}`}
                            contentStyle={{
                              backgroundColor: 'hsl(var(--background))',
                              border: '1px solid hsl(var(--border))',
                              borderRadius: '6px',
                            }}
                          />
                          <Bar dataKey="totalCost" fill={COLORS[2]} radius={[4, 4, 0, 0]} />
                        </BarChart>
                      </ResponsiveContainer>
                    )}
                  </CardContent>
                </Card>
              </div>
            </TabsContent>
          </Tabs>

          {/* Token Stats */}
          <Card>
            <CardHeader>
              <CardTitle>Token Usage</CardTitle>
              <CardDescription>Breakdown of token consumption</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-3">
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium text-muted-foreground">Total Tokens</p>
                  <p className="mt-2 text-2xl font-bold">{coreMetrics.totalTokens.toLocaleString()}</p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium text-muted-foreground">Prompt Tokens</p>
                  <p className="mt-2 text-2xl font-bold">{coreMetrics.totalPromptTokens.toLocaleString()}</p>
                  <p className="text-xs text-muted-foreground">
                    {((coreMetrics.totalPromptTokens / coreMetrics.totalTokens) * 100).toFixed(1)}% of total
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium text-muted-foreground">Completion Tokens</p>
                  <p className="mt-2 text-2xl font-bold">{coreMetrics.totalCompletionTokens.toLocaleString()}</p>
                  <p className="text-xs text-muted-foreground">
                    {((coreMetrics.totalCompletionTokens / coreMetrics.totalTokens) * 100).toFixed(1)}% of total
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

interface MetricCardProps {
  title: string;
  value: string;
  icon: React.ReactNode;
  description?: string;
  variant?: 'default' | 'success' | 'error';
}

function MetricCard({ title, value, icon, description, variant = 'default' }: MetricCardProps) {
  const variantStyles = {
    default: 'border-border',
    success: 'border-green-200 bg-green-50 dark:border-green-900 dark:bg-green-950',
    error: 'border-red-200 bg-red-50 dark:border-red-900 dark:bg-red-950',
  };

  return (
    <Card className={variantStyles[variant]}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {description && (
          <p className="text-xs text-muted-foreground mt-1">{description}</p>
        )}
      </CardContent>
    </Card>
  );
}
