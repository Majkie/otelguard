import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tantml:parameter>
<parameter name="api } from '@/lib/api';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
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
import { Plus, Save, ArrowLeft, Settings, Trash2 } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { MetricCard, LineChart, BarChart, PieChart } from '@/components/charts';

interface DashboardWidget {
  id: string;
  dashboardId: string;
  widgetType: string;
  title: string;
  config: any;
  position: {
    x: number;
    y: number;
    w: number;
    h: number;
  };
}

interface Dashboard {
  id: string;
  name: string;
  description: string;
  isPublic: boolean;
}

const WIDGET_TYPES = [
  { value: 'metric_card', label: 'Metric Card' },
  { value: 'line_chart', label: 'Line Chart' },
  { value: 'bar_chart', label: 'Bar Chart' },
  { value: 'pie_chart', label: 'Pie Chart' },
  { value: 'table', label: 'Table' },
];

export function DashboardBuilderPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { toast } = useToast();
  const queryClient = useQueryClient();

  const [addWidgetOpen, setAddWidgetOpen] = useState(false);
  const [newWidget, setNewWidget] = useState({
    type: 'metric_card',
    title: '',
  });

  // Fetch dashboard
  const { data: dashboardData } = useQuery({
    queryKey: ['dashboard', id],
    queryFn: () =>
      api.get<{ dashboard: Dashboard; widgets: DashboardWidget[] }>(
        `/v1/dashboards/${id}`
      ),
    enabled: !!id,
  });

  // Add widget mutation
  const addWidgetMutation = useMutation({
    mutationFn: (data: { widgetType: string; title: string; config: any; position: any }) =>
      api.post(`/v1/dashboards/${id}/widgets`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard', id] });
      setAddWidgetOpen(false);
      setNewWidget({ type: 'metric_card', title: '' });
      toast({
        title: 'Widget added',
        description: 'Widget has been added successfully.',
      });
    },
  });

  // Delete widget mutation
  const deleteWidgetMutation = useMutation({
    mutationFn: (widgetId: string) =>
      api.delete(`/v1/dashboards/${id}/widgets/${widgetId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dashboard', id] });
      toast({
        title: 'Widget deleted',
        description: 'Widget has been deleted successfully.',
      });
    },
  });

  const handleAddWidget = () => {
    if (!newWidget.title.trim()) {
      toast({
        title: 'Validation error',
        description: 'Widget title is required.',
        variant: 'destructive',
      });
      return;
    }

    const widgetCount = dashboardData?.widgets?.length || 0;
    const position = {
      x: (widgetCount % 3) * 4,
      y: Math.floor(widgetCount / 3) * 4,
      w: 4,
      h: 4,
    };

    addWidgetMutation.mutate({
      widgetType: newWidget.type,
      title: newWidget.title,
      config: {},
      position,
    });
  };

  const handleDeleteWidget = (widgetId: string) => {
    if (confirm('Are you sure you want to delete this widget?')) {
      deleteWidgetMutation.mutate(widgetId);
    }
  };

  const renderWidget = (widget: DashboardWidget) => {
    // Simple demo rendering - would connect to real data sources in production
    switch (widget.widgetType) {
      case 'metric_card':
        return (
          <MetricCard
            title={widget.title}
            value="1,234"
            trend={{ value: 12, label: 'vs last week' }}
          />
        );
      case 'line_chart':
        return (
          <Card>
            <CardHeader>
              <CardTitle>{widget.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Configure data source in widget settings
              </div>
            </CardContent>
          </Card>
        );
      case 'bar_chart':
        return (
          <Card>
            <CardHeader>
              <CardTitle>{widget.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Configure data source in widget settings
              </div>
            </CardContent>
          </Card>
        );
      case 'pie_chart':
        return (
          <Card>
            <CardHeader>
              <CardTitle>{widget.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Configure data source in widget settings
              </div>
            </CardContent>
          </Card>
        );
      default:
        return (
          <Card>
            <CardHeader>
              <CardTitle>{widget.title}</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">Widget type: {widget.widgetType}</div>
            </CardContent>
          </Card>
        );
    }
  };

  if (!dashboardData) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-muted-foreground">Loading...</div>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate('/dashboards')}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold tracking-tight">
              {dashboardData.dashboard.name}
            </h1>
            {dashboardData.dashboard.description && (
              <p className="text-muted-foreground">{dashboardData.dashboard.description}</p>
            )}
          </div>
        </div>

        <div className="flex gap-2">
          <Dialog open={addWidgetOpen} onOpenChange={setAddWidgetOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="mr-2 h-4 w-4" />
                Add Widget
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add Widget</DialogTitle>
                <DialogDescription>
                  Add a new widget to your dashboard.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="widget-type">Widget Type</Label>
                  <Select
                    value={newWidget.type}
                    onValueChange={(value) =>
                      setNewWidget({ ...newWidget, type: value })
                    }
                  >
                    <SelectTrigger id="widget-type">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {WIDGET_TYPES.map((type) => (
                        <SelectItem key={type.value} value={type.value}>
                          {type.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="widget-title">Title</Label>
                  <Input
                    id="widget-title"
                    placeholder="Widget title"
                    value={newWidget.title}
                    onChange={(e) =>
                      setNewWidget({ ...newWidget, title: e.target.value })
                    }
                  />
                </div>
              </div>
              <DialogFooter>
                <Button variant="outline" onClick={() => setAddWidgetOpen(false)}>
                  Cancel
                </Button>
                <Button onClick={handleAddWidget} disabled={addWidgetMutation.isPending}>
                  {addWidgetMutation.isPending ? 'Adding...' : 'Add'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {dashboardData.widgets && dashboardData.widgets.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {dashboardData.widgets.map((widget) => (
            <div key={widget.id} className="relative group">
              <div className="absolute top-2 right-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity">
                <Button
                  variant="destructive"
                  size="icon"
                  className="h-8 w-8"
                  onClick={() => handleDeleteWidget(widget.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
              {renderWidget(widget)}
            </div>
          ))}
        </div>
      ) : (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Plus className="h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 text-lg font-semibold">No widgets yet</h3>
            <p className="text-sm text-muted-foreground text-center max-w-sm mt-2">
              Add your first widget to start building your dashboard.
            </p>
            <Button className="mt-4" onClick={() => setAddWidgetOpen(true)}>
              <Plus className="mr-2 h-4 w-4" />
              Add Widget
            </Button>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
