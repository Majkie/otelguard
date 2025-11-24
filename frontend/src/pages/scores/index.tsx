import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useScores } from '@/api/scores';
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
import { formatDate } from '@/lib/utils';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Filter,
  X,
  BarChart3,
} from 'lucide-react';

type ScoresPageParams = {
  limit?: number;
  offset?: number;
  traceId?: string;
  spanId?: string;
  name?: string;
  source?: 'api' | 'llm_judge' | 'human' | 'user_feedback';
  dataType?: 'numeric' | 'boolean' | 'categorical';
  startTime?: string;
  endTime?: string;
};

const ITEMS_PER_PAGE = 50;

function ScoresPage() {
  const [params, setParams] = useState<ScoresPageParams>({
    limit: ITEMS_PER_PAGE,
    offset: 0,
  });

  const [searchQuery, setSearchQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);

  const { data: scoresResponse, isLoading, error } = useScores(params);

  const scores = scoresResponse?.scores || [];
  const total = scoresResponse?.pagination.total || 0;
  const currentPage = Math.floor((params.offset || 0) / ITEMS_PER_PAGE) + 1;
  const totalPages = Math.ceil(total / ITEMS_PER_PAGE);

  const handleSearch = useCallback(() => {
    if (searchQuery.trim()) {
      // Search by name or trace ID
      setParams(prev => ({
        ...prev,
        name: searchQuery,
        traceId: undefined,
        offset: 0,
      }));
    } else {
      setParams(prev => ({
        ...prev,
        name: undefined,
        traceId: undefined,
        offset: 0,
      }));
    }
  }, [searchQuery]);

  const handleFilterChange = useCallback((key: keyof ScoresPageParams, value: any) => {
    setParams(prev => ({
      ...prev,
      [key]: value || undefined,
      offset: 0, // Reset to first page when filtering
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

  const formatScoreValue = (score: any) => {
    if (score.dataType === 'boolean') {
      return score.value === 1 ? 'True' : 'False';
    }
    if (score.dataType === 'categorical') {
      return score.stringValue || score.value;
    }
    return score.value.toFixed(3);
  };

  const getSourceBadgeVariant = (source: string) => {
    switch (source) {
      case 'api': return 'default';
      case 'llm_judge': return 'secondary';
      case 'human': return 'outline';
      case 'user_feedback': return 'destructive';
      default: return 'default';
    }
  };

  const getDataTypeBadgeVariant = (dataType: string) => {
    switch (dataType) {
      case 'numeric': return 'default';
      case 'boolean': return 'secondary';
      case 'categorical': return 'outline';
      default: return 'default';
    }
  };

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h3 className="text-lg font-semibold text-destructive">Error loading scores</h3>
          <p className="text-muted-foreground">Please try again later</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Scores</h1>
          <p className="text-muted-foreground">
            View and analyze evaluation scores for your traces
          </p>
        </div>
        <div className="flex gap-2">
          <Link to="/scores/analytics">
            <Button variant="outline">
              <BarChart3 className="h-4 w-4 mr-2" />
              Analytics
            </Button>
          </Link>
        </div>
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
                placeholder="Search by score name or trace ID..."
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
                <label className="text-sm font-medium mb-1 block">Source</label>
                <Select
                  value={params.source || ''}
                  onValueChange={(value) => handleFilterChange('source', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All sources" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="api">API</SelectItem>
                    <SelectItem value="llm_judge">LLM Judge</SelectItem>
                    <SelectItem value="human">Human</SelectItem>
                    <SelectItem value="user_feedback">User Feedback</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">Data Type</label>
                <Select
                  value={params.dataType || ''}
                  onValueChange={(value) => handleFilterChange('dataType', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All types" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="numeric">Numeric</SelectItem>
                    <SelectItem value="boolean">Boolean</SelectItem>
                    <SelectItem value="categorical">Categorical</SelectItem>
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

      {/* Results */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>
              Scores {total > 0 && `(${total})`}
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
                <p className="text-muted-foreground">Loading scores...</p>
              </div>
            </div>
          ) : scores.length === 0 ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <h3 className="text-lg font-semibold">No scores found</h3>
                <p className="text-muted-foreground">
                  {Object.values(params).some(v => v) ? 'Try adjusting your filters' : 'Scores will appear here once evaluations are run'}
                </p>
              </div>
            </div>
          ) : (
            <>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Value</TableHead>
                      <TableHead>Source</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>Trace ID</TableHead>
                      <TableHead>Created</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {scores.map((score) => (
                      <TableRow key={score.id}>
                        <TableCell className="font-medium">
                          <Link
                            to={`/scores/${score.id}`}
                            className="text-primary hover:underline"
                          >
                            {score.name}
                          </Link>
                        </TableCell>
                        <TableCell>
                          <span className="font-mono">
                            {formatScoreValue(score)}
                          </span>
                        </TableCell>
                        <TableCell>
                          <Badge variant={getSourceBadgeVariant(score.source)}>
                            {score.source.replace('_', ' ')}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={getDataTypeBadgeVariant(score.dataType)}>
                            {score.dataType}
                          </Badge>
                        </TableCell>
                        <TableCell className="font-mono text-sm">
                          <Link
                            to={`/traces/${score.traceId}`}
                            className="text-primary hover:underline"
                          >
                            {score.traceId.slice(0, 8)}...
                          </Link>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(score.createdAt)}
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
                    {Math.min(currentPage * ITEMS_PER_PAGE, total)} of {total} scores
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
    </div>
  );
}

export default ScoresPage;
