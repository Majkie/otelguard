import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useSearchParams, useNavigate } from "react-router-dom";
import { ArrowLeft, X, Save, Clock, User, CheckCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Slider } from "@/components/ui/slider";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";

import { annotationsApi } from "@/api/annotations";
import { useAuth } from "@/hooks/use-auth";

interface AnnotationQueue {
  id: string;
  name: string;
  description: string;
  instructions: string;
  scoreConfigs: any[];
}

interface AnnotationAssignment {
  id: string;
  queueItemId: string;
  userId: string;
  status: string;
  assignedAt: string;
  startedAt?: string;
  completedAt?: string;
  skippedAt?: string;
}

interface QueueItem {
  id: string;
  queueId: string;
  itemType: string;
  itemId: string;
  itemData: any;
  metadata: any;
}

interface AnnotationData {
  scores: Record<string, any>;
  labels: string[];
  notes: string;
  confidenceScore?: number;
}

export default function AnnotatePage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const queueId = searchParams.get("queueId");
  const [currentAssignment, setCurrentAssignment] = useState<AnnotationAssignment | null>(null);
  const [queueItem, setQueueItem] = useState<QueueItem | null>(null);
  const [annotationData, setAnnotationData] = useState<AnnotationData>({
    scores: {},
    labels: [],
    notes: "",
  });
  const [startTime, setStartTime] = useState<Date | null>(null);

  // Fetch queue details
  const { data: queue } = useQuery({
    queryKey: ["annotation-queue", queueId],
    queryFn: async () => {
      return await annotationsApi.queues.get(queueId!);
    },
    enabled: !!queueId,
  });

  // Get next assignment
  const assignNextItemMutation = useMutation({
    mutationFn: async () => {
      return await annotationsApi.assignments.assignNext(queueId!);
    },
    onSuccess: (assignment) => {
      setCurrentAssignment(assignment);
      setStartTime(new Date());

      // Mark assignment as started
      startAssignmentMutation.mutate(assignment.id);
    },
    onError: (error: any) => {
      toast({
        title: "No items available",
        description: "There are no more items to annotate in this queue.",
        variant: "destructive",
      });
      navigate("/annotations");
    },
  });

  // Start assignment
  const startAssignmentMutation = useMutation({
    mutationFn: async (assignmentId: string) => {
      await annotationsApi.assignments.start(assignmentId);
    },
  });

  // Submit annotation
  const submitAnnotationMutation = useMutation({
    mutationFn: async (data: { assignmentId: string; annotation: AnnotationData }) => {
      return await annotationsApi.annotations.create({
        assignmentId: data.assignmentId,
        scores: data.annotation.scores,
        labels: data.annotation.labels,
        notes: data.annotation.notes,
        confidenceScore: data.annotation.confidenceScore,
        annotationTime: startTime ? `${Date.now() - startTime.getTime()}ms` : undefined,
      });
    },
    onSuccess: () => {
      toast({
        title: "Annotation submitted",
        description: "Your annotation has been saved successfully.",
      });

      // Get next item
      setTimeout(() => {
        assignNextItemMutation.mutate();
      }, 1500);
    },
    onError: (error: any) => {
      toast({
        title: "Error",
        description: error.response?.data?.message || "Failed to submit annotation",
        variant: "destructive",
      });
    },
  });

  // Skip assignment
  const skipAssignmentMutation = useMutation({
    mutationFn: async (assignmentId: string) => {
      await annotationsApi.assignments.skip(assignmentId, {
        notes: "Skipped by user",
      });
    },
    onSuccess: () => {
      toast({
        title: "Item skipped",
        description: "The item has been skipped.",
      });

      // Get next item
      setTimeout(() => {
        assignNextItemMutation.mutate();
      }, 1000);
    },
  });

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      // Skip with 's' key
      if (e.key === 's' && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        if (currentAssignment && !submitAnnotationMutation.isPending) {
          skipAssignmentMutation.mutate(currentAssignment.id);
        }
      }

      // Submit with Ctrl/Cmd + Enter
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        if (currentAssignment && !submitAnnotationMutation.isPending) {
          handleSubmitAnnotation();
        }
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => window.removeEventListener('keydown', handleKeyPress);
  }, [currentAssignment, annotationData, submitAnnotationMutation.isPending]);

  const handleSubmitAnnotation = () => {
    if (!currentAssignment) return;

    // Basic validation
    const hasValidScores = queue?.scoreConfigs?.some(config => {
      const score = annotationData.scores[config.name];
      return score !== undefined && score !== null;
    });

    if (!hasValidScores) {
      toast({
        title: "Validation Error",
        description: "Please provide at least one score before submitting.",
        variant: "destructive",
      });
      return;
    }

    submitAnnotationMutation.mutate({
      assignmentId: currentAssignment.id,
      annotation: annotationData,
    });
  };

  const handleSkip = () => {
    if (!currentAssignment) return;
    skipAssignmentMutation.mutate(currentAssignment.id);
  };

  const updateScore = (configName: string, value: any) => {
    setAnnotationData(prev => ({
      ...prev,
      scores: {
        ...prev.scores,
        [configName]: value,
      },
    }));
  };

  const toggleLabel = (label: string) => {
    setAnnotationData(prev => ({
      ...prev,
      labels: prev.labels.includes(label)
        ? prev.labels.filter(l => l !== label)
        : [...prev.labels, label],
    }));
  };

  // Initialize by getting first assignment
  useEffect(() => {
    if (queueId && !currentAssignment && !assignNextItemMutation.isPending) {
      assignNextItemMutation.mutate();
    }
  }, [queueId, currentAssignment, assignNextItemMutation]);

  if (!queueId) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">No queue selected</div>
      </div>
    );
  }

  if (assignNextItemMutation.isPending) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading next item to annotate...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <Button
                variant="ghost"
                onClick={() => navigate("/annotations")}
              >
                <ArrowLeft className="h-4 w-4 mr-2" />
                Back to Queues
              </Button>
              <div>
                <h1 className="text-xl font-semibold">{queue?.name || "Annotation Queue"}</h1>
                <p className="text-sm text-muted-foreground">
                  Annotating item {currentAssignment ? "in progress" : "loading..."}
                </p>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <div className="flex items-center text-sm text-muted-foreground">
                <User className="h-4 w-4 mr-1" />
                {user?.name}
              </div>
              <Badge variant="outline">
                <Clock className="h-3 w-3 mr-1" />
                {startTime ? `${Math.floor((Date.now() - startTime.getTime()) / 1000)}s` : "0s"}
              </Badge>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Instructions Panel */}
          <div className="lg:col-span-1">
            <Card>
              <CardHeader>
                <CardTitle className="text-lg">Instructions</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="prose prose-sm max-w-none">
                  {queue?.instructions ? (
                    <div dangerouslySetInnerHTML={{ __html: queue.instructions.replace(/\n/g, '<br>') }} />
                  ) : (
                    <p className="text-muted-foreground">
                      Please carefully review the content and provide accurate annotations based on the scoring criteria below.
                    </p>
                  )}
                </div>

                <div className="mt-6 space-y-4">
                  <h4 className="font-semibold">Keyboard Shortcuts</h4>
                  <div className="text-sm space-y-1">
                    <div><kbd className="px-1 py-0.5 bg-gray-100 rounded text-xs">S</kbd> Skip item</div>
                    <div><kbd className="px-1 py-0.5 bg-gray-100 rounded text-xs">Ctrl</kbd> + <kbd className="px-1 py-0.5 bg-gray-100 rounded text-xs">Enter</kbd> Submit annotation</div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Main Content */}
          <div className="lg:col-span-2 space-y-6">
            {/* Item Content */}
            <Card>
              <CardHeader>
                <CardTitle>Content to Annotate</CardTitle>
                <CardDescription>
                  Review this content and provide your annotation below
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="bg-gray-50 p-4 rounded-lg border">
                  {/* This would be replaced with actual content rendering based on itemType */}
                  <div className="text-sm text-muted-foreground mb-2">
                    Item Type: {queueItem?.itemType || "Loading..."}
                  </div>
                  <div className="prose prose-sm max-w-none">
                    {/* Placeholder content - would be dynamically rendered based on item data */}
                    <p>This is sample content to annotate. In a real implementation, this would display the actual trace, span, prompt, or other content based on the item type and data.</p>
                    <pre className="bg-white p-3 rounded border text-xs overflow-x-auto">
{JSON.stringify(queueItem?.itemData || {}, null, 2)}
                    </pre>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Annotation Form */}
            <Card>
              <CardHeader>
                <CardTitle>Your Annotation</CardTitle>
                <CardDescription>
                  Provide scores and feedback for this content
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {/* Score Inputs */}
                {queue?.scoreConfigs?.map((config) => (
                  <div key={config.name} className="space-y-2">
                    <Label className="text-base font-medium">
                      {config.name}
                      {config.description && (
                        <span className="text-sm text-muted-foreground ml-2">
                          ({config.description})
                        </span>
                      )}
                    </Label>

                    {config.dataType === 'numeric' && (
                      <div className="space-y-2">
                        <Slider
                          value={[annotationData.scores[config.name] || 0]}
                          onValueChange={([value]) => updateScore(config.name, value)}
                          min={config.minValue || 0}
                          max={config.maxValue || 10}
                          step={0.1}
                          className="w-full"
                        />
                        <div className="flex justify-between text-sm text-muted-foreground">
                          <span>{config.minValue || 0}</span>
                          <span className="font-medium">
                            {annotationData.scores[config.name]?.toFixed(1) || '0.0'}
                          </span>
                          <span>{config.maxValue || 10}</span>
                        </div>
                      </div>
                    )}

                    {config.dataType === 'categorical' && config.categories && (
                      <Select
                        value={annotationData.scores[config.name] || ""}
                        onValueChange={(value) => updateScore(config.name, value)}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select a category" />
                        </SelectTrigger>
                        <SelectContent>
                          {config.categories.map((category: string) => (
                            <SelectItem key={category} value={category}>
                              {category}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                  </div>
                ))}

                {/* Labels */}
                <div className="space-y-2">
                  <Label className="text-base font-medium">Labels</Label>
                  <div className="flex flex-wrap gap-2">
                    {['positive', 'negative', 'neutral', 'helpful', 'unhelpful', 'accurate', 'inaccurate'].map((label) => (
                      <Badge
                        key={label}
                        variant={annotationData.labels.includes(label) ? "default" : "outline"}
                        className="cursor-pointer"
                        onClick={() => toggleLabel(label)}
                      >
                        {label}
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* Confidence Score */}
                <div className="space-y-2">
                  <Label className="text-base font-medium">Confidence Score</Label>
                  <Slider
                    value={[annotationData.confidenceScore || 0.5]}
                    onValueChange={([value]) => setAnnotationData(prev => ({ ...prev, confidenceScore: value }))}
                    min={0}
                    max={1}
                    step={0.1}
                    className="w-full"
                  />
                  <div className="flex justify-between text-sm text-muted-foreground">
                    <span>Not confident</span>
                    <span className="font-medium">
                      {(annotationData.confidenceScore || 0.5).toFixed(1)}
                    </span>
                    <span>Very confident</span>
                  </div>
                </div>

                {/* Notes */}
                <div className="space-y-2">
                  <Label htmlFor="notes">Additional Notes (Optional)</Label>
                  <Textarea
                    id="notes"
                    value={annotationData.notes}
                    onChange={(e) => setAnnotationData(prev => ({ ...prev, notes: e.target.value }))}
                    placeholder="Any additional comments or observations..."
                    rows={3}
                  />
                </div>

                {/* Action Buttons */}
                <div className="flex justify-between pt-4 border-t">
                  <Button
                    variant="outline"
                    onClick={handleSkip}
                    disabled={submitAnnotationMutation.isPending || skipAssignmentMutation.isPending}
                  >
                    <X className="h-4 w-4 mr-2" />
                    Skip (S)
                  </Button>

                  <Button
                    onClick={handleSubmitAnnotation}
                    disabled={submitAnnotationMutation.isPending || skipAssignmentMutation.isPending}
                  >
                    {submitAnnotationMutation.isPending ? (
                      <>
                        <CheckCircle className="h-4 w-4 mr-2 animate-spin" />
                        Submitting...
                      </>
                    ) : (
                      <>
                        <Save className="h-4 w-4 mr-2" />
                        Submit (Ctrl+Enter)
                      </>
                    )}
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  );
}
