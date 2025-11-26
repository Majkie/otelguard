import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import {
  useScoreDistribution,
  useScoreTrend,
  useScoreBreakdown,
  useF1Score,
  type ScoreAnalyticsParams,
} from '@/api/score-analytics';

const COLORS = ['#0088FE', '#00C49F', '#FFBB28', '#FF8042', '#8884d8', '#82ca9d'];

interface AdvancedAnalyticsProps {
  scoreName: string;
  dimension?: string;
  interval?: '1h' | '6h' | '12h' | '1d' | '1w';
}

export function DistributionChart({ scoreName }: { scoreName: string }) {
  const { data: distribution, isLoading } = useScoreDistribution({ scoreName });

  if (isLoading) {
    return (
      <div className="h-64 flex items-center justify-center text-muted-foreground">
        Loading distribution...
      </div>
    );
  }

  if (!distribution) return null;

  return (
    <div className="space-y-4">
      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Mean</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{distribution.mean.toFixed(3)}</div>
            <p className="text-xs text-muted-foreground">σ = {distribution.stdDev.toFixed(3)}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Median</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{distribution.median.toFixed(3)}</div>
            <p className="text-xs text-muted-foreground">50th percentile</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Range</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {distribution.min.toFixed(2)} - {distribution.max.toFixed(2)}
            </div>
            <p className="text-xs text-muted-foreground">Min - Max</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Count</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{distribution.count.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">samples</p>
          </CardContent>
        </Card>
      </div>

      {/* Histogram */}
      <Card>
        <CardHeader>
          <CardTitle>Distribution Histogram</CardTitle>
          <CardDescription>Frequency distribution across value ranges</CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={distribution.histogram}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="min"
                tickFormatter={(value) => value.toFixed(2)}
                label={{ value: 'Score Range', position: 'insideBottom', offset: -5 }}
              />
              <YAxis label={{ value: 'Frequency', angle: -90, position: 'insideLeft' }} />
              <Tooltip
                formatter={(value: any) => [value, 'Count']}
                labelFormatter={(label) => `Range: ${label.toFixed(2)}`}
              />
              <Bar dataKey="count" fill="#8884d8" />
            </BarChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      {/* Percentiles */}
      <Card>
        <CardHeader>
          <CardTitle>Percentiles</CardTitle>
          <CardDescription>Distribution percentile values</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {Object.entries(distribution.percentiles).map(([key, value]) => (
              <div key={key} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Badge variant="outline">{key.toUpperCase()}</Badge>
                  <span className="text-sm text-muted-foreground">
                    {key === 'p25' && '25th percentile'}
                    {key === 'p50' && '50th percentile (median)'}
                    {key === 'p75' && '75th percentile'}
                    {key === 'p90' && '90th percentile'}
                    {key === 'p95' && '95th percentile'}
                    {key === 'p99' && '99th percentile'}
                  </span>
                </div>
                <span className="font-mono font-semibold">{value.toFixed(3)}</span>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export function TrendChart({ scoreName, interval = '1d' }: { scoreName: string; interval?: '1h' | '6h' | '12h' | '1d' | '1w' }) {
  const { data: trend, isLoading } = useScoreTrend({ scoreName, interval });

  if (isLoading) {
    return (
      <div className="h-96 flex items-center justify-center text-muted-foreground">
        Loading trend...
      </div>
    );
  }

  if (!trend) return null;

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Score Trends Over Time</CardTitle>
            <CardDescription>Track how scores change over time</CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Badge
              variant={
                trend.trend === 'increasing'
                  ? 'default'
                  : trend.trend === 'decreasing'
                  ? 'destructive'
                  : 'secondary'
              }
            >
              {trend.trend === 'increasing' ? (
                <TrendingUp className="h-3 w-3 mr-1" />
              ) : trend.trend === 'decreasing' ? (
                <TrendingDown className="h-3 w-3 mr-1" />
              ) : (
                <Minus className="h-3 w-3 mr-1" />
              )}
              {trend.trend}
            </Badge>
            <span className="text-sm text-muted-foreground">
              {trend.changeRate > 0 ? '+' : ''}
              {trend.changeRate.toFixed(1)}%
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={400}>
          <LineChart data={trend.datapoints}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="timestamp"
              tickFormatter={(value) => new Date(value).toLocaleDateString()}
            />
            <YAxis domain={['dataMin - 0.1', 'dataMax + 0.1']} />
            <Tooltip
              labelFormatter={(label) => new Date(label).toLocaleString()}
              formatter={(value: any) => [value.toFixed(3), 'Mean Score']}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="mean"
              stroke="#8884d8"
              strokeWidth={2}
              dot={{ r: 4 }}
              activeDot={{ r: 6 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  );
}

export function BreakdownChart({ scoreName, dimension = 'model' }: { scoreName: string; dimension?: string }) {
  const { data: breakdown, isLoading } = useScoreBreakdown({ scoreName, dimension });

  if (isLoading) {
    return (
      <div className="h-96 flex items-center justify-center text-muted-foreground">
        Loading breakdown...
      </div>
    );
  }

  if (!breakdown?.values) return null;

  const chartData = Object.entries(breakdown.values).map(([key, stats]) => ({
    name: key,
    mean: stats.mean,
    stdDev: stats.stdDev,
    count: stats.count,
  }));

  return (
    <Card>
      <CardHeader>
        <CardTitle>Score Breakdown by {dimension}</CardTitle>
        <CardDescription>Compare score statistics across different {dimension} values</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Legend />
              <Bar dataKey="mean" fill="#8884d8" name="Mean Score" />
            </BarChart>
          </ResponsiveContainer>

          <div className="grid gap-4 md:grid-cols-3">
            {Object.entries(breakdown.values).map(([key, stats], index) => (
              <Card key={key}>
                <CardHeader className="pb-3">
                  <CardTitle className="text-sm flex items-center gap-2">
                    <div
                      className="h-3 w-3 rounded-full"
                      style={{ backgroundColor: COLORS[index % COLORS.length] }}
                    />
                    {key}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Mean:</span>
                    <span className="font-mono font-semibold">{stats.mean.toFixed(3)}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Std Dev:</span>
                    <span className="font-mono">{stats.stdDev.toFixed(3)}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Count:</span>
                    <span className="font-mono">{stats.count.toLocaleString()}</span>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function F1ScoreMetrics({ scoreName, threshold = 0.5 }: { scoreName: string; threshold?: number }) {
  const { data: f1Result, isLoading } = useF1Score({ scoreName, threshold });

  if (isLoading) {
    return (
      <div className="h-64 flex items-center justify-center text-muted-foreground">
        Loading metrics...
      </div>
    );
  }

  if (!f1Result) return null;

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Classification Metrics</CardTitle>
          <CardDescription>F1 Score and related metrics</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-muted-foreground">F1 Score</p>
                <p className="text-2xl font-bold">{f1Result.f1Score.toFixed(3)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Accuracy</p>
                <p className="text-2xl font-bold">{f1Result.accuracy.toFixed(3)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Precision</p>
                <p className="text-2xl font-bold">{f1Result.precision.toFixed(3)}</p>
              </div>
              <div>
                <p className="text-sm text-muted-foreground">Recall</p>
                <p className="text-2xl font-bold">{f1Result.recall.toFixed(3)}</p>
              </div>
            </div>

            <div className="pt-4 border-t">
              <p className="text-sm font-medium mb-3">Confusion Matrix</p>
              <div className="grid grid-cols-2 gap-2 text-center">
                <div className="p-4 bg-green-50 border border-green-200 rounded">
                  <p className="text-xs text-muted-foreground">True Positives</p>
                  <p className="text-xl font-bold">{f1Result.truePositives}</p>
                </div>
                <div className="p-4 bg-red-50 border border-red-200 rounded">
                  <p className="text-xs text-muted-foreground">False Positives</p>
                  <p className="text-xl font-bold">{f1Result.falsePositives}</p>
                </div>
                <div className="p-4 bg-red-50 border border-red-200 rounded">
                  <p className="text-xs text-muted-foreground">False Negatives</p>
                  <p className="text-xl font-bold">{f1Result.falseNegatives}</p>
                </div>
                <div className="p-4 bg-green-50 border border-green-200 rounded">
                  <p className="text-xs text-muted-foreground">True Negatives</p>
                  <p className="text-xl font-bold">{f1Result.trueNegatives}</p>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Performance Visualization</CardTitle>
          <CardDescription>Visual representation of classification metrics</CardDescription>
        </CardHeader>
        <CardContent>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie
                data={[
                  { name: 'True Positives', value: f1Result.truePositives },
                  { name: 'True Negatives', value: f1Result.trueNegatives },
                  { name: 'False Positives', value: f1Result.falsePositives },
                  { name: 'False Negatives', value: f1Result.falseNegatives },
                ]}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, percent }) => `${name.split(' ')[0]}: ${(percent * 100).toFixed(0)}%`}
                outerRadius={100}
                fill="#8884d8"
                dataKey="value"
              >
                <Cell fill="#10b981" />
                <Cell fill="#6366f1" />
                <Cell fill="#ef4444" />
                <Cell fill="#f97316" />
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>
    </div>
  );
}
