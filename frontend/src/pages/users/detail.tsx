import { useParams, Link } from 'react-router-dom';
import { useUser } from '@/api/users';
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
import { Badge } from '@/components/ui/badge';
import { Progress } from '@/components/ui/progress';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';
import {
  ArrowLeft,
  User,
  Clock,
  Hash,
  Coins,
  CheckCircle,
  XCircle,
  MessageSquare,
  Calendar,
  Activity,
} from 'lucide-react';

export function UserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isLoading, error } = useUser(id || '', 20, 0, 10, 0);

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="space-y-4">
        <Link to="/users">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Users
          </Button>
        </Link>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            User not found or error loading user data.
          </CardContent>
        </Card>
      </div>
    );
  }

  const { user, traces, sessions } = data;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/users">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
        </Link>
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <User className="h-8 w-8" />
            User Details
          </h1>
          <p className="text-muted-foreground font-mono">{user.userId}</p>
        </div>
      </div>

      {/* User Metrics */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Traces</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{user.traceCount}</div>
            <p className="text-xs text-muted-foreground">
              Across {user.sessionCount} sessions
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Avg Latency</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatLatency(user.avgLatencyMs)}</div>
            <p className="text-xs text-muted-foreground">
              Total: {formatLatency(user.totalLatencyMs)}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatCost(user.totalCost)}</div>
            <p className="text-xs text-muted-foreground">
              {formatTokens(user.totalTokens)} tokens used
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{(user.successRate * 100).toFixed(1)}%</div>
            <div className="flex items-center gap-2 mt-2">
              <span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-sm">
                <CheckCircle className="h-3 w-3" />
                {user.successCount}
              </span>
              {user.errorCount > 0 && (
                <span className="flex items-center gap-1 text-red-600 dark:text-red-400 text-sm">
                  <XCircle className="h-3 w-3" />
                  {user.errorCount}
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Activity Period */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calendar className="h-5 w-5" />
            Activity Period
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-muted-foreground">First Seen</p>
              <p className="font-medium">{formatDate(user.firstSeenTime)}</p>
            </div>
            <div className="flex-1 mx-8">
              <Progress value={100} className="h-2" />
            </div>
            <div className="text-right">
              <p className="text-sm text-muted-foreground">Last Seen</p>
              <p className="font-medium">{formatDate(user.lastSeenTime)}</p>
            </div>
          </div>
          {user.models && user.models.length > 0 && (
            <div className="mt-4">
              <p className="text-sm text-muted-foreground mb-2">Models Used</p>
              <div className="flex gap-2 flex-wrap">
                {user.models.map((model) => (
                  <Badge key={model} variant="secondary">
                    {model}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Sessions */}
      <Card>
        <CardHeader>
          <CardTitle>Sessions ({sessions.total})</CardTitle>
        </CardHeader>
        <CardContent>
          {sessions.data.length === 0 ? (
            <p className="text-muted-foreground text-center py-4">No sessions found</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session ID</TableHead>
                  <TableHead>Traces</TableHead>
                  <TableHead>Latency</TableHead>
                  <TableHead>Tokens</TableHead>
                  <TableHead>Cost</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Time</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.data.map((session) => (
                  <TableRow key={session.sessionId}>
                    <TableCell>
                      <Link
                        to={`/sessions/${session.sessionId}`}
                        className="font-mono text-sm hover:underline"
                      >
                        {session.sessionId.slice(0, 12)}...
                      </Link>
                    </TableCell>
                    <TableCell>{session.traceCount}</TableCell>
                    <TableCell>{formatLatency(session.totalLatencyMs)}</TableCell>
                    <TableCell>{formatTokens(session.totalTokens)}</TableCell>
                    <TableCell>{formatCost(session.totalCost)}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <span className="flex items-center gap-1 text-green-600 dark:text-green-400 text-xs">
                          <CheckCircle className="h-3 w-3" />
                          {session.successCount}
                        </span>
                        {session.errorCount > 0 && (
                          <span className="flex items-center gap-1 text-red-600 dark:text-red-400 text-xs">
                            <XCircle className="h-3 w-3" />
                            {session.errorCount}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(session.lastTraceTime)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Recent Traces */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Traces ({traces.total})</CardTitle>
        </CardHeader>
        <CardContent>
          {traces.data.length === 0 ? (
            <p className="text-muted-foreground text-center py-4">No traces found</p>
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
                        {trace.name.length > 30 ? `${trace.name.slice(0, 30)}...` : trace.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{trace.model || '-'}</Badge>
                    </TableCell>
                    <TableCell>{formatLatency(trace.latencyMs)}</TableCell>
                    <TableCell>{formatTokens(trace.totalTokens)}</TableCell>
                    <TableCell>{formatCost(trace.cost)}</TableCell>
                    <TableCell>
                      {trace.status === 'success' ? (
                        <Badge variant="default" className="bg-green-500">Success</Badge>
                      ) : trace.status === 'error' ? (
                        <Badge variant="destructive">Error</Badge>
                      ) : (
                        <Badge variant="secondary">{trace.status}</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatDate(trace.startTime)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          {traces.total > traces.limit && (
            <div className="text-center mt-4">
              <Link to={`/traces?userId=${encodeURIComponent(user.userId)}`}>
                <Button variant="outline" size="sm">
                  View All Traces
                </Button>
              </Link>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
