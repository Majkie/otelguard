import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  useEvaluator,
  useUpdateEvaluator,
  useDeleteEvaluator,
  useRunEvaluation,
  useEvaluationResults,
  useEvaluationStats,
  type UpdateEvaluatorRequest,
} from '@/api/evaluators';
import { useTraces } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { formatDate } from '@/lib/utils';
import { useToast } from '@/hooks/use-toast';
import {
  ArrowLeft,
  Save,
  Trash2,
  Play,
  DollarSign,
  Clock,
  CheckCircle,
  XCircle,
  BarChart3,
  Zap,
} from 'lucide-react';

function EvaluatorDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();

  const [isEditing, setIsEditing] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [runDialogOpen, setRunDialogOpen] = useState(false);
  const [selectedTraceId, setSelectedTraceId] = useState('');
  const [formData, setFormData] = useState<UpdateEvaluatorRequest>({});

  const { data: evaluator, isLoading, error } = useEvaluator(id!);
  const { data: statsData } = useEvaluationStats(id);
  const { data: resultsData } = useEvaluationResults({ evaluatorId: id, limit: 10 });
  const { data: tracesData } = useTraces({ limit: 50 });

  const updateMutation = useUpdateEvaluator();
  const deleteMutation = useDeleteEvaluator();
  const runMutation = useRunEvaluation();

  const results = resultsData?.results || [];
  const traces = tracesData?.data || [];
  const stats = statsData;

  const handleStartEdit = () => {
    if (!evaluator) return;
    setFormData({
      name: evaluator.name,
      description: evaluator.description,
      provider: evaluator.provider,
      model: evaluator.model,
      template: evaluator.template,
      outputType: evaluator.outputType,
      minValue: evaluator.minValue,
      maxValue: evaluator.maxValue,
      categories: evaluator.categories,
      enabled: evaluator.enabled,
    });
    setIsEditing(true);
  };

  const handleSave = async () => {
    if (!id) return;
    try {
      await updateMutation.mutateAsync({ id, data: formData });
      toast({
        title: 'Evaluator updated',
        description: 'Changes have been saved successfully.',
      });
      setIsEditing(false);
    } catch (error) {
      toast({
        title: 'Error updating evaluator',
        description: 'Please try again.',
        variant: 'destructive',
      });
    }
  };

  const handleDelete = async () => {
    if (!id) return;
    try {
      await deleteMutation.mutateAsync(id);
      toast({
        title: 'Evaluator deleted',
        description: 'The evaluator has been deleted.',
      });
      navigate('/evaluators');
    } catch (error) {
      toast({
        title: 'Error deleting evaluator',
        description: 'Please try again.',
        variant: 'destructive',
      });
    }
  };

  const handleRunEvaluation = async () => {
    if (!id || !selectedTraceId) return;
    try {
      const result = await runMutation.mutateAsync({
        evaluatorId: id,
        traceId: selectedTraceId,
      });
      toast({
        title: 'Evaluation completed',
        description: `Score: ${result.score.toFixed(3)}`,
      });
      setRunDialogOpen(false);
      setSelectedTraceId('');
    } catch (error) {
      toast({
        title: 'Evaluation failed',
        description: 'Please try again.',
        variant: 'destructive',
      });
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
          <p className="text-muted-foreground">Loading evaluator...</p>
        </div>
      </div>
    );
  }

  if (error || !evaluator) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h3 className="text-lg font-semibold text-destructive">Evaluator not found</h3>
          <p className="text-muted-foreground mb-4">The evaluator you're looking for doesn't exist.</p>
          <Button onClick={() => navigate('/evaluators')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Evaluators
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="sm" onClick={() => navigate('/evaluators')}>
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{evaluator.name}</h1>
            {evaluator.description && (
              <p className="text-muted-foreground">{evaluator.description}</p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setRunDialogOpen(true)}>
            <Play className="h-4 w-4 mr-2" />
            Run Evaluation
          </Button>
          {isEditing ? (
            <>
              <Button variant="outline" onClick={() => setIsEditing(false)}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={updateMutation.isPending}>
                <Save className="h-4 w-4 mr-2" />
                {updateMutation.isPending ? 'Saving...' : 'Save'}
              </Button>
            </>
          ) : (
            <>
              <Button variant="outline" onClick={handleStartEdit}>
                Edit
              </Button>
              <Button
                variant="destructive"
                onClick={() => setDeleteDialogOpen(true)}
              >
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Stats Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Evaluations</CardTitle>
              <Zap className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.totalEvaluations}</div>
              <p className="text-xs text-muted-foreground">
                {stats.successCount} successful, {stats.errorCount} failed
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Average Score</CardTitle>
              <BarChart3 className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.avgScore.toFixed(3)}</div>
              <p className="text-xs text-muted-foreground">
                Range: {stats.minScore.toFixed(2)} - {stats.maxScore.toFixed(2)}
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
              <DollarSign className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">${stats.totalCost.toFixed(4)}</div>
              <p className="text-xs text-muted-foreground">
                {stats.totalTokens.toLocaleString()} tokens used
              </p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.avgLatencyMs.toFixed(0)}ms</div>
              <p className="text-xs text-muted-foreground">
                Per evaluation
              </p>
            </CardContent>
          </Card>
        </div>
      )}

      <Tabs defaultValue="config" className="w-full">
        <TabsList>
          <TabsTrigger value="config">Configuration</TabsTrigger>
          <TabsTrigger value="results">Recent Results</TabsTrigger>
        </TabsList>

        <TabsContent value="config" className="space-y-6">
          {/* Configuration */}
          <Card>
            <CardHeader>
              <CardTitle>Configuration</CardTitle>
              <CardDescription>
                Evaluator settings and model configuration
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {isEditing ? (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="name">Name</Label>
                      <Input
                        id="name"
                        value={formData.name || ''}
                        onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="provider">Provider</Label>
                      <Select
                        value={formData.provider || ''}
                        onValueChange={(value) => setFormData(prev => ({ ...prev, provider: value }))}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="openai">OpenAI</SelectItem>
                          <SelectItem value="anthropic">Anthropic</SelectItem>
                          <SelectItem value="google">Google</SelectItem>
                          <SelectItem value="ollama">Ollama</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="model">Model</Label>
                      <Input
                        id="model"
                        value={formData.model || ''}
                        onChange={(e) => setFormData(prev => ({ ...prev, model: e.target.value }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="outputType">Output Type</Label>
                      <Select
                        value={formData.outputType || ''}
                        onValueChange={(value) => setFormData(prev => ({ ...prev, outputType: value }))}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="numeric">Numeric</SelectItem>
                          <SelectItem value="boolean">Boolean</SelectItem>
                          <SelectItem value="categorical">Categorical</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="description">Description</Label>
                    <Input
                      id="description"
                      value={formData.description || ''}
                      onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="template">Prompt Template</Label>
                    <Textarea
                      id="template"
                      value={formData.template || ''}
                      onChange={(e) => setFormData(prev => ({ ...prev, template: e.target.value }))}
                      className="min-h-[200px] font-mono text-sm"
                    />
                  </div>

                  <div className="flex items-center space-x-2">
                    <Switch
                      id="enabled"
                      checked={formData.enabled}
                      onCheckedChange={(checked) => setFormData(prev => ({ ...prev, enabled: checked }))}
                    />
                    <Label htmlFor="enabled">Enabled</Label>
                  </div>
                </>
              ) : (
                <>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div>
                      <Label className="text-muted-foreground">Provider</Label>
                      <p className="font-medium capitalize">{evaluator.provider}</p>
                    </div>
                    <div>
                      <Label className="text-muted-foreground">Model</Label>
                      <p className="font-mono text-sm">{evaluator.model}</p>
                    </div>
                    <div className="flex flex-col gap-1 items-start">
                      <Label className="text-muted-foreground">Output Type</Label>
                      <Badge variant="outline">{evaluator.outputType}</Badge>
                    </div>
                    <div className="flex flex-col gap-1 items-start">
                      <Label className="text-muted-foreground">Status</Label>
                      <Badge variant={evaluator.enabled ? 'default' : 'secondary'}>
                        {evaluator.enabled ? 'Enabled' : 'Disabled'}
                      </Badge>
                    </div>
                  </div>

                  {evaluator.outputType === 'numeric' && (
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label className="text-muted-foreground">Min Value</Label>
                        <p className="font-mono">{evaluator.minValue}</p>
                      </div>
                      <div>
                        <Label className="text-muted-foreground">Max Value</Label>
                        <p className="font-mono">{evaluator.maxValue}</p>
                      </div>
                    </div>
                  )}

                  <div>
                    <Label className="text-muted-foreground">Prompt Template</Label>
                    <pre className="mt-2 p-4 bg-muted rounded-md text-sm font-mono whitespace-pre-wrap overflow-x-auto">
                      {evaluator.template}
                    </pre>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="results" className="space-y-6">
          {/* Recent Results */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Recent Results</CardTitle>
                  <CardDescription>
                    Last 10 evaluation results for this evaluator
                  </CardDescription>
                </div>
                <Link to={`/evaluators/results?evaluatorId=${id}`}>
                  <Button variant="outline" size="sm">
                    View All
                  </Button>
                </Link>
              </div>
            </CardHeader>
            <CardContent>
              {results.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-muted-foreground">No evaluation results yet.</p>
                  <Button
                    variant="outline"
                    className="mt-4"
                    onClick={() => setRunDialogOpen(true)}
                  >
                    <Play className="h-4 w-4 mr-2" />
                    Run First Evaluation
                  </Button>
                </div>
              ) : (
                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Trace</TableHead>
                        <TableHead>Score</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Latency</TableHead>
                        <TableHead>Cost</TableHead>
                        <TableHead>Created</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {results.map((result) => (
                        <TableRow key={result.id}>
                          <TableCell className="font-mono text-sm">
                            <Link
                              to={`/traces/${result.traceId}`}
                              className="text-primary hover:underline"
                            >
                              {result.traceId.slice(0, 8)}...
                            </Link>
                          </TableCell>
                          <TableCell>
                            <span className="font-mono font-medium">
                              {result.score.toFixed(3)}
                            </span>
                          </TableCell>
                          <TableCell>
                            {result.status === 'success' ? (
                              <Badge variant="default" className="gap-1">
                                <CheckCircle className="h-3 w-3" />
                                Success
                              </Badge>
                            ) : (
                              <Badge variant="destructive" className="gap-1">
                                <XCircle className="h-3 w-3" />
                                Error
                              </Badge>
                            )}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {result.latencyMs}ms
                          </TableCell>
                          <TableCell className="font-mono text-sm">
                            ${result.cost.toFixed(6)}
                          </TableCell>
                          <TableCell className="text-muted-foreground">
                            {formatDate(result.createdAt)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Evaluator</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{evaluator.name}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Run Evaluation Dialog */}
      <Dialog open={runDialogOpen} onOpenChange={setRunDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Run Evaluation</DialogTitle>
            <DialogDescription>
              Select a trace to evaluate with "{evaluator.name}"
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="trace">Select Trace</Label>
              <Select value={selectedTraceId} onValueChange={setSelectedTraceId}>
                <SelectTrigger>
                  <SelectValue placeholder="Choose a trace..." />
                </SelectTrigger>
                <SelectContent>
                  {traces.map((trace) => (
                    <SelectItem key={trace.id} value={trace.id}>
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-xs">{trace.id.slice(0, 8)}...</span>
                        <span className="text-muted-foreground">{trace.name || 'Unnamed trace'}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setRunDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleRunEvaluation}
              disabled={!selectedTraceId || runMutation.isPending}
            >
              <Play className="h-4 w-4 mr-2" />
              {runMutation.isPending ? 'Running...' : 'Run Evaluation'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default EvaluatorDetailPage;
export { EvaluatorDetailPage };
