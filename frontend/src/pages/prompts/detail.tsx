import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { format } from 'date-fns';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import {
  ArrowLeft,
  Save,
  Copy,
  Tag,
  Clock,
  History,
  Play,
  Settings,
} from 'lucide-react';
import {
  usePrompt,
  useUpdatePrompt,
  usePromptVersions,
} from '@/api/prompts';

export function PromptDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: prompt, isLoading, error } = usePrompt(id || '');
  const { data: versionsData } = usePromptVersions(id || '');
  const updatePrompt = useUpdatePrompt();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [tags, setTags] = useState('');
  const [content, setContent] = useState('');
  const [isDirty, setIsDirty] = useState(false);

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
      setIsDirty(false);
    } catch {
      // Error handled by mutation
    }
  };

  const handleCopyContent = () => {
    navigator.clipboard.writeText(content);
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
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            onClick={handleSave}
            disabled={!isDirty || updatePrompt.isPending}
          >
            <Save className="h-4 w-4 mr-2" />
            {updatePrompt.isPending ? 'Saving...' : 'Save Changes'}
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
                Versions ({versionsData?.data?.length || 0})
              </TabsTrigger>
            </TabsList>

            <TabsContent value="editor">
              <Card>
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">Prompt Template</CardTitle>
                    <Button variant="ghost" size="sm" onClick={handleCopyContent}>
                      <Copy className="h-4 w-4 mr-1" />
                      Copy
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <Textarea
                    placeholder="Enter your prompt template here...

You can use variables like {{variable_name}} to create dynamic prompts.

Example:
You are a helpful assistant. The user's name is {{user_name}}.
Please help them with: {{user_query}}"
                    value={content}
                    onChange={(e) => {
                      setContent(e.target.value);
                      setIsDirty(true);
                    }}
                    className="min-h-[400px] font-mono text-sm"
                  />
                  <p className="text-xs text-muted-foreground mt-2">
                    Use {`{{variable}}`} syntax for template variables
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
                  {!versionsData?.data?.length ? (
                    <div className="text-center py-8 text-muted-foreground">
                      <History className="h-8 w-8 mx-auto mb-2 opacity-50" />
                      <p>No versions yet</p>
                      <p className="text-sm">
                        Save your prompt to create the first version
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {versionsData.data.map((version) => (
                        <div
                          key={version.id}
                          className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 cursor-pointer"
                        >
                          <div>
                            <p className="font-medium">Version {version.version}</p>
                            <p className="text-sm text-muted-foreground">
                              {format(new Date(version.createdAt), 'MMM d, yyyy h:mm a')}
                            </p>
                          </div>
                          <div className="flex gap-1">
                            {version.labels?.map((label) => (
                              <Badge key={label} variant="secondary">
                                {label}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      ))}
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
                Test your prompt with different variables and models.
              </p>
              <Button className="w-full" disabled>
                <Play className="h-4 w-4 mr-2" />
                Open Playground
              </Button>
              <p className="text-xs text-muted-foreground mt-2 text-center">
                Coming soon
              </p>
            </CardContent>
          </Card>

          {/* Quick Info */}
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">Info</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">ID</span>
                <span className="font-mono text-xs">{prompt.id.slice(0, 8)}...</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Created</span>
                <span>{format(new Date(prompt.createdAt), 'MMM d, yyyy')}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Versions</span>
                <span>{versionsData?.data?.length || 0}</span>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
