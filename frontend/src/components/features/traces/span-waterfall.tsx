import { useState, useMemo } from 'react';
import { ChevronRight, ChevronDown, Clock, Coins, Hash } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  buildSpanTree,
  flattenSpanTree,
  getSpanTypeColor,
  getSpanStatusColor,
  formatSpanDuration,
  type SpanTreeNode,
} from '@/lib/span-tree';
import type { Span } from '@/api/traces';
import { formatCost, formatTokens } from '@/lib/utils';

interface SpanWaterfallProps {
  spans: Span[];
  traceStartTime: string;
  traceDurationMs: number;
  onSelectSpan?: (span: SpanTreeNode) => void;
  selectedSpanId?: string;
}

export function SpanWaterfall({
  spans,
  traceStartTime,
  traceDurationMs,
  onSelectSpan,
  selectedSpanId,
}: SpanWaterfallProps) {
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set());
  const [hoveredSpan, setHoveredSpan] = useState<string | null>(null);

  const spanTree = useMemo(
    () => buildSpanTree(spans, traceStartTime, traceDurationMs),
    [spans, traceStartTime, traceDurationMs]
  );

  // Start with all spans expanded
  useMemo(() => {
    const allIds = new Set(spans.map((s) => s.id));
    setExpandedSpans(allIds);
  }, [spans]);

  const toggleExpand = (spanId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setExpandedSpans((prev) => {
      const next = new Set(prev);
      if (next.has(spanId)) {
        next.delete(spanId);
      } else {
        next.add(spanId);
      }
      return next;
    });
  };

  const renderSpanRow = (span: SpanTreeNode) => {
    const hasChildren = span.children.length > 0;
    const isExpanded = expandedSpans.has(span.id);
    const isSelected = selectedSpanId === span.id;
    const isHovered = hoveredSpan === span.id;

    // Calculate waterfall bar position
    const startPercent = traceDurationMs > 0
      ? (span.relativeStartMs / traceDurationMs) * 100
      : 0;
    const widthPercent = Math.max(span.percentOfTrace, 0.5); // Minimum 0.5% width for visibility

    return (
      <div key={span.id}>
        <div
          className={cn(
            'flex items-center gap-2 py-1.5 px-2 rounded cursor-pointer transition-colors',
            isSelected && 'bg-primary/10 border border-primary/50',
            isHovered && !isSelected && 'bg-muted/50',
            !isSelected && !isHovered && 'hover:bg-muted/30'
          )}
          onClick={() => onSelectSpan?.(span)}
          onMouseEnter={() => setHoveredSpan(span.id)}
          onMouseLeave={() => setHoveredSpan(null)}
        >
          {/* Tree indent and expander */}
          <div
            className="flex items-center shrink-0"
            style={{ paddingLeft: `${span.depth * 16}px` }}
          >
            {hasChildren ? (
              <button
                onClick={(e) => toggleExpand(span.id, e)}
                className="p-0.5 hover:bg-muted rounded"
              >
                {isExpanded ? (
                  <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                )}
              </button>
            ) : (
              <span className="w-4" />
            )}
          </div>

          {/* Span type indicator */}
          <div
            className={cn(
              'w-2 h-2 rounded-full shrink-0',
              getSpanTypeColor(span.type)
            )}
            title={span.type}
          />

          {/* Span name */}
          <span className="text-sm font-medium truncate min-w-[120px] max-w-[200px]">
            {span.name}
          </span>

          {/* Type badge */}
          <span className="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded shrink-0">
            {span.type}
          </span>

          {/* Status indicator */}
          <span className={cn('text-xs shrink-0', getSpanStatusColor(span.status))}>
            {span.status === 'success' ? '✓' : span.status === 'error' ? '✗' : '⋯'}
          </span>

          {/* Waterfall bar container */}
          <div className="flex-1 h-5 bg-muted/30 rounded relative overflow-hidden min-w-[200px]">
            <div
              className={cn(
                'absolute h-full rounded transition-all',
                getSpanTypeColor(span.type),
                'opacity-80 hover:opacity-100'
              )}
              style={{
                left: `${startPercent}%`,
                width: `${widthPercent}%`,
              }}
            />
            {/* Duration label on bar */}
            <span
              className="absolute text-[10px] font-medium text-white px-1 truncate"
              style={{
                left: `${startPercent}%`,
                top: '50%',
                transform: 'translateY(-50%)',
              }}
            >
              {formatSpanDuration(span.latencyMs)}
            </span>
          </div>

          {/* Metrics */}
          <div className="flex items-center gap-3 text-xs text-muted-foreground shrink-0">
            <span className="flex items-center gap-1" title="Latency">
              <Clock className="h-3 w-3" />
              {formatSpanDuration(span.latencyMs)}
            </span>
            {span.tokens > 0 && (
              <span className="flex items-center gap-1" title="Tokens">
                <Hash className="h-3 w-3" />
                {formatTokens(span.tokens)}
              </span>
            )}
            {span.cost > 0 && (
              <span className="flex items-center gap-1" title="Cost">
                <Coins className="h-3 w-3" />
                {formatCost(span.cost)}
              </span>
            )}
          </div>
        </div>

        {/* Render children if expanded */}
        {hasChildren && isExpanded && (
          <div>{span.children.map((child) => renderSpanRow(child))}</div>
        )}
      </div>
    );
  };

  if (spanTree.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground">
        No spans available for this trace
      </div>
    );
  }

  return (
    <div className="space-y-1">
      {/* Header */}
      <div className="flex items-center gap-2 py-2 px-2 text-xs font-medium text-muted-foreground border-b">
        <div style={{ width: '24px' }} />
        <div className="w-2" />
        <div className="min-w-[120px] max-w-[200px]">Name</div>
        <div className="w-16">Type</div>
        <div className="w-4">Status</div>
        <div className="flex-1 min-w-[200px]">Timeline</div>
        <div className="w-36">Metrics</div>
      </div>

      {/* Span rows */}
      {spanTree.map((span) => renderSpanRow(span))}

      {/* Legend */}
      <div className="flex items-center gap-4 pt-4 text-xs text-muted-foreground border-t mt-4">
        <span className="font-medium">Span Types:</span>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-blue-500" />
          <span>LLM</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-green-500" />
          <span>Retrieval</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-purple-500" />
          <span>Tool</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-orange-500" />
          <span>Agent</span>
        </div>
        <div className="flex items-center gap-1">
          <div className="w-2 h-2 rounded-full bg-cyan-500" />
          <span>Embedding</span>
        </div>
      </div>
    </div>
  );
}
