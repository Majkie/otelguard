import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useSessions, type ListSessionsParams } from '@/api/sessions';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';
import {
  ChevronLeft,
  ChevronRight,
  Users,
  Clock,
  Hash,
  Coins,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { cn } from '@/lib/utils';

export function SessionsPage() {
  const [params, setParams] = useState<ListSessionsParams>({
    limit: 20,
    offset: 0,
  });

  const { data, isLoading, error } = useSessions(params);

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
        <h1 className="text-3xl font-bold flex items-center gap-2">
          <Users className="h-8 w-8" />
          Sessions
        </h1>
        <p className="text-muted-foreground">
          View user sessions and their aggregated metrics
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {data?.total !== undefined ? `Sessions (${data.total})` : 'Sessions'}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <p className="text-destructive">Error loading sessions</p>
          ) : isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          ) : !data?.data?.length ? (
            <div className="text-center py-8 text-muted-foreground">
              No sessions found. Sessions are automatically created when traces include a sessionId.
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Session ID</TableHead>
                    <TableHead>User</TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Hash className="h-4 w-4" />
                        Traces
                      </div>
                    </TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Clock className="h-4 w-4" />
                        Total Latency
                      </div>
                    </TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Hash className="h-4 w-4" />
                        Tokens
                      </div>
                    </TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Coins className="h-4 w-4" />
                        Cost
                      </div>
                    </TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Models</TableHead>
                    <TableHead>Time</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((session) => (
                    <TableRow key={session.sessionId}>
                      <TableCell>
                        <Link
                          to={`/sessions/${session.sessionId}`}
                          className="font-medium font-mono text-sm hover:underline"
                        >
                          {session.sessionId.slice(0, 12)}...
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground font-mono text-sm">
                        {session.userId ? session.userId.slice(0, 12) : '-'}
                      </TableCell>
                      <TableCell>{session.traceCount}</TableCell>
                      <TableCell>{formatLatency(session.totalLatencyMs)}</TableCell>
                      <TableCell>{formatTokens(session.totalTokens)}</TableCell>
                      <TableCell>{formatCost(session.totalCost)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
                            <CheckCircle className="h-3 w-3" />
                            {session.successCount}
                          </span>
                          {session.errorCount > 0 && (
                            <span className="flex items-center gap-1 text-red-600 dark:text-red-400">
                              <XCircle className="h-3 w-3" />
                              {session.errorCount}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1 flex-wrap max-w-[150px]">
                          {session.models?.slice(0, 2).map((model) => (
                            <span
                              key={model}
                              className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs"
                            >
                              {model}
                            </span>
                          ))}
                          {session.models && session.models.length > 2 && (
                            <span className="text-xs text-muted-foreground">
                              +{session.models.length - 2}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        <div>{formatDate(session.firstTraceTime)}</div>
                        <div className="text-xs">to {formatDate(session.lastTraceTime)}</div>
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
                  of {data.total} sessions
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
