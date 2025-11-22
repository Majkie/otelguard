import { useState } from 'react';
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
import { Search, ChevronLeft, ChevronRight } from 'lucide-react';

export function TracesPage() {
  const [search, setSearch] = useState('');
  const [params, setParams] = useState<ListTracesParams>({
    limit: 20,
    offset: 0,
  });

  const { data, isLoading, error } = useTraces(params);

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
          <div className="flex gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search traces..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Traces table */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Traces</CardTitle>
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
                    <TableHead>Name</TableHead>
                    <TableHead>Model</TableHead>
                    <TableHead>Latency</TableHead>
                    <TableHead>Tokens</TableHead>
                    <TableHead>Cost</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Time</TableHead>
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
                              ? 'bg-green-100 text-green-700'
                              : 'bg-red-100 text-red-700'
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
