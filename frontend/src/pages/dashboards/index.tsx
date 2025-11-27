import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Plus, LayoutDashboard, Share2, Copy, Trash2 } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useToast } from '@/hooks/use-toast';

interface Dashboard {
  id: string;
  projectId: string;
  name: string;
  description: string;
  isTemplate: boolean;
  isPublic: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export function DashboardsPage() {
  const [createOpen, setCreateOpen] = useState(false);
  const [newDashboard, setNewDashboard] = useState({
    name: '',
    description: '',
  });
  const projectId = 'test-project'; // TODO: Get from context
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Fetch dashboards
  const { data: dashboardsData, isLoading } = useQuery({
    queryKey: ['dashboards', projectId],
    queryFn: () =>
      api.get<{ data: Dashboard[] }>('/v1/dashboards', {
        params: { projectId, includeTemplates: false },
      }),
  });

  // Create dashboard mutation
  const createMutation = useMutation({
    mutationFn: (data: { projectId: string; name: string; description: string }) =>
      api.post('/v1/dashboards', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboards'] });
      setCreateOpen(false);
      setNewDashboard({ name: '', description: '' });
      toast({
        title: 'Dashboard created',
        description: 'Your dashboard has been created successfully.',
      });
    },
    onError: () => {
      toast({
        title: 'Error',
        description: 'Failed to create dashboard.',
        variant: 'destructive',
      });
    },
  });

  // Delete dashboard mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/v1/dashboards/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboards'] });
      toast({
        title: 'Dashboard deleted',
        description: 'Dashboard has been deleted successfully.',
      });
    },
    onError: () => {
      toast({
        title: 'Error',
        description: 'Failed to delete dashboard.',
        variant: 'destructive',
      });
    },
  });

  // Clone dashboard mutation
  const cloneMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api.post(`/v1/dashboards/${id}/clone`, { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboards'] });
      toast({
        title: 'Dashboard cloned',
        description: 'Dashboard has been cloned successfully.',
      });
    },
    onError: () => {
      toast({
        title: 'Error',
        description: 'Failed to clone dashboard.',
        variant: 'destructive',
      });
    },
  });

  const handleCreate = () => {
    if (!newDashboard.name.trim()) {
      toast({
        title: 'Validation error',
        description: 'Dashboard name is required.',
        variant: 'destructive',
      });
      return;
    }

    createMutation.mutate({
      projectId,
      name: newDashboard.name,
      description: newDashboard.description,
    });
  };

  const handleDelete = (id: string, name: string) => {
    if (confirm(`Are you sure you want to delete "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  const handleClone = (id: string, name: string) => {
    const cloneName = prompt(`Enter name for cloned dashboard:`, `${name} (Copy)`);
    if (cloneName) {
      cloneMutation.mutate({ id, name: cloneName });
    }
  };

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboards</h1>
          <p className="text-muted-foreground">
            Create and manage custom dashboards for your analytics
          </p>
        </div>

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              New Dashboard
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Dashboard</DialogTitle>
              <DialogDescription>
                Create a new custom dashboard to visualize your metrics.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  placeholder="My Dashboard"
                  value={newDashboard.name}
                  onChange={(e) =>
                    setNewDashboard({ ...newDashboard, name: e.target.value })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  placeholder="Dashboard description..."
                  value={newDashboard.description}
                  onChange={(e) =>
                    setNewDashboard({ ...newDashboard, description: e.target.value })
                  }
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setCreateOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreate} disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Creating...' : 'Create'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(6)].map((_, i) => (
            <Card key={i} className="h-[200px] animate-pulse">
              <CardHeader>
                <div className="h-6 w-3/4 rounded bg-muted" />
                <div className="h-4 w-full rounded bg-muted" />
              </CardHeader>
              <CardContent>
                <div className="h-4 w-1/2 rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : dashboardsData?.data && dashboardsData.data.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {dashboardsData.data.map((dashboard) => (
            <Card key={dashboard.id} className="flex flex-col">
              <CardHeader>
                <div className="flex items-start justify-between">
                  <LayoutDashboard className="h-5 w-5 text-muted-foreground" />
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => handleClone(dashboard.id, dashboard.name)}
                      title="Clone dashboard"
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-destructive hover:text-destructive"
                      onClick={() => handleDelete(dashboard.id, dashboard.name)}
                      title="Delete dashboard"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
                <CardTitle className="mt-4">
                  <Link
                    to={`/dashboards/${dashboard.id}`}
                    className="hover:underline"
                  >
                    {dashboard.name}
                  </Link>
                </CardTitle>
                {dashboard.description && (
                  <CardDescription className="line-clamp-2">
                    {dashboard.description}
                  </CardDescription>
                )}
              </CardHeader>
              <CardContent className="flex-1">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <span>Updated {new Date(dashboard.updatedAt).toLocaleDateString()}</span>
                  {dashboard.isPublic && (
                    <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
                      <Share2 className="h-3 w-3" />
                      Public
                    </span>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <LayoutDashboard className="h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 text-lg font-semibold">No dashboards yet</h3>
            <p className="text-sm text-muted-foreground text-center max-w-sm mt-2">
              Create your first dashboard to start visualizing your analytics data.
            </p>
            <Button className="mt-4" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Dashboard
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
