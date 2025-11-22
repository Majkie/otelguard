import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Slider } from '@/components/ui/slider';
import {
  Play,
  Pause,
  SkipBack,
  SkipForward,
  ChevronLeft,
  ChevronRight,
  Clock,
  DollarSign,
  Zap,
} from 'lucide-react';

interface ReplayStep {
  index: number;
  trace: {
    id: string;
    name: string;
    input: string;
    output: string;
    model: string;
    status: string;
    latencyMs: number;
    totalTokens: number;
    cost: number;
    startTime: string;
  };
  timeSinceStart: number;
  deltaFromPrev: number;
  cumulativeCost: number;
  cumulativeTokens: number;
}

interface SessionReplayProps {
  sessionId: string;
  steps: ReplayStep[];
  totalDuration: number;
  totalCost: number;
  totalTokens: number;
}

export function SessionReplay({
  sessionId,
  steps,
  totalDuration,
  totalCost,
  totalTokens,
}: SessionReplayProps) {
  const [currentStep, setCurrentStep] = useState(0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [playbackSpeed, setPlaybackSpeed] = useState(1);

  const step = steps[currentStep];

  const goToStep = (index: number) => {
    if (index >= 0 && index < steps.length) {
      setCurrentStep(index);
    }
  };

  const goToPrevious = () => goToStep(currentStep - 1);
  const goToNext = () => goToStep(currentStep + 1);
  const goToStart = () => goToStep(0);
  const goToEnd = () => goToStep(steps.length - 1);

  const togglePlayback = () => {
    setIsPlaying(!isPlaying);
  };

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const formatCost = (cost: number) => {
    return `$${cost.toFixed(4)}`;
  };

  if (!step) {
    return (
      <Card>
        <CardContent className="p-6">
          <p className="text-muted-foreground">No steps to replay</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Playback Controls */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={goToStart}
                disabled={currentStep === 0}
              >
                <SkipBack className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={goToPrevious}
                disabled={currentStep === 0}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="default"
                size="icon"
                onClick={togglePlayback}
              >
                {isPlaying ? (
                  <Pause className="h-4 w-4" />
                ) : (
                  <Play className="h-4 w-4" />
                )}
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={goToNext}
                disabled={currentStep === steps.length - 1}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={goToEnd}
                disabled={currentStep === steps.length - 1}
              >
                <SkipForward className="h-4 w-4" />
              </Button>
            </div>

            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span>
                Step {currentStep + 1} of {steps.length}
              </span>
              <span>|</span>
              <span>{formatDuration(step.timeSinceStart / 1000000)}</span>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">Speed:</span>
              <select
                value={playbackSpeed}
                onChange={(e) => setPlaybackSpeed(Number(e.target.value))}
                className="border rounded px-2 py-1 text-sm"
              >
                <option value={0.5}>0.5x</option>
                <option value={1}>1x</option>
                <option value={2}>2x</option>
                <option value={4}>4x</option>
              </select>
            </div>
          </div>

          {/* Progress Bar */}
          <Slider
            value={[currentStep]}
            max={steps.length - 1}
            step={1}
            onValueChange={([value]) => goToStep(value)}
            className="w-full"
          />

          {/* Cumulative Stats */}
          <div className="flex gap-4 mt-4 text-sm">
            <div className="flex items-center gap-1">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span>{formatDuration(step.timeSinceStart / 1000000)}</span>
            </div>
            <div className="flex items-center gap-1">
              <DollarSign className="h-4 w-4 text-muted-foreground" />
              <span>{formatCost(step.cumulativeCost)}</span>
            </div>
            <div className="flex items-center gap-1">
              <Zap className="h-4 w-4 text-muted-foreground" />
              <span>{step.cumulativeTokens.toLocaleString()} tokens</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Current Step Details */}
      <Card>
        <CardHeader className="pb-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">{step.trace.name}</CardTitle>
            <div className="flex items-center gap-2">
              <Badge variant={step.trace.status === 'success' ? 'default' : 'destructive'}>
                {step.trace.status}
              </Badge>
              {step.trace.model && (
                <Badge variant="outline">{step.trace.model}</Badge>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Step Metrics */}
          <div className="grid grid-cols-3 gap-4 text-sm">
            <div>
              <span className="text-muted-foreground">Latency</span>
              <p className="font-medium">{step.trace.latencyMs}ms</p>
            </div>
            <div>
              <span className="text-muted-foreground">Tokens</span>
              <p className="font-medium">{step.trace.totalTokens.toLocaleString()}</p>
            </div>
            <div>
              <span className="text-muted-foreground">Cost</span>
              <p className="font-medium">{formatCost(step.trace.cost)}</p>
            </div>
          </div>

          {/* Input */}
          <div>
            <h4 className="text-sm font-medium mb-2">Input</h4>
            <div className="bg-muted rounded-md p-3 max-h-48 overflow-auto">
              <pre className="text-sm whitespace-pre-wrap font-mono">
                {step.trace.input || '(empty)'}
              </pre>
            </div>
          </div>

          {/* Output */}
          <div>
            <h4 className="text-sm font-medium mb-2">Output</h4>
            <div className="bg-muted rounded-md p-3 max-h-48 overflow-auto">
              <pre className="text-sm whitespace-pre-wrap font-mono">
                {step.trace.output || '(empty)'}
              </pre>
            </div>
          </div>

          {/* Time Delta */}
          {step.deltaFromPrev > 0 && (
            <div className="text-sm text-muted-foreground">
              <Clock className="inline h-3 w-3 mr-1" />
              {formatDuration(step.deltaFromPrev / 1000000)} since previous step
            </div>
          )}
        </CardContent>
      </Card>

      {/* Step List */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">All Steps</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-1 max-h-64 overflow-auto">
            {steps.map((s, i) => (
              <button
                key={s.trace.id}
                onClick={() => goToStep(i)}
                className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                  i === currentStep
                    ? 'bg-primary text-primary-foreground'
                    : 'hover:bg-muted'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium truncate">{s.trace.name}</span>
                  <span className="text-xs opacity-70">
                    {formatDuration(s.timeSinceStart / 1000000)}
                  </span>
                </div>
              </button>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
