import { useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
  useEvaluators,
  useDeleteEvaluator,
  useEvaluatorTemplates,
  useCreateEvaluator,
  type Evaluator,
  type EvaluatorTemplate,
  type CreateEvaluatorRequest,
} from '@/api/evaluators';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { formatDate } from '@/lib/utils';
import { useToast } from '@/hooks/use-toast';
import {
  Search,
  ChevronLeft,
  ChevronRight,
  Filter,
  X,
  Plus,
  Trash2,
  Settings2,
  Play,
  BarChart3,
  Sparkles,
} from 'lucide-react';

type EvaluatorsPageParams = {
  limit?: number;
  offset?: number;
  type?: string;
  provider?: string;
  outputType?: string;
  enabled?: boolean;
  search?: string;
};

const ITEMS_PER_PAGE = 50;

function EvaluatorsPage() {
  const { toast } = useToast();
  const [params, setParams] = useState<EvaluatorsPageParams>({
    limit: ITEMS_PER_PAGE,
    offset: 0,
  });

  const [searchQuery, setSearchQuery] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedEvaluator, setSelectedEvaluator] = useState<Evaluator | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<EvaluatorTemplate | null>(null);

  // Form state
  const [formData, setFormData] = useState<CreateEvaluatorRequest>({
    name: '',
    description: '',
    type: 'llm_judge',
    provider: 'openai',
    model: 'gpt-4o-mini',
    template: '',
    outputType: 'numeric',
    minValue: 0,
    maxValue: 1,
    enabled: true,
  });

  const { data: evaluatorsResponse, isLoading, error } = useEvaluators(params);
  const { data: templatesResponse } = useEvaluatorTemplates();
  const createMutation = useCreateEvaluator();
  const deleteMutation = useDeleteEvaluator();

  const evaluators = evaluatorsResponse?.evaluators || [];
  const templates = templatesResponse?.templates || [];
  const total = evaluatorsResponse?.pagination.total || 0;
  const currentPage = Math.floor((params.offset || 0) / ITEMS_PER_PAGE) + 1;
  const totalPages = Math.ceil(total / ITEMS_PER_PAGE);

  const handleSearch = useCallback(() => {
    setParams(prev => ({
      ...prev,
      search: searchQuery || undefined,
      offset: 0,
    }));
  }, [searchQuery]);

  const handleFilterChange = useCallback((key: keyof EvaluatorsPageParams, value: any) => {
    setParams(prev => ({
      ...prev,
      [key]: value || undefined,
      offset: 0,
    }));
  }, []);

  const clearFilters = useCallback(() => {
    setParams({
      limit: ITEMS_PER_PAGE,
      offset: 0,
    });
    setSearchQuery('');
  }, []);

  const goToPage = useCallback((page: number) => {
    setParams(prev => ({
      ...prev,
      offset: (page - 1) * ITEMS_PER_PAGE,
    }));
  }, []);

  const handleSelectTemplate = useCallback((template: EvaluatorTemplate) => {
    setSelectedTemplate(template);
    setFormData(prev => ({
      ...prev,
      name: template.name,
      description: template.description,
      template: template.template,
      outputType: template.outputType,
      minValue: template.minValue,
      maxValue: template.maxValue,
      categories: template.categories,
    }));
  }, []);

  const handleCreate = async () => {
    try {
      await createMutation.mutateAsync(formData);
      toast({
        title: 'Evaluator created',
        description: `${formData.name} has been created successfully.`,
      });
      setCreateDialogOpen(false);
      resetForm();
    } catch (error) {
      toast({
        title: 'Error creating evaluator',
        description: 'Please try again.',
        variant: 'destructive',
      });
    }
  };

  const handleDelete = async () => {
    if (!selectedEvaluator) return;
    try {
      await deleteMutation.mutateAsync(selectedEvaluator.id);
      toast({
        title: 'Evaluator deleted',
        description: `${selectedEvaluator.name} has been deleted.`,
      });
      setDeleteDialogOpen(false);
      setSelectedEvaluator(null);
    } catch (error) {
      toast({
        title: 'Error deleting evaluator',
        description: 'Please try again.',
        variant: 'destructive',
      });
    }
  };

  const resetForm = () => {
    setFormData({
      name: '',
      description: '',
      type: 'llm_judge',
      provider: 'openai',
      model: 'gpt-4o-mini',
      template: '',
      outputType: 'numeric',
      minValue: 0,
      maxValue: 1,
      enabled: true,
    });
    setSelectedTemplate(null);
  };

  const getProviderBadgeVariant = (provider: string) => {
    switch (provider) {
      case 'openai': return 'default';
      case 'anthropic': return 'secondary';
      case 'google': return 'outline';
      case 'ollama': return 'destructive';
      default: return 'default';
    }
  };

  const getOutputTypeBadgeVariant = (outputType: string) => {
    switch (outputType) {
      case 'numeric': return 'default';
      case 'boolean': return 'secondary';
      case 'categorical': return 'outline';
      default: return 'default';
    }
  };

  // Group templates by category
  const templatesByCategory = templates.reduce((acc, template) => {
    if (!acc[template.category]) {
      acc[template.category] = [];
    }
    acc[template.category].push(template);
    return acc;
  }, {} as Record<string, EvaluatorTemplate[]>);

  if (error) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h3 className="text-lg font-semibold text-destructive">Error loading evaluators</h3>
          <p className="text-muted-foreground">Please try again later</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Evaluators</h1>
          <p className="text-muted-foreground">
            Configure LLM-as-a-Judge evaluators for automated trace scoring
          </p>
        </div>
        <div className="flex gap-2">
          <Link to="/evaluators/results">
            <Button variant="outline">
              <BarChart3 className="h-4 w-4 mr-2" />
              Results
            </Button>
          </Link>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button onClick={resetForm}>
                <Plus className="h-4 w-4 mr-2" />
                Create Evaluator
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>Create Evaluator</DialogTitle>
                <DialogDescription>
                  Configure a new LLM-as-a-Judge evaluator from scratch or from a template.
                </DialogDescription>
              </DialogHeader>

              <Tabs defaultValue="templates" className="w-full">
                <TabsList>
                  <TabsTrigger value="templates">
                    <Sparkles className="h-4 w-4 mr-2" />
                    From Template
                  </TabsTrigger>
                  <TabsTrigger value="custom">
                    <Settings2 className="h-4 w-4 mr-2" />
                    Custom
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="templates" className="space-y-4">
                  <div className="grid grid-cols-1 gap-4">
                    {Object.entries(templatesByCategory).map(([category, categoryTemplates]) => (
                      <Card key={category}>
                        <CardHeader className="py-3">
                          <CardTitle className="text-sm capitalize">{category}</CardTitle>
                        </CardHeader>
                        <CardContent className="py-2">
                          <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
                            {categoryTemplates.map((template) => (
                              <Button
                                key={template.id}
                                variant={selectedTemplate?.id === template.id ? 'default' : 'outline'}
                                className="h-auto py-2 px-3 justify-start"
                                onClick={() => handleSelectTemplate(template)}
                              >
                                <div className="text-left">
                                  <div className="font-medium text-sm">{template.name}</div>
                                  <div className="text-xs text-muted-foreground truncate max-w-[150px]">
                                    {template.description}
                                  </div>
                                </div>
                              </Button>
                            ))}
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                </TabsContent>

                <TabsContent value="custom" className="space-y-4">
                  <p className="text-sm text-muted-foreground">
                    Create a custom evaluator with your own prompt template.
                  </p>
                </TabsContent>
              </Tabs>

              <div className="space-y-4 mt-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="name">Name</Label>
                    <Input
                      id="name"
                      value={formData.name}
                      onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                      placeholder="Enter evaluator name"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="provider">Provider</Label>
                    <Select
                      value={formData.provider}
                      onValueChange={(value: any) => setFormData(prev => ({ ...prev, provider: value }))}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="openai">OpenAI</SelectItem>
                        <SelectItem value="anthropic">Anthropic</SelectItem>
                        <SelectItem value="google">Google</SelectItem>
                        <SelectItem value="ollama">Ollama</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="model">Model</Label>
                    <Input
                      id="model"
                      value={formData.model}
                      onChange={(e) => setFormData(prev => ({ ...prev, model: e.target.value }))}
                      placeholder="gpt-4o-mini"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="outputType">Output Type</Label>
                    <Select
                      value={formData.outputType}
                      onValueChange={(value: any) => setFormData(prev => ({ ...prev, outputType: value }))}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="numeric">Numeric</SelectItem>
                        <SelectItem value="boolean">Boolean</SelectItem>
                        <SelectItem value="categorical">Categorical</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                {formData.outputType === 'numeric' && (
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="minValue">Min Value</Label>
                      <Input
                        id="minValue"
                        type="number"
                        value={formData.minValue}
                        onChange={(e) => setFormData(prev => ({ ...prev, minValue: parseFloat(e.target.value) }))}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="maxValue">Max Value</Label>
                      <Input
                        id="maxValue"
                        type="number"
                        value={formData.maxValue}
                        onChange={(e) => setFormData(prev => ({ ...prev, maxValue: parseFloat(e.target.value) }))}
                      />
                    </div>
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    value={formData.description}
                    onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="Brief description of what this evaluator measures"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="template">Prompt Template</Label>
                  <Textarea
                    id="template"
                    value={formData.template}
                    onChange={(e) => setFormData(prev => ({ ...prev, template: e.target.value }))}
                    placeholder="Enter your evaluation prompt template..."
                    className="min-h-[200px] font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground">
                    Use {'{{input}}'} and {'{{output}}'} as placeholders for trace input/output.
                  </p>
                </div>

                <div className="flex items-center space-x-2">
                  <Switch
                    id="enabled"
                    checked={formData.enabled}
                    onCheckedChange={(checked) => setFormData(prev => ({ ...prev, enabled: checked }))}
                  />
                  <Label htmlFor="enabled">Enabled</Label>
                </div>
              </div>

              <DialogFooter>
                <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                  Cancel
                </Button>
                <Button
                  onClick={handleCreate}
                  disabled={!formData.name || !formData.template || createMutation.isPending}
                >
                  {createMutation.isPending ? 'Creating...' : 'Create Evaluator'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Search and Filters */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">Search & Filter</CardTitle>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
            >
              <Filter className="h-4 w-4 mr-2" />
              Filters
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                placeholder="Search evaluators..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
              />
            </div>
            <Button onClick={handleSearch}>
              <Search className="h-4 w-4 mr-2" />
              Search
            </Button>
            {(Object.values(params).some(v => v !== undefined && v !== 0 && v !== ITEMS_PER_PAGE) || searchQuery) && (
              <Button variant="outline" onClick={clearFilters}>
                <X className="h-4 w-4 mr-2" />
                Clear
              </Button>
            )}
          </div>

          {showFilters && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 pt-4 border-t">
              <div>
                <label className="text-sm font-medium mb-1 block">Provider</label>
                <Select
                  value={params.provider || ''}
                  onValueChange={(value) => handleFilterChange('provider', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All providers" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="anthropic">Anthropic</SelectItem>
                    <SelectItem value="google">Google</SelectItem>
                    <SelectItem value="ollama">Ollama</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">Output Type</label>
                <Select
                  value={params.outputType || ''}
                  onValueChange={(value) => handleFilterChange('outputType', value || undefined)}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All types" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="numeric">Numeric</SelectItem>
                    <SelectItem value="boolean">Boolean</SelectItem>
                    <SelectItem value="categorical">Categorical</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div>
                <label className="text-sm font-medium mb-1 block">Status</label>
                <Select
                  value={params.enabled === undefined ? '' : params.enabled ? 'true' : 'false'}
                  onValueChange={(value) => handleFilterChange('enabled', value === '' ? undefined : value === 'true')}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="All statuses" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="true">Enabled</SelectItem>
                    <SelectItem value="false">Disabled</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Evaluators Table */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>
              Evaluators {total > 0 && `(${total})`}
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
                <p className="text-muted-foreground">Loading evaluators...</p>
              </div>
            </div>
          ) : evaluators.length === 0 ? (
            <div className="flex items-center justify-center h-64">
              <div className="text-center">
                <Sparkles className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                <h3 className="text-lg font-semibold">No evaluators found</h3>
                <p className="text-muted-foreground mb-4">
                  Create your first evaluator to start automated scoring
                </p>
                <Button onClick={() => setCreateDialogOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Evaluator
                </Button>
              </div>
            </div>
          ) : (
            <>
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Provider</TableHead>
                      <TableHead>Model</TableHead>
                      <TableHead>Output</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {evaluators.map((evaluator) => (
                      <TableRow key={evaluator.id}>
                        <TableCell className="font-medium">
                          <Link
                            to={`/evaluators/${evaluator.id}`}
                            className="text-primary hover:underline"
                          >
                            {evaluator.name}
                          </Link>
                          {evaluator.description && (
                            <p className="text-xs text-muted-foreground truncate max-w-[200px]">
                              {evaluator.description}
                            </p>
                          )}
                        </TableCell>
                        <TableCell>
                          <Badge variant={getProviderBadgeVariant(evaluator.provider)}>
                            {evaluator.provider}
                          </Badge>
                        </TableCell>
                        <TableCell className="font-mono text-sm">
                          {evaluator.model}
                        </TableCell>
                        <TableCell>
                          <Badge variant={getOutputTypeBadgeVariant(evaluator.outputType)}>
                            {evaluator.outputType}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <Badge variant={evaluator.enabled ? 'default' : 'secondary'}>
                            {evaluator.enabled ? 'Enabled' : 'Disabled'}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(evaluator.createdAt)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex items-center justify-end gap-2">
                            <Link to={`/evaluators/${evaluator.id}`}>
                              <Button variant="ghost" size="sm">
                                <Settings2 className="h-4 w-4" />
                              </Button>
                            </Link>
                            <Link to={`/evaluators/${evaluator.id}/run`}>
                              <Button variant="ghost" size="sm">
                                <Play className="h-4 w-4" />
                              </Button>
                            </Link>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => {
                                setSelectedEvaluator(evaluator);
                                setDeleteDialogOpen(true);
                              }}
                            >
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    Showing {((currentPage - 1) * ITEMS_PER_PAGE) + 1} to{' '}
                    {Math.min(currentPage * ITEMS_PER_PAGE, total)} of {total} evaluators
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage - 1)}
                      disabled={currentPage === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                      Previous
                    </Button>
                    <span className="text-sm">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => goToPage(currentPage + 1)}
                      disabled={currentPage === totalPages}
                    >
                      Next
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Evaluator</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{selectedEvaluator?.name}"? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export default EvaluatorsPage;
export { EvaluatorsPage };
