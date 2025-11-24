import { useMemo } from 'react';
import { X, Clock, Coins, MessageSquare, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import type { GraphNode, GraphEdge, Bottleneck } from '@/types/agent';

interface NodeDetailPanelProps {
  node: GraphNode | null;
  edges: GraphEdge[];
  bottlenecks?: Bottleneck[];
  criticalPath?: string[];
  onClose: () => void;
  onNavigateToNode?: (nodeId: string) => void;
}

const statusConfig: Record<
  string,
  { icon: React.ComponentType<{ className?: string }>; color: string; label: string }
> = {
  success: { icon: CheckCircle2, color: 'text-green-500', label: 'Success' },
  error: { icon: AlertCircle, color: 'text-red-500', label: 'Error' },
  running: { icon: Loader2, color: 'text-blue-500', label: 'Running' },
  timeout: { icon: Clock, color: 'text-amber-500', label: 'Timeout' },
  pending: { icon: Clock, color: 'text-gray-400', label: 'Pending' },
};

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}

function formatCost(cost: number): string {
  if (cost < 0.0001) return `$${cost.toFixed(6)}`;
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString();
}

export function NodeDetailPanel({
  node,
  edges,
  bottlenecks,
  criticalPath,
  onClose,
  onNavigateToNode,
}: NodeDetailPanelProps) {
  const statusInfo = statusConfig[node?.status || 'pending'] || statusConfig.pending;
  const StatusIcon = statusInfo.icon;

  const isOnCriticalPath = useMemo(
    () => node && criticalPath?.includes(node.id),
    [node, criticalPath]
  );

  const bottleneckInfo = useMemo(
    () => bottlenecks?.find((b) => b.nodeId === node?.id),
    [bottlenecks, node]
  );

  const incomingEdges = useMemo(
    () => edges.filter((e) => e.target === node?.id),
    [edges, node]
  );

  const outgoingEdges = useMemo(
    () => edges.filter((e) => e.source === node?.id),
    [edges, node]
  );

  const metadata = useMemo(() => {
    if (!node?.metadata) return null;
    try {
      return JSON.parse(node.metadata);
    } catch {
      return null;
    }
  }, [node?.metadata]);

  if (!node) {
    return (
      <div className="w-80 ml-4 border border-border rounded-lg bg-card p-4 flex items-center justify-center text-muted-foreground">
        Select a node to view details
      </div>
    );
  }

  return (
    <div className="w-80 ml-4 border border-border rounded-lg bg-card flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-border bg-muted/30">
        <div className="flex items-center gap-2">
          <StatusIcon
            className={cn(
              'h-5 w-5',
              statusInfo.color,
              node.status === 'running' && 'animate-spin'
            )}
          />
          <h3 className="font-semibold truncate">{node.label}</h3>
        </div>
        <Button variant="ghost" size="icon" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4 space-y-3">
          {/* Type and Status */}
          <div className="flex items-center gap-2 flex-wrap">
            <Badge variant="outline" className="capitalize">
              {node.type}
            </Badge>
            <Badge
              variant={
                node.status === 'success'
                  ? 'default'
                  : node.status === 'error'
                  ? 'destructive'
                  : 'secondary'
              }
            >
              {statusInfo.label}
            </Badge>
            {isOnCriticalPath && (
              <Badge variant="destructive" className="text-xs">
                Critical Path
              </Badge>
            )}
            {bottleneckInfo && (
              <Badge className="bg-orange-500 text-xs">Bottleneck</Badge>
            )}
          </div>

          {/* Metrics */}
          <div className="space-y-2 pt-3 border-t border-border">
            <h4 className="text-sm font-medium text-foreground">Metrics</h4>
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Latency
                </div>
                <div className="font-medium">{formatLatency(node.latencyMs)}</div>
              </div>
              {node.tokens !== undefined && node.tokens > 0 && (
                <div className="space-y-1">
                  <div className="text-xs text-muted-foreground flex items-center gap-1">
                    <MessageSquare className="h-3 w-3" />
                    Tokens
                  </div>
                  <div className="font-medium">{node.tokens.toLocaleString()}</div>
                </div>
              )}
              {node.cost !== undefined && node.cost > 0 && (
                <div className="space-y-1">
                  <div className="text-xs text-muted-foreground flex items-center gap-1">
                    <Coins className="h-3 w-3" />
                    Cost
                  </div>
                  <div className="font-medium">{formatCost(node.cost)}</div>
                </div>
              )}
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">Depth</div>
                <div className="font-medium">{node.depth}</div>
              </div>
            </div>
            {node.parallelGroup > 0 && (
              <div className="space-y-1">
                <div className="text-xs text-muted-foreground">Parallel Group</div>
                <div className="font-medium">Group {node.parallelGroup}</div>
              </div>
            )}
          </div>

          {/* Bottleneck Info */}
          {bottleneckInfo && (
            <div className="space-y-2 pt-3 border-t border-border">
              <h4 className="text-sm font-medium text-orange-600">Bottleneck Analysis</h4>
                <div className="text-sm text-muted-foreground">
                  <p>
                    This node accounts for{' '}
                    <span className="font-medium text-foreground">
                      {bottleneckInfo.percentage.toFixed(1)}%
                    </span>{' '}
                    of total execution time.
                  </p>
                  <p className="mt-1 text-xs">{bottleneckInfo.reason}</p>
                </div>
              </div>
          )}

          {/* Model */}
          {node.model && (
            <div className="space-y-2 pt-3 border-t border-border">
              <h4 className="text-sm font-medium text-foreground">Model</h4>
              <Badge variant="outline">{node.model}</Badge>
            </div>
          )}

          {/* Timing */}
          <div className="space-y-2 pt-3 border-t border-border">
            <h4 className="text-sm font-medium text-foreground">Timing</h4>
            <div className="space-y-1 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Start:</span>
                <span className="font-mono text-xs">{formatDate(node.startTime)}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">End:</span>
                <span className="font-mono text-xs">{formatDate(node.endTime)}</span>
              </div>
            </div>
          </div>

          {/* Connections */}
          {(incomingEdges.length > 0 || outgoingEdges.length > 0) && (
            <div className="space-y-2 pt-3 border-t border-border">
              <h4 className="text-sm font-medium text-foreground">Connections</h4>
              <div className="space-y-2">

                {incomingEdges.length > 0 && (
                  <div className="space-y-1">
                    <div className="text-xs text-muted-foreground">
                      Incoming ({incomingEdges.length})
                    </div>
                    <div className="space-y-1">
                      {incomingEdges.slice(0, 5).map((edge) => (
                        <button
                          key={edge.id}
                          onClick={() => onNavigateToNode?.(edge.source)}
                          className="w-full text-left text-xs px-2 py-1 rounded bg-muted hover:bg-muted/80 truncate"
                        >
                          <span className="text-muted-foreground capitalize">
                            {edge.type.replace('_', ' ')}
                          </span>
                          {edge.label && (
                            <span className="ml-1 text-foreground">: {edge.label}</span>
                          )}
                        </button>
                      ))}
                      {incomingEdges.length > 5 && (
                        <div className="text-xs text-muted-foreground px-2">
                          +{incomingEdges.length - 5} more
                        </div>
                      )}
                    </div>
                  </div>
                )}

                {outgoingEdges.length > 0 && (
                  <div className="space-y-1">
                    <div className="text-xs text-muted-foreground">
                      Outgoing ({outgoingEdges.length})
                    </div>
                    <div className="space-y-1">
                      {outgoingEdges.slice(0, 5).map((edge) => (
                        <button
                          key={edge.id}
                          onClick={() => onNavigateToNode?.(edge.target)}
                          className="w-full text-left text-xs px-2 py-1 rounded bg-muted hover:bg-muted/80 truncate"
                        >
                          <span className="text-muted-foreground capitalize">
                            {edge.type.replace('_', ' ')}
                          </span>
                          {edge.label && (
                            <span className="ml-1 text-foreground">: {edge.label}</span>
                          )}
                        </button>
                      ))}
                      {outgoingEdges.length > 5 && (
                        <div className="text-xs text-muted-foreground px-2">
                          +{outgoingEdges.length - 5} more
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Metadata */}
          {metadata && Object.keys(metadata).length > 0 && (
            <div className="space-y-2 pt-3 border-t border-border">
              <h4 className="text-sm font-medium text-foreground">Metadata</h4>
                <pre className="text-xs bg-muted p-2 rounded overflow-auto max-h-40 font-mono">
                  {JSON.stringify(metadata, null, 2)}
                </pre>
            </div>
          )}

          {/* IDs */}
          <div className="space-y-2 pt-3 border-t border-border">
            <h4 className="text-sm font-medium text-foreground">Identifiers</h4>
            <div className="space-y-1 text-xs font-mono">
              <div className="flex flex-col gap-0.5">
                <span className="text-muted-foreground">Node ID:</span>
                <span className="truncate">{node.id}</span>
              </div>
              {node.spanId && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-muted-foreground">Span ID:</span>
                  <span className="truncate">{node.spanId}</span>
                </div>
              )}
              {node.agentId && (
                <div className="flex flex-col gap-0.5">
                  <span className="text-muted-foreground">Agent ID:</span>
                  <span className="truncate">{node.agentId}</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </ScrollArea>
    </div>
  );
}
