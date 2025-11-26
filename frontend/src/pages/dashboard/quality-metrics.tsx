import React, { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQualityMetrics, useCoreMetrics, type MetricsFilter } from '@/api/metrics';
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
import { Loader2, Star, ThumbsUp, ThumbsDown, TrendingUp, AlertCircle } from 'lucide-react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  RadarChart,
  PolarGrid,
  PolarAngleAxis,
  PolarRadiusAxis,
  Radar,
} from 'recharts';

const COLORS = [
  'hsl(var(--chart-1))',
  'hsl(var(--chart-2))',
  'hsl(var(--chart-3))',
  'hsl(var(--chart-4))',
  'hsl(var(--chart-5))',
];

export default function QualityMetricsDashboard() {
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

  const { data: qualityMetrics, isLoading: isLoadingQuality } = useQualityMetrics(filter);
  const { data: coreMetrics, isLoading: isLoadingCore } = useCoreMetrics(filter);

  if (!projectId) {
    return (
      <div className="container mx-auto py-8">
        <div className="rounded-md border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900 dark:bg-yellow-950">
          <p className="text-sm">Please select a project to view quality metrics.</p>
        </div>
      </div>
    );
  }

  const isLoading = isLoadingQuality || isLoadingCore;

  // Prepare radar chart data
  const radarData =
    qualityMetrics && Object.keys(qualityMetrics.scoresByName).length > 0
      ? Object.entries(qualityMetrics.scoresByName).map(([name, value]) => ({
          metric: name,
          value: value,
        }))
      : [];

  // Prepare bar chart data
  const barData =
    qualityMetrics && Object.keys(qualityMetrics.scoresByName).length > 0
      ? Object.entries(qualityMetrics.scoresByName).map(([name, value]) => ({
          name,
          score: value,
        }))
      : [];

  return (
    <div className="container mx-auto space-y-6 py-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Quality Metrics</h1>
          <p className="mt-1 text-muted-foreground">
            Monitor the quality and performance of your LLM outputs
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

      {!isLoading && qualityMetrics && coreMetrics && (
        <>
          {/* Key Quality Metrics */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Average Score</CardTitle>
                <Star className="h-4 w-4 text-yellow-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {qualityMetrics.avgScore ? qualityMetrics.avgScore.toFixed(2) : 'N/A'}
                </div>
                <p className="text-xs text-muted-foreground">
                  From {qualityMetrics.totalScores.toLocaleString()} evaluations
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Error Rate</CardTitle>
                <AlertCircle className="h-4 w-4 text-red-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(coreMetrics.errorRate * 100).toFixed(2)}%
                </div>
                <p className="text-xs text-muted-foreground">
                  {coreMetrics.errorCount} errors / {coreMetrics.totalTraces} traces
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Positive Feedback</CardTitle>
                <ThumbsUp className="h-4 w-4 text-green-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {qualityMetrics.positiveFeedback.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  {qualityMetrics.feedbackCount > 0
                    ? `${((qualityMetrics.positiveFeedback / qualityMetrics.feedbackCount) * 100).toFixed(1)}% of feedback`
                    : 'No feedback yet'}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Negative Feedback</CardTitle>
                <ThumbsDown className="h-4 w-4 text-red-500" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {qualityMetrics.negativeFeedback.toLocaleString()}
                </div>
                <p className="text-xs text-muted-foreground">
                  {qualityMetrics.feedbackCount > 0
                    ? `${((qualityMetrics.negativeFeedback / qualityMetrics.feedbackCount) * 100).toFixed(1)}% of feedback`
                    : 'No feedback yet'}
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Score Breakdown */}
          {radarData.length > 0 && (
            <div className="grid gap-4 md:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle>Score Distribution (Radar)</CardTitle>
                  <CardDescription>
                    Comparison of different evaluation metrics
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ResponsiveContainer width="100%" height={350}>
                    <RadarChart data={radarData}>
                      <PolarGrid />
                      <PolarAngleAxis dataKey="metric" />
                      <PolarRadiusAxis domain={[0, 1]} />
                      <Radar
                        name="Score"
                        dataKey="value"
                        stroke={COLORS[0]}
                        fill={COLORS[0]}
                        fillOpacity={0.6}
                      />
                      <Tooltip />
                    </RadarChart>
                  </ResponsiveContainer>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>Score Breakdown (Bar Chart)</CardTitle>
                  <CardDescription>
                    Average scores across evaluation dimensions
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ResponsiveContainer width="100%" height={350}>
                    <BarChart data={barData}>
                      <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                      <XAxis dataKey="name" className="text-xs" />
                      <YAxis domain={[0, 1]} className="text-xs" />
                      <Tooltip
                        formatter={(value: number) => value.toFixed(3)}
                        contentStyle={{
                          backgroundColor: 'hsl(var(--background))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '6px',
                        }}
                      />
                      <Bar dataKey="score" fill={COLORS[1]} radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </CardContent>
              </Card>
            </div>
          )}

          {/* Score Details Table */}
          {Object.keys(qualityMetrics.scoresByName).length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Score Details</CardTitle>
                <CardDescription>
                  Detailed breakdown of all evaluation scores
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {Object.entries(qualityMetrics.scoresByName)
                    .sort(([, a], [, b]) => b - a)
                    .map(([name, value]) => (
                      <div key={name} className="space-y-2">
                        <div className="flex items-center justify-between">
                          <p className="text-sm font-medium capitalize">{name.replace(/_/g, ' ')}</p>
                          <p className="text-sm font-mono">{value.toFixed(3)}</p>
                        </div>
                        <div className="h-2 w-full rounded-full bg-muted">
                          <div
                            className="h-2 rounded-full bg-primary transition-all"
                            style={{ width: `${value * 100}%` }}
                          />
                        </div>
                      </div>
                    ))}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Feedback Analysis */}
          {qualityMetrics.feedbackCount > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Feedback Analysis</CardTitle>
                <CardDescription>
                  User feedback sentiment breakdown
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="grid gap-4 md:grid-cols-3">
                    <div className="rounded-lg border p-4">
                      <p className="text-sm font-medium text-muted-foreground">Total Feedback</p>
                      <p className="mt-2 text-2xl font-bold">{qualityMetrics.feedbackCount.toLocaleString()}</p>
                    </div>
                    <div className="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-900 dark:bg-green-950">
                      <p className="text-sm font-medium text-muted-foreground">Positive</p>
                      <p className="mt-2 text-2xl font-bold text-green-700 dark:text-green-400">
                        {qualityMetrics.positiveFeedback.toLocaleString()}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {((qualityMetrics.positiveFeedback / qualityMetrics.feedbackCount) * 100).toFixed(1)}%
                      </p>
                    </div>
                    <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900 dark:bg-red-950">
                      <p className="text-sm font-medium text-muted-foreground">Negative</p>
                      <p className="mt-2 text-2xl font-bold text-red-700 dark:text-red-400">
                        {qualityMetrics.negativeFeedback.toLocaleString()}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {((qualityMetrics.negativeFeedback / qualityMetrics.feedbackCount) * 100).toFixed(1)}%
                      </p>
                    </div>
                  </div>

                  <div className="rounded-lg border p-4">
                    <p className="text-sm font-medium">Feedback Rate</p>
                    <p className="mt-1 text-lg font-bold">
                      {(qualityMetrics.feedbackRate * 100).toFixed(2)}%
                    </p>
                    <p className="text-sm text-muted-foreground">
                      {qualityMetrics.feedbackCount} feedback / {coreMetrics.totalTraces} traces
                    </p>
                    <div className="mt-2 h-2 w-full rounded-full bg-muted">
                      <div
                        className="h-2 rounded-full bg-primary transition-all"
                        style={{ width: `${qualityMetrics.feedbackRate * 100}%` }}
                      />
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Quality Insights */}
          <Card>
            <CardHeader>
              <CardTitle>
                <TrendingUp className="mr-2 inline h-5 w-5" />
                Quality Insights
              </CardTitle>
              <CardDescription>
                Recommendations to improve output quality
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {coreMetrics.errorRate > 0.05 && (
                <div className="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900 dark:bg-red-950">
                  <p className="text-sm font-medium text-red-900 dark:text-red-100">
                    High Error Rate Detected
                  </p>
                  <p className="text-sm text-red-700 dark:text-red-300">
                    Your error rate of {(coreMetrics.errorRate * 100).toFixed(2)}% is higher than recommended.
                    Consider implementing better error handling and input validation.
                  </p>
                </div>
              )}

              {qualityMetrics.feedbackCount > 0 &&
                qualityMetrics.negativeFeedback / qualityMetrics.feedbackCount > 0.3 && (
                  <div className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-900 dark:bg-yellow-950">
                    <p className="text-sm font-medium text-yellow-900 dark:text-yellow-100">
                      Low User Satisfaction
                    </p>
                    <p className="text-sm text-yellow-700 dark:text-yellow-300">
                      {((qualityMetrics.negativeFeedback / qualityMetrics.feedbackCount) * 100).toFixed(1)}%
                      of user feedback is negative. Review your prompt templates and consider A/B testing improvements.
                    </p>
                  </div>
                )}

              {qualityMetrics.feedbackRate < 0.1 && coreMetrics.totalTraces > 100 && (
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium">Low Feedback Collection</p>
                  <p className="text-sm text-muted-foreground">
                    Only {(qualityMetrics.feedbackRate * 100).toFixed(2)}% of traces have feedback.
                    Consider making feedback collection more prominent in your application.
                  </p>
                </div>
              )}

              {Object.keys(qualityMetrics.scoresByName).length === 0 && (
                <div className="rounded-lg border p-4">
                  <p className="text-sm font-medium">No Evaluation Scores</p>
                  <p className="text-sm text-muted-foreground">
                    Set up automated evaluations using LLM-as-a-Judge to continuously monitor quality metrics.
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
