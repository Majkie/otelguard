import { useState, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table';
import {
  Bot,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Eye,
  Search,
  Network,
  Clock,
  Coins,
} from 'lucide-react';

import { useAgents } from '@/api/agents';
import { useTraces, type Trace } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { cn } from '@/lib/utils';

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function formatCost(cost: number): string {
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
}

// Page showing traces that have agents with ability to view graph
export function AgentsPage() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState('');
  const [sorting, setSorting] = useState<SortingState>([{ id: 'startTime', desc: true }]);

  // Get traces (agent graphs are per-trace)
  const { data: tracesData, isLoading: tracesLoading } = useTraces({
    limit: 100,
    sortBy: 'start_time',
    sortOrder: 'DESC',
  });

  // Get agents data
  const { data: agentsData, isLoading: agentsLoading } = useAgents({ limit: 100 });

  // Group agents by trace for the visualization feature
  const traceAgentCounts = useMemo(() => {
    if (!agentsData?.data) return new Map<string, number>();
    const counts = new Map<string, number>();
    agentsData.data.forEach((agent) => {
      counts.set(agent.traceId, (counts.get(agent.traceId) || 0) + 1);
    });
    return counts;
  }, [agentsData]);

  // Filter traces that have agents
  const tracesWithAgents = useMemo(() => {
    if (!tracesData?.data) return [];
    return tracesData.data.filter(
      (trace) => traceAgentCounts.get(trace.id) && traceAgentCounts.get(trace.id)! > 0
    );
  }, [tracesData, traceAgentCounts]);

  const columns: ColumnDef<Trace>[] = useMemo(
    () => [
      {
        accessorKey: 'name',
        header: 'Trace Name',
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Bot className="h-4 w-4 text-muted-foreground" />
            <span className="font-medium">{row.original.name}</span>
          </div>
        ),
      },
      {
        id: 'agentCount',
        header: 'Agents',
        cell: ({ row }) => {
          const count = traceAgentCounts.get(row.original.id) || 0;
          return (
            <Badge variant="secondary" className="font-mono">
              {count}
            </Badge>
          );
        },
      },
      {
        accessorKey: 'latencyMs',
        header: ({ column }) => {
          const sorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              className="-ml-4"
              onClick={() => column.toggleSorting()}
            >
              <Clock className="mr-1 h-4 w-4" />
              Latency
              {sorted === 'asc' ? (
                <ChevronUp className="ml-1 h-4 w-4" />
              ) : sorted === 'desc' ? (
                <ChevronDown className="ml-1 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-1 h-4 w-4" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => formatLatency(row.original.latencyMs),
      },
      {
        accessorKey: 'totalTokens',
        header: 'Tokens',
        cell: ({ row }) => row.original.totalTokens.toLocaleString(),
      },
      {
        accessorKey: 'cost',
        header: ({ column }) => {
          const sorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              className="-ml-4"
              onClick={() => column.toggleSorting()}
            >
              <Coins className="mr-1 h-4 w-4" />
              Cost
              {sorted === 'asc' ? (
                <ChevronUp className="ml-1 h-4 w-4" />
              ) : sorted === 'desc' ? (
                <ChevronDown className="ml-1 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-1 h-4 w-4" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => formatCost(row.original.cost),
      },
      {
        accessorKey: 'status',
        header: 'Status',
        cell: ({ row }) => (
          <Badge
            className={cn(
              'capitalize',
              row.original.status === 'success' &&
                'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
              row.original.status === 'error' &&
                'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
              row.original.status === 'pending' &&
                'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
            )}
          >
            {row.original.status}
          </Badge>
        ),
      },
      {
        accessorKey: 'startTime',
        header: ({ column }) => {
          const sorted = column.getIsSorted();
          return (
            <Button
              variant="ghost"
              className="-ml-4"
              onClick={() => column.toggleSorting()}
            >
              Time
              {sorted === 'asc' ? (
                <ChevronUp className="ml-1 h-4 w-4" />
              ) : sorted === 'desc' ? (
                <ChevronDown className="ml-1 h-4 w-4" />
              ) : (
                <ChevronsUpDown className="ml-1 h-4 w-4" />
              )}
            </Button>
          );
        },
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">
            {new Date(row.original.startTime).toLocaleString()}
          </span>
        ),
      },
      {
        id: 'actions',
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate(`/agents/${row.original.id}`)}
            >
              <Network className="mr-1 h-4 w-4" />
              View Graph
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate(`/traces/${row.original.id}`)}
            >
              <Eye className="h-4 w-4" />
            </Button>
          </div>
        ),
      },
    ],
    [traceAgentCounts, navigate]
  );

  const filteredData = useMemo(() => {
    if (!searchQuery) return tracesWithAgents;
    const query = searchQuery.toLowerCase();
    return tracesWithAgents.filter(
      (trace) =>
        trace.name.toLowerCase().includes(query) ||
        trace.model.toLowerCase().includes(query)
    );
  }, [tracesWithAgents, searchQuery]);

  const table = useReactTable({
    data: filteredData,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    initialState: {
      pagination: { pageSize: 20 },
    },
  });

  const isLoading = tracesLoading || agentsLoading;

  if (isLoading) {
    return (
      <div className="container py-6 space-y-6">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-10 w-64" />
        </div>
        <Card>
          <CardContent className="p-6">
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <Skeleton key={i} className="h-16 w-full" />
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container py-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Network className="h-6 w-6" />
            Agent Graphs
          </h1>
          <p className="text-muted-foreground">
            Visualize multi-agent system execution flows
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="relative w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search traces..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Traces with Agents</CardDescription>
            <CardTitle className="text-2xl">{tracesWithAgents.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Agents</CardDescription>
            <CardTitle className="text-2xl">{agentsData?.total || 0}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Avg Latency</CardDescription>
            <CardTitle className="text-2xl">
              {formatLatency(
                Math.round(
                  tracesWithAgents.reduce((acc, t) => acc + t.latencyMs, 0) /
                    (tracesWithAgents.length || 1)
                )
              )}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>Total Cost</CardDescription>
            <CardTitle className="text-2xl">
              {formatCost(tracesWithAgents.reduce((acc, t) => acc + t.cost, 0))}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              {table.getHeaderGroups().map((headerGroup) => (
                <TableRow key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <TableHead key={header.id}>
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                    </TableHead>
                  ))}
                </TableRow>
              ))}
            </TableHeader>
            <TableBody>
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <TableRow
                    key={row.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/agents/${row.original.id}`)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={columns.length} className="h-24 text-center">
                    <div className="flex flex-col items-center gap-2 text-muted-foreground">
                      <Network className="h-8 w-8" />
                      <p>No traces with agents found</p>
                      <p className="text-sm">
                        Agent data will appear here when multi-agent traces are captured
                      </p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Pagination */}
      {table.getPageCount() > 1 && (
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            Showing {table.getState().pagination.pageIndex * table.getState().pagination.pageSize + 1} to{' '}
            {Math.min(
              (table.getState().pagination.pageIndex + 1) * table.getState().pagination.pageSize,
              filteredData.length
            )}{' '}
            of {filteredData.length} results
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default AgentsPage;
