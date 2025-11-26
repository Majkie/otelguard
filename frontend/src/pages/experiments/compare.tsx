import { useState } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ArrowLeft, GitCompare, BarChart3, Table as TableIcon } from 'lucide-react';
import {
  useExperiments,
  useExperimentRuns,
  useExperimentComparison,
  useStatisticalComparison,
} from '@/api/experiments';
import {
  ExperimentComparisonSummary,
  MetricsComparisonChart,
  StatisticalSignificanceTable,
} from '@/components/features/experiments/experiment-comparison';
import { Badge } from '@/components/ui/badge';
import { useProjectContext } from '@/contexts/project-context';

function ExperimentComparePage() {
  const [searchParams] = useSearchParams();
  const datasetId = searchParams.get('datasetId');
  const { currentProject } = useProjectContext();

  const [selectedExperimentId, setSelectedExperimentId] = useState<string>('');
  const [selectedRunIds, setSelectedRunIds] = useState<string[]>([]);

  // Fetch experiments
  const { data: experimentsData } = useExperiments({
    projectId: currentProject?.id || '',
  });

  // Fetch runs for selected experiment
  const { data: runsData } = useExperimentRuns(selectedExperimentId);

  // Fetch comparison data
  const { data: comparison, isLoading: comparisonLoading } =
    useExperimentComparison(selectedRunIds);
  const { data: statisticalComparison, isLoading: statisticalLoading } =
    useStatisticalComparison(selectedRunIds);

  const handleExperimentSelect = (experimentId: string) => {
    setSelectedExperimentId(experimentId);
    setSelectedRunIds([]); // Reset run selection when experiment changes
  };

  const handleRunToggle = (runId: string) => {
    setSelectedRunIds((prev) =>
      prev.includes(runId)
        ? prev.filter((id) => id !== runId)
        : [...prev, runId]
    );
  };

  const isComparisonReady = selectedRunIds.length >= 2;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/experiments">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Experiments
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">Compare Experiments</h1>
          <p className="text-muted-foreground">
            Compare multiple experiment runs with statistical analysis
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Selection Sidebar */}
        <div className="lg:col-span-1 space-y-4">
          {/* Experiment Selector */}
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Select Experiment</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                {experimentsData?.data.map((experiment) => (
                  <div
                    key={experiment.id}
                    className={`p-3 rounded-lg border cursor-pointer hover:bg-muted transition-colors ${
                      selectedExperimentId === experiment.id
                        ? 'border-primary bg-muted'
                        : 'border-border'
                    }`}
                    onClick={() => handleExperimentSelect(experiment.id)}
                  >
                    <p className="font-medium text-sm">{experiment.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {experiment.config.model}
                    </p>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Run Selector */}
          {selectedExperimentId && (
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Select Runs to Compare</CardTitle>
                <CardDescription className="text-xs">
                  Choose at least 2 runs
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {runsData?.data.map((run) => (
                    <div
                      key={run.id}
                      className="flex items-center space-x-2 p-2 rounded hover:bg-muted"
                    >
                      <Checkbox
                        id={run.id}
                        checked={selectedRunIds.includes(run.id)}
                        onCheckedChange={() => handleRunToggle(run.id)}
                      />
                      <label
                        htmlFor={run.id}
                        className="flex-1 text-sm font-medium cursor-pointer"
                      >
                        <div className="flex items-center justify-between">
                          <span>Run #{run.runNumber}</span>
                          <Badge
                            variant={
                              run.status === 'completed' ? 'outline' : 'secondary'
                            }
                          >
                            {run.status}
                          </Badge>
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {run.completedItems}/{run.totalItems} items •{' '}
                          {run.avgLatencyMs.toFixed(0)}ms
                        </div>
                      </label>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          )}

          {selectedRunIds.length > 0 && (
            <div className="p-3 bg-muted rounded-lg">
              <p className="text-sm font-medium">
                {selectedRunIds.length} runs selected
              </p>
              {isComparisonReady ? (
                <p className="text-xs text-muted-foreground">
                  Ready to compare
                </p>
              ) : (
                <p className="text-xs text-yellow-600">
                  Select at least 2 runs
                </p>
              )}
            </div>
          )}
        </div>

        {/* Comparison Results */}
        <div className="lg:col-span-3">
          {!isComparisonReady ? (
            <Card>
              <CardContent className="py-12 text-center">
                <GitCompare className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-muted-foreground">
                  Select an experiment and at least 2 runs to begin comparison
                </p>
              </CardContent>
            </Card>
          ) : comparisonLoading || statisticalLoading ? (
            <Card>
              <CardContent className="py-12 text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto"></div>
                <p className="text-muted-foreground mt-4">
                  Analyzing experiment results...
                </p>
              </CardContent>
            </Card>
          ) : (
            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="overview" className="flex items-center gap-2">
                  <TableIcon className="h-4 w-4" />
                  Overview
                </TabsTrigger>
                <TabsTrigger value="metrics" className="flex items-center gap-2">
                  <BarChart3 className="h-4 w-4" />
                  Metrics
                </TabsTrigger>
                <TabsTrigger value="statistical" className="flex items-center gap-2">
                  <GitCompare className="h-4 w-4" />
                  Statistical
                </TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4">
                {comparison && (
                  <ExperimentComparisonSummary comparison={comparison} />
                )}
              </TabsContent>

              <TabsContent value="metrics" className="space-y-4">
                {comparison && (
                  <>
                    <MetricsComparisonChart
                      comparison={comparison}
                      metricName="latency"
                      metricLabel="Latency (milliseconds)"
                      format={(v) => `${v.toFixed(0)}ms`}
                    />
                    <MetricsComparisonChart
                      comparison={comparison}
                      metricName="cost"
                      metricLabel="Cost per Item"
                      format={(v) => `$${v.toFixed(4)}`}
                    />
                  </>
                )}
              </TabsContent>

              <TabsContent value="statistical" className="space-y-4">
                {statisticalComparison && (
                  <>
                    <Card>
                      <CardHeader>
                        <CardTitle>Statistical Analysis</CardTitle>
                        <CardDescription>
                          Pairwise comparisons using Student's t-test with effect size
                          (Cohen's d) calculations
                        </CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2 text-sm">
                          <div className="flex items-center gap-2">
                            <Badge variant="default" className="bg-green-600">
                              ** p &lt; 0.01
                            </Badge>
                            <span className="text-muted-foreground">
                              Highly significant (99% confidence)
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge variant="default" className="bg-green-500">
                              * p &lt; 0.05
                            </Badge>
                            <span className="text-muted-foreground">
                              Significant (95% confidence)
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge variant="secondary">Not Significant</Badge>
                            <span className="text-muted-foreground">
                              Difference could be due to chance
                            </span>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                    <StatisticalSignificanceTable
                      comparison={statisticalComparison}
                    />
                  </>
                )}
              </TabsContent>
            </Tabs>
          )}
        </div>
      </div>
    </div>
  );
}

export default ExperimentComparePage;
