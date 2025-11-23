import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
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
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  DropdownMenuSub,
  DropdownMenuSubTrigger,
  DropdownMenuSubContent,
  DropdownMenuPortal,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  ArrowLeft,
  Save,
  Copy,
  Tag,
  Clock,
  History,
  Play,
  Settings,
  MoreVertical,
  GitCompare,
  CopyPlus,
  Check,
  AlertCircle,
  Rocket,
  BarChart3,
} from 'lucide-react';
import {
  usePrompt,
  useUpdatePrompt,
  usePromptVersions,
  useCreateVersion,
  useUpdateVersionLabels,
  useDuplicatePrompt,
  usePromoteVersion,
  usePromptAnalytics,
  type PromptVersion,
} from '@/api/prompts';
import { PromptPlayground } from '@/components/features/prompts/playground';

// Available version labels
const VERSION_LABELS = ['production', 'staging', 'development', 'archived'];

export function PromptDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: prompt, isLoading, error } = usePrompt(id || '');
  const { data: versionsData, refetch: refetchVersions } = usePromptVersions(id || '');
  const { data: analytics } = usePromptAnalytics(id || '');
  const updatePrompt = useUpdatePrompt();
  const createVersion = useCreateVersion();
  const updateVersionLabels = useUpdateVersionLabels();
  const duplicatePrompt = useDuplicatePrompt();
  const promoteVersion = usePromoteVersion();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState('');
  const [content, setContent] = useState('');
  const [isDirty, setIsDirty] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<number | null>(null);
  const [compareMode, setCompareMode] = useState(false);
  const [compareV1, setCompareV1] = useState<number | null>(null);
  const [compareV2, setCompareV2] = useState<number | null>(null);
  const [duplicateDialogOpen, setDuplicateDialogOpen] = useState(false);
  const [duplicateName, setDuplicateName] = useState('');
  const [labelDialogOpen, setLabelDialogOpen] = useState(false);
  const [editingVersion, setEditingVersion] = useState<PromptVersion | null>(null);
  const [newLabels, setNewLabels] = useState<string[]>([]);
  const [copied, setCopied] = useState(false);
  const [playgroundOpen, setPlaygroundOpen] = useState(false);

  // Load the latest version content when versions are loaded
  useEffect(() => {
    if (versionsData?.data?.length) {
      const latestVersion = versionsData.data[0]; // Already sorted DESC
      setContent(latestVersion.content);
      setSelectedVersion(latestVersion.version);
    }
  }, [versionsData]);

  useEffect(() => {
    if (prompt) {
      setName(prompt.name);
      setDescription(prompt.description || '');
      setTags(prompt.tags?.join(', ') || '');
    }
  }, [prompt]);

  const handleSave = async () => {
    if (!id) return;

    try {
      // Update prompt metadata
      await updatePrompt.mutateAsync({
        id,
        data: {
          name,
          description,
          tags: tags
            .split(',')
            .map((t) => t.trim())
            .filter(Boolean),
        },
      });

      // Create a new version if content changed
      if (content && content !== versionsData?.data?.[0]?.content) {
        await createVersion.mutateAsync({
          promptId: id,
          data: {
            content,
          },
        });
        await refetchVersions();
      }

      setIsDirty(false);
    } catch {
      // Error handled by mutation
    }
  };

  const handleCopyContent = async () => {
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleLoadVersion = (version: PromptVersion) => {
    setContent(version.content);
    setSelectedVersion(version.version);
    setIsDirty(true);
  };

  const handleDuplicate = async () => {
    if (!id || !duplicateName) return;

    try {
      const newPrompt = await duplicatePrompt.mutateAsync({
        id,
        name: duplicateName,
      });
      setDuplicateDialogOpen(false);
      setDuplicateName('');
      navigate(`/prompts/${newPrompt.id}`);
    } catch {
      // Error handled by mutation
    }
  };

  const handleUpdateLabels = async () => {
    if (!id || !editingVersion) return;

    try {
      await updateVersionLabels.mutateAsync({
        promptId: id,
        version: editingVersion.version,
        data: { labels: newLabels },
      });
      setLabelDialogOpen(false);
      setEditingVersion(null);
      setNewLabels([]);
      await refetchVersions();
    } catch {
      // Error handled by mutation
    }
  };

  const openLabelDialog = (version: PromptVersion) => {
    setEditingVersion(version);
    setNewLabels(version.labels || []);
    setLabelDialogOpen(true);
  };

  const toggleLabel = (label: string) => {
    setNewLabels((prev) =>
      prev.includes(label) ? prev.filter((l) => l !== label) : [...prev, label]
    );
  };

  const handlePromote = async (
    version: number,
    target: 'production' | 'staging' | 'development'
  ) => {
    if (!id) return;

    try {
      await promoteVersion.mutateAsync({
        promptId: id,
        version,
        target,
      });
      await refetchVersions();
    } catch {
      // Error handled by mutation
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="animate-pulse">
            <div className="h-8 w-48 bg-muted rounded" />
          </div>
        </div>
        <Card>
          <CardContent className="pt-6">
            <div className="animate-pulse space-y-4">
              <div className="h-4 w-3/4 bg-muted rounded" />
              <div className="h-4 w-1/2 bg-muted rounded" />
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (error || !prompt) {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <h1 className="text-3xl font-bold">Prompt Not Found</h1>
        </div>
        <Card>
          <CardContent className="pt-6">
            <p className="text-muted-foreground">
              The prompt you&apos;re looking for doesn&apos;t exist or has been deleted.
            </p>
            <Button className="mt-4" onClick={() => navigate('/prompts')}>
              Back to Prompts
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const versions = versionsData?.data || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{prompt.name}</h1>
            <p className="text-muted-foreground flex items-center gap-2 mt-1">
              <Clock className="h-3 w-3" />
              Last updated {format(new Date(prompt.updatedAt), 'MMM d, yyyy h:mm a')}
              {selectedVersion && (
                <Badge variant="outline" className="ml-2">
                  v{selectedVersion}
                </Badge>
              )}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Dialog open={duplicateDialogOpen} onOpenChange={setDuplicateDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <CopyPlus className="h-4 w-4 mr-2" />
                Duplicate
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Duplicate Prompt</DialogTitle>
                <DialogDescription>
                  Create a copy of this prompt with all its versions.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                <div className="space-y-2">
                  <Label htmlFor="duplicate-name">New Prompt Name</Label>
                  <Input
                    id="duplicate-name"
                    value={duplicateName}
                    onChange={(e) => setDuplicateName(e.target.value)}
                    placeholder={`${prompt.name} (copy)`}
                  />
                </div>
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setDuplicateDialogOpen(false)}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleDuplicate}
                  disabled={!duplicateName || duplicatePrompt.isPending}
                >
                  {duplicatePrompt.isPending ? 'Duplicating...' : 'Duplicate'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
          <Button
            onClick={handleSave}
            disabled={!isDirty || updatePrompt.isPending || createVersion.isPending}
          >
            <Save className="h-4 w-4 mr-2" />
            {updatePrompt.isPending || createVersion.isPending
              ? 'Saving...'
              : 'Save Changes'}
          </Button>
        </div>
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Editor */}
        <div className="lg:col-span-2 space-y-6">
          <Tabs defaultValue="editor">
            <TabsList>
              <TabsTrigger value="editor">Editor</TabsTrigger>
              <TabsTrigger value="preview">Preview</TabsTrigger>
              <TabsTrigger value="versions">
                <History className="h-3 w-3 mr-1" />
                Versions ({versions.length})
              </TabsTrigger>
              <TabsTrigger value="compare" disabled={versions.length < 2}>
                <GitCompare className="h-3 w-3 mr-1" />
                Compare
              </TabsTrigger>
            </TabsList>

            <TabsContent value="editor">
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">Prompt Template</CardTitle>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleCopyContent}
                    >
                      {copied ? (
                        <>
                          <Check className="h-4 w-4 mr-1 text-green-500" />
                          Copied
                        </>
                      ) : (
                        <>
                          <Copy className="h-4 w-4 mr-1" />
                          Copy
                        </>
                      )}
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <Textarea
                    placeholder="Enter your prompt template here...

Variables: {{variable_name}}
Conditionals: {{#if condition}}...{{else}}...{{/if}}
Loops: {{#each items}}{{this}}{{/each}}
Negation: {{#unless condition}}...{{/unless}}

Example:
You are a helpful assistant. The user's name is {{user_name}}.
{{#if is_premium}}You have premium access.{{/if}}
Please help them with: {{user_query}}"
                    value={content}
                    onChange={(e) => {
                      setContent(e.target.value);
                      setIsDirty(true);
                    }}
                    className="min-h-[400px] font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground mt-2">
                    Syntax: {`{{variable}}`} | {`{{#if var}}...{{/if}}`} | {`{{#each arr}}...{{/each}}`}
                  </p>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="preview">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Preview</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="bg-muted rounded-lg p-4 min-h-[400px]">
                    <pre className="whitespace-pre-wrap font-mono text-sm">
                      {content || 'No content to preview'}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="versions">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Version History</CardTitle>
                </CardHeader>
                <CardContent>
                  {!versions.length ? (
                    <div className="text-center py-8 text-muted-foreground">
                      <History className="h-8 w-8 mx-auto mb-2 opacity-50" />
                      <p>No versions yet</p>
                      <p className="text-sm">
                        Save your prompt to create the first version
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {versions.map((version) => (
                        <div
                          key={version.id}
                          className={`flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 ${
                            selectedVersion === version.version
                              ? 'border-primary bg-primary/5'
                              : ''
                          }`}
                        >
                          <div
                            className="flex-1 cursor-pointer"
                            onClick={() => handleLoadVersion(version)}
                          >
                            <p className="font-medium">
                              Version {version.version}
                              {selectedVersion === version.version && (
                                <Badge variant="secondary" className="ml-2">
                                  Current
                                </Badge>
                              )}
                            </p>
                            <p className="text-sm text-muted-foreground">
                              {format(
                                new Date(version.createdAt),
                                'MMM d, yyyy h:mm a'
                              )}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            {version.labels?.map((label) => (
                              <Badge
                                key={label}
                                className={
                                  label === 'production'
                                    ? 'bg-green-500/20 text-green-700 dark:text-green-400 border-green-500/50'
                                    : label === 'staging'
                                    ? 'bg-yellow-500/20 text-yellow-700 dark:text-yellow-400 border-yellow-500/50'
                                    : label === 'development'
                                    ? 'bg-blue-500/20 text-blue-700 dark:text-blue-400 border-blue-500/50'
                                    : ''
                                }
                                variant="outline"
                              >
                                {label}
                              </Badge>
                            ))}
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon">
                                  <MoreVertical className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onClick={() => handleLoadVersion(version)}
                                >
                                  Load this version
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onClick={() => openLabelDialog(version)}
                                >
                                  <Tag className="h-4 w-4 mr-2" />
                                  Manage labels
                                </DropdownMenuItem>
                                <DropdownMenuSub>
                                  <DropdownMenuSubTrigger>
                                    <Rocket className="h-4 w-4 mr-2" />
                                    Promote to...
                                  </DropdownMenuSubTrigger>
                                  <DropdownMenuPortal>
                                    <DropdownMenuSubContent>
                                      <DropdownMenuItem
                                        onClick={() =>
                                          handlePromote(version.version, 'production')
                                        }
                                        disabled={version.labels?.includes('production')}
                                      >
                                        <span className="h-2 w-2 rounded-full bg-green-500 mr-2" />
                                        Production
                                        {version.labels?.includes('production') && (
                                          <Check className="h-3 w-3 ml-2 text-muted-foreground" />
                                        )}
                                      </DropdownMenuItem>
                                      <DropdownMenuItem
                                        onClick={() =>
                                          handlePromote(version.version, 'staging')
                                        }
                                        disabled={version.labels?.includes('staging')}
                                      >
                                        <span className="h-2 w-2 rounded-full bg-yellow-500 mr-2" />
                                        Staging
                                        {version.labels?.includes('staging') && (
                                          <Check className="h-3 w-3 ml-2 text-muted-foreground" />
                                        )}
                                      </DropdownMenuItem>
                                      <DropdownMenuItem
                                        onClick={() =>
                                          handlePromote(version.version, 'development')
                                        }
                                        disabled={version.labels?.includes('development')}
                                      >
                                        <span className="h-2 w-2 rounded-full bg-blue-500 mr-2" />
                                        Development
                                        {version.labels?.includes('development') && (
                                          <Check className="h-3 w-3 ml-2 text-muted-foreground" />
                                        )}
                                      </DropdownMenuItem>
                                    </DropdownMenuSubContent>
                                  </DropdownMenuPortal>
                                </DropdownMenuSub>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  onClick={() => {
                                    navigator.clipboard.writeText(version.content);
                                  }}
                                >
                                  <Copy className="h-4 w-4 mr-2" />
                                  Copy content
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="compare">
              <Card>
                <CardHeader>
                  <CardTitle className="text-lg">Compare Versions</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="flex gap-4 mb-4">
                    <div className="flex-1">
                      <Label>Version 1</Label>
                      <Select
                        value={compareV1?.toString() || ''}
                        onValueChange={(v) => setCompareV1(parseInt(v, 10))}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select version" />
                        </SelectTrigger>
                        <SelectContent>
                          {versions.map((v) => (
                            <SelectItem key={v.version} value={v.version.toString()}>
                              Version {v.version}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="flex-1">
                      <Label>Version 2</Label>
                      <Select
                        value={compareV2?.toString() || ''}
                        onValueChange={(v) => setCompareV2(parseInt(v, 10))}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select version" />
                        </SelectTrigger>
                        <SelectContent>
                          {versions.map((v) => (
                            <SelectItem key={v.version} value={v.version.toString()}>
                              Version {v.version}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  {compareV1 && compareV2 ? (
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <p className="text-sm font-medium mb-2">
                          Version {compareV1}
                        </p>
                        <div className="bg-muted rounded-lg p-3 min-h-[300px]">
                          <pre className="whitespace-pre-wrap font-mono text-xs">
                            {versions.find((v) => v.version === compareV1)?.content}
                          </pre>
                        </div>
                      </div>
                      <div>
                        <p className="text-sm font-medium mb-2">
                          Version {compareV2}
                        </p>
                        <div className="bg-muted rounded-lg p-3 min-h-[300px]">
                          <pre className="whitespace-pre-wrap font-mono text-xs">
                            {versions.find((v) => v.version === compareV2)?.content}
                          </pre>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      <GitCompare className="h-8 w-8 mx-auto mb-2 opacity-50" />
                      <p>Select two versions to compare</p>
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Settings */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Settings className="h-4 w-4" />
                Settings
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    setIsDirty(true);
                  }}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Input
                  id="description"
                  value={description}
                  onChange={(e) => {
                    setDescription(e.target.value);
                    setIsDirty(true);
                  }}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tags">
                  <Tag className="h-3 w-3 inline mr-1" />
                  Tags
                </Label>
                <Input
                  id="tags"
                  value={tags}
                  placeholder="Comma-separated tags"
                  onChange={(e) => {
                    setTags(e.target.value);
                    setIsDirty(true);
                  }}
                />
              </div>
            </CardContent>
          </Card>

          {/* Playground */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <Play className="h-4 w-4" />
                Playground
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">
                Test your prompt with different variables and preview the
                compiled output.
              </p>
              <Button
                className="w-full"
                onClick={() => setPlaygroundOpen(true)}
                disabled={!content}
              >
                <Play className="h-4 w-4 mr-2" />
                Open Playground
              </Button>
            </CardContent>
          </Card>

          {/* Quick Info */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <BarChart3 className="h-4 w-4" />
                Info & Analytics
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">ID</span>
                  <span className="font-mono text-xs">
                    {prompt.id.slice(0, 8)}...
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Created</span>
                  <span>{format(new Date(prompt.createdAt), 'MMM d, yyyy')}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Versions</span>
                  <span>{versions.length}</span>
                </div>
                {versions.length > 0 && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Latest</span>
                    <span>v{versions[0].version}</span>
                  </div>
                )}
              </div>

              {/* Deployment Status */}
              {analytics && (
                <div className="border-t pt-4">
                  <p className="text-sm font-medium mb-2">Deployment Status</p>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="h-2 w-2 rounded-full bg-green-500" />
                        Production
                      </span>
                      {analytics.productionVersion ? (
                        <Badge variant="outline" className="bg-green-500/10 text-green-700">
                          v{analytics.productionVersion}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground text-xs">Not deployed</span>
                      )}
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="h-2 w-2 rounded-full bg-yellow-500" />
                        Staging
                      </span>
                      {analytics.stagingVersion ? (
                        <Badge variant="outline" className="bg-yellow-500/10 text-yellow-700">
                          v{analytics.stagingVersion}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground text-xs">Not deployed</span>
                      )}
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="flex items-center gap-2">
                        <span className="h-2 w-2 rounded-full bg-blue-500" />
                        Development
                      </span>
                      {analytics.developmentVersion ? (
                        <Badge variant="outline" className="bg-blue-500/10 text-blue-700">
                          v{analytics.developmentVersion}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground text-xs">Not deployed</span>
                      )}
                    </div>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Label Management Dialog */}
      <Dialog open={labelDialogOpen} onOpenChange={setLabelDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Manage Version Labels</DialogTitle>
            <DialogDescription>
              Add or remove labels from version {editingVersion?.version}
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <div className="flex flex-wrap gap-2">
              {VERSION_LABELS.map((label) => (
                <Button
                  key={label}
                  variant={newLabels.includes(label) ? 'default' : 'outline'}
                  size="sm"
                  onClick={() => toggleLabel(label)}
                >
                  {newLabels.includes(label) && (
                    <Check className="h-3 w-3 mr-1" />
                  )}
                  {label}
                </Button>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setLabelDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleUpdateLabels}
              disabled={updateVersionLabels.isPending}
            >
              {updateVersionLabels.isPending ? 'Saving...' : 'Save Labels'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Prompt Playground */}
      <PromptPlayground
        promptId={id || ''}
        content={content}
        open={playgroundOpen}
        onOpenChange={setPlaygroundOpen}
      />
    </div>
  );
}
