import type { Span } from '@/api/traces';

export interface SpanTreeNode extends Span {
  children: SpanTreeNode[];
  depth: number;
  // Calculated relative timing within trace
  relativeStartMs: number;
  percentOfTrace: number;
}

/**
 * Builds a tree structure from flat span array based on parentSpanId relationships
 */
export function buildSpanTree(
  spans: Span[],
  traceStartTime: string,
  traceDurationMs: number
): SpanTreeNode[] {
  if (!spans || spans.length === 0) return [];

  const traceStart = new Date(traceStartTime).getTime();

  // Create map of spans by ID
  const spanMap = new Map<string, SpanTreeNode>();
  spans.forEach((span) => {
    const spanStart = new Date(span.startTime).getTime();
    const relativeStartMs = spanStart - traceStart;
    const percentOfTrace = traceDurationMs > 0
      ? (span.latencyMs / traceDurationMs) * 100
      : 0;

    spanMap.set(span.id, {
      ...span,
      children: [],
      depth: 0,
      relativeStartMs,
      percentOfTrace,
    });
  });

  // Build tree structure
  const roots: SpanTreeNode[] = [];
  spanMap.forEach((node) => {
    if (node.parentSpanId && spanMap.has(node.parentSpanId)) {
      const parent = spanMap.get(node.parentSpanId)!;
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  });

  // Sort children by start time
  const sortChildren = (nodes: SpanTreeNode[]) => {
    nodes.sort((a, b) => a.relativeStartMs - b.relativeStartMs);
    nodes.forEach((node) => sortChildren(node.children));
  };
  sortChildren(roots);

  // Calculate depths
  const setDepths = (nodes: SpanTreeNode[], depth: number) => {
    nodes.forEach((node) => {
      node.depth = depth;
      setDepths(node.children, depth + 1);
    });
  };
  setDepths(roots, 0);

  return roots;
}

/**
 * Flattens the tree back to a list while preserving tree order (depth-first)
 */
export function flattenSpanTree(roots: SpanTreeNode[]): SpanTreeNode[] {
  const result: SpanTreeNode[] = [];

  const traverse = (nodes: SpanTreeNode[]) => {
    nodes.forEach((node) => {
      result.push(node);
      traverse(node.children);
    });
  };

  traverse(roots);
  return result;
}

/**
 * Get span type color for visualization
 */
export function getSpanTypeColor(type: string): string {
  const colors: Record<string, string> = {
    llm: 'bg-blue-500',
    retrieval: 'bg-green-500',
    tool: 'bg-purple-500',
    agent: 'bg-orange-500',
    embedding: 'bg-cyan-500',
    custom: 'bg-gray-500',
  };
  return colors[type] || colors.custom;
}

/**
 * Get span status color
 */
export function getSpanStatusColor(status: string): string {
  switch (status) {
    case 'success':
      return 'text-green-600 dark:text-green-400';
    case 'error':
      return 'text-red-600 dark:text-red-400';
    default:
      return 'text-yellow-600 dark:text-yellow-400';
  }
}

/**
 * Format duration for display
 */
export function formatSpanDuration(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}
