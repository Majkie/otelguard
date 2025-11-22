import { useParams, Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { useTrace, useTraceSpans } from '@/api/traces';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatDate, formatCost, formatLatency, formatTokens } from '@/lib/utils';

export function TraceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: trace, isLoading, error } = useTrace(id || '');
  const { data: spansData } = useTraceSpans(id || '');

  if (isLoading) {
    return (
      <div className="flex justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
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

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/traces">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold">{trace.name}</h1>
          <p className="text-sm text-muted-foreground">{trace.id}</p>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Latency</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {formatLatency(trace.latencyMs)}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Tokens</CardTitle>
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
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Cost</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatCost(trace.cost)}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Model</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{trace.model || '-'}</div>
          </CardContent>
        </Card>
      </div>

      {/* Details */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Input</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="whitespace-pre-wrap text-sm bg-muted p-4 rounded-lg overflow-auto max-h-96">
              {trace.input || 'No input'}
            </pre>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Output</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="whitespace-pre-wrap text-sm bg-muted p-4 rounded-lg overflow-auto max-h-96">
              {trace.output || 'No output'}
            </pre>
          </CardContent>
        </Card>
      </div>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle>Metadata</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <dt className="text-muted-foreground">Trace ID</dt>
              <dd className="font-mono">{trace.id}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Project ID</dt>
              <dd className="font-mono">{trace.projectId}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Session ID</dt>
              <dd className="font-mono">{trace.sessionId || '-'}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">User ID</dt>
              <dd className="font-mono">{trace.userId || '-'}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Start Time</dt>
              <dd>{formatDate(trace.startTime)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">End Time</dt>
              <dd>{formatDate(trace.endTime)}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Status</dt>
              <dd>
                <span
                  className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
                    trace.status === 'success'
                      ? 'bg-green-100 text-green-700'
                      : 'bg-red-100 text-red-700'
                  }`}
                >
                  {trace.status}
                </span>
              </dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Tags</dt>
              <dd className="flex gap-1 flex-wrap">
                {trace.tags?.length ? (
                  trace.tags.map((tag) => (
                    <span
                      key={tag}
                      className="inline-flex items-center rounded-full bg-muted px-2 py-1 text-xs"
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
        </CardContent>
      </Card>

      {/* Spans */}
      {spansData?.data?.length ? (
        <Card>
          <CardHeader>
            <CardTitle>Spans ({spansData.data.length})</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {spansData.data.map((span) => (
                <div
                  key={span.id}
                  className="border rounded-lg p-4 space-y-2"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{span.name}</span>
                    <span className="text-sm text-muted-foreground">
                      {formatLatency(span.latencyMs)}
                    </span>
                  </div>
                  <div className="flex gap-4 text-sm text-muted-foreground">
                    <span>Type: {span.type}</span>
                    <span>Tokens: {formatTokens(span.tokens)}</span>
                    {span.model && <span>Model: {span.model}</span>}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
