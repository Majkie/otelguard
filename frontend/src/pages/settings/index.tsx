import { useState } from 'react';
import { useAuth } from '@/hooks/use-auth';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
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
import { Copy, Plus, Eye, EyeOff } from 'lucide-react';
import { useToast } from '@/hooks/use-toast';
import { useProjectContext } from '@/contexts/project-context';
import { useAPIKeys, useCreateAPIKey, useRevokeAPIKey } from '@/api/api-keys';

export function SettingsPage() {
  const { user } = useAuth();
  const { toast } = useToast();
  const { selectedProject } = useProjectContext();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const [showKey, setShowKey] = useState(false);
  const [keyToRevoke, setKeyToRevoke] = useState<{ id: string; name: string } | null>(null);

  const { data: apiKeysData, isLoading } = useAPIKeys(selectedProject?.id || '');
  const createAPIKey = useCreateAPIKey(selectedProject?.id || '');
  const revokeAPIKey = useRevokeAPIKey(selectedProject?.id || '');

  const apiKeys = apiKeysData?.data || [];

  const handleCopy = (text: string) => {
    navigator.clipboard.writeText(text);
    toast({
      title: 'Copied to clipboard',
    });
  };

  const handleCreateKey = async () => {
    if (!newKeyName.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a name for the API key',
        variant: 'destructive',
      });
      return;
    }

    try {
      const result = await createAPIKey.mutateAsync({
        name: newKeyName,
        scopes: ['trace:write', 'prompt:read', 'guardrail:evaluate'],
      });

      setCreatedKey(result.key);
      setNewKeyName('');
      toast({
        title: 'API Key Created',
        description: 'Save this key securely. It will not be shown again.',
      });
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to create API key',
        variant: 'destructive',
      });
    }
  };

  const handleCloseCreateDialog = () => {
    setCreateDialogOpen(false);
    setCreatedKey(null);
    setNewKeyName('');
    setShowKey(false);
  };

  const handleRevoke = async () => {
    if (!keyToRevoke) return;

    try {
      await revokeAPIKey.mutateAsync(keyToRevoke.id);
      toast({
        title: 'API Key Revoked',
        description: `API key "${keyToRevoke.name}" has been revoked`,
      });
      setKeyToRevoke(null);
    } catch (error) {
      toast({
        title: 'Error',
        description: 'Failed to revoke API key',
        variant: 'destructive',
      });
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Manage your account and API keys
        </p>
      </div>

      {/* Profile */}
      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Your account information</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label>Name</Label>
              <Input value={user?.name || ''} disabled />
            </div>
            <div className="space-y-2">
              <Label>Email</Label>
              <Input value={user?.email || ''} disabled />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* API Keys */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>API Keys</CardTitle>
              <CardDescription>
                Manage API keys for SDK authentication
              </CardDescription>
            </div>
            <Button onClick={() => setCreateDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Create API Key
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8 text-muted-foreground">
              <p>Loading API keys...</p>
            </div>
          ) : apiKeys.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <p>No API keys yet</p>
              <p className="text-sm mt-1">
                Create an API key to start sending traces from your application
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {apiKeys.map((key) => (
                <div
                  key={key.id}
                  className="flex items-center justify-between border rounded-lg p-4"
                >
                  <div className="flex-1">
                    <p className="font-medium">{key.name}</p>
                    <p className="text-sm text-muted-foreground font-mono">
                      {key.keyPrefix}...
                    </p>
                    {key.lastUsedAt && (
                      <p className="text-xs text-muted-foreground mt-1">
                        Last used: {new Date(key.lastUsedAt).toLocaleDateString()}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => handleCopy(key.keyPrefix)}
                      title="Copy prefix"
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setKeyToRevoke({ id: key.id, name: key.name })}
                    >
                      Revoke
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* SDK Installation */}
      <Card>
        <CardHeader>
          <CardTitle>SDK Installation</CardTitle>
          <CardDescription>Install the SDK in your project</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label>Python</Label>
            <div className="relative">
              <pre className="bg-muted p-3 rounded-lg font-mono text-sm">
                pip install otelguard-sdk
              </pre>
              <Button
                variant="ghost"
                size="icon"
                className="absolute top-2 right-2"
                onClick={() => handleCopy('pip install otelguard-sdk')}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <div className="space-y-2">
            <Label>JavaScript / TypeScript</Label>
            <div className="relative">
              <pre className="bg-muted p-3 rounded-lg font-mono text-sm">
                npm install @otelguard/sdk
              </pre>
              <Button
                variant="ghost"
                size="icon"
                className="absolute top-2 right-2"
                onClick={() => handleCopy('npm install @otelguard/sdk')}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <div className="space-y-2">
            <Label>Go</Label>
            <div className="relative">
              <pre className="bg-muted p-3 rounded-lg font-mono text-sm">
                go get github.com/otelguard/otelguard-go
              </pre>
              <Button
                variant="ghost"
                size="icon"
                className="absolute top-2 right-2"
                onClick={() => handleCopy('go get github.com/otelguard/otelguard-go')}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Create API Key Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={handleCloseCreateDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {createdKey ? 'API Key Created' : 'Create API Key'}
            </DialogTitle>
            <DialogDescription>
              {createdKey
                ? 'Save this key securely. It will not be shown again.'
                : 'Create a new API key to authenticate your SDK requests.'}
            </DialogDescription>
          </DialogHeader>

          {createdKey ? (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Your API Key</Label>
                <div className="flex items-center gap-2">
                  <Input
                    readOnly
                    type={showKey ? 'text' : 'password'}
                    value={createdKey}
                    className="font-mono text-sm"
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => setShowKey(!showKey)}
                    title={showKey ? 'Hide key' : 'Show key'}
                  >
                    {showKey ? (
                      <EyeOff className="h-4 w-4" />
                    ) : (
                      <Eye className="h-4 w-4" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleCopy(createdKey)}
                    title="Copy key"
                  >
                    <Copy className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-3">
                <p className="text-sm text-yellow-800 dark:text-yellow-200">
                  ⚠️ Make sure to copy your API key now. You won't be able to see it again!
                </p>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="key-name">Key Name</Label>
                <Input
                  id="key-name"
                  placeholder="e.g., Production API Key"
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      handleCreateKey();
                    }
                  }}
                />
              </div>
            </div>
          )}

          <DialogFooter>
            {createdKey ? (
              <Button onClick={handleCloseCreateDialog}>Done</Button>
            ) : (
              <>
                <Button
                  variant="outline"
                  onClick={handleCloseCreateDialog}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleCreateKey}
                  disabled={createAPIKey.isPending}
                >
                  {createAPIKey.isPending ? 'Creating...' : 'Create Key'}
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Revoke API Key Confirmation Dialog */}
      <AlertDialog open={!!keyToRevoke} onOpenChange={() => setKeyToRevoke(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API Key?</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to revoke the API key "{keyToRevoke?.name}"?
              This action cannot be undone and any applications using this key will
              lose access immediately.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleRevoke}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {revokeAPIKey.isPending ? 'Revoking...' : 'Revoke Key'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
