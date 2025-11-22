import { useState, useMemo } from 'react';
import { useSearchParams, Link } from 'react-router-dom';
import { ArrowLeft, GitCompare, Search, X } from 'lucide-react';
import { useTrace, useTraces } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { DiffViewer } from '@/components/features/traces/diff-viewer';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';
import { cn } from '@/lib/utils';

export function TraceComparePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const leftId = searchParams.get('left') || '';
  const rightId = searchParams.get('right') || '';

  const [leftSearch, setLeftSearch] = useState(leftId);
  const [rightSearch, setRightSearch] = useState(rightId);

  const { data: leftTrace, isLoading: leftLoading } = useTrace(leftId);
  const { data: rightTrace, isLoading: rightLoading } = useTrace(rightId);

  // Recent traces for quick selection
  const { data: recentTraces } = useTraces({ limit: 10, sortBy: 'start_time', sortOrder: 'DESC' });

  const handleSetLeft = (id: string) => {
    setLeftSearch(id);
    setSearchParams((prev) => {
      prev.set('left', id);
      return prev;
    });
  };

  const handleSetRight = (id: string) => {
    setRightSearch(id);
    setSearchParams((prev) => {
      prev.set('right', id);
      return prev;
    });
  };

  const handleApply = () => {
    const params = new URLSearchParams();
    if (leftSearch) params.set('left', leftSearch);
    if (rightSearch) params.set('right', rightSearch);
    setSearchParams(params);
  };

  const bothLoaded = leftTrace && rightTrace;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/traces">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <GitCompare className="h-6 w-6" />
            Compare Traces
          </h1>
          <p className="text-sm text-muted-foreground">
            Compare two traces to see differences in inputs, outputs, and metadata
          </p>
        </div>
      </div>

      {/* Trace Selection */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Select Traces to Compare</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid md:grid-cols-2 gap-6">
            {/* Left trace selection */}
            <div className="space-y-3">
              <label className="text-sm font-medium flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-red-500" />
                Original Trace
              </label>
              <div className="flex gap-2">
                <Input
                  placeholder="Enter trace ID..."
                  value={leftSearch}
                  onChange={(e) => setLeftSearch(e.target.value)}
                  className="font-mono text-sm"
                />
                {leftSearch && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      setLeftSearch('');
                      handleSetLeft('');
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </div>
              {leftTrace && (
                <div className="p-3 bg-muted rounded-lg text-sm">
                  <p className="font-medium">{leftTrace.name}</p>
                  <p className="text-muted-foreground text-xs">
                    {formatDate(leftTrace.startTime)} · {leftTrace.model}
                  </p>
                </div>
              )}
            </div>

            {/* Right trace selection */}
            <div className="space-y-3">
              <label className="text-sm font-medium flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-green-500" />
                Comparison Trace
              </label>
              <div className="flex gap-2">
                <Input
                  placeholder="Enter trace ID..."
                  value={rightSearch}
                  onChange={(e) => setRightSearch(e.target.value)}
                  className="font-mono text-sm"
                />
                {rightSearch && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      setRightSearch('');
                      handleSetRight('');
                    }}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                )}
              </div>
              {rightTrace && (
                <div className="p-3 bg-muted rounded-lg text-sm">
                  <p className="font-medium">{rightTrace.name}</p>
                  <p className="text-muted-foreground text-xs">
                    {formatDate(rightTrace.startTime)} · {rightTrace.model}
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Recent traces quick select */}
          {recentTraces?.data && recentTraces.data.length > 0 && (
            <div className="mt-6 pt-4 border-t">
              <p className="text-sm font-medium mb-3">Recent Traces (click to select)</p>
              <div className="flex flex-wrap gap-2">
                {recentTraces.data.slice(0, 8).map((trace) => (
                  <div key={trace.id} className="flex gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-xs"
                      onClick={() => handleSetLeft(trace.id)}
                      disabled={trace.id === leftId}
                    >
                      <span className="w-2 h-2 rounded-full bg-red-500 mr-1" />
                      {trace.name.slice(0, 20)}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="text-xs"
                      onClick={() => handleSetRight(trace.id)}
                      disabled={trace.id === rightId}
                    >
                      <span className="w-2 h-2 rounded-full bg-green-500 mr-1" />
                      {trace.name.slice(0, 20)}
                    </Button>
                  </div>
                ))}
              </div>
            </div>
          )}

          <div className="mt-4 flex justify-end">
            <Button onClick={handleApply} disabled={!leftSearch || !rightSearch}>
              <Search className="h-4 w-4 mr-2" />
              Compare
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Loading state */}
      {(leftLoading || rightLoading) && (leftId || rightId) && (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
        </div>
      )}

      {/* Comparison Results */}
      {bothLoaded && (
        <>
          {/* Metrics Comparison */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Metrics Comparison</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <MetricComparison
                  label="Latency"
                  left={leftTrace.latencyMs}
                  right={rightTrace.latencyMs}
                  format={(v) => formatLatency(v)}
                  lowerIsBetter
                />
                <MetricComparison
                  label="Tokens"
                  left={leftTrace.totalTokens}
                  right={rightTrace.totalTokens}
                  format={(v) => formatTokens(v)}
                  lowerIsBetter
                />
                <MetricComparison
                  label="Cost"
                  left={leftTrace.cost}
                  right={rightTrace.cost}
                  format={(v) => formatCost(v)}
                  lowerIsBetter
                />
                <MetricComparison
                  label="Status"
                  left={leftTrace.status}
                  right={rightTrace.status}
                  format={(v) => v}
                  isStatus
                />
              </div>
            </CardContent>
          </Card>

          {/* Content Diff */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Content Differences</CardTitle>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="input">
                <TabsList>
                  <TabsTrigger value="input">Input</TabsTrigger>
                  <TabsTrigger value="output">Output</TabsTrigger>
                  <TabsTrigger value="metadata">Metadata</TabsTrigger>
                </TabsList>

                <TabsContent value="input" className="mt-4">
                  <DiffViewer
                    left={formatForDiff(leftTrace.input)}
                    right={formatForDiff(rightTrace.input)}
                    leftLabel={`${leftTrace.name} (Input)`}
                    rightLabel={`${rightTrace.name} (Input)`}
                  />
                </TabsContent>

                <TabsContent value="output" className="mt-4">
                  <DiffViewer
                    left={formatForDiff(leftTrace.output)}
                    right={formatForDiff(rightTrace.output)}
                    leftLabel={`${leftTrace.name} (Output)`}
                    rightLabel={`${rightTrace.name} (Output)`}
                  />
                </TabsContent>

                <TabsContent value="metadata" className="mt-4">
                  <div className="grid md:grid-cols-2 gap-4">
                    <div>
                      <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-red-500" />
                        {leftTrace.name}
                      </h4>
                      <TraceMetadata trace={leftTrace} />
                    </div>
                    <div>
                      <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                        <span className="w-2 h-2 rounded-full bg-green-500" />
                        {rightTrace.name}
                      </h4>
                      <TraceMetadata trace={rightTrace} />
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </>
      )}

      {/* Empty state */}
      {!leftId && !rightId && (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            <GitCompare className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Select two traces above to compare them</p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

interface MetricComparisonProps {
  label: string;
  left: number | string;
  right: number | string;
  format: (value: number | string) => string;
  lowerIsBetter?: boolean;
  isStatus?: boolean;
}

function MetricComparison({
  label,
  left,
  right,
  format,
  lowerIsBetter,
  isStatus,
}: MetricComparisonProps) {
  const leftNum = typeof left === 'number' ? left : 0;
  const rightNum = typeof right === 'number' ? right : 0;
  const diff = rightNum - leftNum;
  const percentChange = leftNum !== 0 ? (diff / leftNum) * 100 : 0;

  let changeColor = 'text-muted-foreground';
  if (!isStatus && diff !== 0) {
    const isImprovement = lowerIsBetter ? diff < 0 : diff > 0;
    changeColor = isImprovement
      ? 'text-green-600 dark:text-green-400'
      : 'text-red-600 dark:text-red-400';
  }

  return (
    <div className="space-y-1">
      <p className="text-xs text-muted-foreground">{label}</p>
      <div className="flex items-baseline gap-2">
        <span className="text-lg font-semibold">{format(right)}</span>
        {!isStatus && diff !== 0 && (
          <span className={cn('text-xs', changeColor)}>
            {diff > 0 ? '+' : ''}
            {percentChange.toFixed(1)}%
          </span>
        )}
      </div>
      <p className="text-xs text-muted-foreground">
        was: {format(left)}
      </p>
    </div>
  );
}

function TraceMetadata({ trace }: { trace: { id: string; projectId: string; sessionId?: string; userId?: string; model: string; startTime: string; tags: string[] } }) {
  return (
    <dl className="text-sm space-y-2 bg-muted p-3 rounded-lg">
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Trace ID</dt>
        <dd className="font-mono text-xs">{trace.id.slice(0, 8)}...</dd>
      </div>
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Model</dt>
        <dd>{trace.model || '-'}</dd>
      </div>
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Session</dt>
        <dd className="font-mono text-xs">{trace.sessionId?.slice(0, 8) || '-'}</dd>
      </div>
      <div className="flex justify-between">
        <dt className="text-muted-foreground">User</dt>
        <dd className="font-mono text-xs">{trace.userId?.slice(0, 8) || '-'}</dd>
      </div>
      <div className="flex justify-between">
        <dt className="text-muted-foreground">Time</dt>
        <dd className="text-xs">{formatDate(trace.startTime)}</dd>
      </div>
      {trace.tags?.length > 0 && (
        <div>
          <dt className="text-muted-foreground mb-1">Tags</dt>
          <dd className="flex flex-wrap gap-1">
            {trace.tags.map((tag) => (
              <span key={tag} className="px-1.5 py-0.5 bg-background rounded text-xs">
                {tag}
              </span>
            ))}
          </dd>
        </div>
      )}
    </dl>
  );
}

function formatForDiff(content: string): string {
  // Try to pretty-print JSON for better diffing
  try {
    const parsed = JSON.parse(content);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return content || '';
  }
}
