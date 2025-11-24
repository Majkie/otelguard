import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Settings, Users, BarChart3, Play, TrendingUp, CheckCircle, Clock, Download } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useToast } from "@/hooks/use-toast";

import { annotationsApi } from "@/api/annotations";
import { useAuth } from "@/hooks/use-auth";
import { useProject } from "@/contexts/project-context";

interface AnnotationQueue {
  id: string;
  name: string;
  description: string;
  scoreConfigs: any[];
  itemSource: string;
  assignmentStrategy: string;
  maxAnnotationsPerItem: number;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface CreateQueueData {
  name: string;
  description: string;
  itemSource: string;
  assignmentStrategy: string;
  maxAnnotationsPerItem: number;
  instructions: string;
}

export default function AnnotationsPage() {
  const { user } = useAuth();
  const { currentProject } = useProject();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [createForm, setCreateForm] = useState<CreateQueueData>({
    name: "",
    description: "",
    itemSource: "manual",
    assignmentStrategy: "round_robin",
    maxAnnotationsPerItem: 1,
    instructions: "",
  });

  // Export function
  const handleExport = async (queueId: string, format: 'json' | 'csv') => {
    try {
      const blob = await annotationsApi.export.exportAnnotations(queueId, format);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `annotations-${queueId}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    } catch (error) {
      toast({
        title: "Export failed",
        description: "Failed to export annotations",
        variant: "destructive",
      });
    }
  };

  // Fetch annotation queues
  const { data: queues, isLoading } = useQuery({
    queryKey: ["annotation-queues", currentProject?.id],
    queryFn: async () => {
      if (!currentProject?.id) return [];
      return await annotationsApi.queues.listByProject(currentProject.id);
    },
    enabled: !!currentProject?.id,
  });

  // Fetch user stats
  const { data: userStats } = useQuery({
    queryKey: ["annotation-user-stats"],
    queryFn: () => annotationsApi.stats.getUserStats(),
  });

  // Fetch queue stats for each queue
  const queueStatsQueries = useQuery({
    queryKey: ["annotation-queue-stats", queues?.map(q => q.id)],
    queryFn: async () => {
      if (!queues?.length) return {};
      const stats: Record<string, any> = {};
      await Promise.all(
        queues.map(async (queue) => {
          try {
            stats[queue.id] = await annotationsApi.queues.getStats(queue.id);
          } catch (error) {
            console.error(`Failed to fetch stats for queue ${queue.id}:`, error);
            stats[queue.id] = null;
          }
        })
      );
      return stats;
    },
    enabled: !!queues?.length,
  });

  // Fetch agreement stats for the first queue (as an example)
  const agreementStatsQuery = useQuery({
    queryKey: ["annotation-agreement-stats", queues?.[0]?.id],
    queryFn: async () => {
      if (!queues?.[0]?.id) return null;
      return await annotationsApi.agreements.getQueueStats(queues[0].id);
    },
    enabled: !!queues?.length,
  });

  // Create queue mutation
  const createQueueMutation = useMutation({
    mutationFn: async (data: CreateQueueData) => {
      return await annotationsApi.queues.create(currentProject!.id, data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["annotation-queues"] });
      setIsCreateDialogOpen(false);
      setCreateForm({
        name: "",
        description: "",
        itemSource: "manual",
        assignmentStrategy: "round_robin",
        maxAnnotationsPerItem: 1,
        instructions: "",
      });
      toast({
        title: "Success",
        description: "Annotation queue created successfully",
      });
    },
    onError: (error: any) => {
      toast({
        title: "Error",
        description: error.response?.data?.message || "Failed to create annotation queue",
        variant: "destructive",
      });
    },
  });

  const handleCreateQueue = () => {
    if (!createForm.name.trim()) {
      toast({
        title: "Error",
        description: "Queue name is required",
        variant: "destructive",
      });
      return;
    }
    createQueueMutation.mutate(createForm);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-lg">Loading annotation queues...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold">Annotation Queues</h1>
          <p className="text-muted-foreground">
            Create and manage human annotation queues for quality assessment
          </p>
        </div>
        <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="h-4 w-4 mr-2" />
              Create Queue
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>Create Annotation Queue</DialogTitle>
              <DialogDescription>
                Set up a new queue for human annotation tasks.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Queue Name</Label>
                <Input
                  id="name"
                  value={createForm.name}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, name: e.target.value }))}
                  placeholder="e.g., Quality Assessment Queue"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={createForm.description}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, description: e.target.value }))}
                  placeholder="Describe what this queue is for..."
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="itemSource">Item Source</Label>
                  <Select
                    value={createForm.itemSource}
                    onValueChange={(value) => setCreateForm(prev => ({ ...prev, itemSource: value }))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="manual">Manual</SelectItem>
                      <SelectItem value="auto">Automatic</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="assignmentStrategy">Assignment Strategy</Label>
                  <Select
                    value={createForm.assignmentStrategy}
                    onValueChange={(value) => setCreateForm(prev => ({ ...prev, assignmentStrategy: value }))}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="round_robin">Round Robin</SelectItem>
                      <SelectItem value="priority">Priority</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="maxAnnotations">Max Annotations per Item</Label>
                <Input
                  id="maxAnnotations"
                  type="number"
                  min="1"
                  value={createForm.maxAnnotationsPerItem}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, maxAnnotationsPerItem: parseInt(e.target.value) || 1 }))}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="instructions">Instructions</Label>
                <Textarea
                  id="instructions"
                  value={createForm.instructions}
                  onChange={(e) => setCreateForm(prev => ({ ...prev, instructions: e.target.value }))}
                  placeholder="Instructions for annotators..."
                />
              </div>
              <div className="flex justify-end space-x-2">
                <Button
                  variant="outline"
                  onClick={() => setIsCreateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreateQueue}
                  disabled={createQueueMutation.isPending}
                >
                  {createQueueMutation.isPending ? "Creating..." : "Create Queue"}
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Progress Overview */}
      {userStats && (
        <div className="grid gap-4 md:grid-cols-3 mb-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Completed Annotations</CardTitle>
              <CheckCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{userStats.totalAnnotations || 0}</div>
              <p className="text-xs text-muted-foreground">
                {userStats.completedAssignments || 0} assignments completed
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Assignments</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{userStats.assignedAssignments || 0}</div>
              <p className="text-xs text-muted-foreground">
                Currently working on
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Success Rate</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {userStats.totalAnnotations > 0
                  ? Math.round((userStats.completedAssignments / (userStats.completedAssignments + userStats.skippedAssignments)) * 100)
                  : 0
                }%
              </div>
              <p className="text-xs text-muted-foreground">
                Completion rate
              </p>
            </CardContent>
          </Card>

          {agreementStatsQuery.data && (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Annotator Agreement</CardTitle>
                <BarChart3 className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {agreementStatsQuery.data.avgAgreement
                    ? `${(agreementStatsQuery.data.avgAgreement * 100).toFixed(1)}%`
                    : 'N/A'
                  }
                </div>
                <p className="text-xs text-muted-foreground">
                  Average agreement across {agreementStatsQuery.data.totalAgreements || 0} metrics
                </p>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {queues?.map((queue) => {
          const stats = queueStatsQueries.data?.[queue.id];
          return (
            <Card key={queue.id} className="cursor-pointer hover:shadow-md transition-shadow">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-lg">{queue.name}</CardTitle>
                  <Badge variant={queue.isActive ? "default" : "secondary"}>
                    {queue.isActive ? "Active" : "Inactive"}
                  </Badge>
                </div>
                <CardDescription>{queue.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex items-center text-sm text-muted-foreground">
                    <Settings className="h-4 w-4 mr-2" />
                    {queue.itemSource} • {queue.assignmentStrategy}
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <Users className="h-4 w-4 mr-2" />
                    Max {queue.maxAnnotationsPerItem} annotations per item
                  </div>

                  {stats && (
                    <div className="space-y-2">
                      <div className="flex justify-between text-sm">
                        <span>Items:</span>
                        <span className="font-medium">{stats.totalItems || 0}</span>
                      </div>
                      <div className="flex justify-between text-sm">
                        <span>Completed:</span>
                        <span className="font-medium">{stats.completedAssignments || 0}</span>
                      </div>
                      {stats.totalItems > 0 && (
                        <div className="w-full bg-gray-200 rounded-full h-2">
                          <div
                            className="bg-blue-600 h-2 rounded-full"
                            style={{
                              width: `${Math.min((stats.completedAssignments / stats.totalItems) * 100, 100)}%`
                            }}
                          ></div>
                        </div>
                      )}
                    </div>
                  )}

                  <div className="flex space-x-2">
                    <Button size="sm" variant="outline" className="flex-1">
                      <Settings className="h-4 w-4 mr-2" />
                      Configure
                    </Button>
                    <Button
                      size="sm"
                      className="flex-1"
                      onClick={() => window.open(`/annotations/annotate?queueId=${queue.id}`, '_blank')}
                    >
                      <Play className="h-4 w-4 mr-2" />
                      Annotate
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleExport(queue.id, 'csv')}
                    >
                      <Download className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}

        {queues?.length === 0 && (
          <div className="col-span-full text-center py-12">
            <BarChart3 className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
            <h3 className="text-lg font-semibold mb-2">No annotation queues yet</h3>
            <p className="text-muted-foreground mb-4">
              Create your first annotation queue to start collecting human feedback on your LLM outputs.
            </p>
            <Button onClick={() => setIsCreateDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Create Your First Queue
            </Button>
          </div>
        )}
      </div>

      {queues && queues.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>All Queues</CardTitle>
            <CardDescription>
              Detailed view of all annotation queues
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Source</TableHead>
                  <TableHead>Strategy</TableHead>
                  <TableHead>Max Annotations</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Progress</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {queues.map((queue) => {
                  const stats = queueStatsQueries.data?.[queue.id];
                  const progressPercent = stats?.totalItems > 0
                    ? Math.min((stats.completedAssignments / stats.totalItems) * 100, 100)
                    : 0;

                  return (
                    <TableRow key={queue.id}>
                      <TableCell className="font-medium">{queue.name}</TableCell>
                      <TableCell>{queue.description}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{queue.itemSource}</Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">{queue.assignmentStrategy}</Badge>
                      </TableCell>
                      <TableCell>{queue.maxAnnotationsPerItem}</TableCell>
                      <TableCell>
                        <Badge variant={queue.isActive ? "default" : "secondary"}>
                          {queue.isActive ? "Active" : "Inactive"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center space-x-2">
                          <div className="flex-1 bg-gray-200 rounded-full h-2">
                            <div
                              className="bg-blue-600 h-2 rounded-full"
                              style={{ width: `${progressPercent}%` }}
                            ></div>
                          </div>
                          <span className="text-sm text-muted-foreground">
                            {stats?.completedAssignments || 0}/{stats?.totalItems || 0}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>{new Date(queue.createdAt).toLocaleDateString()}</TableCell>
                      <TableCell>
                      <div className="flex space-x-2">
                        <Button size="sm" variant="outline">
                          <Settings className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          onClick={() => window.open(`/annotations/annotate?queueId=${queue.id}`, '_blank')}
                        >
                          <Play className="h-4 w-4" />
                        </Button>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => handleExport(queue.id, 'csv')}
                        >
                          <Download className="h-4 w-4" />
                        </Button>
                      </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
