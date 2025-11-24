import type { GraphNode, ExecutionLane } from '@/types/agent';

export interface TimelineItem {
  id: string;
  label: string;
  type: GraphNode['type'];
  status: string;
  startTime: Date;
  endTime: Date;
  latencyMs: number;
  tokens?: number;
  cost?: number;
  model?: string;
  laneId: number;
  depth: number;
  parallelGroup: number;
  // Computed positioning (percentages relative to total duration)
  startPercent: number;
  widthPercent: number;
  // Original node reference
  nodeId: string;
}

export interface TimelineData {
  items: TimelineItem[];
  lanes: ExecutionLane[];
  totalDurationMs: number;
  startTime: Date;
  endTime: Date;
  maxLanes: number;
}

/**
 * Convert GraphNodes to TimelineItems organized by execution lanes
 */
export function buildTimelineData(
  nodes: GraphNode[],
  lanes?: ExecutionLane[]
): TimelineData {
  if (!nodes || nodes.length === 0) {
    return {
      items: [],
      lanes: [],
      totalDurationMs: 0,
      startTime: new Date(),
      endTime: new Date(),
      maxLanes: 0,
    };
  }

  // Find the overall time range
  const startTimes = nodes.map((n) => new Date(n.startTime).getTime());
  const endTimes = nodes.map((n) => new Date(n.endTime).getTime());
  const minStartTime = Math.min(...startTimes);
  const maxEndTime = Math.max(...endTimes);
  const totalDurationMs = maxEndTime - minStartTime;

  // Build node-to-lane mapping
  const nodeLaneMap = new Map<string, number>();
  if (lanes && lanes.length > 0) {
    lanes.forEach((lane) => {
      lane.nodes.forEach((nodeId) => {
        nodeLaneMap.set(nodeId, lane.laneId);
      });
    });
  }

  // Assign lanes to nodes that aren't in lanes (based on parallel groups)
  const maxLaneFromLanes = lanes?.reduce((max, l) => Math.max(max, l.laneId), 0) || 0;
  let nextLane = maxLaneFromLanes;
  const parallelGroupToLane = new Map<number, number>();

  nodes.forEach((node) => {
    if (!nodeLaneMap.has(node.id)) {
      // Use parallel group for lane assignment
      if (node.parallelGroup > 0) {
        if (!parallelGroupToLane.has(node.parallelGroup)) {
          parallelGroupToLane.set(node.parallelGroup, nextLane++);
        }
        nodeLaneMap.set(node.id, parallelGroupToLane.get(node.parallelGroup)!);
      } else {
        // Sequential nodes go to lane 0
        nodeLaneMap.set(node.id, 0);
      }
    }
  });

  // Build timeline items
  const items: TimelineItem[] = nodes.map((node) => {
    const nodeStart = new Date(node.startTime).getTime();
    const startPercent = totalDurationMs > 0
      ? ((nodeStart - minStartTime) / totalDurationMs) * 100
      : 0;
    const widthPercent = totalDurationMs > 0
      ? Math.max((node.latencyMs / totalDurationMs) * 100, 0.5) // Minimum 0.5% width for visibility
      : 100;

    return {
      id: node.id,
      label: node.label,
      type: node.type,
      status: node.status,
      startTime: new Date(node.startTime),
      endTime: new Date(node.endTime),
      latencyMs: node.latencyMs,
      tokens: node.tokens,
      cost: node.cost,
      model: node.model,
      laneId: nodeLaneMap.get(node.id) || 0,
      depth: node.depth,
      parallelGroup: node.parallelGroup,
      startPercent,
      widthPercent,
      nodeId: node.id,
    };
  });

  // Sort items by start time within each lane
  items.sort((a, b) => {
    if (a.laneId !== b.laneId) return a.laneId - b.laneId;
    return a.startTime.getTime() - b.startTime.getTime();
  });

  const maxLanes = Math.max(...items.map((i) => i.laneId), 0) + 1;

  return {
    items,
    lanes: lanes || [],
    totalDurationMs,
    startTime: new Date(minStartTime),
    endTime: new Date(maxEndTime),
    maxLanes,
  };
}

/**
 * Get timeline items grouped by lane
 */
export function getItemsByLane(items: TimelineItem[]): Map<number, TimelineItem[]> {
  const byLane = new Map<number, TimelineItem[]>();

  items.forEach((item) => {
    if (!byLane.has(item.laneId)) {
      byLane.set(item.laneId, []);
    }
    byLane.get(item.laneId)!.push(item);
  });

  return byLane;
}

/**
 * Format time for the timeline ruler
 */
export function formatTimelineTime(ms: number): string {
  if (ms < 1) return '0ms';
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

/**
 * Generate time scale markers
 */
export function generateTimeMarkers(
  totalDurationMs: number,
  numMarkers: number = 10
): { percent: number; label: string; ms: number }[] {
  if (totalDurationMs <= 0) return [];

  const markers: { percent: number; label: string; ms: number }[] = [];
  const step = totalDurationMs / numMarkers;

  for (let i = 0; i <= numMarkers; i++) {
    const ms = i * step;
    markers.push({
      percent: (ms / totalDurationMs) * 100,
      label: formatTimelineTime(ms),
      ms,
    });
  }

  return markers;
}

/**
 * Get color for timeline item based on type
 */
export function getTimelineItemColor(type: string): {
  bg: string;
  border: string;
  text: string;
} {
  const colors: Record<string, { bg: string; border: string; text: string }> = {
    agent: {
      bg: 'bg-purple-500/80',
      border: 'border-purple-600',
      text: 'text-white',
    },
    tool: {
      bg: 'bg-amber-500/80',
      border: 'border-amber-600',
      text: 'text-white',
    },
    llm: {
      bg: 'bg-blue-500/80',
      border: 'border-blue-600',
      text: 'text-white',
    },
    retrieval: {
      bg: 'bg-green-500/80',
      border: 'border-green-600',
      text: 'text-white',
    },
    embedding: {
      bg: 'bg-cyan-500/80',
      border: 'border-cyan-600',
      text: 'text-white',
    },
    message: {
      bg: 'bg-gray-500/80',
      border: 'border-gray-600',
      text: 'text-white',
    },
    start: {
      bg: 'bg-emerald-500/80',
      border: 'border-emerald-600',
      text: 'text-white',
    },
    end: {
      bg: 'bg-red-500/80',
      border: 'border-red-600',
      text: 'text-white',
    },
    custom: {
      bg: 'bg-slate-500/80',
      border: 'border-slate-600',
      text: 'text-white',
    },
  };
  return colors[type] || colors.custom;
}

/**
 * Get status indicator style
 */
export function getStatusStyle(status: string): string {
  switch (status) {
    case 'success':
      return 'ring-2 ring-green-400 ring-offset-1';
    case 'error':
      return 'ring-2 ring-red-400 ring-offset-1';
    case 'running':
      return 'ring-2 ring-blue-400 ring-offset-1 animate-pulse';
    case 'timeout':
      return 'ring-2 ring-amber-400 ring-offset-1';
    default:
      return '';
  }
}

/**
 * Calculate zoom level to show a specific time range
 */
export function calculateZoomForRange(
  rangeMs: number,
  totalDurationMs: number,
  minZoom: number = 1,
  maxZoom: number = 10
): number {
  if (totalDurationMs <= 0 || rangeMs <= 0) return minZoom;
  const zoom = totalDurationMs / rangeMs;
  return Math.max(minZoom, Math.min(maxZoom, zoom));
}
