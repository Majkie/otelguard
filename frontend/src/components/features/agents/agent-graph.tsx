import { useCallback, useMemo, useState, useEffect } from 'react';
import dagre from 'dagre';
import {
  ReactFlow,
  MiniMap,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
  Panel,
  useReactFlow,
  MarkerType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  ZoomIn,
  ZoomOut,
  Maximize2,
  Route,
  AlertTriangle,
  Layers,
} from 'lucide-react';

import { AgentNode, type AgentNodeData } from './agent-node';
import { AgentEdge, type AgentEdgeData } from './agent-edge';
import { NodeDetailPanel } from './node-detail-panel';
import type { AgentGraph as AgentGraphType, GraphNode, GraphEdge } from '@/types/agent';

const nodeTypes = {
  agentNode: AgentNode,
};

const edgeTypes = {
  agentEdge: AgentEdge,
};

// Auto-layout algorithm using dagre
function calculateLayout(graphNodes: GraphNode[], graphEdges: GraphEdge[]) {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));

  // Node dimensions (must match or exceed the actual rendered size)
  const NODE_WIDTH = 220;
  const NODE_HEIGHT = 100;

  // Set graph layout options
  dagreGraph.setGraph({
    rankdir: 'TB',
    nodesep: 50,
    ranksep: 80
  });

  // Add nodes
  graphNodes.forEach((node) => {
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });

  // Add edges
  graphEdges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  // Calculate layout
  dagre.layout(dagreGraph);

  // Get positions (convert from center-based to top-left)
  const positions: Map<string, { x: number; y: number }> = new Map();
  graphNodes.forEach((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    positions.set(node.id, {
      x: nodeWithPosition.x - NODE_WIDTH / 2,
      y: nodeWithPosition.y - NODE_HEIGHT / 2,
    });
  });

  return positions;
}

interface AgentGraphProps {
  graph: AgentGraphType;
  className?: string;
  showMinimap?: boolean;
  showControls?: boolean;
  onNodeClick?: (node: GraphNode) => void;
}

export function AgentGraphVisualization({
  graph,
  className,
  showMinimap = true,
  onNodeClick,
}: AgentGraphProps) {
  const { fitView, zoomIn, zoomOut, setCenter } = useReactFlow();
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [highlightCriticalPath, setHighlightCriticalPath] = useState(false);
  const [highlightBottlenecks, setHighlightBottlenecks] = useState(false);
  const [showParallelGroups, setShowParallelGroups] = useState(false);

  // Calculate node positions
  const nodePositions = useMemo(
    () => calculateLayout(graph.nodes, graph.edges),
    [graph.nodes, graph.edges]
  );

  // Create React Flow nodes
  const initialNodes: Node<AgentNodeData>[] = useMemo(() => {
    const criticalPathSet = new Set(graph.metadata.criticalPath || []);
    const bottleneckSet = new Set(graph.metadata.bottlenecks?.map((b) => b.nodeId) || []);

    return graph.nodes.map((node) => {
      const position = nodePositions.get(node.id) || { x: 0, y: 0 };

      return {
        id: node.id,
        type: 'agentNode',
        position: node.position || position,
        data: {
          label: node.label,
          type: node.type,
          status: node.status,
          latencyMs: node.latencyMs,
          tokens: node.tokens,
          cost: node.cost,
          model: node.model,
          depth: node.depth,
          parallelGroup: node.parallelGroup,
          isOnCriticalPath: highlightCriticalPath && criticalPathSet.has(node.id),
          isHighlighted: selectedNodeId === node.id,
          isBottleneck: highlightBottlenecks && bottleneckSet.has(node.id),
        },
      };
    });
  }, [graph.nodes, graph.metadata, nodePositions, highlightCriticalPath, highlightBottlenecks, selectedNodeId]);

  // Create React Flow edges
  const initialEdges: Edge<AgentEdgeData>[] = useMemo(() => {
    const criticalPathSet = new Set(graph.metadata.criticalPath || []);

    return graph.edges.map((edge) => {
      const isOnCriticalPath =
        highlightCriticalPath &&
        criticalPathSet.has(edge.source) &&
        criticalPathSet.has(edge.target);

      return {
        id: edge.id,
        source: edge.source,
        target: edge.target,
        type: 'agentEdge',
        markerEnd: {
          type: MarkerType.ArrowClosed,
          width: 15,
          height: 15,
        },
        data: {
          type: edge.type,
          label: edge.label,
          latencyMs: edge.latencyMs,
          order: edge.order,
          isOnCriticalPath,
          isHighlighted: false,
        },
      };
    });
  }, [graph.edges, graph.metadata, highlightCriticalPath]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update nodes when graph changes
  useEffect(() => {
    setNodes(initialNodes);
    setEdges(initialEdges);
  }, [initialNodes, initialEdges, setNodes, setEdges]);

  // Handle node clicks directly (more reliable than selection)
  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node) => {
      setSelectedNodeId(node.id);
      const graphNode = graph.nodes.find((n) => n.id === node.id);
      if (graphNode && onNodeClick) {
        onNodeClick(graphNode);
      }
    },
    [graph.nodes, onNodeClick]
  );

  // Navigate to a node
  const navigateToNode = useCallback(
    (nodeId: string) => {
      const node = nodes.find((n) => n.id === nodeId);
      if (node) {
        setCenter(node.position.x + 100, node.position.y + 50, { zoom: 1.5, duration: 500 });
        setSelectedNodeId(nodeId);
      }
    },
    [nodes, setCenter]
  );

  // Get selected graph node
  const selectedGraphNode = useMemo(
    () => graph.nodes.find((n) => n.id === selectedNodeId) || null,
    [graph.nodes, selectedNodeId]
  );

  // Minimap colors by node type
  const minimapNodeColor = useCallback((node: Node) => {
    const typeColors: Record<string, string> = {
      agent: '#8b5cf6',
      tool: '#f59e0b',
      llm: '#3b82f6',
      retrieval: '#10b981',
      embedding: '#06b6d4',
      message: '#6b7280',
      start: '#10b981',
      end: '#ef4444',
      custom: '#64748b',
    };
    return typeColors[(node.data as AgentNodeData).type] || '#64748b';
  }, []);

  return (
    <div className={cn('flex h-full', className)}>
      <div className="flex-1 relative border border-border rounded-lg overflow-hidden">
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={handleNodeClick}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          fitView
          proOptions={{hideAttribution: true}}
          fitViewOptions={{ padding: 0.2 }}
          minZoom={0.1}
          maxZoom={2}
          defaultEdgeOptions={{
            animated: false,
          }}
          className="bg-background"
        >
          {/* Controls Panel */}
          <Panel position="top-left" className="flex flex-col gap-2">
            <div className="flex items-center gap-2 p-2 bg-background/90 border rounded-lg shadow-sm">
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => zoomIn()}
                    >
                      <ZoomIn className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Zoom In</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => zoomOut()}
                    >
                      <ZoomOut className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Zoom Out</TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => fitView({ padding: 0.2, duration: 500 })}
                    >
                      <Maximize2 className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Fit View</TooltipContent>
                </Tooltip>
              </TooltipProvider>

              <Separator orientation="vertical" className="h-6" />

              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={highlightCriticalPath ? 'secondary' : 'ghost'}
                      size="icon"
                      onClick={() => setHighlightCriticalPath(!highlightCriticalPath)}
                    >
                      <Route className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {highlightCriticalPath ? 'Hide' : 'Show'} Critical Path
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={highlightBottlenecks ? 'secondary' : 'ghost'}
                      size="icon"
                      onClick={() => setHighlightBottlenecks(!highlightBottlenecks)}
                    >
                      <AlertTriangle className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {highlightBottlenecks ? 'Hide' : 'Show'} Bottlenecks
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={showParallelGroups ? 'secondary' : 'ghost'}
                      size="icon"
                      onClick={() => setShowParallelGroups(!showParallelGroups)}
                    >
                      <Layers className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {showParallelGroups ? 'Hide' : 'Show'} Parallel Groups
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </Panel>

          {/* Stats Panel */}
          <Panel position="top-right" className="p-2 bg-background/90 border rounded-lg shadow-sm">
            <div className="flex items-center gap-3 text-sm">
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Nodes:</span>
                <Badge variant="secondary">{graph.metadata.totalNodes}</Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Edges:</span>
                <Badge variant="secondary">{graph.metadata.totalEdges}</Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Depth:</span>
                <Badge variant="secondary">{graph.metadata.maxDepth}</Badge>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-muted-foreground">Parallel:</span>
                <Badge variant="secondary">{graph.metadata.maxParallelism}</Badge>
              </div>
              {graph.metadata.hasCycles && (
                <Badge variant="destructive" className="text-xs">
                  Has Cycles
                </Badge>
              )}
            </div>
          </Panel>

          {/* Legend */}
          <Panel position="bottom-left" className="p-2 bg-background/90 border rounded-lg shadow-sm">
            <div className="text-xs space-y-1">
              <div className="font-medium mb-1">Node Types</div>
              <div className="grid grid-cols-3 gap-x-4 gap-y-1">
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-purple-500" />
                  <span>Agent</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-amber-500" />
                  <span>Tool</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-blue-500" />
                  <span>LLM</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-green-500" />
                  <span>Retrieval</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-cyan-500" />
                  <span>Embedding</span>
                </div>
                <div className="flex items-center gap-1">
                  <div className="w-3 h-3 rounded bg-slate-500" />
                  <span>Custom</span>
                </div>
              </div>
            </div>
          </Panel>

          {showMinimap && (
            <MiniMap
              nodeColor={minimapNodeColor}
              maskColor="rgba(0, 0, 0, 0.1)"
              className="!bg-background border"
              pannable
              zoomable
            />
          )}

          <Background variant={BackgroundVariant.Dots} gap={20} size={1} />
        </ReactFlow>
      </div>

      {/* Detail Panel */}
      <NodeDetailPanel
        node={selectedGraphNode}
        edges={graph.edges}
        bottlenecks={graph.metadata.bottlenecks}
        criticalPath={graph.metadata.criticalPath}
        onClose={() => setSelectedNodeId(null)}
        onNavigateToNode={navigateToNode}
      />
    </div>
  );
}

// Wrapper component with ReactFlowProvider
import { ReactFlowProvider } from '@xyflow/react';

export function AgentGraph(props: AgentGraphProps) {
  return (
    <ReactFlowProvider>
      <AgentGraphVisualization {...props} />
    </ReactFlowProvider>
  );
}
