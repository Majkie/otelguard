import { useState, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  ArrowLeft,
  Network,
  Clock,
  Coins,
  MessageSquare,
  Bot,
  AlertTriangle,
  Download,
  GanttChart,
  Route,
} from 'lucide-react';

import { useAgentGraph, useTraceAgents, useAgentMessages } from '@/api/agents';
import { useTrace } from '@/api/traces';
import { AgentGraph, AgentTimeline } from '@/components/features/agents';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import type { GraphNode } from '@/types/agent';

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

export function AgentGraphDetailPage() {
  const { traceId } = useParams<{ traceId: string }>();
  const navigate = useNavigate();

  // Synchronized selection state between graph and timeline
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [highlightCriticalPath, setHighlightCriticalPath] = useState(false);
  const [highlightBottlenecks, setHighlightBottlenecks] = useState(false);

  // Fetch data
  const { data: trace, isLoading: traceLoading } = useTrace(traceId || '');
  const { data: graphData, isLoading: graphLoading, error: graphError } = useAgentGraph(traceId || '');
  const { data: agentsData, isLoading: agentsLoading } = useTraceAgents(traceId || '');
  const { data: messagesData } = useAgentMessages(traceId || '');

  const isLoading = traceLoading || graphLoading || agentsLoading;

  // Handle node selection from graph - syncs to timeline
  const handleGraphNodeClick = useCallback((node: GraphNode) => {
    setSelectedNodeId(node.id === selectedNodeId ? null : node.id);
  }, [selectedNodeId]);

  // Handle node selection from timeline - syncs to graph
  const handleTimelineNodeSelect = useCallback((nodeId: string | null) => {
    setSelectedNodeId(nodeId);
  }, []);

  const handleExportGraph = () => {
    if (!graphData) return;
    const dataStr = JSON.stringify(graphData, null, 2);
    const blob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `agent-graph-${traceId}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (isLoading) {
    return (
      <div className="flex flex-col" style={{ height: 'calc(100vh - 8rem)' }}>
        <div className="border-b p-4">
          <div className="flex items-center gap-4">
            <Skeleton className="h-8 w-8" />
            <div className="space-y-2">
              <Skeleton className="h-6 w-64" />
              <Skeleton className="h-4 w-48" />
            </div>
          </div>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center space-y-4">
            <div className="animate-pulse">
              <Network className="h-16 w-16 mx-auto text-muted-foreground" />
            </div>
            <p className="text-muted-foreground">Loading agent graph...</p>
          </div>
        </div>
      </div>
    );
  }

  if (graphError || !graphData) {
    return (
      <div className="flex flex-col" style={{ height: 'calc(100vh - 8rem)' }}>
        <div className="border-b p-4">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => navigate('/agents')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-xl font-semibold">{trace?.name || 'Agent Graph'}</h1>
              <p className="text-sm text-muted-foreground">Trace: {traceId}</p>
            </div>
          </div>
        </div>
        <div className="flex-1 flex items-center justify-center">
          <div className="text-center space-y-4">
            <AlertTriangle className="h-16 w-16 mx-auto text-amber-500" />
            <p className="text-lg font-medium">No agent graph available</p>
            <p className="text-muted-foreground">
              This trace may not have agent data or the graph could not be generated.
            </p>
            <Button onClick={() => navigate('/agents')}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Agent Graphs
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col" style={{ height: 'calc(100vh - 8rem)' }}>
      {/* Header */}
      <div className="bg-background shrink-0">
        <div className="flex items-center justify-between p-4">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => navigate('/agents')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-xl font-semibold flex items-center gap-2">
                <Network className="h-5 w-5" />
                {trace?.name || 'Agent Graph'}
              </h1>
              <div className="flex items-center gap-3 text-sm text-muted-foreground">
                <span>Trace: {traceId?.slice(0, 8)}...</span>
                <span>•</span>
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  {formatLatency(graphData.metadata.totalLatencyMs)}
                </span>
                <span>•</span>
                <span className="flex items-center gap-1">
                  <Bot className="h-3 w-3" />
                  {agentsData?.data?.length || 0} {agentsData?.data?.length === 1 ? 'agent' : 'agents'}
                </span>
                {trace?.cost != null && trace.cost > 0 && (
                  <>
                    <span>•</span>
                    <span className="flex items-center gap-1">
                      <Coins className="h-3 w-3" />
                      {formatCost(trace.cost)}
                    </span>
                  </>
                )}
                {graphData.metadata.bottlenecks && graphData.metadata.bottlenecks.length > 0 && (
                  <>
                    <span>•</span>
                    <Badge variant="outline" className="bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-950 dark:text-orange-400 dark:border-orange-800">
                      <AlertTriangle className="h-3 w-3 mr-1" />
                      {graphData.metadata.bottlenecks.length} bottleneck{graphData.metadata.bottlenecks.length === 1 ? '' : 's'}
                    </Badge>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={handleExportGraph}>
              <Download className="mr-2 h-4 w-4" />
              Export
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link to={`/traces/${traceId}`}>
                View Trace
              </Link>
            </Button>
          </div>
        </div>

        {/* Tabs */}
        <Tabs defaultValue="graph" className="flex-1">
          <div className="px-4 pt-2 flex items-center justify-between">
            <TabsList className="h-10">
              <TabsTrigger value="graph" className="gap-2">
                <Network className="h-4 w-4" />
                Graph
              </TabsTrigger>
              <TabsTrigger value="timeline" className="gap-2">
                <GanttChart className="h-4 w-4" />
                Timeline
              </TabsTrigger>
              <TabsTrigger value="agents" className="gap-2">
                <Bot className="h-4 w-4" />
                Agents ({agentsData?.data?.length || 0})
              </TabsTrigger>
              <TabsTrigger value="messages" className="gap-2">
                <MessageSquare className="h-4 w-4" />
                Messages ({messagesData?.data?.length || 0})
              </TabsTrigger>
            </TabsList>

            {/* Highlight controls */}
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Switch
                  id="critical-path"
                  checked={highlightCriticalPath}
                  onCheckedChange={setHighlightCriticalPath}
                />
                <Label htmlFor="critical-path" className="text-xs flex items-center gap-1 cursor-pointer">
                  <Route className="h-3 w-3" />
                  Critical Path
                </Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="bottlenecks"
                  checked={highlightBottlenecks}
                  onCheckedChange={setHighlightBottlenecks}
                />
                <Label htmlFor="bottlenecks" className="text-xs flex items-center gap-1 cursor-pointer">
                  <AlertTriangle className="h-3 w-3" />
                  Bottlenecks
                </Label>
              </div>
            </div>
          </div>

          <TabsContent value="graph" className="mt-0 p-4" style={{ height: 'calc(100vh - 16rem)' }}>
            <AgentGraph
              graph={graphData}
              className="h-full"
              showMinimap
              showControls
              onNodeClick={handleGraphNodeClick}
            />
          </TabsContent>

          <TabsContent value="timeline" className="mt-0 p-0" style={{ height: 'calc(100vh - 16rem)' }}>
            <AgentTimeline
              graph={graphData}
              className="h-full"
              selectedNodeId={selectedNodeId}
              onSelectNode={handleTimelineNodeSelect}
              highlightCriticalPath={highlightCriticalPath}
              highlightBottlenecks={highlightBottlenecks}
            />
          </TabsContent>

          <TabsContent value="agents" className="mt-0 p-0" style={{ height: 'calc(100vh - 16rem)' }}>
            <ScrollArea className="h-full">
              <div className="p-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {agentsData?.data?.map((agent) => (
                  <Card key={agent.id} className="cursor-pointer hover:shadow-md transition-shadow">
                    <CardHeader className="pb-2">
                      <div className="flex items-center justify-between">
                        <CardTitle className="text-base flex items-center gap-2">
                          <Bot className="h-4 w-4" />
                          {agent.name}
                        </CardTitle>
                        <Badge
                          className={cn(
                            'capitalize text-xs',
                            agent.status === 'success' && 'bg-green-100 text-green-800',
                            agent.status === 'error' && 'bg-red-100 text-red-800',
                            agent.status === 'running' && 'bg-blue-100 text-blue-800',
                            agent.status === 'timeout' && 'bg-amber-100 text-amber-800'
                          )}
                        >
                          {agent.status}
                        </Badge>
                      </div>
                      <CardDescription className="capitalize">
                        {agent.agentType.replace('_', ' ')}
                        {agent.role && ` - ${agent.role}`}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <Clock className="h-3 w-3" />
                          {formatLatency(agent.latencyMs)}
                        </span>
                        {agent.totalTokens > 0 && (
                          <span className="flex items-center gap-1">
                            <MessageSquare className="h-3 w-3" />
                            {agent.totalTokens.toLocaleString()}
                          </span>
                        )}
                        {agent.cost > 0 && (
                          <span className="flex items-center gap-1">
                            <Coins className="h-3 w-3" />
                            {formatCost(agent.cost)}
                          </span>
                        )}
                      </div>
                      {agent.model && (
                        <div className="mt-2">
                          <Badge variant="outline" className="text-xs">
                            {agent.model}
                          </Badge>
                        </div>
                      )}
                    </CardContent>
                  </Card>
                ))}
                {(!agentsData?.data || agentsData.data.length === 0) && (
                  <div className="col-span-full text-center py-8 text-muted-foreground">
                    <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>No agents detected in this trace</p>
                  </div>
                )}
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="messages" className="mt-0 p-0" style={{ height: 'calc(100vh - 16rem)' }}>
            <ScrollArea className="h-full">
              <div className="p-4 space-y-3">
                {messagesData?.data?.map((msg) => (
                  <div
                    key={msg.id}
                    className={cn(
                      'p-3 rounded-lg border',
                      msg.role === 'user' && 'bg-blue-50 dark:bg-blue-950 border-blue-200',
                      msg.role === 'assistant' && 'bg-green-50 dark:bg-green-950 border-green-200',
                      msg.role === 'system' && 'bg-gray-50 dark:bg-gray-900 border-gray-200',
                      msg.role === 'tool' && 'bg-amber-50 dark:bg-amber-950 border-amber-200'
                    )}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="capitalize text-xs">
                          {msg.role}
                        </Badge>
                        <Badge variant="secondary" className="text-xs">
                          {msg.messageType}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          #{msg.sequenceNum}
                        </span>
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {new Date(msg.timestamp).toLocaleTimeString()}
                      </span>
                    </div>
                    <p className="text-sm whitespace-pre-wrap line-clamp-3">
                      {msg.content}
                    </p>
                    {msg.tokenCount > 0 && (
                      <div className="mt-2 text-xs text-muted-foreground">
                        {msg.tokenCount} tokens
                      </div>
                    )}
                  </div>
                ))}
                {(!messagesData?.data || messagesData.data.length === 0) && (
                  <div className="text-center py-8 text-muted-foreground">
                    <MessageSquare className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>No agent messages in this trace</p>
                  </div>
                )}
              </div>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}

export default AgentGraphDetailPage;
