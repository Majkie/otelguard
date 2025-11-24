import { memo } from 'react';
import { Handle, Position, type Node } from '@xyflow/react';
import { cn } from '@/lib/utils';
import {
  Bot,
  Wrench,
  Brain,
  Database,
  Layers,
  MessageSquare,
  Play,
  Square,
  Box,
  AlertCircle,
  CheckCircle2,
  Clock,
  Loader2,
} from 'lucide-react';
import type { NodeType } from '@/types/agent';

export interface AgentNodeData extends Record<string, unknown> {
  label: string;
  type: NodeType;
  status: string;
  latencyMs: number;
  tokens?: number;
  cost?: number;
  model?: string;
  depth: number;
  parallelGroup: number;
  isOnCriticalPath?: boolean;
  isHighlighted?: boolean;
  isBottleneck?: boolean;
}

export type AgentNodeType = Node<AgentNodeData>;

interface AgentNodeProps {
  data: AgentNodeData;
  selected?: boolean;
}

const nodeTypeConfig: Record<
  NodeType,
  {
    icon: React.ComponentType<{ className?: string }>;
    bgColor: string;
    borderColor: string;
    iconColor: string;
  }
> = {
  agent: {
    icon: Bot,
    bgColor: 'bg-purple-50 dark:bg-purple-950',
    borderColor: 'border-purple-300 dark:border-purple-700',
    iconColor: 'text-purple-600 dark:text-purple-400',
  },
  tool: {
    icon: Wrench,
    bgColor: 'bg-amber-50 dark:bg-amber-950',
    borderColor: 'border-amber-300 dark:border-amber-700',
    iconColor: 'text-amber-600 dark:text-amber-400',
  },
  llm: {
    icon: Brain,
    bgColor: 'bg-blue-50 dark:bg-blue-950',
    borderColor: 'border-blue-300 dark:border-blue-700',
    iconColor: 'text-blue-600 dark:text-blue-400',
  },
  retrieval: {
    icon: Database,
    bgColor: 'bg-green-50 dark:bg-green-950',
    borderColor: 'border-green-300 dark:border-green-700',
    iconColor: 'text-green-600 dark:text-green-400',
  },
  embedding: {
    icon: Layers,
    bgColor: 'bg-cyan-50 dark:bg-cyan-950',
    borderColor: 'border-cyan-300 dark:border-cyan-700',
    iconColor: 'text-cyan-600 dark:text-cyan-400',
  },
  message: {
    icon: MessageSquare,
    bgColor: 'bg-gray-50 dark:bg-gray-900',
    borderColor: 'border-gray-300 dark:border-gray-700',
    iconColor: 'text-gray-600 dark:text-gray-400',
  },
  start: {
    icon: Play,
    bgColor: 'bg-emerald-50 dark:bg-emerald-950',
    borderColor: 'border-emerald-300 dark:border-emerald-700',
    iconColor: 'text-emerald-600 dark:text-emerald-400',
  },
  end: {
    icon: Square,
    bgColor: 'bg-red-50 dark:bg-red-950',
    borderColor: 'border-red-300 dark:border-red-700',
    iconColor: 'text-red-600 dark:text-red-400',
  },
  custom: {
    icon: Box,
    bgColor: 'bg-slate-50 dark:bg-slate-900',
    borderColor: 'border-slate-300 dark:border-slate-700',
    iconColor: 'text-slate-600 dark:text-slate-400',
  },
};

const statusConfig: Record<
  string,
  { icon: React.ComponentType<{ className?: string }>; color: string }
> = {
  success: { icon: CheckCircle2, color: 'text-green-500' },
  error: { icon: AlertCircle, color: 'text-red-500' },
  running: { icon: Loader2, color: 'text-blue-500' },
  timeout: { icon: Clock, color: 'text-amber-500' },
  pending: { icon: Clock, color: 'text-gray-400' },
};

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function formatCost(cost: number): string {
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
}

function AgentNodeComponent({ data, selected }: AgentNodeProps) {
  const config = nodeTypeConfig[data.type] || nodeTypeConfig.custom;
  const Icon = config.icon;
  const statusInfo = statusConfig[data.status] || statusConfig.pending;
  const StatusIcon = statusInfo.icon;

  return (
    <div
      className={cn(
        'relative min-w-[180px] rounded-lg border-2 p-3 shadow-sm transition-all',
        config.bgColor,
        config.borderColor,
        selected && 'ring-2 ring-primary ring-offset-2',
        data.isHighlighted && 'ring-2 ring-yellow-400 ring-offset-2',
        data.isOnCriticalPath && 'border-red-500 dark:border-red-400',
        data.isBottleneck && 'border-dashed border-orange-500'
      )}
    >
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-gray-400 !border-2 !border-gray-200 !w-3 !h-3"
      />

      {/* Header */}
      <div className="flex items-center gap-2 mb-2">
        <div className={cn('p-1.5 rounded-md', config.bgColor)}>
          <Icon className={cn('h-4 w-4', config.iconColor)} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="font-medium text-sm truncate text-foreground">{data.label}</div>
          <div className="text-xs text-muted-foreground capitalize">{data.type}</div>
        </div>
        <StatusIcon
          className={cn(
            'h-4 w-4 flex-shrink-0',
            statusInfo.color,
            data.status === 'running' && 'animate-spin'
          )}
        />
      </div>

      {/* Metrics */}
      <div className="flex items-center gap-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {formatLatency(data.latencyMs)}
        </span>
        {data.tokens !== undefined && data.tokens > 0 && (
          <span className="flex items-center gap-1">
            <MessageSquare className="h-3 w-3" />
            {data.tokens.toLocaleString()}
          </span>
        )}
        {data.cost !== undefined && data.cost > 0 && (
          <span className="flex items-center gap-1">{formatCost(data.cost)}</span>
        )}
      </div>

      {/* Model Badge */}
      {data.model && (
        <div className="mt-2">
          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300">
            {data.model}
          </span>
        </div>
      )}

      {/* Parallel Group Indicator */}
      {data.parallelGroup > 0 && (
        <div className="absolute -top-2 -right-2 w-5 h-5 rounded-full bg-blue-500 text-white text-[10px] font-medium flex items-center justify-center">
          {data.parallelGroup}
        </div>
      )}

      {/* Bottleneck Indicator */}
      {data.isBottleneck && (
        <div className="absolute -bottom-2 left-1/2 -translate-x-1/2 px-1.5 py-0.5 rounded bg-orange-500 text-white text-[9px] font-medium">
          Bottleneck
        </div>
      )}

      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-gray-400 !border-2 !border-gray-200 !w-3 !h-3"
      />
    </div>
  );
}

export const AgentNode = memo(AgentNodeComponent);
