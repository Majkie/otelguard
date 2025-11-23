import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useUsers } from '@/api/users';
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
  User,
  Clock,
  Hash,
  Coins,
  CheckCircle,
  XCircle,
  MessageSquare,
  Calendar,
} from 'lucide-react';
import { Progress } from '@/components/ui/progress';

type UsersPageParams = {
  limit?: number;
  offset?: number;
  startTime?: string;
  endTime?: string;
};

export function UsersPage() {
  const [params, setParams] = useState<UsersPageParams>({
    limit: 20,
    offset: 0,
  });

  const { data, isLoading, error } = useUsers(params);

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
          <User className="h-8 w-8" />
          Users
        </h1>
        <p className="text-muted-foreground">
          Track user activity and metrics across your LLM application
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>
            {data?.total !== undefined ? `Users (${data.total})` : 'Users'}
          </CardTitle>
        </CardHeader>
        <CardContent>
          {error ? (
            <p className="text-destructive">Error loading users</p>
          ) : isLoading ? (
            <div className="flex justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
          ) : !data?.data?.length ? (
            <div className="text-center py-8 text-muted-foreground">
              No users found. Users are automatically tracked when traces include a userId.
            </div>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>User ID</TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <MessageSquare className="h-4 w-4" />
                        Traces
                      </div>
                    </TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Hash className="h-4 w-4" />
                        Sessions
                      </div>
                    </TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Clock className="h-4 w-4" />
                        Avg Latency
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
                    <TableHead>Success Rate</TableHead>
                    <TableHead>Models</TableHead>
                    <TableHead>
                      <div className="flex items-center gap-1">
                        <Calendar className="h-4 w-4" />
                        Activity
                      </div>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.data.map((user) => (
                    <TableRow key={user.userId}>
                      <TableCell>
                        <Link
                          to={`/users/${encodeURIComponent(user.userId)}`}
                          className="font-medium font-mono text-sm hover:underline"
                        >
                          {user.userId.length > 20 ? `${user.userId.slice(0, 20)}...` : user.userId}
                        </Link>
                      </TableCell>
                      <TableCell>{user.traceCount}</TableCell>
                      <TableCell>{user.sessionCount}</TableCell>
                      <TableCell>{formatLatency(user.avgLatencyMs)}</TableCell>
                      <TableCell>{formatTokens(user.totalTokens)}</TableCell>
                      <TableCell>{formatCost(user.totalCost)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <Progress
                            value={user.successRate * 100}
                            className="w-16 h-2"
                          />
                          <span className="text-sm text-muted-foreground">
                            {(user.successRate * 100).toFixed(0)}%
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1 flex-wrap max-w-[150px]">
                          {user.models?.slice(0, 2).map((model) => (
                            <span
                              key={model}
                              className="inline-flex items-center rounded-md bg-muted px-2 py-0.5 text-xs"
                            >
                              {model}
                            </span>
                          ))}
                          {user.models && user.models.length > 2 && (
                            <span className="text-xs text-muted-foreground">
                              +{user.models.length - 2}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm">
                        <div>First: {formatDate(user.firstSeenTime)}</div>
                        <div className="text-xs">Last: {formatDate(user.lastSeenTime)}</div>
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
                  of {data.total} users
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
