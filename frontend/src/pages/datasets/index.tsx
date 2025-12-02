import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Plus,
  Database,
  Search,
  Trash2,
  ChevronLeft,
  ChevronRight,
  FileText,
} from 'lucide-react';
import {
  useDatasets,
  useCreateDataset,
  useDeleteDataset,
  type Dataset,
} from '@/api/datasets';
import {useProjectContext} from "@/contexts/project-context.tsx";

export function DatasetsPage() {
  const navigate = useNavigate();
  const { selectedProject } = useProjectContext();
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [selectedDataset, setSelectedDataset] = useState<Dataset | null>(null);
  const [newDatasetName, setNewDatasetName] = useState('');
  const [newDatasetDescription, setNewDatasetDescription] = useState('');

  const limit = 20;
  const { data, isLoading, error } = useDatasets({
    projectId: selectedProject?.id || '',
    limit,
    offset: page * limit,
  });
  const createDataset = useCreateDataset();
  const deleteDataset = useDeleteDataset();

  const filteredDatasets = data?.data.filter(
    (dataset) =>
      dataset.name.toLowerCase().includes(search.toLowerCase()) ||
      dataset.description?.toLowerCase().includes(search.toLowerCase())
  );

  const handleCreate = async () => {
    if (!newDatasetName.trim() || !selectedProject) return;

    try {
      const result = await createDataset.mutateAsync({
        projectId: selectedProject.id,
        name: newDatasetName,
        description: newDatasetDescription,
      });
      setCreateDialogOpen(false);
      setNewDatasetName('');
      setNewDatasetDescription('');
      // Navigate to the dataset detail page
      navigate(`/datasets/${result.id}`);
    } catch (err) {
      console.error('Failed to create dataset:', err);
    }
  };

  const handleDelete = async () => {
    if (!selectedDataset) return;

    try {
      await deleteDataset.mutateAsync(selectedDataset.id);
      setDeleteDialogOpen(false);
      setSelectedDataset(null);
    } catch (err) {
      console.error('Failed to delete dataset:', err);
    }
  };

  if (error) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-destructive">Failed to load datasets</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Database className="h-8 w-8" />
            Datasets
          </h1>
          <p className="text-muted-foreground mt-1">
            Manage test datasets for evaluations and experiments
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          New Dataset
        </Button>
      </div>

      {/* Search and Filters */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-4">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search datasets..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Datasets Table */}
      <Card>
        <CardContent className="pt-6">
          {isLoading ? (
            <div className="flex items-center justify-center h-64">
              <p className="text-muted-foreground">Loading datasets...</p>
            </div>
          ) : filteredDatasets && filteredDatasets.length > 0 ? (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead className="text-right">Items</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredDatasets.map((dataset) => (
                    <TableRow key={dataset.id}>
                      <TableCell>
                        <Link
                          to={`/datasets/${dataset.id}`}
                          className="font-medium hover:underline flex items-center gap-2"
                        >
                          <FileText className="h-4 w-4" />
                          {dataset.name}
                        </Link>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {dataset.description || '—'}
                      </TableCell>
                      <TableCell className="text-right">
                        {dataset.itemCount ?? 0}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {format(new Date(dataset.createdAt), 'MMM d, yyyy')}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.preventDefault();
                            setSelectedDataset(dataset);
                            setDeleteDialogOpen(true);
                          }}
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              {data && data.total > limit && (
                <div className="flex items-center justify-between mt-4">
                  <p className="text-sm text-muted-foreground">
                    Showing {page * limit + 1} to{' '}
                    {Math.min((page + 1) * limit, data.total)} of {data.total}{' '}
                    datasets
                  </p>
                  <div className="flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage(Math.max(0, page - 1))}
                      disabled={page === 0}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setPage(page + 1)}
                      disabled={(page + 1) * limit >= data.total}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col items-center justify-center h-64 text-center">
              <Database className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">No datasets found</h3>
              <p className="text-muted-foreground mb-4">
                {search
                  ? 'Try adjusting your search criteria'
                  : 'Get started by creating your first dataset'}
              </p>
              {!search && (
                <Button onClick={() => setCreateDialogOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Dataset
                </Button>
              )}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Create Dataset Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Dataset</DialogTitle>
            <DialogDescription>
              Create a new dataset to organize test cases for evaluation
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                placeholder="Customer support evaluation"
                value={newDatasetName}
                onChange={(e) => setNewDatasetName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description (optional)</Label>
              <Textarea
                id="description"
                placeholder="Test cases for customer support responses..."
                value={newDatasetDescription}
                onChange={(e) => setNewDatasetDescription(e.target.value)}
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setCreateDialogOpen(false);
                setNewDatasetName('');
                setNewDatasetDescription('');
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreate}
              disabled={!newDatasetName.trim() || createDataset.isPending}
            >
              {createDataset.isPending ? 'Creating...' : 'Create Dataset'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Dataset</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete "{selectedDataset?.name}"? This
              will also delete all items in this dataset. This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setSelectedDataset(null)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteDataset.isPending}
            >
              {deleteDataset.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
