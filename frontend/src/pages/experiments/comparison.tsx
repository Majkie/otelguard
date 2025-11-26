import React, { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  useExperimentComparison,
  useStatisticalComparison,
} from '@/api/experiments';
import {
  MetricsComparisonChart,
  StatisticalSignificanceTable,
  ExperimentComparisonSummary,
} from '@/components/features/experiments/experiment-comparison';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Loader2, AlertCircle, TrendingUp } from 'lucide-react';
import { Checkbox } from '@/components/ui/checkbox';

/**
 * ExperimentComparisonPage
 *
 * Displays side-by-side comparison of multiple experiment runs with:
 * - Basic metrics comparison (latency, cost, tokens)
 * - Statistical significance testing (t-tests, p-values)
 * - Effect size analysis (Cohen's d)
 * - Visual charts and tables
 *
 * Usage:
 *   Navigate to /experiments/compare?runs=run1-id,run2-id,run3-id
 */
export default function ExperimentComparisonPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [showStatistical, setShowStatistical] = useState(true);

  // Get run IDs from URL query params
  const runIdsParam = searchParams.get('runs') || '';
  const runIds = runIdsParam
    .split(',')
    .map((id) => id.trim())
    .filter((id) => id.length > 0);

  // Fetch comparison data
  const {
    data: basicComparison,
    isLoading: isLoadingBasic,
    error: basicError,
  } = useExperimentComparison(runIds);

  const {
    data: statisticalComparison,
    isLoading: isLoadingStats,
    error: statsError,
  } = useStatisticalComparison(runIds);

  // Loading state
  if (isLoadingBasic || (showStatistical && isLoadingStats)) {
    return (
      <div className="flex h-96 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // Error state
  if (basicError || (showStatistical && statsError)) {
    return (
      <div className="container mx-auto py-8">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            {basicError?.message ||
              statsError?.message ||
              'Failed to load comparison data'}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // No runs selected
  if (runIds.length === 0) {
    return (
      <div className="container mx-auto py-8">
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            No experiment runs selected for comparison. Please select at least 2
            runs from the experiments list.
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  // Need at least 2 runs
  if (runIds.length < 2) {
    return (
      <div className="container mx-auto py-8">
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            At least 2 experiment runs are required for comparison. Currently
            selected: {runIds.length}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!basicComparison) {
    return null;
  }

  return (
    <div className="container mx-auto space-y-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Experiment Comparison</h1>
          <p className="mt-1 text-muted-foreground">
            Comparing {runIds.length} experiment runs
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center space-x-2">
            <Checkbox
              id="show-statistical"
              checked={showStatistical}
              onCheckedChange={(checked) => setShowStatistical(!!checked)}
            />
            <label
              htmlFor="show-statistical"
              className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              Show Statistical Analysis
            </label>
          </div>
          <Button variant="outline" onClick={() => window.history.back()}>
            Back to Experiments
          </Button>
        </div>
      </div>

      {/* Runs Overview */}
      <ExperimentComparisonSummary comparison={basicComparison} />

      {/* Metrics Comparison */}
      <Tabs defaultValue="overview" className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="metrics">Metrics</TabsTrigger>
          {showStatistical && (
            <TabsTrigger value="statistical">
              <TrendingUp className="mr-2 h-4 w-4" />
              Statistical Analysis
            </TabsTrigger>
          )}
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="space-y-6">
          <div className="grid gap-6 md:grid-cols-2">
            <MetricsComparisonChart
              comparison={basicComparison}
              metricName="latency"
              metricLabel="Average Latency"
              format={(v) => `${v.toFixed(0)}ms`}
            />
            <MetricsComparisonChart
              comparison={basicComparison}
              metricName="cost"
              metricLabel="Average Cost per Item"
              format={(v) => `$${v.toFixed(4)}`}
            />
          </div>
          <MetricsComparisonChart
            comparison={basicComparison}
            metricName="tokens"
            metricLabel="Average Tokens"
            format={(v) => v.toFixed(0)}
          />
        </TabsContent>

        {/* Metrics Details Tab */}
        <TabsContent value="metrics" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Detailed Metrics Breakdown</CardTitle>
              <CardDescription>
                Statistical measures for each metric across all runs
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-6">
                {Object.entries(basicComparison.metrics).map(
                  ([metricName, metrics]) => (
                    <div key={metricName}>
                      <h3 className="mb-2 text-lg font-semibold capitalize">
                        {metricName}
                      </h3>
                      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-6">
                        <MetricCard label="Mean" value={metrics.mean} />
                        <MetricCard label="Median" value={metrics.median} />
                        <MetricCard label="Std Dev" value={metrics.stdDev} />
                        <MetricCard label="Min" value={metrics.min} />
                        <MetricCard label="Max" value={metrics.max} />
                        <MetricCard label="Sample Size" value={metrics.n} />
                      </div>
                    </div>
                  )
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Statistical Analysis Tab */}
        {showStatistical && statisticalComparison && (
          <TabsContent value="statistical" className="space-y-6">
            <Alert>
              <TrendingUp className="h-4 w-4" />
              <AlertDescription>
                Statistical significance tests use two-tailed t-tests with
                Welch's correction for unequal variances. Effect sizes are
                calculated using Cohen's d. * indicates p &lt; 0.05, **
                indicates p &lt; 0.01.
              </AlertDescription>
            </Alert>

            <StatisticalSignificanceTable comparison={statisticalComparison} />

            {/* Additional Statistical Info */}
            <Card>
              <CardHeader>
                <CardTitle>Interpretation Guide</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <h4 className="font-semibold">P-Value</h4>
                  <p className="text-sm text-muted-foreground">
                    The probability that the observed difference occurred by
                    chance. Lower values indicate more confidence that the
                    difference is real.
                    <br />
                    • p &lt; 0.05: Significant (*)
                    <br />
                    • p &lt; 0.01: Highly significant (**)
                    <br />• p ≥ 0.05: Not statistically significant
                  </p>
                </div>
                <div>
                  <h4 className="font-semibold">Effect Size (Cohen's d)</h4>
                  <p className="text-sm text-muted-foreground">
                    Measures the magnitude of the difference, independent of
                    sample size.
                    <br />
                    • |d| &lt; 0.2: Negligible
                    <br />
                    • |d| ≥ 0.2: Small
                    <br />
                    • |d| ≥ 0.5: Medium
                    <br />• |d| ≥ 0.8: Large
                  </p>
                </div>
                <div>
                  <h4 className="font-semibold">Practical Interpretation</h4>
                  <p className="text-sm text-muted-foreground">
                    A result can be statistically significant (low p-value) but
                    have negligible practical impact (small effect size), or
                    vice versa. Consider both metrics when making decisions.
                  </p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        )}
      </Tabs>
    </div>
  );
}

// ============================================================================
// Helper Components
// ============================================================================

interface MetricCardProps {
  label: string;
  value: number;
  format?: (value: number) => string;
}

function MetricCard({ label, value, format }: MetricCardProps) {
  const displayValue = format ? format(value) : value.toFixed(2);

  return (
    <div className="rounded-lg border p-3">
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-semibold">{displayValue}</p>
    </div>
  );
}
