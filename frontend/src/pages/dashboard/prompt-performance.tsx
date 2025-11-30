import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { LineChart, BarChart, DonutChart, MetricCard } from '@/components/charts';
import { FileText, TrendingUp, DollarSign, Clock } from 'lucide-react';
import { format, subDays, subHours } from 'date-fns';
import { Link } from 'react-router-dom';
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
];

export function PromptPerformancePage() {
  const { selectedProject } = useProjectContext();
  const [timeRange, setTimeRange] = useState<TimeRange>(TIME_RANGES[1]);
  const projectId = selectedProject;

  // Fetch all prompts
  const { data: promptsData } = useQuery({
    queryKey: ['prompts', projectId],
    queryFn: () =>
      api.get('/v1/prompts', {
        params: { projectId },
      }),
  });

  // Fetch prompt performance aggregates
  const { data: promptPerformance, isLoading: perfLoading } = useQuery({
    queryKey: ['prompts', 'performance', 'aggregate', projectId, timeRange.value],
    queryFn: async () => {
      const prompts = promptsData?.data || [];
      const performances = await Promise.all(
        prompts.slice(0, 20).map((prompt: any) =>
          api.get(`/v1/prompts/${prompt.id}/performance`, {
            params: {
              startTime: timeRange.startTime.toISOString(),
              endTime: timeRange.endTime.toISOString(),
            },
          }).catch(() => null)
        )
      );
      return performances
        .filter((p) => p !== null)
        .map((p: any, i) => ({
          ...p,
          prompt_id: prompts[i].id,
          prompt_name: prompts[i].name,
        }));
    },
    enabled: !!promptsData?.data,
  });

  // Calculate aggregate metrics
  const aggregateMetrics = promptPerformance
    ? {
        total_prompts: promptPerformance.length,
        total_executions: promptPerformance.reduce(
          (sum: number, p: any) => sum + (p.total_executions || 0),
          0
        ),
        avg_latency: promptPerformance.reduce(
          (sum: number, p: any) => sum + (p.avg_latency_ms || 0),
          0
        ) / (promptPerformance.length || 1),
        total_cost: promptPerformance.reduce(
          (sum: number, p: any) => sum + (p.total_cost || 0),
          0
        ),
      }
    : null;

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
          <h1 className="text-3xl font-bold tracking-tight">Prompt Performance</h1>
          <p className="text-muted-foreground">
            Monitor prompt usage, performance, and effectiveness across versions
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
          title="Active Prompts"
          value={aggregateMetrics?.total_prompts?.toLocaleString() || '0'}
          icon={<FileText className="h-4 w-4" />}
          loading={perfLoading}
        />
        <MetricCard
          title="Total Executions"
          value={aggregateMetrics?.total_executions?.toLocaleString() || '0'}
          icon={<TrendingUp className="h-4 w-4" />}
          loading={perfLoading}
        />
        <MetricCard
          title="Avg Response Time"
          value={`${aggregateMetrics?.avg_latency?.toFixed(0) || '0'}ms`}
          icon={<Clock className="h-4 w-4" />}
          loading={perfLoading}
        />
        <MetricCard
          title="Total Cost"
          value={`$${aggregateMetrics?.total_cost?.toFixed(2) || '0.00'}`}
          icon={<DollarSign className="h-4 w-4" />}
          loading={perfLoading}
        />
      </div>

      <Tabs defaultValue="usage" className="space-y-4">
        <TabsList>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="cost">Cost</TabsTrigger>
          <TabsTrigger value="versions">Versions</TabsTrigger>
        </TabsList>

        <TabsContent value="usage" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Usage Distribution</CardTitle>
                <CardDescription>Execution count by prompt</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <DonutChart
                    data={promptPerformance
                      .sort((a: any, b: any) => b.total_executions - a.total_executions)
                      .slice(0, 10)
                      .map((p: any) => ({
                        name: p.prompt_name,
                        value: p.total_executions || 0,
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
                <CardTitle>Top Prompts by Volume</CardTitle>
                <CardDescription>Most frequently executed prompts</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <BarChart
                    data={promptPerformance
                      .sort((a: any, b: any) => b.total_executions - a.total_executions)
                      .slice(0, 10)}
                    xKey="prompt_name"
                    bars={[
                      {
                        dataKey: 'total_executions',
                        color: 'hsl(var(--chart-1))',
                        label: 'Executions',
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
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Latency by Prompt</CardTitle>
                <CardDescription>Average response time for each prompt</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <BarChart
                    data={promptPerformance
                      .sort((a: any, b: any) => b.avg_latency_ms - a.avg_latency_ms)
                      .slice(0, 10)}
                    xKey="prompt_name"
                    bars={[
                      {
                        dataKey: 'avg_latency_ms',
                        color: 'hsl(var(--chart-2))',
                        label: 'Avg Latency (ms)',
                      },
                    ]}
                    height={300}
                    layout="horizontal"
                    formatYAxis={(value) => `${value}ms`}
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
                <CardTitle>Token Usage</CardTitle>
                <CardDescription>Average tokens per prompt</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <BarChart
                    data={promptPerformance
                      .sort((a: any, b: any) => b.avg_total_tokens - a.avg_total_tokens)
                      .slice(0, 10)}
                    xKey="prompt_name"
                    bars={[
                      {
                        dataKey: 'avg_total_tokens',
                        color: 'hsl(var(--chart-3))',
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

        <TabsContent value="cost" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Cost Distribution</CardTitle>
                <CardDescription>Total cost by prompt</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <DonutChart
                    data={promptPerformance
                      .sort((a: any, b: any) => b.total_cost - a.total_cost)
                      .slice(0, 10)
                      .map((p: any) => ({
                        name: p.prompt_name,
                        value: p.total_cost || 0,
                      }))}
                    height={300}
                    formatTooltip={(value) => `$${value.toFixed(4)}`}
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
                <CardTitle>Cost per Execution</CardTitle>
                <CardDescription>Average cost efficiency by prompt</CardDescription>
              </CardHeader>
              <CardContent>
                {promptPerformance ? (
                  <BarChart
                    data={promptPerformance
                      .map((p: any) => ({
                        ...p,
                        cost_per_exec: p.total_executions > 0 ? p.total_cost / p.total_executions : 0,
                      }))
                      .sort((a: any, b: any) => b.cost_per_exec - a.cost_per_exec)
                      .slice(0, 10)}
                    xKey="prompt_name"
                    bars={[
                      {
                        dataKey: 'cost_per_exec',
                        color: 'hsl(var(--chart-4))',
                        label: 'Cost per Execution',
                      },
                    ]}
                    height={300}
                    layout="horizontal"
                    formatYAxis={(value) => `$${value.toFixed(4)}`}
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

        <TabsContent value="versions" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Prompt Performance Comparison</CardTitle>
              <CardDescription>
                Compare key metrics across all prompts
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-md border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="p-3 text-left font-medium">Prompt</th>
                      <th className="p-3 text-right font-medium">Executions</th>
                      <th className="p-3 text-right font-medium">Avg Latency</th>
                      <th className="p-3 text-right font-medium">Avg Tokens</th>
                      <th className="p-3 text-right font-medium">Total Cost</th>
                    </tr>
                  </thead>
                  <tbody>
                    {promptPerformance && promptPerformance.length > 0 ? (
                      promptPerformance
                        .sort((a: any, b: any) => b.total_executions - a.total_executions)
                        .slice(0, 20)
                        .map((p: any) => (
                          <tr key={p.prompt_id} className="border-b hover:bg-muted/30">
                            <td className="p-3">
                              <Link
                                to={`/prompts/${p.prompt_id}`}
                                className="font-medium text-primary hover:underline"
                              >
                                {p.prompt_name}
                              </Link>
                            </td>
                            <td className="p-3 text-right">{p.total_executions?.toLocaleString() || 0}</td>
                            <td className="p-3 text-right">{p.avg_latency_ms?.toFixed(0) || 0}ms</td>
                            <td className="p-3 text-right">{p.avg_total_tokens?.toFixed(0) || 0}</td>
                            <td className="p-3 text-right">${p.total_cost?.toFixed(4) || '0.0000'}</td>
                          </tr>
                        ))
                    ) : (
                      <tr>
                        <td colSpan={5} className="p-8 text-center text-muted-foreground">
                          No prompt performance data available
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
