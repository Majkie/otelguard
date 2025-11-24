import { useState, useMemo, useCallback, useRef, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Slider } from '@/components/ui/slider';
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
  Clock,
  Coins,
  Hash,
  Route,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import {
  buildTimelineData,
  getItemsByLane,
  generateTimeMarkers,
  getTimelineItemColor,
  getStatusStyle,
  formatTimelineTime,
  type TimelineItem,
  type TimelineData,
} from '@/lib/timeline';
import type { AgentGraph, Bottleneck } from '@/types/agent';

interface AgentTimelineProps {
  graph: AgentGraph;
  className?: string;
  selectedNodeId?: string | null;
  onSelectNode?: (nodeId: string | null) => void;
  criticalPath?: string[];
  bottlenecks?: Bottleneck[];
  highlightCriticalPath?: boolean;
  highlightBottlenecks?: boolean;
}

const LANE_HEIGHT = 48;
const RULER_HEIGHT = 32;
const MIN_ZOOM = 1;
const MAX_ZOOM = 10;

export function AgentTimeline({
  graph,
  className,
  selectedNodeId,
  onSelectNode,
  criticalPath,
  bottlenecks,
  highlightCriticalPath = false,
  highlightBottlenecks = false,
}: AgentTimelineProps) {
  const [zoom, setZoom] = useState(1);
  const [panOffset, setPanOffset] = useState(0);
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const timelineRef = useRef<HTMLDivElement>(null);

  // Build timeline data from graph
  const timelineData: TimelineData = useMemo(
    () => buildTimelineData(graph.nodes, graph.metadata.executionLanes),
    [graph.nodes, graph.metadata.executionLanes]
  );

  // Group items by lane
  const itemsByLane = useMemo(
    () => getItemsByLane(timelineData.items),
    [timelineData.items]
  );

  // Generate time markers
  const timeMarkers = useMemo(
    () => generateTimeMarkers(timelineData.totalDurationMs, 10),
    [timelineData.totalDurationMs]
  );

  // Critical path and bottleneck sets for quick lookup
  const criticalPathSet = useMemo(
    () => new Set(criticalPath || graph.metadata.criticalPath || []),
    [criticalPath, graph.metadata.criticalPath]
  );

  const bottleneckMap = useMemo(() => {
    const map = new Map<string, Bottleneck>();
    (bottlenecks || graph.metadata.bottlenecks || []).forEach((b) => {
      map.set(b.nodeId, b);
    });
    return map;
  }, [bottlenecks, graph.metadata.bottlenecks]);

  // Zoom controls
  const handleZoomIn = useCallback(() => {
    setZoom((prev) => Math.min(MAX_ZOOM, prev + 0.5));
  }, []);

  const handleZoomOut = useCallback(() => {
    setZoom((prev) => Math.max(MIN_ZOOM, prev - 0.5));
  }, []);

  const handleResetZoom = useCallback(() => {
    setZoom(1);
    setPanOffset(0);
  }, []);

  // Pan controls
  const handlePanLeft = useCallback(() => {
    setPanOffset((prev) => Math.max(0, prev - 10));
  }, []);

  const handlePanRight = useCallback(() => {
    const maxPan = (zoom - 1) * 100;
    setPanOffset((prev) => Math.min(maxPan, prev + 10));
  }, [zoom]);

  // Handle scroll/wheel for panning
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleWheel = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) {
        // Zoom with ctrl/cmd + scroll
        e.preventDefault();
        const delta = e.deltaY > 0 ? -0.2 : 0.2;
        setZoom((prev) => Math.max(MIN_ZOOM, Math.min(MAX_ZOOM, prev + delta)));
      } else if (zoom > 1) {
        // Pan with regular scroll when zoomed in
        const maxPan = (zoom - 1) * 100;
        setPanOffset((prev) =>
          Math.max(0, Math.min(maxPan, prev + e.deltaX * 0.1))
        );
      }
    };

    container.addEventListener('wheel', handleWheel, { passive: false });
    return () => container.removeEventListener('wheel', handleWheel);
  }, [zoom]);

  // Scroll selected item into view
  useEffect(() => {
    if (selectedNodeId && timelineRef.current) {
      const item = timelineData.items.find((i) => i.nodeId === selectedNodeId);
      if (item) {
        // Calculate if we need to adjust pan to show selected item
        const itemStartPercent = item.startPercent / zoom + panOffset / zoom;
        const visibleStart = panOffset;
        const visibleEnd = panOffset + 100 / zoom;

        if (itemStartPercent < visibleStart) {
          setPanOffset(Math.max(0, item.startPercent - 5));
        } else if (itemStartPercent > visibleEnd) {
          const maxPan = (zoom - 1) * 100;
          setPanOffset(Math.min(maxPan, item.startPercent - 50));
        }
      }
    }
  }, [selectedNodeId, timelineData.items, zoom, panOffset]);

  const handleItemClick = useCallback(
    (item: TimelineItem) => {
      onSelectNode?.(item.nodeId === selectedNodeId ? null : item.nodeId);
    },
    [onSelectNode, selectedNodeId]
  );

  // Render a single timeline item
  const renderTimelineItem = (item: TimelineItem) => {
    const colors = getTimelineItemColor(item.type);
    const isSelected = selectedNodeId === item.nodeId;
    const isHovered = hoveredItem === item.id;
    const isOnCriticalPath = highlightCriticalPath && criticalPathSet.has(item.nodeId);
    const bottleneckInfo = highlightBottlenecks ? bottleneckMap.get(item.nodeId) : null;

    // Calculate position with zoom and pan
    const left = (item.startPercent * zoom) - panOffset;
    const width = item.widthPercent * zoom;

    // Skip items outside visible area
    if (left + width < 0 || left > 100) return null;

    return (
      <TooltipProvider key={item.id}>
        <Tooltip delayDuration={200}>
          <TooltipTrigger asChild>
            <button
              className={cn(
                'absolute h-8 rounded cursor-pointer transition-all border',
                'flex items-center justify-start px-2 overflow-hidden',
                colors.bg,
                colors.border,
                colors.text,
                isSelected && 'ring-2 ring-primary ring-offset-2',
                isHovered && !isSelected && 'brightness-110 shadow-lg',
                isOnCriticalPath && 'ring-2 ring-yellow-400',
                bottleneckInfo && 'ring-2 ring-orange-500',
                getStatusStyle(item.status)
              )}
              style={{
                left: `${Math.max(0, left)}%`,
                width: `${Math.min(100 - left, width)}%`,
                minWidth: '24px',
                top: '8px',
              }}
              onClick={() => handleItemClick(item)}
              onMouseEnter={() => setHoveredItem(item.id)}
              onMouseLeave={() => setHoveredItem(null)}
            >
              <span className="text-xs font-medium truncate">{item.label}</span>
            </button>
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-xs">
            <div className="space-y-1.5">
              <div className="font-medium">{item.label}</div>
              <div className="flex items-center gap-2 text-xs">
                <Badge variant="outline" className="capitalize">
                  {item.type}
                </Badge>
                <Badge
                  variant={item.status === 'success' ? 'default' : 'destructive'}
                  className="capitalize"
                >
                  {item.status}
                </Badge>
              </div>
              <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                <div className="flex items-center gap-1 text-muted-foreground">
                  <Clock className="h-3 w-3" />
                  <span>{formatTimelineTime(item.latencyMs)}</span>
                </div>
                {item.tokens && item.tokens > 0 && (
                  <div className="flex items-center gap-1 text-muted-foreground">
                    <Hash className="h-3 w-3" />
                    <span>{item.tokens.toLocaleString()}</span>
                  </div>
                )}
                {item.cost && item.cost > 0 && (
                  <div className="flex items-center gap-1 text-muted-foreground">
                    <Coins className="h-3 w-3" />
                    <span>${item.cost.toFixed(4)}</span>
                  </div>
                )}
                {item.model && (
                  <div className="text-muted-foreground col-span-2 truncate">
                    Model: {item.model}
                  </div>
                )}
              </div>
              {isOnCriticalPath && (
                <div className="flex items-center gap-1 text-xs text-yellow-600">
                  <Route className="h-3 w-3" />
                  <span>On critical path</span>
                </div>
              )}
              {bottleneckInfo && (
                <div className="flex items-center gap-1 text-xs text-orange-600">
                  <AlertTriangle className="h-3 w-3" />
                  <span>
                    Bottleneck ({bottleneckInfo.percentage.toFixed(1)}%)
                  </span>
                </div>
              )}
            </div>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  };

  // Get lanes to display
  const laneIds = Array.from(itemsByLane.keys()).sort((a, b) => a - b);

  if (timelineData.items.length === 0) {
    return (
      <div className={cn('flex items-center justify-center h-64', className)}>
        <div className="text-center text-muted-foreground">
          <Clock className="h-12 w-12 mx-auto mb-2 opacity-50" />
          <p>No timeline data available</p>
        </div>
      </div>
    );
  }

  return (
    <div className={cn('flex flex-col', className)}>
      {/* Controls */}
      <div className="flex items-center justify-between p-2 border-b bg-muted/30">
        <div className="flex items-center gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handleZoomOut}
                  disabled={zoom <= MIN_ZOOM}
                >
                  <ZoomOut className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Zoom Out</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <div className="w-24">
            <Slider
              value={[zoom]}
              min={MIN_ZOOM}
              max={MAX_ZOOM}
              step={0.1}
              onValueChange={([value]) => setZoom(value)}
            />
          </div>

          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handleZoomIn}
                  disabled={zoom >= MAX_ZOOM}
                >
                  <ZoomIn className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Zoom In</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" onClick={handleResetZoom}>
                  <Maximize2 className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Reset View</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <div className="h-4 border-l mx-2" />

          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handlePanLeft}
                  disabled={zoom <= 1 || panOffset <= 0}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Pan Left</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={handlePanRight}
                  disabled={zoom <= 1 || panOffset >= (zoom - 1) * 100}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Pan Right</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <span className="text-xs text-muted-foreground ml-2">
            {zoom.toFixed(1)}x
          </span>
        </div>

        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span>
            Duration: <strong>{formatTimelineTime(timelineData.totalDurationMs)}</strong>
          </span>
          <span>
            Lanes: <strong>{timelineData.maxLanes}</strong>
          </span>
          <span>
            Items: <strong>{timelineData.items.length}</strong>
          </span>
        </div>
      </div>

      {/* Timeline container */}
      <div
        ref={containerRef}
        className="flex-1 overflow-hidden bg-background"
      >
        {/* Time ruler */}
        <div
          className="sticky top-0 z-10 bg-muted/50 border-b"
          style={{ height: RULER_HEIGHT }}
        >
          <div className="relative h-full" ref={timelineRef}>
            {timeMarkers.map((marker, idx) => {
              const left = (marker.percent * zoom) - panOffset;
              if (left < 0 || left > 100) return null;
              return (
                <div
                  key={idx}
                  className="absolute top-0 h-full flex flex-col items-center"
                  style={{ left: `${left}%` }}
                >
                  <div className="h-2 w-px bg-border" />
                  <span className="text-[10px] text-muted-foreground mt-0.5">
                    {marker.label}
                  </span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Lanes */}
        <div className="relative">
          {/* Grid lines */}
          <div className="absolute inset-0 pointer-events-none">
            {timeMarkers.map((marker, idx) => {
              const left = (marker.percent * zoom) - panOffset;
              if (left < 0 || left > 100) return null;
              return (
                <div
                  key={idx}
                  className="absolute top-0 bottom-0 w-px bg-border/30"
                  style={{ left: `${left}%` }}
                />
              );
            })}
          </div>

          {/* Lane rows */}
          {laneIds.map((laneId) => {
            const laneItems = itemsByLane.get(laneId) || [];
            return (
              <div
                key={laneId}
                className="relative border-b hover:bg-muted/20 transition-colors"
                style={{ height: LANE_HEIGHT }}
              >
                {/* Lane label */}
                <div className="absolute left-0 top-0 bottom-0 w-16 flex items-center justify-center bg-muted/30 border-r z-10">
                  <span className="text-xs text-muted-foreground font-medium">
                    Lane {laneId}
                  </span>
                </div>

                {/* Lane content */}
                <div className="ml-16 relative h-full overflow-hidden">
                  {laneItems.map((item) => renderTimelineItem(item))}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 p-2 border-t text-xs text-muted-foreground bg-muted/30">
        <span className="font-medium">Types:</span>
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
        {highlightCriticalPath && criticalPathSet.size > 0 && (
          <>
            <div className="h-4 border-l mx-1" />
            <div className="flex items-center gap-1">
              <Route className="h-3 w-3 text-yellow-500" />
              <span>Critical Path</span>
            </div>
          </>
        )}
        {highlightBottlenecks && bottleneckMap.size > 0 && (
          <div className="flex items-center gap-1">
            <AlertTriangle className="h-3 w-3 text-orange-500" />
            <span>Bottleneck</span>
          </div>
        )}
      </div>
    </div>
  );
}

export default AgentTimeline;
