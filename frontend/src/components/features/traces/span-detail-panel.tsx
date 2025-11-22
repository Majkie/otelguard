import { X, Clock, Hash, Coins, Calendar, AlertCircle } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { JsonViewer, RawViewer } from './json-viewer';
import { formatSpanDuration, getSpanTypeColor, getSpanStatusColor } from '@/lib/span-tree';
import type { SpanTreeNode } from '@/lib/span-tree';
import { formatCost, formatDate, formatTokens } from '@/lib/utils';

interface SpanDetailPanelProps {
  span: SpanTreeNode;
  onClose: () => void;
  className?: string;
}

export function SpanDetailPanel({ span, onClose, className }: SpanDetailPanelProps) {
  return (
    <div
      className={cn(
        'border-l bg-background overflow-hidden flex flex-col',
        className
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-3 min-w-0">
          <div
            className={cn(
              'w-3 h-3 rounded-full shrink-0',
              getSpanTypeColor(span.type)
            )}
          />
          <div className="min-w-0">
            <h3 className="font-semibold truncate">{span.name}</h3>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {span.id}
            </p>
          </div>
        </div>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Quick stats */}
      <div className="grid grid-cols-2 gap-4 p-4 border-b bg-muted/30">
        <div className="flex items-center gap-2">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <div>
            <p className="text-xs text-muted-foreground">Duration</p>
            <p className="font-medium">{formatSpanDuration(span.latencyMs)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Hash className="h-4 w-4 text-muted-foreground" />
          <div>
            <p className="text-xs text-muted-foreground">Tokens</p>
            <p className="font-medium">{formatTokens(span.tokens)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Coins className="h-4 w-4 text-muted-foreground" />
          <div>
            <p className="text-xs text-muted-foreground">Cost</p>
            <p className="font-medium">{formatCost(span.cost)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Calendar className="h-4 w-4 text-muted-foreground" />
          <div>
            <p className="text-xs text-muted-foreground">Start Time</p>
            <p className="font-medium text-xs">{formatDate(span.startTime)}</p>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="io" className="flex-1 flex flex-col overflow-hidden">
        <TabsList className="w-full justify-start rounded-none border-b px-4 h-10">
          <TabsTrigger value="io" className="text-xs">Input/Output</TabsTrigger>
          <TabsTrigger value="metadata" className="text-xs">Metadata</TabsTrigger>
          <TabsTrigger value="info" className="text-xs">Details</TabsTrigger>
        </TabsList>

        <TabsContent value="io" className="flex-1 overflow-auto p-4 space-y-4 mt-0">
          <div>
            <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
              Input
              <span className="text-xs text-muted-foreground font-normal">
                ({span.input?.length || 0} chars)
              </span>
            </h4>
            <JsonViewer data={span.input || ''} defaultExpanded maxDepth={3} />
          </div>
          <div>
            <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
              Output
              <span className="text-xs text-muted-foreground font-normal">
                ({span.output?.length || 0} chars)
              </span>
            </h4>
            <JsonViewer data={span.output || ''} defaultExpanded maxDepth={3} />
          </div>
        </TabsContent>

        <TabsContent value="metadata" className="flex-1 overflow-auto p-4 mt-0">
          <h4 className="text-sm font-medium mb-2">Metadata</h4>
          <JsonViewer data="{}" defaultExpanded />
        </TabsContent>

        <TabsContent value="info" className="flex-1 overflow-auto p-4 mt-0">
          <dl className="space-y-4">
            <div>
              <dt className="text-xs text-muted-foreground">Span ID</dt>
              <dd className="font-mono text-sm">{span.id}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Trace ID</dt>
              <dd className="font-mono text-sm">{span.traceId}</dd>
            </div>
            {span.parentSpanId && (
              <div>
                <dt className="text-xs text-muted-foreground">Parent Span ID</dt>
                <dd className="font-mono text-sm">{span.parentSpanId}</dd>
              </div>
            )}
            <div>
              <dt className="text-xs text-muted-foreground">Type</dt>
              <dd className="flex items-center gap-2">
                <div
                  className={cn(
                    'w-2 h-2 rounded-full',
                    getSpanTypeColor(span.type)
                  )}
                />
                <span className="capitalize">{span.type}</span>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Status</dt>
              <dd className={cn('font-medium', getSpanStatusColor(span.status))}>
                {span.status}
              </dd>
            </div>
            {span.model && (
              <div>
                <dt className="text-xs text-muted-foreground">Model</dt>
                <dd>{span.model}</dd>
              </div>
            )}
            <div>
              <dt className="text-xs text-muted-foreground">Start Time</dt>
              <dd>{formatDate(span.startTime)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">End Time</dt>
              <dd>{formatDate(span.endTime)}</dd>
            </div>
            <div>
              <dt className="text-xs text-muted-foreground">Tree Depth</dt>
              <dd>{span.depth}</dd>
            </div>
            {span.children.length > 0 && (
              <div>
                <dt className="text-xs text-muted-foreground">Child Spans</dt>
                <dd>{span.children.length}</dd>
              </div>
            )}
          </dl>
        </TabsContent>
      </Tabs>

      {/* Error message if present */}
      {span.status === 'error' && (
        <div className="p-4 border-t bg-destructive/10">
          <div className="flex items-start gap-2 text-destructive">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <div className="text-sm">
              <p className="font-medium">Error</p>
              <p className="text-xs opacity-90">
                This span encountered an error during execution.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
