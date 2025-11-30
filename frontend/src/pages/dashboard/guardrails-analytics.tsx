import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '@/api/client';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { LineChart, BarChart, DonutChart, MetricCard } from '@/components/charts';
import { Shield, AlertTriangle, CheckCircle, Clock } from 'lucide-react';
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
];

export function GuardrailsAnalyticsPage() {
  const { selectedProject } = useProjectContext();
  const [timeRange, setTimeRange] = useState<TimeRange>(TIME_RANGES[1]);
  const projectId = selectedProject;

  // Fetch trigger statistics
  const { data: triggerStats, isLoading: triggerLoading } = useQuery({
    queryKey: ['guardrails', 'analytics', 'triggers', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/guardrails/analytics/triggers', {
        params: {
          projectId,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch violation trends
  const { data: violationTrends } = useQuery({
    queryKey: ['guardrails', 'analytics', 'trends', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/guardrails/analytics/trends', {
        params: {
          projectId,
          interval: timeRange.value === '24h' ? '1h' : '1d',
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch remediation success rates
  const { data: remediationSuccess } = useQuery({
    queryKey: ['guardrails', 'analytics', 'remediation', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/guardrails/analytics/remediation-success', {
        params: {
          projectId,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch cost impact
  const { data: costImpact } = useQuery({
    queryKey: ['guardrails', 'analytics', 'cost', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/guardrails/analytics/cost-impact', {
        params: {
          projectId,
          startTime: timeRange.startTime.toISOString(),
          endTime: timeRange.endTime.toISOString(),
        },
      }),
  });

  // Fetch latency impact
  const { data: latencyImpact } = useQuery({
    queryKey: ['guardrails', 'analytics', 'latency', projectId, timeRange.value],
    queryFn: () =>
      api.get('/v1/guardrails/analytics/latency-impact', {
        params: {
          projectId,
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

  const calculateTriggerRate = () => {
    if (!triggerStats) return '0';
    const total = triggerStats.total_evaluations || 0;
    const triggered = triggerStats.total_triggered || 0;
    if (total === 0) return '0';
    return ((triggered / total) * 100).toFixed(1);
  };

  const calculateRemediationRate = () => {
    if (!triggerStats) return '0';
    const triggered = triggerStats.total_triggered || 0;
    const remediated = triggerStats.total_remediated || 0;
    if (triggered === 0) return '0';
    return ((remediated / triggered) * 100).toFixed(1);
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Guardrails Analytics</h1>
          <p className="text-muted-foreground">
            Monitor guardrail performance, violations, and remediation effectiveness
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
          title="Total Evaluations"
          value={triggerStats?.total_evaluations?.toLocaleString() || '0'}
          icon={<Shield className="h-4 w-4" />}
          loading={triggerLoading}
        />
        <MetricCard
          title="Violations Detected"
          value={triggerStats?.total_triggered?.toLocaleString() || '0'}
          icon={<AlertTriangle className="h-4 w-4" />}
          subtitle={`${calculateTriggerRate()}% trigger rate`}
          loading={triggerLoading}
        />
        <MetricCard
          title="Auto-Remediated"
          value={triggerStats?.total_remediated?.toLocaleString() || '0'}
          icon={<CheckCircle className="h-4 w-4" />}
          subtitle={`${calculateRemediationRate()}% success rate`}
          loading={triggerLoading}
        />
        <MetricCard
          title="Avg Processing Time"
          value={`${latencyImpact?.avg_guardrail_latency_ms?.toFixed(0) || '0'}ms`}
          icon={<Clock className="h-4 w-4" />}
          loading={!latencyImpact}
        />
      </div>

      <Tabs defaultValue="violations" className="space-y-4">
        <TabsList>
          <TabsTrigger value="violations">Violations</TabsTrigger>
          <TabsTrigger value="policies">Policies</TabsTrigger>
          <TabsTrigger value="remediation">Remediation</TabsTrigger>
          <TabsTrigger value="impact">Impact</TabsTrigger>
        </TabsList>

        <TabsContent value="violations" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Violation Trends</CardTitle>
              <CardDescription>Violations detected over time</CardDescription>
            </CardHeader>
            <CardContent>
              {violationTrends?.data ? (
                <LineChart
                  data={violationTrends.data}
                  xKey="timestamp"
                  lines={[
                    {
                      dataKey: 'triggered_count',
                      color: 'hsl(var(--destructive))',
                      label: 'Violations',
                    },
                    {
                      dataKey: 'remediated_count',
                      color: 'hsl(var(--chart-2))',
                      label: 'Remediated',
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

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Violations by Type</CardTitle>
                <CardDescription>Distribution of violation types</CardDescription>
              </CardHeader>
              <CardContent>
                {triggerStats?.violations_by_type ? (
                  <DonutChart
                    data={triggerStats.violations_by_type.map((v: any) => ({
                      name: v.rule_type,
                      value: v.count,
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
                <CardTitle>Top Violating Rules</CardTitle>
                <CardDescription>Most frequently triggered rules</CardDescription>
              </CardHeader>
              <CardContent>
                {triggerStats?.violations_by_type ? (
                  <BarChart
                    data={triggerStats.violations_by_type.slice(0, 10)}
                    xKey="rule_type"
                    bars={[
                      {
                        dataKey: 'count',
                        color: 'hsl(var(--destructive))',
                        label: 'Violations',
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

        <TabsContent value="policies" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Policy Performance</CardTitle>
              <CardDescription>Evaluation count and trigger rate by policy</CardDescription>
            </CardHeader>
            <CardContent>
              {triggerStats?.policies ? (
                <BarChart
                  data={triggerStats.policies}
                  xKey="policy_name"
                  bars={[
                    {
                      dataKey: 'evaluation_count',
                      color: 'hsl(var(--chart-1))',
                      label: 'Evaluations',
                      stackId: 'a',
                    },
                    {
                      dataKey: 'triggered_count',
                      color: 'hsl(var(--destructive))',
                      label: 'Violations',
                      stackId: 'a',
                    },
                  ]}
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

        <TabsContent value="remediation" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Remediation Actions</CardTitle>
                <CardDescription>Distribution of remediation types</CardDescription>
              </CardHeader>
              <CardContent>
                {remediationSuccess?.data ? (
                  <DonutChart
                    data={remediationSuccess.data.map((r: any) => ({
                      name: r.action,
                      value: r.count,
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
                <CardTitle>Success Rates</CardTitle>
                <CardDescription>Remediation success rate by action type</CardDescription>
              </CardHeader>
              <CardContent>
                {remediationSuccess?.data ? (
                  <BarChart
                    data={remediationSuccess.data.map((r: any) => ({
                      action: r.action,
                      success_rate: (r.success_count / r.count) * 100,
                    }))}
                    xKey="action"
                    bars={[
                      {
                        dataKey: 'success_rate',
                        color: 'hsl(var(--chart-2))',
                        label: 'Success Rate (%)',
                      },
                    ]}
                    height={300}
                    formatYAxis={(value) => `${value}%`}
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

        <TabsContent value="impact" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Cost Impact</CardTitle>
                <CardDescription>Cost savings from guardrails</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <div className="text-2xl font-bold">
                      ${costImpact?.total_cost_saved?.toFixed(2) || '0.00'}
                    </div>
                    <p className="text-xs text-muted-foreground">Total cost saved</p>
                  </div>
                  <div>
                    <div className="text-xl font-semibold">
                      ${costImpact?.avg_cost_per_block?.toFixed(4) || '0.0000'}
                    </div>
                    <p className="text-xs text-muted-foreground">Average cost per blocked request</p>
                  </div>
                  <div>
                    <div className="text-xl font-semibold">
                      {costImpact?.total_requests_blocked?.toLocaleString() || '0'}
                    </div>
                    <p className="text-xs text-muted-foreground">Total requests blocked</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Latency Impact</CardTitle>
                <CardDescription>Performance overhead of guardrails</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div>
                    <div className="text-2xl font-bold">
                      {latencyImpact?.avg_guardrail_latency_ms?.toFixed(0) || '0'}ms
                    </div>
                    <p className="text-xs text-muted-foreground">Average guardrail latency</p>
                  </div>
                  <div>
                    <div className="text-xl font-semibold">
                      {latencyImpact?.p95_latency_ms?.toFixed(0) || '0'}ms
                    </div>
                    <p className="text-xs text-muted-foreground">P95 latency</p>
                  </div>
                  <div>
                    <div className="text-xl font-semibold">
                      {latencyImpact?.p99_latency_ms?.toFixed(0) || '0'}ms
                    </div>
                    <p className="text-xs text-muted-foreground">P99 latency</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
