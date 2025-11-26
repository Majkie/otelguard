import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import {
  ArrowLeft,
  Shield,
  TrendingUp,
  AlertTriangle,
  Clock,
  DollarSign,
  Activity,
} from 'lucide-react';
import {
  BarChart,
  Bar,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import {
  useTriggerStats,
  useViolationTrends,
  useRemediationSuccessRates,
  useCostImpactAnalysis,
  useLatencyImpact,
  type GuardrailAnalyticsParams,
} from '@/api/guardrail-analytics';
import { useProjectContext } from '@/contexts/project-context';

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884d8', '#82ca9d'];

function GuardrailAnalyticsPage() {
  const { currentProject } = useProjectContext();
  const [timeRange, setTimeRange] = useState<'24h' | '7d' | '30d' | '90d'>('7d');
  const [interval, setInterval] = useState<'1h' | '6h' | '12h' | '1d' | '1w'>('1d');

  // Calculate time range
  const now = new Date();
  const startTime = new Date();
  switch (timeRange) {
    case '24h':
      startTime.setHours(now.getHours() - 24);
      break;
    case '7d':
      startTime.setDate(now.getDate() - 7);
      break;
    case '30d':
      startTime.setDate(now.getDate() - 30);
      break;
    case '90d':
      startTime.setDate(now.getDate() - 90);
      break;
  }

  const params: GuardrailAnalyticsParams = {
    projectId: currentProject?.id || '',
    startTime: startTime.toISOString(),
    endTime: now.toISOString(),
    interval,
  };

  const { data: triggerStats, isLoading: statsLoading } = useTriggerStats(params);
  const { data: trendsData, isLoading: trendsLoading } = useViolationTrends(params);
  const { data: remediationData, isLoading: remediationLoading } =
    useRemediationSuccessRates(params);
  const { data: costImpact, isLoading: costLoading } = useCostImpactAnalysis(params);
  const { data: latencyImpact, isLoading: latencyLoading } = useLatencyImpact(params);

  const formatNumber = (num: number) => {
    if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
    if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`;
    return num.toString();
  };

  const formatPercentage = (rate: number) => `${(rate * 100).toFixed(2)}%`;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/guardrails">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Guardrails
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">Guardrails Analytics</h1>
          <p className="text-muted-foreground">
            Monitor guardrail performance, triggers, and impact
          </p>
        </div>
      </div>

      {/* Time Range Selector */}
      <div className="flex gap-4">
        <div className="flex-1">
          <label className="text-sm font-medium mb-2 block">Time Range</label>
          <Select value={timeRange} onValueChange={(v: any) => setTimeRange(v)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="24h">Last 24 Hours</SelectItem>
              <SelectItem value="7d">Last 7 Days</SelectItem>
              <SelectItem value="30d">Last 30 Days</SelectItem>
              <SelectItem value="90d">Last 90 Days</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="flex-1">
          <label className="text-sm font-medium mb-2 block">Interval</label>
          <Select value={interval} onValueChange={(v: any) => setInterval(v)}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">Hourly</SelectItem>
              <SelectItem value="6h">6 Hours</SelectItem>
              <SelectItem value="12h">12 Hours</SelectItem>
              <SelectItem value="1d">Daily</SelectItem>
              <SelectItem value="1w">Weekly</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Overview Stats */}
      {statsLoading ? (
        <div className="grid gap-4 md:grid-cols-4">
          {[...Array(4)].map((_, i) => (
            <Card key={i}>
              <CardContent className="p-6">
                <div className="animate-pulse space-y-2">
                  <div className="h-4 bg-muted rounded w-20"></div>
                  <div className="h-8 bg-muted rounded w-24"></div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : triggerStats ? (
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Shield className="h-4 w-4 text-primary" />
                <CardTitle className="text-sm font-medium">Evaluations</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatNumber(triggerStats.totalEvaluations)}
              </div>
              <p className="text-xs text-muted-foreground mt-1">Total evaluations</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-yellow-500" />
                <CardTitle className="text-sm font-medium">Triggers</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatNumber(triggerStats.totalTriggered)}
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {formatPercentage(triggerStats.triggerRate)} trigger rate
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Activity className="h-4 w-4 text-blue-500" />
                <CardTitle className="text-sm font-medium">Actions</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {formatNumber(triggerStats.totalActioned)}
              </div>
              <p className="text-xs text-muted-foreground mt-1">
                {formatPercentage(triggerStats.actionRate)} action rate
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <DollarSign className="h-4 w-4 text-green-500" />
                <CardTitle className="text-sm font-medium">Cost Savings</CardTitle>
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                ${costImpact?.estimatedCostSavings.toFixed(2) || '0.00'}
              </div>
              <p className="text-xs text-muted-foreground mt-1">Estimated savings</p>
            </CardContent>
          </Card>
        </div>
      ) : null}

      {/* Analytics Tabs */}
      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
          <TabsTrigger value="policies">By Policy</TabsTrigger>
          <TabsTrigger value="rules">By Rule Type</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          {/* Action Type Distribution */}
          <Card>
            <CardHeader>
              <CardTitle>Action Distribution</CardTitle>
              <CardDescription>Breakdown of remediation actions taken</CardDescription>
            </CardHeader>
            <CardContent>
              {triggerStats && Object.keys(triggerStats.byAction).length > 0 ? (
                <div className="grid md:grid-cols-2 gap-6">
                  <ResponsiveContainer width="100%" height={300}>
                    <PieChart>
                      <Pie
                        data={Object.entries(triggerStats.byAction).map(
                          ([action, stats]) => ({
                            name: action,
                            value: stats.actionCount,
                          })
                        )}
                        cx="50%"
                        cy="50%"
                        labelLine={false}
                        label={({ name, percent }) =>
                          `${name}: ${(percent * 100).toFixed(0)}%`
                        }
                        outerRadius={100}
                        fill="#8884d8"
                        dataKey="value"
                      >
                        {Object.keys(triggerStats.byAction).map((_, index) => (
                          <Cell
                            key={`cell-${index}`}
                            fill={COLORS[index % COLORS.length]}
                          />
                        ))}
                      </Pie>
                      <Tooltip />
                    </PieChart>
                  </ResponsiveContainer>

                  <div className="space-y-3">
                    {Object.entries(triggerStats.byAction).map(([action, stats]) => (
                      <div
                        key={action}
                        className="flex items-center justify-between p-3 border rounded"
                      >
                        <div>
                          <p className="font-medium capitalize">{action}</p>
                          <p className="text-sm text-muted-foreground">
                            {formatPercentage(stats.successRate)} success rate
                          </p>
                        </div>
                        <Badge variant="outline">{stats.actionCount} actions</Badge>
                      </div>
                    ))}
                  </div>
                </div>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  No action data available
                </div>
              )}
            </CardContent>
          </Card>

          {/* Remediation Success Rates */}
          {remediationData && remediationData.data.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Remediation Success Rates</CardTitle>
                <CardDescription>
                  Success rates and latency by action type
                </CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart data={remediationData.data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="actionType" />
                    <YAxis yAxisId="left" orientation="left" />
                    <YAxis yAxisId="right" orientation="right" />
                    <Tooltip />
                    <Legend />
                    <Bar
                      yAxisId="left"
                      dataKey="successRate"
                      fill="#82ca9d"
                      name="Success Rate"
                    />
                    <Bar
                      yAxisId="right"
                      dataKey="avgLatencyMs"
                      fill="#8884d8"
                      name="Avg Latency (ms)"
                    />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Violation Trends Over Time</CardTitle>
              <CardDescription>Track guardrail triggers and actions</CardDescription>
            </CardHeader>
            <CardContent>
              {trendsLoading ? (
                <div className="h-80 flex items-center justify-center">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              ) : trendsData && trendsData.data.length > 0 ? (
                <ResponsiveContainer width="100%" height={400}>
                  <LineChart data={trendsData.data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="timestamp"
                      tickFormatter={(value) =>
                        new Date(value).toLocaleDateString(undefined, {
                          month: 'short',
                          day: 'numeric',
                        })
                      }
                    />
                    <YAxis />
                    <Tooltip
                      labelFormatter={(value) => new Date(value).toLocaleString()}
                    />
                    <Legend />
                    <Line
                      type="monotone"
                      dataKey="evaluationCount"
                      stroke="#8884d8"
                      name="Evaluations"
                      strokeWidth={2}
                    />
                    <Line
                      type="monotone"
                      dataKey="triggerCount"
                      stroke="#ff8042"
                      name="Triggers"
                      strokeWidth={2}
                    />
                    <Line
                      type="monotone"
                      dataKey="actionCount"
                      stroke="#00c49f"
                      name="Actions"
                      strokeWidth={2}
                    />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <div className="h-80 flex items-center justify-center text-muted-foreground">
                  No trend data available
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="policies" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Performance by Policy</CardTitle>
              <CardDescription>Compare guardrail policies</CardDescription>
            </CardHeader>
            <CardContent>
              {triggerStats && Object.keys(triggerStats.byPolicy).length > 0 ? (
                <div className="space-y-4">
                  {Object.entries(triggerStats.byPolicy).map(([_, policy]) => (
                    <Card key={policy.policyId}>
                      <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                          <div className="flex-1">
                            <h3 className="font-semibold">{policy.policyName}</h3>
                            <div className="grid grid-cols-4 gap-4 mt-3">
                              <div>
                                <p className="text-sm text-muted-foreground">Evaluations</p>
                                <p className="text-lg font-semibold">
                                  {formatNumber(policy.evaluationCount)}
                                </p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Trigger Rate</p>
                                <p className="text-lg font-semibold">
                                  {formatPercentage(policy.triggerRate)}
                                </p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Avg Latency</p>
                                <p className="text-lg font-semibold">
                                  {policy.avgLatencyMs.toFixed(1)}ms
                                </p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Actions</p>
                                <p className="text-lg font-semibold">
                                  {formatNumber(policy.actionCount)}
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  No policy data available
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="rules" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Triggers by Rule Type</CardTitle>
              <CardDescription>See which rules trigger most frequently</CardDescription>
            </CardHeader>
            <CardContent>
              {triggerStats && Object.keys(triggerStats.byRuleType).length > 0 ? (
                <ResponsiveContainer width="100%" height={400}>
                  <BarChart
                    data={Object.entries(triggerStats.byRuleType).map(
                      ([ruleType, stats]) => ({
                        ruleType,
                        triggers: stats.triggerCount,
                        avgLatency: stats.avgLatencyMs,
                      })
                    )}
                    layout="vertical"
                  >
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis type="number" />
                    <YAxis type="category" dataKey="ruleType" width={150} />
                    <Tooltip />
                    <Legend />
                    <Bar dataKey="triggers" fill="#8884d8" name="Trigger Count" />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="text-center py-12 text-muted-foreground">
                  No rule type data available
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            {/* Latency Impact */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Clock className="h-5 w-5" />
                  Latency Impact
                </CardTitle>
                <CardDescription>Performance overhead from guardrails</CardDescription>
              </CardHeader>
              <CardContent>
                {latencyLoading ? (
                  <div className="h-48 flex items-center justify-center">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                  </div>
                ) : latencyImpact ? (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-sm text-muted-foreground">Average</p>
                        <p className="text-2xl font-bold">
                          {latencyImpact.avgLatencyMs.toFixed(1)}ms
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-muted-foreground">P95</p>
                        <p className="text-2xl font-bold">
                          {latencyImpact.p95LatencyMs.toFixed(1)}ms
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-muted-foreground">Median</p>
                        <p className="text-2xl font-bold">
                          {latencyImpact.medianLatencyMs.toFixed(1)}ms
                        </p>
                      </div>
                      <div>
                        <p className="text-sm text-muted-foreground">P99</p>
                        <p className="text-2xl font-bold">
                          {latencyImpact.p99LatencyMs.toFixed(1)}ms
                        </p>
                      </div>
                    </div>
                    <div className="pt-4 border-t">
                      <p className="text-sm text-muted-foreground">Impact on Total Latency</p>
                      <div className="flex items-center gap-2 mt-2">
                        <div className="flex-1 bg-muted rounded-full h-2">
                          <div
                            className="bg-primary h-2 rounded-full"
                            style={{
                              width: `${Math.min(latencyImpact.impactPercentage, 100)}%`,
                            }}
                          />
                        </div>
                        <span className="text-sm font-semibold">
                          {latencyImpact.impactPercentage.toFixed(1)}%
                        </span>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-12 text-muted-foreground">
                    No latency data available
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Cost Impact */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <DollarSign className="h-5 w-5" />
                  Cost Impact
                </CardTitle>
                <CardDescription>Cost savings from guardrails</CardDescription>
              </CardHeader>
              <CardContent>
                {costLoading ? (
                  <div className="h-48 flex items-center justify-center">
                    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                  </div>
                ) : costImpact ? (
                  <div className="space-y-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Total Estimated Savings</p>
                      <p className="text-3xl font-bold text-green-600">
                        ${costImpact.estimatedCostSavings.toFixed(2)}
                      </p>
                    </div>
                    <div className="pt-4 border-t space-y-3">
                      <p className="text-sm font-medium">Savings by Policy</p>
                      {Object.entries(costImpact.byPolicy).map(([_, policy]) => (
                        <div
                          key={policy.policyId}
                          className="flex items-center justify-between"
                        >
                          <span className="text-sm">{policy.policyName}</span>
                          <span className="font-semibold">
                            ${policy.estimatedCostSavings.toFixed(2)}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="text-center py-12 text-muted-foreground">
                    No cost data available
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default GuardrailAnalyticsPage;
