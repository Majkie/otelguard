import { useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Clock, Hash, Coins, Cpu, Download } from 'lucide-react';
import { useTrace, useTraceSpans } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SpanWaterfall } from '@/components/features/traces/span-waterfall';
import { SpanDetailPanel } from '@/components/features/traces/span-detail-panel';
import { JsonViewer } from '@/components/features/traces/json-viewer';
import { TraceDetailSkeleton } from '@/components/features/traces/traces-skeleton';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';
import { exportTraceDetailToJson } from '@/lib/export';
import type { SpanTreeNode } from '@/lib/span-tree';
import { cn } from '@/lib/utils';

export function TraceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: trace, isLoading, error } = useTrace(id || '');
  const { data: spansData } = useTraceSpans(id || '');
  const [selectedSpan, setSelectedSpan] = useState<SpanTreeNode | null>(null);

  if (isLoading) {
    return <TraceDetailSkeleton />;
  }

  if (error || !trace) {
    return (
      <div className="space-y-4">
        <Link to="/traces">
          <Button variant="ghost">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Traces
          </Button>
        </Link>
        <p className="text-destructive">Error loading trace</p>
      </div>
    );
  }

  const hasSpans = spansData?.data && spansData.data.length > 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/traces">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold truncate">{trace.name}</h1>
            <span
              className={cn(
                'inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium',
                trace.status === 'success'
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                  : trace.status === 'error'
                    ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                    : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400'
              )}
            >
              {trace.status}
            </span>
          </div>
          <p className="text-sm text-muted-foreground font-mono truncate">
            {trace.id}
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="text-right text-sm text-muted-foreground">
            {formatDate(trace.startTime)}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => exportTraceDetailToJson(trace, spansData?.data || [])}
          >
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Latency</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatLatency(trace.latencyMs)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Tokens</CardTitle>
            <Hash className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatTokens(trace.totalTokens)}
            </div>
            <p className="text-xs text-muted-foreground">
              {trace.promptTokens} prompt / {trace.completionTokens} completion
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Cost</CardTitle>
            <Coins className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatCost(trace.cost)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Model</CardTitle>
            <Cpu className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold truncate">
              {trace.model || '-'}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Main content area */}
      <div className={cn('flex gap-6', selectedSpan && 'pr-0')}>
        {/* Left side - main content */}
        <div className={cn('flex-1 space-y-6 min-w-0', selectedSpan && 'max-w-[calc(100%-400px)]')}>
          <Tabs defaultValue={hasSpans ? 'spans' : 'io'}>
            <TabsList>
              {hasSpans && <TabsTrigger value="spans">Spans ({spansData.data.length})</TabsTrigger>}
              <TabsTrigger value="io">Input/Output</TabsTrigger>
              <TabsTrigger value="metadata">Metadata</TabsTrigger>
            </TabsList>

            {hasSpans && (
              <TabsContent value="spans">
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                      <span>Span Timeline</span>
                      <span className="text-sm font-normal text-muted-foreground">
                        Total: {formatLatency(trace.latencyMs)}
                      </span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <SpanWaterfall
                      spans={spansData.data}
                      traceStartTime={trace.startTime}
                      traceDurationMs={trace.latencyMs}
                      onSelectSpan={setSelectedSpan}
                      selectedSpanId={selectedSpan?.id}
                    />
                  </CardContent>
                </Card>
              </TabsContent>
            )}

            <TabsContent value="io">
              <div className="grid gap-6 lg:grid-cols-2">
                <Card>
                  <CardHeader>
                    <CardTitle>Input</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <JsonViewer data={trace.input || ''} defaultExpanded maxDepth={4} />
                  </CardContent>
                </Card>
                <Card>
                  <CardHeader>
                    <CardTitle>Output</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <JsonViewer data={trace.output || ''} defaultExpanded maxDepth={4} />
                  </CardContent>
                </Card>
              </div>
            </TabsContent>

            <TabsContent value="metadata">
              <Card>
                <CardHeader>
                  <CardTitle>Trace Metadata</CardTitle>
                </CardHeader>
                <CardContent>
                  <dl className="grid grid-cols-2 md:grid-cols-3 gap-x-6 gap-y-4 text-sm">
                    <div>
                      <dt className="text-muted-foreground">Trace ID</dt>
                      <dd className="font-mono text-xs mt-1 break-all">{trace.id}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Project ID</dt>
                      <dd className="font-mono text-xs mt-1 break-all">{trace.projectId}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Session ID</dt>
                      <dd className="font-mono text-xs mt-1 break-all">{trace.sessionId || '-'}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">User ID</dt>
                      <dd className="font-mono text-xs mt-1 break-all">{trace.userId || '-'}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Start Time</dt>
                      <dd className="mt-1">{formatDate(trace.startTime)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">End Time</dt>
                      <dd className="mt-1">{formatDate(trace.endTime)}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Model</dt>
                      <dd className="mt-1">{trace.model || '-'}</dd>
                    </div>
                    <div>
                      <dt className="text-muted-foreground">Status</dt>
                      <dd className="mt-1">
                        <span
                          className={cn(
                            'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
                            trace.status === 'success'
                              ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                              : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                          )}
                        >
                          {trace.status}
                        </span>
                      </dd>
                    </div>
                    <div className="col-span-2 md:col-span-1">
                      <dt className="text-muted-foreground">Tags</dt>
                      <dd className="flex gap-1 flex-wrap mt-1">
                        {trace.tags?.length ? (
                          trace.tags.map((tag) => (
                            <span
                              key={tag}
                              className="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs"
                            >
                              {tag}
                            </span>
                          ))
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </dd>
                    </div>
                  </dl>

                  {trace.errorMessage && (
                    <div className="mt-6 p-4 bg-destructive/10 rounded-lg border border-destructive/30">
                      <h4 className="text-sm font-medium text-destructive mb-2">Error Message</h4>
                      <p className="text-sm text-destructive/90">{trace.errorMessage}</p>
                    </div>
                  )}

                  {trace.metadata && trace.metadata !== '{}' && (
                    <div className="mt-6">
                      <h4 className="text-sm font-medium mb-2">Custom Metadata</h4>
                      <JsonViewer data={trace.metadata} defaultExpanded maxDepth={3} />
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>

        {/* Right side - span detail panel */}
        {selectedSpan && (
          <SpanDetailPanel
            span={selectedSpan}
            onClose={() => setSelectedSpan(null)}
            className="w-[400px] shrink-0 sticky top-20 h-[calc(100vh-120px)]"
          />
        )}
      </div>
    </div>
  );
}
