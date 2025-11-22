import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { formatDate, formatLatency, cn } from '@/lib/utils';
import type { Trace } from '@/api/traces';
import { CheckCircle, XCircle, Clock, MessageSquare } from 'lucide-react';

interface SessionTimelineProps {
  traces: Trace[];
  sessionStart: string;
  sessionEnd: string;
}

export function SessionTimeline({ traces, sessionStart, sessionEnd }: SessionTimelineProps) {
  // Sort traces by start time
  const sortedTraces = useMemo(() => {
    return [...traces].sort(
      (a, b) => new Date(a.startTime).getTime() - new Date(b.startTime).getTime()
    );
  }, [traces]);

  // Calculate timeline bounds
  const { minTime, maxTime, totalDuration } = useMemo(() => {
    if (sortedTraces.length === 0) {
      return { minTime: 0, maxTime: 0, totalDuration: 1 };
    }
    const min = new Date(sessionStart).getTime();
    const max = new Date(sessionEnd).getTime();
    return {
      minTime: min,
      maxTime: max,
      totalDuration: max - min || 1,
    };
  }, [sortedTraces, sessionStart, sessionEnd]);

  // Calculate position and width for each trace
  const traceItems = useMemo(() => {
    return sortedTraces.map((trace) => {
      const start = new Date(trace.startTime).getTime();
      const end = new Date(trace.endTime).getTime();
      const left = ((start - minTime) / totalDuration) * 100;
      const width = Math.max(((end - start) / totalDuration) * 100, 0.5); // Minimum 0.5% width
      return {
        trace,
        left: Math.min(left, 99), // Ensure it stays within bounds
        width: Math.min(width, 100 - left),
      };
    });
  }, [sortedTraces, minTime, totalDuration]);

  if (traces.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5" />
            Session Timeline
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-muted-foreground text-center py-4">
            No traces in this session
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Clock className="h-5 w-5" />
          Session Timeline
        </CardTitle>
      </CardHeader>
      <CardContent>
        {/* Timeline header */}
        <div className="flex justify-between text-xs text-muted-foreground mb-2">
          <span>{formatDate(sessionStart)}</span>
          <span>{formatDate(sessionEnd)}</span>
        </div>

        {/* Timeline bar */}
        <div className="relative h-8 bg-muted rounded-lg mb-4 overflow-hidden">
          {traceItems.map(({ trace, left, width }) => (
            <Link
              key={trace.id}
              to={`/traces/${trace.id}`}
              className={cn(
                'absolute top-1 bottom-1 rounded transition-all hover:opacity-80',
                trace.status === 'success'
                  ? 'bg-green-500/70'
                  : trace.status === 'error'
                  ? 'bg-red-500/70'
                  : 'bg-blue-500/70'
              )}
              style={{
                left: `${left}%`,
                width: `${width}%`,
                minWidth: '4px',
              }}
              title={`${trace.name} - ${formatLatency(trace.latencyMs)}`}
            />
          ))}
        </div>

        {/* Trace list */}
        <div className="space-y-2">
          {sortedTraces.map((trace, index) => (
            <div
              key={trace.id}
              className="flex items-center gap-3 p-2 rounded-lg hover:bg-muted/50 transition-colors"
            >
              {/* Timeline indicator */}
              <div className="relative flex items-center">
                <div
                  className={cn(
                    'w-3 h-3 rounded-full',
                    trace.status === 'success'
                      ? 'bg-green-500'
                      : trace.status === 'error'
                      ? 'bg-red-500'
                      : 'bg-blue-500'
                  )}
                />
                {index < sortedTraces.length - 1 && (
                  <div className="absolute top-3 left-1.5 w-0.5 h-8 bg-border -translate-x-1/2" />
                )}
              </div>

              {/* Trace info */}
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <Link
                    to={`/traces/${trace.id}`}
                    className="font-medium text-sm hover:underline truncate"
                  >
                    {trace.name}
                  </Link>
                  {trace.status === 'success' ? (
                    <CheckCircle className="h-4 w-4 text-green-500 flex-shrink-0" />
                  ) : trace.status === 'error' ? (
                    <XCircle className="h-4 w-4 text-red-500 flex-shrink-0" />
                  ) : null}
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{formatDate(trace.startTime)}</span>
                  <span>|</span>
                  <span>{formatLatency(trace.latencyMs)}</span>
                  {trace.model && (
                    <>
                      <span>|</span>
                      <Badge variant="outline" className="text-xs py-0">
                        {trace.model}
                      </Badge>
                    </>
                  )}
                </div>
              </div>

              {/* Metrics */}
              <div className="text-right text-sm">
                <div className="flex items-center gap-1 text-muted-foreground">
                  <MessageSquare className="h-3 w-3" />
                  <span>{trace.totalTokens} tokens</span>
                </div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
