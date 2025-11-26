import React from 'react';
import {
  LineChart,
  Line,
  BarChart,
  Bar,
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
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import {
  AlertTriangle,
  Shield,
  TrendingDown,
  TrendingUp,
  DollarSign,
  Clock,
} from 'lucide-react';
import type {
  TriggerStats,
  ViolationTrend,
  RemediationSuccessRate,
  CostImpactAnalysis,
  LatencyImpact,
} from '@/api/guardrail-analytics';

// ============================================================================
// Colors
// ============================================================================

const COLORS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
];

// ============================================================================
// Trigger Dashboard
// ============================================================================

interface TriggerDashboardProps {
  stats: TriggerStats;
}

export function TriggerDashboard({ stats }: TriggerDashboardProps) {
  return (
    <div className="space-y-6">
      {/* Overview Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <MetricCard
          icon={<Shield className="h-4 w-4" />}
          label="Total Evaluations"
          value={stats.totalEvaluations.toLocaleString()}
          trend={null}
        />
        <MetricCard
          icon={<AlertTriangle className="h-4 w-4" />}
          label="Triggers"
          value={stats.totalTriggered.toLocaleString()}
          subtitle={`${(stats.triggerRate * 100).toFixed(2)}% trigger rate`}
          trend={null}
        />
        <MetricCard
          icon={<Shield className="h-4 w-4" />}
          label="Actions Taken"
          value={stats.totalActioned.toLocaleString()}
          subtitle={`${(stats.actionRate * 100).toFixed(2)}% action rate`}
          trend={null}
        />
        <MetricCard
          icon={<Shield className="h-4 w-4" />}
          label="Blocked Requests"
          value={stats.totalActioned.toLocaleString()}
          trend={null}
        />
      </div>

      {/* By Policy Stats */}
      <Card>
        <CardHeader>
          <CardTitle>Triggers by Policy</CardTitle>
          <CardDescription>
            Policy-level trigger and action statistics
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Policy</TableHead>
                <TableHead className="text-right">Evaluations</TableHead>
                <TableHead className="text-right">Triggers</TableHead>
                <TableHead className="text-right">Actions</TableHead>
                <TableHead className="text-right">Trigger Rate</TableHead>
                <TableHead className="text-right">Avg Latency</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.values(stats.byPolicy).map((policy) => (
                <TableRow key={policy.policyId}>
                  <TableCell className="font-medium">
                    {policy.policyName}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.evaluationCount.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.triggerCount.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.actionCount.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Badge variant="outline">
                      {(policy.triggerRate * 100).toFixed(2)}%
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.avgLatencyMs.toFixed(1)}ms
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* By Rule Type */}
      <Card>
        <CardHeader>
          <CardTitle>Triggers by Rule Type</CardTitle>
          <CardDescription>
            Rule type breakdown with latency metrics
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart
              data={Object.values(stats.byRuleType).map((rule) => ({
                name: rule.ruleType.replace(/_/g, ' '),
                triggers: rule.triggerCount,
                actions: rule.actionCount,
              }))}
            >
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="name" className="text-xs" angle={-45} height={100} textAnchor="end" />
              <YAxis className="text-xs" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                }}
              />
              <Legend />
              <Bar dataKey="triggers" fill={COLORS[0]} radius={[4, 4, 0, 0]} />
              <Bar dataKey="actions" fill={COLORS[1]} radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* By Action Type */}
      <Card>
        <CardHeader>
          <CardTitle>Actions Breakdown</CardTitle>
          <CardDescription>Distribution of remediation actions</CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={Object.values(stats.byAction).map((action) => ({
                  name: action.actionType,
                  value: action.actionCount,
                }))}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={(entry) => `${entry.name}: ${entry.value}`}
                outerRadius={100}
                fill="#8884d8"
                dataKey="value"
              >
                {Object.values(stats.byAction).map((_, index) => (
                  <Cell
                    key={`cell-${index}`}
                    fill={COLORS[index % COLORS.length]}
                  />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  );
}

// ============================================================================
// Violation Trends Chart
// ============================================================================

interface ViolationTrendsChartProps {
  trends: ViolationTrend[];
}

export function ViolationTrendsChart({ trends }: ViolationTrendsChartProps) {
  const data = trends.map((trend) => ({
    timestamp: new Date(trend.timestamp).toLocaleTimeString(),
    evaluations: trend.evaluationCount,
    triggers: trend.triggerCount,
    actions: trend.actionCount,
    triggerRate: trend.triggerRate * 100,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle>Violation Trends</CardTitle>
        <CardDescription>
          Guardrail triggers and actions over time
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={400}>
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey="timestamp" className="text-xs" />
            <YAxis className="text-xs" />
            <Tooltip
              contentStyle={{
                backgroundColor: 'hsl(var(--background))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '6px',
              }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="triggers"
              stroke={COLORS[0]}
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="actions"
              stroke={COLORS[1]}
              strokeWidth={2}
              dot={false}
            />
            <Line
              type="monotone"
              dataKey="triggerRate"
              stroke={COLORS[2]}
              strokeWidth={2}
              dot={false}
              yAxisId="right"
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Remediation Success Rates
// ============================================================================

interface RemediationSuccessTableProps {
  rates: RemediationSuccessRate[];
}

export function RemediationSuccessTable({ rates }: RemediationSuccessTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Remediation Success Rates</CardTitle>
        <CardDescription>
          Effectiveness of each remediation action
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Action Type</TableHead>
              <TableHead className="text-right">Total Attempts</TableHead>
              <TableHead className="text-right">Successful</TableHead>
              <TableHead className="text-right">Success Rate</TableHead>
              <TableHead className="text-right">Avg Latency</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rates.map((rate) => (
              <TableRow key={rate.actionType}>
                <TableCell className="font-medium capitalize">
                  {rate.actionType.replace(/_/g, ' ')}
                </TableCell>
                <TableCell className="text-right">
                  {rate.totalAttempts.toLocaleString()}
                </TableCell>
                <TableCell className="text-right">
                  {rate.successfulCount.toLocaleString()}
                </TableCell>
                <TableCell className="text-right">
                  <SuccessRateBadge rate={rate.successRate} />
                </TableCell>
                <TableCell className="text-right">
                  {rate.avgLatencyMs.toFixed(1)}ms
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Cost Impact Analysis
// ============================================================================

interface CostImpactCardProps {
  analysis: CostImpactAnalysis;
}

export function CostImpactCard({ analysis }: CostImpactCardProps) {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard
          icon={<DollarSign className="h-4 w-4" />}
          label="Estimated Cost Savings"
          value={`$${analysis.estimatedCostSavings.toFixed(2)}`}
          subtitle="From blocked requests"
          trend={null}
        />
        <MetricCard
          icon={<Shield className="h-4 w-4" />}
          label="Total Evaluations"
          value={analysis.totalEvaluations.toLocaleString()}
          trend={null}
        />
        <MetricCard
          icon={<Clock className="h-4 w-4" />}
          label="Avg Evaluation Latency"
          value={`${analysis.avgLatencyMs.toFixed(1)}ms`}
          trend={null}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Cost Impact by Policy</CardTitle>
          <CardDescription>
            Cost savings and latency impact per policy
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Policy</TableHead>
                <TableHead className="text-right">Evaluations</TableHead>
                <TableHead className="text-right">Blocked</TableHead>
                <TableHead className="text-right">Cost Savings</TableHead>
                <TableHead className="text-right">Latency Impact</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.values(analysis.byPolicy).map((policy) => (
                <TableRow key={policy.policyId}>
                  <TableCell className="font-medium">
                    {policy.policyName}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.evaluationCount.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.blockedCount.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right text-green-600">
                    ${policy.estimatedCostSavings.toFixed(2)}
                  </TableCell>
                  <TableCell className="text-right">
                    {policy.avgLatencyMs.toFixed(1)}ms
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

// ============================================================================
// Latency Impact
// ============================================================================

interface LatencyImpactCardProps {
  impact: LatencyImpact;
}

export function LatencyImpactCard({ impact }: LatencyImpactCardProps) {
  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-5">
        <MetricCard
          label="Avg Latency"
          value={`${impact.avgLatencyMs.toFixed(1)}ms`}
          trend={null}
        />
        <MetricCard
          label="Median"
          value={`${impact.medianLatencyMs.toFixed(1)}ms`}
          trend={null}
        />
        <MetricCard
          label="P95"
          value={`${impact.p95LatencyMs.toFixed(1)}ms`}
          trend={null}
        />
        <MetricCard
          label="P99"
          value={`${impact.p99LatencyMs.toFixed(1)}ms`}
          trend={null}
        />
        <MetricCard
          label="Impact"
          value={`${impact.impactPercentage.toFixed(1)}%`}
          subtitle="Of total latency"
          trend={null}
        />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Latency by Rule Type</CardTitle>
          <CardDescription>
            Average latency and evaluation count by rule type
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart
              data={Object.entries(impact.byRuleType).map(([type, data]) => ({
                name: type.replace(/_/g, ' '),
                latency: data.avgLatencyMs,
                count: data.count,
              }))}
            >
              <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
              <XAxis dataKey="name" className="text-xs" angle={-45} height={100} textAnchor="end" />
              <YAxis className="text-xs" />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'hsl(var(--background))',
                  border: '1px solid hsl(var(--border))',
                  borderRadius: '6px',
                }}
              />
              <Legend />
              <Bar dataKey="latency" fill={COLORS[0]} name="Avg Latency (ms)" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

interface MetricCardProps {
  icon?: React.ReactNode;
  label: string;
  value: string;
  subtitle?: string;
  trend?: 'up' | 'down' | null;
}

function MetricCard({ icon, label, value, subtitle, trend }: MetricCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{label}</CardTitle>
        {icon && <div className="text-muted-foreground">{icon}</div>}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {subtitle && (
          <p className="text-xs text-muted-foreground">{subtitle}</p>
        )}
        {trend && (
          <div className="flex items-center pt-1">
            {trend === 'up' && <TrendingUp className="mr-1 h-4 w-4 text-green-500" />}
            {trend === 'down' && <TrendingDown className="mr-1 h-4 w-4 text-red-500" />}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function SuccessRateBadge({ rate }: { rate: number }) {
  const percentage = rate * 100;
  let variant: 'default' | 'destructive' | 'outline' = 'default';

  if (percentage >= 95) {
    variant = 'default';
  } else if (percentage >= 80) {
    variant = 'outline';
  } else {
    variant = 'destructive';
  }

  return <Badge variant={variant}>{percentage.toFixed(1)}%</Badge>;
}
