import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { useTraces, type ListTracesParams } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';
import { exportTracesToJson, exportTracesToCsv } from '@/lib/export';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Filter,
  X,
  Download,
  Calendar,
} from 'lucide-react';

type SortField = 'start_time' | 'latency_ms' | 'cost' | 'total_tokens' | 'name' | 'model';

export function TracesPage() {
  const [params, setParams] = useState<ListTracesParams>({
    limit: 20,
    offset: 0,
    sortBy: 'start_time',
    sortOrder: 'DESC',
  });

  const [showFilters, setShowFilters] = useState(false);
  const [nameFilter, setNameFilter] = useState('');
  const [modelFilter, setModelFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState<'' | 'success' | 'error' | 'pending'>('');
  const [startTimeFilter, setStartTimeFilter] = useState('');
  const [endTimeFilter, setEndTimeFilter] = useState('');
  const [showExportMenu, setShowExportMenu] = useState(false);

  const { data, isLoading, error } = useTraces(params);

  const handleSort = useCallback((field: SortField) => {
    setParams((prev) => ({
      ...prev,
      offset: 0,
      sortBy: field,
      sortOrder:
        prev.sortBy === field && prev.sortOrder === 'DESC' ? 'ASC' : 'DESC',
    }));
  }, []);

  const handleSearch = useCallback(() => {
    setParams((prev) => ({
      ...prev,
      offset: 0,
      name: nameFilter || undefined,
      model: modelFilter || undefined,
      status: statusFilter || undefined,
      startTime: startTimeFilter || undefined,
      endTime: endTimeFilter || undefined,
    }));
  }, [nameFilter, modelFilter, statusFilter, startTimeFilter, endTimeFilter]);

  const handleClearFilters = useCallback(() => {
    setNameFilter('');
    setModelFilter('');
    setStatusFilter('');
    setStartTimeFilter('');
    setEndTimeFilter('');
    setParams((prev) => ({
      ...prev,
      offset: 0,
      name: undefined,
      model: undefined,
      status: undefined,
      startTime: undefined,
      endTime: undefined,
    }));
  }, []);

  const handleExportJson = useCallback(() => {
    if (data?.data) {
      exportTracesToJson(data.data, `traces-export-${Date.now()}`);
    }
    setShowExportMenu(false);
  }, [data]);

  const handleExportCsv = useCallback(() => {
    if (data?.data) {
      exportTracesToCsv(data.data, `traces-export-${Date.now()}`);
    }
    setShowExportMenu(false);
  }, [data]);

  const handleNextPage = () => {
    setParams((prev) => ({
      ...prev,
      offset: (prev.offset || 0) + (prev.limit || 20),
    }));
  };

  const handlePrevPage = () => {
    setParams((prev) => ({
      ...prev,
      offset: Math.max(0, (prev.offset || 0) - (prev.limit || 20)),
    }));
  };

  const getSortIcon = (field: SortField) => {
    if (params.sortBy !== field) {
      return <ArrowUpDown className="ml-1 h-4 w-4 text-muted-foreground" />;
    }
    return params.sortOrder === 'ASC' ? (
      <ArrowUp className="ml-1 h-4 w-4" />
    ) : (
      <ArrowDown className="ml-1 h-4 w-4" />
    );
  };

  const hasActiveFilters = params.name || params.model || params.status || params.startTime || params.endTime;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Traces</h1>
        <p className="text-muted-foreground">
          View and analyze your LLM application traces
        </p>
      </div>

      {/* Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col gap-4">
            <div className="flex gap-4 items-center">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search by name..."
                  value={nameFilter}
                  onChange={(e) => setNameFilter(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
                  className="pl-9"
                />
              </div>
              <Button onClick={handleSearch}>Search</Button>
              <Button
                variant="outline"
                onClick={() => setShowFilters(!showFilters)}
              >
                <Filter className="h-4 w-4 mr-2" />
                Filters
                {hasActiveFilters && (
                  <span className="ml-2 rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">
                    Active
                  </span>
                )}
              </Button>
              {hasActiveFilters && (
                <Button variant="ghost" size="sm" onClick={handleClearFilters}>
                  <X className="h-4 w-4 mr-1" />
                  Clear
                </Button>
              )}
              {/* Export dropdown */}
              <div className="relative">
                <Button
                  variant="outline"
                  onClick={() => setShowExportMenu(!showExportMenu)}
                  disabled={!data?.data?.length}
                >
                  <Download className="h-4 w-4 mr-2" />
                  Export
                </Button>
                {showExportMenu && (
                  <div className="absolute right-0 mt-2 w-40 rounded-md shadow-lg bg-popover border z-10">
                    <div className="py-1">
                      <button
                        onClick={handleExportJson}
                        className="block w-full text-left px-4 py-2 text-sm hover:bg-muted"
                      >
                        Export as JSON
                      </button>
                      <button
                        onClick={handleExportCsv}
                        className="block w-full text-left px-4 py-2 text-sm hover:bg-muted"
                      >
                        Export as CSV
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {showFilters && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t">
                <div>
                  <label className="text-sm font-medium mb-1 block">Model</label>
                  <Input
                    placeholder="Filter by model..."
                    value={modelFilter}
                    onChange={(e) => setModelFilter(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-sm font-medium mb-1 block">Status</label>
                  <select
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                    value={statusFilter}
                    onChange={(e) =>
                      setStatusFilter(e.target.value as '' | 'success' | 'error' | 'pending')
                    }
                  >
                    <option value="">All statuses</option>
                    <option value="success">Success</option>
                    <option value="error">Error</option>
                    <option value="pending">Pending</option>
                  </select>
                </div>
                <div>
                  <label className="text-sm font-medium mb-1 block flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    Start Time
                  </label>
                  <Input
                    type="datetime-local"
                    value={startTimeFilter}
                    onChange={(e) => setStartTimeFilter(e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-sm font-medium mb-1 block flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    End Time
                  </label>
                  <Input
                    type="datetime-local"
                    value={endTimeFilter}
                    onChange={(e) => setEndTimeFilter(e.target.value)}
                  />
                </div>
                <div className="md:col-span-4 flex justify-end">
                  <Button onClick={handleSearch}>
                    Apply Filters
                  </Button>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Traces table */}
      <Card>
        <CardHeader>
          <CardTitle>
            {data?.total !== undefined ? `Traces (${data.total})` : 'Traces'}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <p className="text-destructive">Error loading traces</p>
          ) : isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          ) : !data?.data?.length ? (
            <div className="text-center py-8 text-muted-foreground">
              No traces found. Start sending traces from your application!
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('name')}
                    >
                      <div className="flex items-center">
                        Name
                        {getSortIcon('name')}
                      </div>
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('model')}
                    >
                      <div className="flex items-center">
                        Model
                        {getSortIcon('model')}
                      </div>
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('latency_ms')}
                    >
                      <div className="flex items-center">
                        Latency
                        {getSortIcon('latency_ms')}
                      </div>
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('total_tokens')}
                    >
                      <div className="flex items-center">
                        Tokens
                        {getSortIcon('total_tokens')}
                      </div>
                    </TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('cost')}
                    >
                      <div className="flex items-center">
                        Cost
                        {getSortIcon('cost')}
                      </div>
                    </TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead
                      className="cursor-pointer select-none"
                      onClick={() => handleSort('start_time')}
                    >
                      <div className="flex items-center">
                        Time
                        {getSortIcon('start_time')}
                      </div>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((trace) => (
                    <TableRow key={trace.id}>
                      <TableCell>
                        <Link
                          to={`/traces/${trace.id}`}
                          className="font-medium hover:underline"
                        >
                          {trace.name}
                        </Link>
                        {trace.tags?.length > 0 && (
                          <div className="flex gap-1 mt-1">
                            {trace.tags.slice(0, 3).map((tag) => (
                              <span
                                key={tag}
                                className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs font-medium"
                              >
                                {tag}
                              </span>
                            ))}
                            {trace.tags.length > 3 && (
                              <span className="text-xs text-muted-foreground">
                                +{trace.tags.length - 3}
                              </span>
                            )}
                          </div>
                        )}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {trace.model || '-'}
                      </TableCell>
                      <TableCell>{formatLatency(trace.latencyMs)}</TableCell>
                      <TableCell>{formatTokens(trace.totalTokens)}</TableCell>
                      <TableCell>{formatCost(trace.cost)}</TableCell>
                      <TableCell>
                        <span
                          className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
                            trace.status === 'success'
                              ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                              : trace.status === 'error'
                              ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                              : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
                          }`}
                        >
                          {trace.status}
                        </span>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDate(trace.startTime)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              <div className="flex items-center justify-between pt-4">
                <p className="text-sm text-muted-foreground">
                  Showing {(params.offset || 0) + 1} to{' '}
                  {Math.min(
                    (params.offset || 0) + (params.limit || 20),
                    data.total
                  )}{' '}
                  of {data.total} traces
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handlePrevPage}
                    disabled={(params.offset || 0) === 0}
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleNextPage}
                    disabled={
                      (params.offset || 0) + (params.limit || 20) >= data.total
                    }
                  >
                    Next
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
