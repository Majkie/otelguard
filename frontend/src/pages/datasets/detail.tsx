import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { format } from 'date-fns';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
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
import {
  ArrowLeft,
  Plus,
  Trash2,
  Upload,
  FileJson,
  FileText,
  Pencil,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import {
  useDataset,
  useDatasetItems,
  useUpdateDataset,
  useDeleteDataset,
  useCreateDatasetItem,
  useDeleteDatasetItem,
  useImportDataset,
  type DatasetItem,
} from '@/api/datasets';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

export function DatasetDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [addItemDialogOpen, setAddItemDialogOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [deleteItemDialogOpen, setDeleteItemDialogOpen] = useState(false);
  const [selectedItem, setSelectedItem] = useState<DatasetItem | null>(null);

  const [editName, setEditName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [itemInput, setItemInput] = useState('{}');
  const [itemExpectedOutput, setItemExpectedOutput] = useState('');
  const [itemMetadata, setItemMetadata] = useState('{}');
  const [importFormat, setImportFormat] = useState<'json' | 'csv'>('json');
  const [importData, setImportData] = useState('');

  const limit = 20;
  const { data: dataset, isLoading: loadingDataset } = useDataset(id || '');
  const { data: itemsData, isLoading: loadingItems } = useDatasetItems(
    id || '',
    { limit, offset: page * limit }
  );
  const updateDataset = useUpdateDataset();
  const deleteDataset = useDeleteDataset();
  const createItem = useCreateDatasetItem();
  const deleteItem = useDeleteDatasetItem();
  const importDataset = useImportDataset();

  const handleEdit = () => {
    if (!dataset) return;
    setEditName(dataset.name);
    setEditDescription(dataset.description || '');
    setEditDialogOpen(true);
  };

  const handleSaveEdit = async () => {
    if (!id || !editName.trim()) return;

    try {
      await updateDataset.mutateAsync({
        id,
        input: {
          name: editName,
          description: editDescription,
        },
      });
      setEditDialogOpen(false);
    } catch (err) {
      console.error('Failed to update dataset:', err);
    }
  };

  const handleDelete = async () => {
    if (!id) return;

    try {
      await deleteDataset.mutateAsync(id);
      navigate('/datasets');
    } catch (err) {
      console.error('Failed to delete dataset:', err);
    }
  };

  const handleAddItem = async () => {
    if (!id) return;

    try {
      const parsedInput = JSON.parse(itemInput);
      const parsedExpectedOutput = itemExpectedOutput
        ? JSON.parse(itemExpectedOutput)
        : undefined;
      const parsedMetadata = JSON.parse(itemMetadata);

      await createItem.mutateAsync({
        datasetId: id,
        input: parsedInput,
        expectedOutput: parsedExpectedOutput,
        metadata: parsedMetadata,
      });

      setAddItemDialogOpen(false);
      setItemInput('{}');
      setItemExpectedOutput('');
      setItemMetadata('{}');
    } catch (err) {
      console.error('Failed to create item:', err);
      alert('Invalid JSON format. Please check your input.');
    }
  };

  const handleDeleteItem = async () => {
    if (!selectedItem || !id) return;

    try {
      await deleteItem.mutateAsync({
        itemId: selectedItem.id,
        datasetId: id,
      });
      setDeleteItemDialogOpen(false);
      setSelectedItem(null);
    } catch (err) {
      console.error('Failed to delete item:', err);
    }
  };

  const handleImport = async () => {
    if (!id || !importData.trim()) return;

    try {
      await importDataset.mutateAsync({
        datasetId: id,
        format: importFormat,
        data: importData,
      });
      setImportDialogOpen(false);
      setImportData('');
    } catch (err) {
      console.error('Failed to import data:', err);
      alert('Failed to import data. Please check the format.');
    }
  };

  if (loadingDataset) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-muted-foreground">Loading dataset...</p>
      </div>
    );
  }

  if (!dataset) {
    return (
      <div className="flex items-center justify-center h-96">
        <p className="text-destructive">Dataset not found</p>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/datasets')}
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back
          </Button>
          <div>
            <h1 className="text-3xl font-bold">{dataset.name}</h1>
            {dataset.description && (
              <p className="text-muted-foreground mt-1">{dataset.description}</p>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={handleEdit}>
            <Pencil className="h-4 w-4 mr-2" />
            Edit
          </Button>
          <Button
            variant="destructive"
            onClick={() => setDeleteDialogOpen(true)}
          >
            <Trash2 className="h-4 w-4 mr-2" />
            Delete
          </Button>
        </div>
      </div>

      {/* Dataset Info */}
      <Card>
        <CardHeader>
          <CardTitle>Dataset Information</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-3 gap-4">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                Total Items
              </dt>
              <dd className="text-2xl font-bold">
                {itemsData?.total ?? dataset.itemCount ?? 0}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                Created
              </dt>
              <dd className="text-lg">
                {format(new Date(dataset.createdAt), 'MMM d, yyyy')}
              </dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">
                Last Updated
              </dt>
              <dd className="text-lg">
                {format(new Date(dataset.updatedAt), 'MMM d, yyyy')}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Items Section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Dataset Items</CardTitle>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setImportDialogOpen(true)}
              >
                <Upload className="h-4 w-4 mr-2" />
                Import
              </Button>
              <Button size="sm" onClick={() => setAddItemDialogOpen(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add Item
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loadingItems ? (
            <div className="flex items-center justify-center h-64">
              <p className="text-muted-foreground">Loading items...</p>
            </div>
          ) : itemsData && itemsData.data.length > 0 ? (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Input</TableHead>
                    <TableHead>Expected Output</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {itemsData.data.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="max-w-md">
                        <pre className="text-xs bg-muted p-2 rounded overflow-auto max-h-24">
                          {JSON.stringify(item.input, null, 2)}
                        </pre>
                      </TableCell>
                      <TableCell className="max-w-md">
                        {item.expectedOutput ? (
                          <pre className="text-xs bg-muted p-2 rounded overflow-auto max-h-24">
                            {JSON.stringify(item.expectedOutput, null, 2)}
                          </pre>
                        ) : (
                          <span className="text-muted-foreground">—</span>
                        )}
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {format(new Date(item.createdAt), 'MMM d, yyyy')}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setSelectedItem(item);
                            setDeleteItemDialogOpen(true);
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
              {itemsData.total > limit && (
                <div className="flex items-center justify-between mt-4">
                  <p className="text-sm text-muted-foreground">
                    Showing {page * limit + 1} to{' '}
                    {Math.min((page + 1) * limit, itemsData.total)} of{' '}
                    {itemsData.total} items
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
                      disabled={(page + 1) * limit >= itemsData.total}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </>
          ) : (
            <div className="flex flex-col items-center justify-center h-64 text-center">
              <FileText className="h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">No items yet</h3>
              <p className="text-muted-foreground mb-4">
                Add items to this dataset to use it in experiments
              </p>
              <div className="flex gap-2">
                <Button onClick={() => setAddItemDialogOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Add Item
                </Button>
                <Button
                  variant="outline"
                  onClick={() => setImportDialogOpen(true)}
                >
                  <Upload className="h-4 w-4 mr-2" />
                  Import Items
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Dataset Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Dataset</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-description">Description</Label>
              <Textarea
                id="edit-description"
                value={editDescription}
                onChange={(e) => setEditDescription(e.target.value)}
                rows={3}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleSaveEdit}
              disabled={!editName.trim() || updateDataset.isPending}
            >
              {updateDataset.isPending ? 'Saving...' : 'Save Changes'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dataset Dialog */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Dataset</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this dataset? All items will be
              permanently removed. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
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

      {/* Add Item Dialog */}
      <Dialog open={addItemDialogOpen} onOpenChange={setAddItemDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Add Dataset Item</DialogTitle>
            <DialogDescription>
              Add a new test case to this dataset
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="item-input">Input (JSON)</Label>
              <Textarea
                id="item-input"
                value={itemInput}
                onChange={(e) => setItemInput(e.target.value)}
                rows={6}
                className="font-mono text-sm"
                placeholder='{"prompt": "What is the capital of France?"}'
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="item-expected">
                Expected Output (JSON, optional)
              </Label>
              <Textarea
                id="item-expected"
                value={itemExpectedOutput}
                onChange={(e) => setItemExpectedOutput(e.target.value)}
                rows={4}
                className="font-mono text-sm"
                placeholder='{"answer": "Paris"}'
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="item-metadata">Metadata (JSON, optional)</Label>
              <Textarea
                id="item-metadata"
                value={itemMetadata}
                onChange={(e) => setItemMetadata(e.target.value)}
                rows={3}
                className="font-mono text-sm"
                placeholder='{"category": "geography"}'
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setAddItemDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button onClick={handleAddItem} disabled={createItem.isPending}>
              {createItem.isPending ? 'Adding...' : 'Add Item'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Import Dialog */}
      <Dialog open={importDialogOpen} onOpenChange={setImportDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Import Dataset Items</DialogTitle>
            <DialogDescription>
              Import multiple items from JSON or CSV format
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="import-format">Format</Label>
              <Select
                value={importFormat}
                onValueChange={(value: 'json' | 'csv') =>
                  setImportFormat(value)
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="json">
                    <div className="flex items-center gap-2">
                      <FileJson className="h-4 w-4" />
                      JSON
                    </div>
                  </SelectItem>
                  <SelectItem value="csv">
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4" />
                      CSV
                    </div>
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="import-data">Data</Label>
              <Textarea
                id="import-data"
                value={importData}
                onChange={(e) => setImportData(e.target.value)}
                rows={12}
                className="font-mono text-sm"
                placeholder={
                  importFormat === 'json'
                    ? '[{"input": {...}, "expectedOutput": {...}}]'
                    : 'input,expectedOutput\n"value1","value2"'
                }
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setImportDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleImport}
              disabled={!importData.trim() || importDataset.isPending}
            >
              {importDataset.isPending ? 'Importing...' : 'Import'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Item Dialog */}
      <AlertDialog
        open={deleteItemDialogOpen}
        onOpenChange={setDeleteItemDialogOpen}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Item</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this item? This action cannot be
              undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setSelectedItem(null)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteItem}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={deleteItem.isPending}
            >
              {deleteItem.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
