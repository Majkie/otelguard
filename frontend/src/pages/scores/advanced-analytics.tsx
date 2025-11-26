import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Badge } from '@/components/ui/badge';
import { ArrowLeft, BarChart, LineChart as LineChartIcon, GitCompare, Users, Target, TrendingUp } from 'lucide-react';
import {
  DistributionChart,
  TrendChart,
  BreakdownChart,
  F1ScoreMetrics,
} from '@/components/features/scores/advanced-analytics';
import {
  useScoreCorrelation,
  useCohenKappa,
  type CorrelationResult,
  type CohenKappaResult,
} from '@/api/score-analytics';
import {
  ScatterChart,
  Scatter,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';

function CorrelationAnalysis() {
  const [score1, setScore1] = useState('');
  const [score2, setScore2] = useState('');

  const { data: correlation, isLoading } = useScoreCorrelation({
    score1,
    score2,
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Correlation Analysis</CardTitle>
        <CardDescription>Analyze relationships between two different scores</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="score1">First Score</Label>
              <Input
                id="score1"
                placeholder="e.g., relevance"
                value={score1}
                onChange={(e) => setScore1(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="score2">Second Score</Label>
              <Input
                id="score2"
                placeholder="e.g., accuracy"
                value={score2}
                onChange={(e) => setScore2(e.target.value)}
              />
            </div>
          </div>

          {isLoading && (
            <div className="h-64 flex items-center justify-center text-muted-foreground">
              Loading correlation analysis...
            </div>
          )}

          {correlation && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Pearson r</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {correlation.pearson.toFixed(3)}
                    </div>
                    <p className="text-xs text-muted-foreground">Linear correlation</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Spearman ρ</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {correlation.spearman.toFixed(3)}
                    </div>
                    <p className="text-xs text-muted-foreground">Rank correlation</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Sample Size</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {correlation.sampleSize}
                    </div>
                    <p className="text-xs text-muted-foreground">data points</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Significance</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Badge variant={correlation.isSignificant ? 'default' : 'secondary'}>
                      {correlation.isSignificant ? 'Significant' : 'Not Significant'}
                    </Badge>
                    <p className="text-xs text-muted-foreground mt-1">
                      p = {correlation.pValue.toFixed(3)}
                    </p>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Interpretation</CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-sm text-muted-foreground">
                    {Math.abs(correlation.pearson) < 0.3 && 'Weak correlation between the two scores.'}
                    {Math.abs(correlation.pearson) >= 0.3 && Math.abs(correlation.pearson) < 0.7 &&
                      'Moderate correlation between the two scores.'}
                    {Math.abs(correlation.pearson) >= 0.7 &&
                      'Strong correlation between the two scores.'}
                    {correlation.pearson > 0 && ' The scores tend to increase together.'}
                    {correlation.pearson < 0 && ' As one score increases, the other tends to decrease.'}
                  </p>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function InterAnnotatorAgreement() {
  const [scoreName, setScoreName] = useState('');
  const [annotator1, setAnnotator1] = useState('');
  const [annotator2, setAnnotator2] = useState('');

  const { data: kappa, isLoading } = useCohenKappa({
    scoreName,
    annotator1,
    annotator2,
  });

  const getInterpretationColor = (kappaValue: number) => {
    if (kappaValue < 0) return 'destructive';
    if (kappaValue < 0.20) return 'secondary';
    if (kappaValue < 0.40) return 'outline';
    if (kappaValue < 0.60) return 'default';
    if (kappaValue < 0.80) return 'default';
    return 'default';
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>Inter-Annotator Agreement</CardTitle>
        <CardDescription>Measure agreement between two annotators using Cohen's Kappa</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label htmlFor="scoreName">Score Name</Label>
              <Input
                id="scoreName"
                placeholder="e.g., sentiment"
                value={scoreName}
                onChange={(e) => setScoreName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="annotator1">Annotator 1 ID</Label>
              <Input
                id="annotator1"
                placeholder="UUID"
                value={annotator1}
                onChange={(e) => setAnnotator1(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="annotator2">Annotator 2 ID</Label>
              <Input
                id="annotator2"
                placeholder="UUID"
                value={annotator2}
                onChange={(e) => setAnnotator2(e.target.value)}
              />
            </div>
          </div>

          {isLoading && (
            <div className="h-64 flex items-center justify-center text-muted-foreground">
              Loading agreement analysis...
            </div>
          )}

          {kappa && (
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Cohen's Kappa</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{kappa.kappa.toFixed(3)}</div>
                    <Badge variant={getInterpretationColor(kappa.kappa)} className="mt-2">
                      {kappa.interpretation}
                    </Badge>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Observed Agreement</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {(kappa.agreement * 100).toFixed(1)}%
                    </div>
                    <p className="text-xs text-muted-foreground">Actual agreement</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm font-medium">Expected Agreement</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">
                      {(kappa.chanceAgreement * 100).toFixed(1)}%
                    </div>
                    <p className="text-xs text-muted-foreground">By chance</p>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Confusion Matrix</CardTitle>
                  <CardDescription>Agreement patterns between annotators</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr>
                          <th className="text-left p-2 border-b">Annotator 1 \ Annotator 2</th>
                          {Object.keys(kappa.confusionMatrix).map((category) => (
                            <th key={category} className="text-center p-2 border-b">
                              {category}
                            </th>
                          ))}
                        </tr>
                      </thead>
                      <tbody>
                        {Object.entries(kappa.confusionMatrix).map(([cat1, row]) => (
                          <tr key={cat1}>
                            <td className="font-medium p-2 border-b">{cat1}</td>
                            {Object.entries(row).map(([cat2, count]) => (
                              <td
                                key={cat2}
                                className={`text-center p-2 border-b ${
                                  cat1 === cat2 ? 'bg-green-50 font-semibold' : ''
                                }`}
                              >
                                {count}
                              </td>
                            ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function AdvancedScoreAnalyticsPage() {
  const [selectedScore, setSelectedScore] = useState('');
  const [dimension, setDimension] = useState('model');
  const [interval, setInterval] = useState<'1h' | '6h' | '12h' | '1d' | '1w'>('1d');

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
          <h1 className="text-3xl font-bold">Advanced Score Analytics</h1>
          <p className="text-muted-foreground">
            Deep dive into score distributions, correlations, and statistical analysis
          </p>
        </div>
      </div>

      {/* Global Score Selector */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Analysis Configuration</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="space-y-2">
              <Label htmlFor="score">Score Name</Label>
              <Input
                id="score"
                placeholder="Enter score name (e.g., relevance)"
                value={selectedScore}
                onChange={(e) => setSelectedScore(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="dimension">Breakdown Dimension</Label>
              <Select value={dimension} onValueChange={setDimension}>
                <SelectTrigger id="dimension">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="model">Model</SelectItem>
                  <SelectItem value="user_id">User</SelectItem>
                  <SelectItem value="session_id">Session</SelectItem>
                  <SelectItem value="prompt_version">Prompt Version</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="interval">Time Interval</Label>
              <Select value={interval} onValueChange={(v: any) => setInterval(v)}>
                <SelectTrigger id="interval">
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
        </CardContent>
      </Card>

      {/* Analytics Tabs */}
      <Tabs defaultValue="distribution" className="space-y-4">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="distribution" className="flex items-center gap-2">
            <BarChart className="h-4 w-4" />
            Distribution
          </TabsTrigger>
          <TabsTrigger value="trends" className="flex items-center gap-2">
            <TrendingUp className="h-4 w-4" />
            Trends
          </TabsTrigger>
          <TabsTrigger value="breakdown" className="flex items-center gap-2">
            <LineChartIcon className="h-4 w-4" />
            Breakdown
          </TabsTrigger>
          <TabsTrigger value="correlation" className="flex items-center gap-2">
            <GitCompare className="h-4 w-4" />
            Correlation
          </TabsTrigger>
          <TabsTrigger value="agreement" className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            Agreement
          </TabsTrigger>
          <TabsTrigger value="f1" className="flex items-center gap-2">
            <Target className="h-4 w-4" />
            F1 Score
          </TabsTrigger>
        </TabsList>

        <TabsContent value="distribution" className="space-y-4">
          {selectedScore ? (
            <DistributionChart scoreName={selectedScore} />
          ) : (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">
                  Enter a score name above to view distribution analytics
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="trends" className="space-y-4">
          {selectedScore ? (
            <TrendChart scoreName={selectedScore} interval={interval} />
          ) : (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">
                  Enter a score name above to view trend analytics
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="breakdown" className="space-y-4">
          {selectedScore ? (
            <BreakdownChart scoreName={selectedScore} dimension={dimension} />
          ) : (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">
                  Enter a score name above to view breakdown analytics
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="correlation" className="space-y-4">
          <CorrelationAnalysis />
        </TabsContent>

        <TabsContent value="agreement" className="space-y-4">
          <InterAnnotatorAgreement />
        </TabsContent>

        <TabsContent value="f1" className="space-y-4">
          {selectedScore ? (
            <F1ScoreMetrics scoreName={selectedScore} threshold={0.5} />
          ) : (
            <Card>
              <CardContent className="py-12 text-center">
                <p className="text-muted-foreground">
                  Enter a score name above to view F1 score metrics
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

export default AdvancedScoreAnalyticsPage;
