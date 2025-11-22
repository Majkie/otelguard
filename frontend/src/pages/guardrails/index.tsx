import { useState } from 'react';
import { format } from 'date-fns';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Plus,
  Shield,
  Search,
  MoreHorizontal,
  Pencil,
  Trash2,
  Play,
  ChevronLeft,
  ChevronRight,
  ShieldCheck,
  ShieldAlert,
  ShieldX,
} from 'lucide-react';
import {
  useGuardrailPolicies,
  useCreatePolicy,
  useUpdatePolicy,
  useDeletePolicy,
  type GuardrailPolicy,
} from '@/api/guardrails';

export function GuardrailsPage() {
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<GuardrailPolicy | null>(null);
  const [newPolicyName, setNewPolicyName] = useState('');
  const [newPolicyDescription, setNewPolicyDescription] = useState('');
  const [newPolicyEnabled, setNewPolicyEnabled] = useState(true);
  const [newPolicyPriority, setNewPolicyPriority] = useState(0);

  const limit = 20;
  const { data, isLoading, error } = useGuardrailPolicies({
    limit,
    offset: page * limit,
  });
  const createPolicy = useCreatePolicy();
  const updatePolicy = useUpdatePolicy();
  const deletePolicy = useDeletePolicy();

  const filteredPolicies = data?.data.filter(
    (policy) =>
      policy.name.toLowerCase().includes(search.toLowerCase()) ||
      policy.description?.toLowerCase().includes(search.toLowerCase())
  );

  const handleCreate = async () => {
    if (!newPolicyName.trim()) return;

    try {
      await createPolicy.mutateAsync({
        name: newPolicyName,
        description: newPolicyDescription,
        enabled: newPolicyEnabled,
        priority: newPolicyPriority,
      });
      setCreateDialogOpen(false);
      setNewPolicyName('');
      setNewPolicyDescription('');
      setNewPolicyEnabled(true);
      setNewPolicyPriority(0);
    } catch {
      // Error handled by mutation
    }
  };

  const handleToggleEnabled = async (policy: GuardrailPolicy) => {
    try {
      await updatePolicy.mutateAsync({
        id: policy.id,
        data: { enabled: !policy.enabled },
      });
    } catch {
      // Error handled by mutation
    }
  };

  const handleDelete = async () => {
    if (!selectedPolicy) return;

    try {
      await deletePolicy.mutateAsync(selectedPolicy.id);
      setDeleteDialogOpen(false);
      setSelectedPolicy(null);
    } catch {
      // Error handled by mutation
    }
  };

  const totalPages = Math.ceil((data?.total || 0) / limit);

  if (error) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold">Guardrails</h1>
          <p className="text-muted-foreground">
            Configure policies to protect your LLM applications
          </p>
        </div>
        <Card>
          <CardContent className="pt-6">
            <p className="text-destructive">Error loading policies</p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Guardrails</h1>
          <p className="text-muted-foreground">
            Configure policies to protect your LLM applications
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Policy
        </Button>
      </div>

      {/* Search */}
      <div className="flex gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search policies..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-8 text-center text-muted-foreground">
              Loading policies...
            </div>
          ) : !filteredPolicies?.length ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Shield className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-medium">No guardrail policies</h3>
              <p className="text-muted-foreground max-w-sm mt-2">
                Create guardrail policies to detect and remediate issues like
                prompt injection, PII exposure, and toxic content.
              </p>
              <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Create Policy
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Policy</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Updated</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredPolicies.map((policy) => (
                  <TableRow key={policy.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        {policy.enabled ? (
                          <ShieldCheck className="h-5 w-5 text-green-500" />
                        ) : (
                          <ShieldX className="h-5 w-5 text-muted-foreground" />
                        )}
                        <div>
                          <p className="font-medium">{policy.name}</p>
                          <p className="text-sm text-muted-foreground">
                            {policy.description || 'No description'}
                          </p>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <Switch
                          checked={policy.enabled}
                          onCheckedChange={() => handleToggleEnabled(policy)}
                        />
                        <Badge variant={policy.enabled ? 'default' : 'secondary'}>
                          {policy.enabled ? 'Active' : 'Disabled'}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{policy.priority}</Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {format(new Date(policy.updatedAt), 'MMM d, yyyy')}
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem>
                            <Pencil className="h-4 w-4 mr-2" />
                            Edit
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <Play className="h-4 w-4 mr-2" />
                            Test
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            className="text-destructive"
                            onClick={() => {
                              setSelectedPolicy(policy);
                              setDeleteDialogOpen(true);
                            }}
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            Delete
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between px-4 py-3 border-t">
            <p className="text-sm text-muted-foreground">
              Showing {page * limit + 1} to{' '}
              {Math.min((page + 1) * limit, data?.total || 0)} of {data?.total}{' '}
              policies
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.max(0, p - 1))}
                disabled={page === 0}
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
                disabled={page >= totalPages - 1}
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </Card>

      {/* Feature overview */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <ShieldAlert className="h-4 w-4" />
              Input Validation
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Detect prompt injection, jailbreak attempts, and PII before they
            reach your LLM.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <ShieldCheck className="h-4 w-4" />
              Output Validation
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Check for toxicity, hallucinations, and ensure responses match
            expected formats.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Auto-Remediation
            </CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Automatically block, sanitize, retry, or fallback when issues are
            detected.
          </CardContent>
        </Card>
      </div>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Policy</DialogTitle>
            <DialogDescription>
              Create a guardrail policy to protect your LLM applications. You
              can add rules after creation.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="e.g., PII Protection"
                value={newPolicyName}
                onChange={(e) => setNewPolicyName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Input
                id="description"
                placeholder="Brief description of this policy"
                value={newPolicyDescription}
                onChange={(e) => setNewPolicyDescription(e.target.value)}
              />
            </div>
            <div className="flex items-center justify-between">
              <Label htmlFor="enabled">Enable policy</Label>
              <Switch
                id="enabled"
                checked={newPolicyEnabled}
                onCheckedChange={setNewPolicyEnabled}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="priority">Priority (higher = evaluated first)</Label>
              <Input
                id="priority"
                type="number"
                value={newPolicyPriority}
                onChange={(e) => setNewPolicyPriority(parseInt(e.target.value) || 0)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!newPolicyName.trim() || createPolicy.isPending}
            >
              {createPolicy.isPending ? 'Creating...' : 'Create Policy'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Policy</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{selectedPolicy?.name}&quot;?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deletePolicy.isPending}
            >
              {deletePolicy.isPending ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
