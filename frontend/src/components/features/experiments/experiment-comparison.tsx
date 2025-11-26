import React from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  Cell,
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
import type {
  ExperimentComparison,
  StatisticalComparison,
  PairwiseComparison,
} from '@/api/experiments';

// ============================================================================
// Metrics Comparison Chart
// ============================================================================

interface MetricsComparisonChartProps {
  comparison: ExperimentComparison;
  metricName: string;
  metricLabel: string;
  format?: (value: number) => string;
}

export function MetricsComparisonChart({
  comparison,
  metricName,
  metricLabel,
  format = (v) => v.toFixed(2),
}: MetricsComparisonChartProps) {
  const metrics = comparison.metrics[metricName];

  if (!metrics) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{metricLabel}</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">No data available</p>
        </CardContent>
      </Card>
    );
  }

  // Prepare data for each run
  const chartData = comparison.runs.map((run, idx) => ({
    name: `Run #${run.runNumber}`,
    value:
      metricName === 'latency'
        ? run.avgLatencyMs
        : metricName === 'cost'
        ? run.totalCost / run.completedItems
        : 0,
    runId: run.id,
  }));

  const colors = [
    'hsl(var(--chart-1))',
    'hsl(var(--chart-2))',
    'hsl(var(--chart-3))',
    'hsl(var(--chart-4))',
    'hsl(var(--chart-5))',
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{metricLabel}</CardTitle>
        <CardDescription>
          Mean: {format(metrics.mean)} | Median: {format(metrics.median)} |
          StdDev: {format(metrics.stdDev)}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={300}>
          <BarChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis dataKey="name" className="text-xs" />
            <YAxis className="text-xs" />
            <Tooltip
              formatter={(value: number) => format(value)}
              contentStyle={{
                backgroundColor: 'hsl(var(--background))',
                border: '1px solid hsl(var(--border))',
                borderRadius: '6px',
              }}
            />
            <Bar dataKey="value" radius={[4, 4, 0, 0]}>
              {chartData.map((_, index) => (
                <Cell key={`cell-${index}`} fill={colors[index % colors.length]} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>

        {/* Summary Statistics */}
        <div className="mt-4 grid grid-cols-2 gap-4 sm:grid-cols-5">
          <div>
            <p className="text-xs text-muted-foreground">Min</p>
            <p className="text-sm font-medium">{format(metrics.min)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">P25</p>
            <p className="text-sm font-medium">{format(metrics.mean - metrics.stdDev * 0.67)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Median</p>
            <p className="text-sm font-medium">{format(metrics.median)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">P75</p>
            <p className="text-sm font-medium">{format(metrics.mean + metrics.stdDev * 0.67)}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">Max</p>
            <p className="text-sm font-medium">{format(metrics.max)}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Statistical Significance Table
// ============================================================================

interface StatisticalSignificanceTableProps {
  comparison: StatisticalComparison;
}

export function StatisticalSignificanceTable({
  comparison,
}: StatisticalSignificanceTableProps) {
  const metrics = ['latency', 'cost', 'tokens'];
  const metricLabels: Record<string, string> = {
    latency: 'Latency (ms)',
    cost: 'Cost ($)',
    tokens: 'Tokens',
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Statistical Significance Tests</CardTitle>
        <CardDescription>
          Pairwise t-tests comparing experiment runs. Values shown are
          two-tailed p-values.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {metrics.map((metricName) => {
            const tests = comparison.pairwiseTests[metricName] || [];
            if (tests.length === 0) return null;

            return (
              <div key={metricName}>
                <h4 className="mb-3 text-sm font-semibold">
                  {metricLabels[metricName]}
                </h4>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Comparison</TableHead>
                      <TableHead className="text-right">Mean Diff</TableHead>
                      <TableHead className="text-right">Effect Size</TableHead>
                      <TableHead className="text-right">t-stat</TableHead>
                      <TableHead className="text-right">p-value</TableHead>
                      <TableHead>Significance</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tests.map((test, idx) => (
                      <TableRow key={idx}>
                        <TableCell className="font-medium">
                          {test.run1Name} vs {test.run2Name}
                        </TableCell>
                        <TableCell className="text-right">
                          {test.meanDifference.toFixed(2)}
                        </TableCell>
                        <TableCell className="text-right">
                          <EffectSizeBadge effectSize={test.effectSize} />
                        </TableCell>
                        <TableCell className="text-right">
                          {test.tStatistic.toFixed(3)}
                        </TableCell>
                        <TableCell className="text-right">
                          {test.pValue.toFixed(4)}
                        </TableCell>
                        <TableCell>
                          <SignificanceBadge
                            significantAt01={test.significantAt01}
                            significantAt05={test.significantAt05}
                            pValue={test.pValue}
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Pairwise Comparison Detail
// ============================================================================

interface PairwiseComparisonDetailProps {
  comparison: PairwiseComparison;
  metricLabel: string;
  format?: (value: number) => string;
}

export function PairwiseComparisonDetail({
  comparison,
  metricLabel,
  format = (v) => v.toFixed(2),
}: PairwiseComparisonDetailProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>
          {comparison.run1Name} vs {comparison.run2Name}
        </CardTitle>
        <CardDescription>{metricLabel}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Mean Difference */}
          <div>
            <p className="text-sm font-medium">Mean Difference</p>
            <p className="text-2xl font-bold">
              {format(comparison.meanDifference)}
            </p>
            <p className="text-xs text-muted-foreground">
              {comparison.run1Name} - {comparison.run2Name}
            </p>
          </div>

          {/* Effect Size */}
          <div>
            <p className="text-sm font-medium">Effect Size (Cohen's d)</p>
            <div className="flex items-center gap-2">
              <p className="text-2xl font-bold">
                {comparison.effectSize.toFixed(3)}
              </p>
              <EffectSizeBadge effectSize={comparison.effectSize} />
            </div>
            <p className="text-xs text-muted-foreground">
              {getEffectSizeInterpretation(comparison.effectSize)}
            </p>
          </div>

          {/* Statistical Significance */}
          <div>
            <p className="text-sm font-medium">Statistical Significance</p>
            <div className="flex items-center gap-2">
              <p className="text-2xl font-bold">
                p = {comparison.pValue.toFixed(4)}
              </p>
              <SignificanceBadge
                significantAt01={comparison.significantAt01}
                significantAt05={comparison.significantAt05}
                pValue={comparison.pValue}
              />
            </div>
            <p className="text-xs text-muted-foreground">
              t({comparison.degreesOfFreedom}) = {comparison.tStatistic.toFixed(3)}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

function SignificanceBadge({
  significantAt01,
  significantAt05,
  pValue,
}: {
  significantAt01: boolean;
  significantAt05: boolean;
  pValue: number;
}) {
  if (significantAt01) {
    return (
      <Badge variant="default" className="bg-green-600">
        p &lt; 0.01 **
      </Badge>
    );
  }
  if (significantAt05) {
    return (
      <Badge variant="default" className="bg-green-500">
        p &lt; 0.05 *
      </Badge>
    );
  }
  return (
    <Badge variant="secondary">Not Significant</Badge>
  );
}

function EffectSizeBadge({ effectSize }: { effectSize: number }) {
  const absEffect = Math.abs(effectSize);
  let label = 'Negligible';
  let variant: 'default' | 'secondary' | 'outline' = 'secondary';

  if (absEffect >= 0.8) {
    label = 'Large';
    variant = 'default';
  } else if (absEffect >= 0.5) {
    label = 'Medium';
    variant = 'default';
  } else if (absEffect >= 0.2) {
    label = 'Small';
    variant = 'outline';
  }

  return <Badge variant={variant}>{label}</Badge>;
}

function getEffectSizeInterpretation(effectSize: number): string {
  const absEffect = Math.abs(effectSize);
  if (absEffect < 0.2) return 'Negligible practical difference';
  if (absEffect < 0.5) return 'Small practical difference';
  if (absEffect < 0.8) return 'Medium practical difference';
  return 'Large practical difference';
}

// ============================================================================
// Overall Comparison Summary
// ============================================================================

interface ExperimentComparisonSummaryProps {
  comparison: ExperimentComparison;
}

export function ExperimentComparisonSummary({
  comparison,
}: ExperimentComparisonSummaryProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Experiment Runs Overview</CardTitle>
        <CardDescription>
          Comparing {comparison.runs.length} experiment runs
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Run</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Items</TableHead>
              <TableHead className="text-right">Success Rate</TableHead>
              <TableHead className="text-right">Avg Latency</TableHead>
              <TableHead className="text-right">Total Cost</TableHead>
              <TableHead>Completed</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {comparison.runs.map((run) => {
              const successRate =
                run.totalItems > 0
                  ? ((run.completedItems / run.totalItems) * 100).toFixed(1)
                  : '0.0';

              return (
                <TableRow key={run.id}>
                  <TableCell className="font-medium">
                    Run #{run.runNumber}
                  </TableCell>
                  <TableCell>
                    <RunStatusBadge status={run.status} />
                  </TableCell>
                  <TableCell className="text-right">
                    {run.completedItems} / {run.totalItems}
                  </TableCell>
                  <TableCell className="text-right">{successRate}%</TableCell>
                  <TableCell className="text-right">
                    {run.avgLatencyMs.toFixed(0)}ms
                  </TableCell>
                  <TableCell className="text-right">
                    ${run.totalCost.toFixed(4)}
                  </TableCell>
                  <TableCell>
                    {run.completedAt
                      ? new Date(run.completedAt).toLocaleString()
                      : 'In progress'}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function RunStatusBadge({ status }: { status: string }) {
  const variants: Record<string, { variant: any; label: string }> = {
    pending: { variant: 'secondary', label: 'Pending' },
    running: { variant: 'default', label: 'Running' },
    completed: { variant: 'outline', label: 'Completed' },
    failed: { variant: 'destructive', label: 'Failed' },
  };

  const config = variants[status] || variants.pending;
  return <Badge variant={config.variant}>{config.label}</Badge>;
}
