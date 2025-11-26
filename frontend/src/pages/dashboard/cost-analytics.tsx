import React, { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  useCostBreakdown,
  useModelBreakdown,
  useUserBreakdown,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Loader2, DollarSign, TrendingUp } from 'lucide-react';
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

export default function CostAnalyticsDashboard() {
  const [searchParams] = useSearchParams();
  const projectId = searchParams.get('projectId') || '';

  const [timeRange, setTimeRange] = useState('7d');

  const getTimeRange = (range: string) => {
    const endTime = new Date();
    const startTime = new Date();

    switch (range) {
      case '24h':
        startTime.setHours(endTime.getHours() - 24);
        break;
      case '7d':
        startTime.setDate(endTime.getDate() - 7);
        break;
      case '30d':
        startTime.setDate(endTime.getDate() - 30);
        break;
      case '90d':
        startTime.setDate(endTime.getDate() - 90);
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

  const { data: costBreakdown, isLoading: isLoadingCost } = useCostBreakdown(filter);
  const { data: modelBreakdown, isLoading: isLoadingModels } = useModelBreakdown(filter);
  const { data: userBreakdown, isLoading: isLoadingUsers } = useUserBreakdown(filter, 20);

  if (!projectId) {
    return (
      <div className="container mx-auto py-8">
        <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900 dark:bg-yellow-950">
          <p className="text-sm">Please select a project to view cost analytics.</p>
        </div>
      </div>
    );
  }

  const isLoading = isLoadingCost || isLoadingModels || isLoadingUsers;

  return (
    <div className="container mx-auto space-y-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Cost Analytics</h1>
          <p className="mt-1 text-muted-foreground">
            Track and optimize your LLM spending
          </p>
        </div>
        <Select value={timeRange} onValueChange={setTimeRange}>
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="Select time range" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="24h">Last 24 hours</SelectItem>
            <SelectItem value="7d">Last 7 days</SelectItem>
            <SelectItem value="30d">Last 30 days</SelectItem>
            <SelectItem value="90d">Last 90 days</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {isLoading && (
        <div className="flex h-96 items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      )}

      {!isLoading && costBreakdown && (
        <>
          {/* Summary Card */}
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <div>
                <CardTitle>Total Cost</CardTitle>
                <CardDescription>Cumulative spending in selected period</CardDescription>
              </div>
              <DollarSign className="h-8 w-8 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-4xl font-bold">${costBreakdown.totalCost.toFixed(2)}</div>
              <p className="mt-2 text-sm text-muted-foreground">
                Across {costBreakdown.topCostModels.reduce((sum, m) => sum + m.traceCount, 0).toLocaleString()} traces
              </p>
            </CardContent>
          </Card>

          {/* Cost Over Time */}
          <Card>
            <CardHeader>
              <CardTitle>Cost Over Time</CardTitle>
              <CardDescription>
                Daily cost trend in the selected period
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={350}>
                <LineChart data={costBreakdown.costOverTime}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                  <XAxis
                    dataKey="timestamp"
                    tickFormatter={(value) => new Date(value).toLocaleDateString()}
                    className="text-xs"
                  />
                  <YAxis className="text-xs" />
                  <Tooltip
                    labelFormatter={(value) => new Date(value).toLocaleDateString()}
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
            </CardContent>
          </Card>

          {/* Cost by Model */}
          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Cost Distribution by Model</CardTitle>
                <CardDescription>Spending breakdown per LLM model</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <PieChart>
                    <Pie
                      data={costBreakdown.topCostModels}
                      dataKey="totalCost"
                      nameKey="model"
                      cx="50%"
                      cy="50%"
                      outerRadius={80}
                      label={(entry) => `$${entry.totalCost.toFixed(2)}`}
                    >
                      {costBreakdown.topCostModels.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value: number) => `$${value.toFixed(4)}`} />
                  </PieChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Top Models by Cost</CardTitle>
                <CardDescription>Models ordered by total spending</CardDescription>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={300}>
                  <BarChart
                    data={costBreakdown.topCostModels.slice(0, 10)}
                    layout="vertical"
                  >
                    <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                    <XAxis type="number" className="text-xs" />
                    <YAxis dataKey="model" type="category" className="text-xs" width={150} />
                    <Tooltip
                      formatter={(value: number) => `$${value.toFixed(4)}`}
                      contentStyle={{
                        backgroundColor: 'hsl(var(--background))',
                        border: '1px solid hsl(var(--border))',
                        borderRadius: '6px',
                      }}
                    />
                    <Bar dataKey="totalCost" fill={COLORS[2]} />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>

          {/* Model Cost Table */}
          <Card>
            <CardHeader>
              <CardTitle>Model Cost Details</CardTitle>
              <CardDescription>
                Detailed cost breakdown and efficiency metrics per model
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Model</TableHead>
                    <TableHead className="text-right">Total Cost</TableHead>
                    <TableHead className="text-right">Traces</TableHead>
                    <TableHead className="text-right">Avg Cost/Trace</TableHead>
                    <TableHead className="text-right">Total Tokens</TableHead>
                    <TableHead className="text-right">Cost/1K Tokens</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {costBreakdown.topCostModels.map((model) => {
                    const costPer1kTokens = (model.totalCost / model.totalTokens) * 1000;
                    return (
                      <TableRow key={model.model}>
                        <TableCell className="font-medium">{model.model}</TableCell>
                        <TableCell className="text-right font-mono">
                          ${model.totalCost.toFixed(4)}
                        </TableCell>
                        <TableCell className="text-right">
                          {model.traceCount.toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          ${model.avgCost.toFixed(6)}
                        </TableCell>
                        <TableCell className="text-right">
                          {model.totalTokens.toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          ${costPer1kTokens.toFixed(6)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* Top Users by Cost */}
          {costBreakdown.topCostUsers && costBreakdown.topCostUsers.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Top Users by Cost</CardTitle>
                <CardDescription>
                  Users with highest spending in selected period
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>User ID</TableHead>
                      <TableHead className="text-right">Total Cost</TableHead>
                      <TableHead className="text-right">Traces</TableHead>
                      <TableHead className="text-right">Avg Cost/Trace</TableHead>
                      <TableHead className="text-right">Total Tokens</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {costBreakdown.topCostUsers.slice(0, 10).map((user) => (
                      <TableRow key={user.userId}>
                        <TableCell className="font-medium">{user.userId}</TableCell>
                        <TableCell className="text-right font-mono">
                          ${user.totalCost.toFixed(4)}
                        </TableCell>
                        <TableCell className="text-right">
                          {user.traceCount.toLocaleString()}
                        </TableCell>
                        <TableCell className="text-right font-mono">
                          ${user.avgCost.toFixed(6)}
                        </TableCell>
                        <TableCell className="text-right">
                          {user.totalTokens.toLocaleString()}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </Card>
          )}

          {/* Cost Insights */}
          <Card>
            <CardHeader>
              <CardTitle>
                <TrendingUp className="mr-2 inline h-5 w-5" />
                Cost Insights
              </CardTitle>
              <CardDescription>
                Recommendations to optimize your spending
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {costBreakdown.topCostModels.length > 0 && (
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium">Most Expensive Model</p>
                  <p className="mt-1 text-lg font-bold">{costBreakdown.topCostModels[0].model}</p>
                  <p className="text-sm text-muted-foreground">
                    Accounts for ${costBreakdown.topCostModels[0].totalCost.toFixed(2)} (
                    {((costBreakdown.topCostModels[0].totalCost / costBreakdown.totalCost) * 100).toFixed(1)}% of total)
                  </p>
                </div>
              )}
              {modelBreakdown && modelBreakdown.length > 0 && (
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium">Average Cost per Trace</p>
                  <p className="mt-1 text-lg font-bold">
                    ${(costBreakdown.totalCost / modelBreakdown.reduce((sum, m) => sum + m.traceCount, 0)).toFixed(6)}
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Consider caching, prompt optimization, or using cheaper models for simpler tasks
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
