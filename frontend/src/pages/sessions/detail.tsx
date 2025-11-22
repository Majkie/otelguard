import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Clock, Hash, Coins, CheckCircle, XCircle, Users, Cpu } from 'lucide-react';
import { useSession } from '@/api/sessions';
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
import { cn } from '@/lib/utils';

export function SessionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading, error } = useSession(id || '');

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="space-y-4">
        <Link to="/sessions">
          <Button variant="ghost">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Sessions
          </Button>
        </Link>
        <p className="text-destructive">Error loading session</p>
      </div>
    );
  }

  const { session, traces } = data;
  const successRate = session.traceCount > 0
    ? ((session.successCount / session.traceCount) * 100).toFixed(1)
    : '0';
  const avgLatency = session.traceCount > 0
    ? Math.round(session.totalLatencyMs / session.traceCount)
    : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/sessions">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Users className="h-6 w-6" />
            Session Details
          </h1>
          <p className="text-sm text-muted-foreground font-mono">
            {session.sessionId}
          </p>
        </div>
        <div className="text-right text-sm text-muted-foreground">
          <div>{formatDate(session.firstTraceTime)}</div>
          <div className="text-xs">to {formatDate(session.lastTraceTime)}</div>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Traces</CardTitle>
            <Hash className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{session.traceCount}</div>
            <p className="text-xs text-muted-foreground">
              {session.successCount} success / {session.errorCount} errors
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Latency</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatLatency(session.totalLatencyMs)}
            </div>
            <p className="text-xs text-muted-foreground">
              Avg: {formatLatency(avgLatency)} per trace
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Tokens</CardTitle>
            <Hash className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatTokens(session.totalTokens)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatCost(session.totalCost)}</div>
          </CardContent>
        </Card>
      </div>

      {/* Summary */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <div className="text-3xl font-bold">{successRate}%</div>
              {parseFloat(successRate) >= 90 ? (
                <CheckCircle className="h-6 w-6 text-green-500" />
              ) : (
                <XCircle className="h-6 w-6 text-red-500" />
              )}
            </div>
            <div className="mt-2 h-2 rounded-full bg-muted overflow-hidden">
              <div
                className="h-full bg-green-500 transition-all"
                style={{ width: `${successRate}%` }}
              />
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">User</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="font-mono text-sm">
              {session.userId || 'No user ID'}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Cpu className="h-4 w-4" />
              Models Used
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-1">
              {session.models?.map((model) => (
                <span
                  key={model}
                  className="inline-flex items-center rounded-md bg-muted px-2 py-1 text-xs font-medium"
                >
                  {model}
                </span>
              )) || <span className="text-muted-foreground">-</span>}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Traces */}
      <Card>
        <CardHeader>
          <CardTitle>Session Traces ({traces.total})</CardTitle>
        </CardHeader>
        <CardContent>
          {traces.data.length === 0 ? (
            <p className="text-center py-8 text-muted-foreground">
              No traces in this session
            </p>
          ) : (
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
                {traces.data.map((trace) => (
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
                        className={cn(
                          'inline-flex items-center rounded-full px-2 py-1 text-xs font-medium',
                          trace.status === 'success'
                            ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
                            : trace.status === 'error'
                              ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
                              : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'
                        )}
                      >
                        {trace.status}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(trace.startTime)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
