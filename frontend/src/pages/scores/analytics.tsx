import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  useScoreAggregations,
  useScoreTrends,
  useScoreComparisons,
} from '@/api/scores';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, BarChart3, TrendingUp, GitCompare } from 'lucide-react';

type AnalyticsParams = {
  name?: string;
  source?: 'api' | 'llm_judge' | 'human' | 'user_feedback';
  startTime?: string;
  endTime?: string;
};

function ScoreAnalyticsPage() {
  const [params, setParams] = useState<AnalyticsParams>({});
  const [trendsGroupBy, setTrendsGroupBy] = useState<'hour' | 'day' | 'week' | 'month'>('day');
  const [comparisonDimension, setComparisonDimension] = useState<'model' | 'user' | 'session' | 'prompt'>('model');

  const { data: aggregations, isLoading: aggregationsLoading } = useScoreAggregations(params);
  const { data: trends, isLoading: trendsLoading } = useScoreTrends({
    ...params,
    groupBy: trendsGroupBy,
  });
  const { data: comparisons, isLoading: comparisonsLoading } = useScoreComparisons({
    ...params,
    dimension: comparisonDimension,
  });

  const handleParamChange = (key: keyof AnalyticsParams, value: any) => {
    setParams(prev => ({
      ...prev,
      [key]: value || undefined,
    }));
  };

  const clearFilters = () => {
    setParams({});
  };

  const formatNumber = (num: number) => num.toFixed(3);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/scores">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Scores
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">Score Analytics</h1>
          <p className="text-muted-foreground">
            Analyze score distributions, trends, and comparisons
          </p>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Filters</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label className="text-sm font-medium mb-1 block">Score Name</label>
              <Input
                placeholder="Filter by score name..."
                value={params.name || ''}
                onChange={(e) => handleParamChange('name', e.target.value)}
              />
            </div>

            <div>
              <label className="text-sm font-medium mb-1 block">Source</label>
              <Select
                value={params.source || ''}
                onValueChange={(value) => handleParamChange('source', value || undefined)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All sources" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All sources</SelectItem>
                  <SelectItem value="api">API</SelectItem>
                  <SelectItem value="llm_judge">LLM Judge</SelectItem>
                  <SelectItem value="human">Human</SelectItem>
                  <SelectItem value="user_feedback">User Feedback</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div>
              <label className="text-sm font-medium mb-1 block">Start Time</label>
              <Input
                type="datetime-local"
                value={params.startTime || ''}
                onChange={(e) => handleParamChange('startTime', e.target.value)}
              />
            </div>

            <div>
              <label className="text-sm font-medium mb-1 block">End Time</label>
              <Input
                type="datetime-local"
                value={params.endTime || ''}
                onChange={(e) => handleParamChange('endTime', e.target.value)}
              />
            </div>
          </div>

          {Object.values(params).some(v => v) && (
            <div className="mt-4">
              <Button variant="outline" onClick={clearFilters}>
                Clear Filters
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Analytics Tabs */}
      <Tabs defaultValue="aggregations" className="space-y-4">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="aggregations" className="flex items-center gap-2">
            <BarChart3 className="h-4 w-4" />
            Aggregations
          </TabsTrigger>
          <TabsTrigger value="trends" className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4" />
            Trends
          </TabsTrigger>
          <TabsTrigger value="comparisons" className="flex items-center gap-2">
            <GitCompare className="h-4 w-4" />
            Comparisons
          </TabsTrigger>
        </TabsList>

        {/* Aggregations Tab */}
        <TabsContent value="aggregations" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Score Aggregations</CardTitle>
              <p className="text-sm text-muted-foreground">
                Statistical summaries of your evaluation scores
              </p>
            </CardHeader>
            <CardContent>
              {aggregationsLoading ? (
                <div className="flex items-center justify-center h-64">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              ) : aggregations?.aggregations.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No score data available</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {aggregations?.aggregations.map((agg) => (
                    <Card key={`${agg.name}-${agg.dataType}`}>
                      <CardHeader className="pb-3">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-lg">{agg.name}</CardTitle>
                          <Badge variant="outline">{agg.dataType}</Badge>
                        </div>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                          <div>
                            <p className="text-2xl font-bold">{agg.count}</p>
                            <p className="text-xs text-muted-foreground">Total Count</p>
                          </div>

                          {agg.dataType === 'numeric' && agg.avgValue !== undefined && (
                            <>
                              <div>
                                <p className="text-2xl font-bold">{formatNumber(agg.avgValue)}</p>
                                <p className="text-xs text-muted-foreground">Average</p>
                              </div>
                              <div>
                                <p className="text-2xl font-bold">{formatNumber(agg.minValue!)}</p>
                                <p className="text-xs text-muted-foreground">Min</p>
                              </div>
                              <div>
                                <p className="text-2xl font-bold">{formatNumber(agg.maxValue!)}</p>
                                <p className="text-xs text-muted-foreground">Max</p>
                              </div>
                            </>
                          )}
                        </div>

                        {agg.dataType === 'categorical' && agg.categories && (
                          <div>
                            <h4 className="font-medium mb-2">Categories</h4>
                            <div className="flex flex-wrap gap-2">
                              {Object.entries(agg.categories)
                                .sort(([,a], [,b]) => b - a)
                                .slice(0, 10)
                                .map(([category, count]) => (
                                  <Badge key={category} variant="secondary">
                                    {category}: {count}
                                  </Badge>
                                ))}
                            </div>
                          </div>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Trends Tab */}
        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Score Trends</CardTitle>
                  <p className="text-sm text-muted-foreground">
                    Score performance over time
                  </p>
                </div>
                <Select value={trendsGroupBy} onValueChange={(value: any) => setTrendsGroupBy(value)}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="hour">Hourly</SelectItem>
                    <SelectItem value="day">Daily</SelectItem>
                    <SelectItem value="week">Weekly</SelectItem>
                    <SelectItem value="month">Monthly</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent>
              {trendsLoading ? (
                <div className="flex items-center justify-center h-64">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              ) : trends?.trends.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No trend data available</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {trends?.trends.map((trend) => (
                    <Card key={`${trend.timePeriod}-${trend.name}`}>
                      <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                          <div>
                            <h3 className="font-medium">{trend.name}</h3>
                            <p className="text-sm text-muted-foreground">
                              {new Date(trend.timePeriod).toLocaleDateString()}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-2xl font-bold">
                              {trend.avgValue !== undefined ? formatNumber(trend.avgValue) : trend.count}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {trend.avgValue !== undefined ? 'Average' : 'Count'}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Comparisons Tab */}
        <TabsContent value="comparisons" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Score Comparisons</CardTitle>
                  <p className="text-sm text-muted-foreground">
                    Compare scores across different dimensions
                  </p>
                </div>
                <Select value={comparisonDimension} onValueChange={(value: any) => setComparisonDimension(value)}>
                  <SelectTrigger className="w-32">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="model">Model</SelectItem>
                    <SelectItem value="user">User</SelectItem>
                    <SelectItem value="session">Session</SelectItem>
                    <SelectItem value="prompt">Prompt</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardHeader>
            <CardContent>
              {comparisonsLoading ? (
                <div className="flex items-center justify-center h-64">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              ) : comparisons?.comparisons.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No comparison data available</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {comparisons?.comparisons.map((comp) => (
                    <Card key={`${comp.dimension}-${comp.value}-${comp.name}`}>
                      <CardContent className="pt-6">
                        <div className="flex items-center justify-between">
                          <div>
                            <h3 className="font-medium">{comp.name}</h3>
                            <p className="text-sm text-muted-foreground">
                              {comp.dimension}: {comp.value}
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="text-2xl font-bold">
                              {comp.avgValue !== undefined ? formatNumber(comp.avgValue) : comp.count}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              {comp.avgValue !== undefined ? 'Average' : 'Count'}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default ScoreAnalyticsPage;
