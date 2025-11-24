import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  useFeedbackAnalytics,
  useFeedbackTrends,
} from '@/api/feedback';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, BarChart3, TrendingUp, ThumbsUp, ThumbsDown, Star, MessageSquare } from 'lucide-react';

type AnalyticsParams = {
  itemType: 'trace' | 'session' | 'span' | 'prompt';
  startDate: string;
  endDate: string;
};

function FeedbackAnalyticsPage() {
  const [params, setParams] = useState<AnalyticsParams>({
    itemType: 'trace',
    startDate: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0], // 30 days ago
    endDate: new Date().toISOString().split('T')[0], // Today
  });

  const [trendsInterval, setTrendsInterval] = useState<'hour' | 'day' | 'week' | 'month'>('day');

  const { data: analytics, isLoading: analyticsLoading } = useFeedbackAnalytics(
    params.itemType,
    params.startDate,
    params.endDate
  );

  const { data: trends, isLoading: trendsLoading } = useFeedbackTrends(
    params.itemType,
    params.startDate,
    params.endDate,
    trendsInterval
  );

  const handleParamChange = (key: keyof AnalyticsParams, value: any) => {
    setParams(prev => ({
      ...prev,
      [key]: value,
    }));
  };

  const formatNumber = (num: number) => num.toLocaleString();
  const formatPercentage = (num: number) => `${(num * 100).toFixed(1)}%`;
  const formatRating = (num: number) => num.toFixed(1);

  const getThumbsUpRate = () => {
    if (!analytics) return 0;
    const total = analytics.thumbsUpCount + analytics.thumbsDownCount;
    return total > 0 ? analytics.thumbsUpCount / total : 0;
  };

  const getRatingDistribution = () => {
    if (!analytics?.ratingCounts) return [];
    return Object.entries(analytics.ratingCounts)
      .map(([rating, count]) => ({
        rating: parseInt(rating),
        count,
        percentage: analytics.totalFeedback > 0 ? (count / analytics.totalFeedback) * 100 : 0,
      }))
      .sort((a, b) => b.rating - a.rating);
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" asChild>
            <Link to="/feedback">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back to Feedback
            </Link>
          </Button>
          <div>
            <h1 className="text-2xl font-bold">Feedback Analytics</h1>
            <p className="text-muted-foreground">
              Analyze user feedback patterns and trends
            </p>
          </div>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Item Type</label>
              <Select
                value={params.itemType}
                onValueChange={(value: any) => handleParamChange('itemType', value)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="trace">Traces</SelectItem>
                  <SelectItem value="session">Sessions</SelectItem>
                  <SelectItem value="span">Spans</SelectItem>
                  <SelectItem value="prompt">Prompts</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Start Date</label>
              <input
                type="date"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                value={params.startDate}
                onChange={(e) => handleParamChange('startDate', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">End Date</label>
              <input
                type="date"
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                value={params.endDate}
                onChange={(e) => handleParamChange('endDate', e.target.value)}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Analytics Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Feedback</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analyticsLoading ? '...' : formatNumber(analytics?.totalFeedback || 0)}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Thumbs Up Rate</CardTitle>
            <ThumbsUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analyticsLoading ? '...' : formatPercentage(getThumbsUpRate())}
            </div>
            <p className="text-xs text-muted-foreground">
              {analytics?.thumbsUpCount || 0} up / {(analytics?.thumbsUpCount || 0) + (analytics?.thumbsDownCount || 0)} total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Average Rating</CardTitle>
            <Star className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analyticsLoading ? '...' : analytics?.averageRating ? formatRating(analytics.averageRating) : 'N/A'}
            </div>
            <p className="text-xs text-muted-foreground">
              Out of 5 stars
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Comments</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {analyticsLoading ? '...' : formatNumber(analytics?.commentCount || 0)}
            </div>
            <p className="text-xs text-muted-foreground">
              {analytics?.totalFeedback ?
                formatPercentage((analytics.commentCount || 0) / analytics.totalFeedback) :
                '0%'
              } have comments
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Detailed Analytics */}
      <Tabs defaultValue="distribution" className="space-y-4">
        <TabsList>
          <TabsTrigger value="distribution">Rating Distribution</TabsTrigger>
          <TabsTrigger value="trends">Trends</TabsTrigger>
        </TabsList>

        <TabsContent value="distribution" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Rating Distribution</CardTitle>
            </CardHeader>
            <CardContent>
              {analyticsLoading ? (
                <div className="text-center py-8">Loading...</div>
              ) : (
                <div className="space-y-4">
                  {getRatingDistribution().map(({ rating, count, percentage }) => (
                    <div key={rating} className="flex items-center gap-4">
                      <div className="flex items-center gap-2 min-w-[80px]">
                        <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
                        <span className="font-medium">{rating}</span>
                      </div>
                      <div className="flex-1">
                        <div className="w-full bg-gray-200 rounded-full h-2">
                          <div
                            className="bg-blue-600 h-2 rounded-full"
                            style={{ width: `${percentage}%` }}
                          ></div>
                        </div>
                      </div>
                      <div className="text-sm text-muted-foreground min-w-[60px] text-right">
                        {count} ({percentage.toFixed(1)}%)
                      </div>
                    </div>
                  ))}
                  {getRatingDistribution().length === 0 && (
                    <div className="text-center py-8 text-muted-foreground">
                      No ratings found for the selected period
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="trends" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2">
                  <TrendingUp className="h-5 w-5" />
                  Feedback Trends
                </CardTitle>
                <Select
                  value={trendsInterval}
                  onValueChange={(value: any) => setTrendsInterval(value)}
                >
                  <SelectTrigger className="w-[120px]">
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
                <div className="text-center py-8">Loading trends...</div>
              ) : trends && trends.length > 0 ? (
                <div className="space-y-4">
                  {trends.slice(-30).map((trend, index) => (
                    <div key={index} className="flex items-center justify-between p-3 border rounded-lg">
                      <div className="font-medium">{trend.date}</div>
                      <div className="flex items-center gap-6 text-sm">
                        <div className="text-center">
                          <div className="font-medium">{trend.totalFeedback}</div>
                          <div className="text-muted-foreground">Feedback</div>
                        </div>
                        <div className="text-center">
                          <div className="font-medium">{formatPercentage(trend.thumbsUpRate)}</div>
                          <div className="text-muted-foreground">Thumbs Up</div>
                        </div>
                        <div className="text-center">
                          <div className="font-medium">{formatRating(trend.averageRating)}</div>
                          <div className="text-muted-foreground">Avg Rating</div>
                        </div>
                        <div className="text-center">
                          <div className="font-medium">{trend.commentCount}</div>
                          <div className="text-muted-foreground">Comments</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  No trend data available for the selected period
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default FeedbackAnalyticsPage;
