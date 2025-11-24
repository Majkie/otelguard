import { memo } from 'react';
import {
  BaseEdge,
  EdgeLabelRenderer,
  getBezierPath,
  type Edge,
  type Position,
} from '@xyflow/react';
import { cn } from '@/lib/utils';
import type { EdgeType } from '@/types/agent';

export interface AgentEdgeData extends Record<string, unknown> {
  type: EdgeType;
  label?: string;
  latencyMs?: number;
  order: number;
  isOnCriticalPath?: boolean;
  isHighlighted?: boolean;
}

export type AgentEdgeType = Edge<AgentEdgeData>;

interface AgentEdgeProps {
  id: string;
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  sourcePosition: Position;
  targetPosition: Position;
  data?: AgentEdgeData;
  selected?: boolean;
  markerEnd?: string;
}

const edgeTypeConfig: Record<
  EdgeType,
  {
    color: string;
    strokeDasharray?: string;
    strokeWidth: number;
    animated?: boolean;
  }
> = {
  delegation: {
    color: '#8b5cf6', // purple
    strokeWidth: 2,
  },
  tool_call: {
    color: '#f59e0b', // amber
    strokeWidth: 2,
  },
  llm_call: {
    color: '#3b82f6', // blue
    strokeWidth: 2,
  },
  message: {
    color: '#6b7280', // gray
    strokeWidth: 1.5,
    strokeDasharray: '5,5',
  },
  sequence: {
    color: '#10b981', // emerald
    strokeWidth: 1.5,
  },
  parallel: {
    color: '#06b6d4', // cyan
    strokeWidth: 1.5,
    strokeDasharray: '3,3',
    animated: true,
  },
  return: {
    color: '#ec4899', // pink
    strokeWidth: 1.5,
    strokeDasharray: '8,4',
  },
  custom: {
    color: '#64748b', // slate
    strokeWidth: 1.5,
  },
};

const edgeTypeLabels: Record<EdgeType, string> = {
  delegation: 'Delegates',
  tool_call: 'Tool Call',
  llm_call: 'LLM Call',
  message: 'Message',
  sequence: 'Then',
  parallel: 'Parallel',
  return: 'Returns',
  custom: 'Custom',
};

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function AgentEdgeComponent({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
  selected,
  markerEnd,
}: AgentEdgeProps) {
  const edgeType = data?.type || 'custom';
  const config = edgeTypeConfig[edgeType];

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const strokeColor = data?.isOnCriticalPath
    ? '#ef4444' // red for critical path
    : data?.isHighlighted
    ? '#eab308' // yellow for highlighted
    : config.color;

  const strokeWidth = data?.isOnCriticalPath || data?.isHighlighted
    ? config.strokeWidth + 1
    : config.strokeWidth;

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        markerEnd={markerEnd}
        style={{
          stroke: strokeColor,
          strokeWidth,
          strokeDasharray: config.strokeDasharray,
        }}
        className={cn(
          'transition-all',
          config.animated && 'animate-pulse',
          selected && 'stroke-[3px]'
        )}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: 'absolute',
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            pointerEvents: 'all',
          }}
          className="nodrag nopan"
        >
          <div
            className={cn(
              'flex flex-col items-center gap-0.5 px-2 py-1 rounded text-[10px] font-medium',
              'bg-background/90 border shadow-sm',
              selected && 'ring-1 ring-primary',
              data?.isOnCriticalPath && 'bg-red-50 dark:bg-red-950 border-red-300',
              data?.isHighlighted && 'bg-yellow-50 dark:bg-yellow-950 border-yellow-300'
            )}
          >
            <span style={{ color: strokeColor }}>
              {data?.label || edgeTypeLabels[data?.type || 'custom']}
            </span>
            {data?.latencyMs !== undefined && data.latencyMs > 0 && (
              <span className="text-[9px] text-muted-foreground">
                +{formatLatency(data.latencyMs)}
              </span>
            )}
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

export const AgentEdge = memo(AgentEdgeComponent);
