import { useState, useCallback } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import {
  useEvaluationResults,
  useEvaluators,
  useEvaluationCosts,
  type EvaluationResult,
} from '@/api/evaluators';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { formatDate } from '@/lib/utils';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Filter,
  X,
  ArrowLeft,
  CheckCircle,
  XCircle,
  DollarSign,
  MessageSquare,
} from 'lucide-react';

type ResultsPageParams = {
  limit?: number;
  offset?: number;
  evaluatorId?: string;
  traceId?: string;
  status?: string;
  startTime?: string;
  endTime?: string;
};

const ITEMS_PER_PAGE = 50;

function EvaluationResultsPage() {
  const [searchParams] = useSearchParams();
  const initialEvaluatorId = searchParams.get('evaluatorId') || undefined;

  const [params, setParams] = useState<ResultsPageParams>({
    limit: ITEMS_PER_PAGE,
    offset: 0,
    evaluatorId: initialEvaluatorId,
  });

  const [searchQuery, setSearchQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [selectedResult, setSelectedResult] = useState<EvaluationResult | null>(null);

  const { data: resultsResponse, isLoading, error } = useEvaluationResults(params);
  const { data: evaluatorsResponse } = useEvaluators({ limit: 100 });
  const { data: costsResponse } = useEvaluationCosts();

  const results = resultsResponse?.results || [];
  const evaluators = evaluatorsResponse?.data || [];
  const costs = costsResponse?.costs || [];
  const total = resultsResponse?.pagination.total || 0;
  const currentPage = Math.floor((params.offset || 0) / ITEMS_PER_PAGE) + 1;
  const totalPages = Math.ceil(total / ITEMS_PER_PAGE);

  // Calculate total cost
  const totalCost = costs.reduce((sum, c) => sum + c.totalCost, 0);
  const totalTokens = costs.reduce((sum, c) => sum + c.totalTokens, 0);

  const handleSearch = useCallback(() => {
    if (searchQuery.trim()) {
      setParams(prev => ({
        ...prev,
        traceId: searchQuery,
        offset: 0,
      }));
    } else {
      setParams(prev => ({
        ...prev,
        traceId: undefined,
        offset: 0,
      }));
    }
  }, [searchQuery]);

  const handleFilterChange = useCallback((key: keyof ResultsPageParams, value: any) => {
    setParams(prev => ({
      ...prev,
      [key]: value || undefined,
      offset: 0,
    }));
  }, []);

  const clearFilters = useCallback(() => {
    setParams({
      limit: ITEMS_PER_PAGE,
      offset: 0,
    });
    setSearchQuery('');
  }, []);

  const goToPage = useCallback((page: number) => {
    setParams(prev => ({
      ...prev,
      offset: (page - 1) * ITEMS_PER_PAGE,
    }));
  }, []);

  const getEvaluatorName = (evaluatorId: string) => {
    const evaluator = evaluators.find(e => e.id === evaluatorId);
    return evaluator?.name || evaluatorId.slice(0, 8) + '...';
  };

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h3 className="text-lg font-semibold text-destructive">Error loading results</h3>
          <p className="text-muted-foreground">Please try again later</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link to="/evaluators">
            <Button variant="ghost" size="sm">
              <ArrowLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold">Evaluation Results</h1>
            <p className="text-muted-foreground">
              View all LLM-as-a-Judge evaluation results
            </p>
          </div>
        </div>
      </div>

      {/* Stats Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Results</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{total.toLocaleString()}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">${totalCost.toFixed(4)}</div>
            <p className="text-xs text-muted-foreground">
              {totalTokens.toLocaleString()} tokens
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Evaluators</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{costs.length}</div>
          </CardContent>
        </Card>
      </div>

      {/* Search and Filters */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">Search & Filter</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                placeholder="Search by trace ID..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
            <Button onClick={handleSearch}>
              <Search className="h-4 w-4 mr-2" />
              Search
            </Button>
            {(Object.values(params).some(v => v !== undefined && v !== 0 && v !== ITEMS_PER_PAGE) || searchQuery) && (
              <Button variant="outline" onClick={clearFilters}>
                <X className="h-4 w-4 mr-2" />
                Clear
              </Button>
            )}
          </div>

          {showFilters && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 pt-4 border-t">
              <div>
                <label className="text-sm font-medium mb-1 block">Evaluator</label>
                <Select
                  value={params.evaluatorId || ''}
                  onValueChange={(value) => handleFilterChange('evaluatorId', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All evaluators" />
                  </SelectTrigger>
                  <SelectContent>
                    {evaluators.map((evaluator) => (
                      <SelectItem key={evaluator.id} value={evaluator.id}>
                        {evaluator.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">Status</label>
                <Select
                  value={params.status || ''}
                  onValueChange={(value) => handleFilterChange('status', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All statuses" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="success">Success</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">Start Time</label>
                <Input
                  type="datetime-local"
                  value={params.startTime || ''}
                  onChange={(e) => handleFilterChange('startTime', e.target.value || undefined)}
                />
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">End Time</label>
                <Input
                  type="datetime-local"
                  value={params.endTime || ''}
                  onChange={(e) => handleFilterChange('endTime', e.target.value || undefined)}
                />
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Results Table */}
      <Card>
        <CardHeader>
          <CardTitle>
            Results {total > 0 && `(${total})`}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
                <p className="text-muted-foreground">Loading results...</p>
              </div>
            </div>
          ) : results.length === 0 ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <h3 className="text-lg font-semibold">No results found</h3>
                <p className="text-muted-foreground">
                  {Object.values(params).some(v => v) ? 'Try adjusting your filters' : 'Run evaluations to see results here'}
                </p>
              </div>
            </div>
          ) : (
            <>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Evaluator</TableHead>
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
                      <TableRow
                        key={result.id}
                        className="cursor-pointer hover:bg-muted/50"
                        onClick={() => setSelectedResult(result)}
                      >
                        <TableCell className="font-medium">
                          <Link
                            to={`/evaluators/${result.evaluatorId}`}
                            className="text-primary hover:underline"
                            onClick={(e) => e.stopPropagation()}
                          >
                            {getEvaluatorName(result.evaluatorId)}
                          </Link>
                        </TableCell>
                        <TableCell className="font-mono text-sm">
                          <Link
                            to={`/traces/${result.traceId}`}
                            className="text-primary hover:underline"
                            onClick={(e) => e.stopPropagation()}
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

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    Showing {((currentPage - 1) * ITEMS_PER_PAGE) + 1} to{' '}
                    {Math.min(currentPage * ITEMS_PER_PAGE, total)} of {total} results
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage - 1)}
                      disabled={currentPage === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                      Previous
                    </Button>
                    <span className="text-sm">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage + 1)}
                      disabled={currentPage === totalPages}
                    >
                      Next
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* Result Detail Sheet */}
      <Sheet open={!!selectedResult} onOpenChange={() => setSelectedResult(null)}>
        <SheetContent className="w-[500px] sm:max-w-[540px]">
          <SheetHeader>
            <SheetTitle>Evaluation Result</SheetTitle>
            <SheetDescription>
              Details for evaluation result
            </SheetDescription>
          </SheetHeader>
          {selectedResult && (
            <div className="mt-6 space-y-6">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-muted-foreground">Score</label>
                  <p className="text-2xl font-bold font-mono">{selectedResult.score.toFixed(3)}</p>
                </div>
                <div>
                  <label className="text-sm text-muted-foreground">Status</label>
                  <div className="mt-1">
                    {selectedResult.status === 'success' ? (
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
                  </div>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-muted-foreground">Latency</label>
                  <p className="font-medium">{selectedResult.latencyMs}ms</p>
                </div>
                <div>
                  <label className="text-sm text-muted-foreground">Cost</label>
                  <p className="font-mono">${selectedResult.cost.toFixed(6)}</p>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm text-muted-foreground">Prompt Tokens</label>
                  <p className="font-mono">{selectedResult.promptTokens.toLocaleString()}</p>
                </div>
                <div>
                  <label className="text-sm text-muted-foreground">Completion Tokens</label>
                  <p className="font-mono">{selectedResult.completionTokens.toLocaleString()}</p>
                </div>
              </div>

              {selectedResult.reasoning && (
                <div>
                  <label className="text-sm text-muted-foreground flex items-center gap-1">
                    <MessageSquare className="h-4 w-4" />
                    Reasoning
                  </label>
                  <p className="mt-2 text-sm p-3 bg-muted rounded-md whitespace-pre-wrap">
                    {selectedResult.reasoning}
                  </p>
                </div>
              )}

              {selectedResult.errorMessage && (
                <div>
                  <label className="text-sm text-destructive">Error Message</label>
                  <p className="mt-2 text-sm p-3 bg-destructive/10 text-destructive rounded-md">
                    {selectedResult.errorMessage}
                  </p>
                </div>
              )}

              <div className="flex gap-2">
                <Link to={`/traces/${selectedResult.traceId}`} className="flex-1">
                  <Button variant="outline" className="w-full">
                    View Trace
                  </Button>
                </Link>
                <Link to={`/evaluators/${selectedResult.evaluatorId}`} className="flex-1">
                  <Button variant="outline" className="w-full">
                    View Evaluator
                  </Button>
                </Link>
              </div>
            </div>
          )}
        </SheetContent>
      </Sheet>
    </div>
  );
}

export default EvaluationResultsPage;
export { EvaluationResultsPage };
