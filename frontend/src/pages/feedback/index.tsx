import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useFeedbackList, useDeleteFeedback, FeedbackFilter } from '@/api/feedback';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
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
import { ThumbsUp, ThumbsDown, Star, MessageSquare, BarChart3, Plus, Search, Filter, Trash2 } from 'lucide-react';

function FeedbackPage() {
  const [filters, setFilters] = useState<FeedbackFilter>({
    limit: 50,
    offset: 0,
  });

  const [searchTerm, setSearchTerm] = useState('');
  const [selectedFeedback, setSelectedFeedback] = useState<any>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [feedbackToDelete, setFeedbackToDelete] = useState<string | null>(null);

  const { data, isLoading } = useFeedbackList(filters);
  const deleteFeedback = useDeleteFeedback();

  const handleFilterChange = (key: keyof FeedbackFilter, value: any) => {
    setFilters(prev => ({
      ...prev,
      [key]: value || undefined,
      offset: 0, // Reset pagination when filters change
    }));
  };

  const handlePageChange = (direction: 'next' | 'prev') => {
    if (!data) return;

    const newOffset = direction === 'next'
      ? filters.offset! + filters.limit!
      : Math.max(0, filters.offset! - filters.limit!);

    setFilters(prev => ({
      ...prev,
      offset: newOffset,
    }));
  };

  const handleDeleteClick = (id: string) => {
    setFeedbackToDelete(id);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (feedbackToDelete) {
      await deleteFeedback.mutateAsync(feedbackToDelete);
      setDeleteDialogOpen(false);
      setFeedbackToDelete(null);
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString();
  };

  const renderThumbs = (thumbsUp?: boolean | null) => {
    if (thumbsUp === true) {
      return <ThumbsUp className="h-4 w-4 text-green-600" />;
    } else if (thumbsUp === false) {
      return <ThumbsDown className="h-4 w-4 text-red-600" />;
    }
    return null;
  };

  const renderRating = (rating?: number) => {
    if (!rating) return null;

    return (
      <div className="flex items-center gap-1">
        <Star className="h-4 w-4 fill-yellow-400 text-yellow-400" />
        <span className="text-sm">{rating}</span>
      </div>
    );
  };

  const getItemTypeColor = (itemType: string) => {
    switch (itemType) {
      case 'trace': return 'bg-blue-100 text-blue-800';
      case 'session': return 'bg-green-100 text-green-800';
      case 'span': return 'bg-purple-100 text-purple-800';
      case 'prompt': return 'bg-orange-100 text-orange-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">User Feedback</h1>
          <p className="text-muted-foreground">
            View and manage user feedback across your application
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" asChild>
            <Link to="/feedback/analytics">
              <BarChart3 className="h-4 w-4 mr-2" />
              Analytics
            </Link>
          </Button>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-5 w-5" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Search</label>
              <div className="relative">
                <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search comments..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Item Type</label>
              <Select
                value={filters.itemType || ''}
                onValueChange={(value) => handleFilterChange('itemType', value)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All types" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="trace">Traces</SelectItem>
                  <SelectItem value="session">Sessions</SelectItem>
                  <SelectItem value="span">Spans</SelectItem>
                  <SelectItem value="prompt">Prompts</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Thumbs</label>
              <Select
                value={filters.thumbsUp === undefined ? '' : filters.thumbsUp.toString()}
                onValueChange={(value) => handleFilterChange('thumbsUp', value === '' ? undefined : value === 'true')}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="true">Up</SelectItem>
                  <SelectItem value="false">Down</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Rating</label>
              <Select
                value={filters.rating?.toString() || ''}
                onValueChange={(value) => handleFilterChange('rating', value === '' ? undefined : parseInt(value))}
              >
                <SelectTrigger>
                  <SelectValue placeholder="All ratings" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="5">5 stars</SelectItem>
                  <SelectItem value="4">4 stars</SelectItem>
                  <SelectItem value="3">3 stars</SelectItem>
                  <SelectItem value="2">2 stars</SelectItem>
                  <SelectItem value="1">1 star</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Feedback List */}
      <Card>
        <CardHeader>
          <CardTitle>Feedback Items</CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="text-center py-8">Loading feedback...</div>
          ) : data?.feedback && data.feedback.length > 0 ? (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Type</TableHead>
                    <TableHead>Item ID</TableHead>
                    <TableHead>Feedback</TableHead>
                    <TableHead>Comment</TableHead>
                    <TableHead>Date</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.feedback.map((feedback) => (
                    <TableRow key={feedback.id}>
                      <TableCell>
                        <Badge className={getItemTypeColor(feedback.itemType)}>
                          {feedback.itemType}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {feedback.itemId.substring(0, 8)}...
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {renderThumbs(feedback.thumbsUp)}
                          {renderRating(feedback.rating)}
                        </div>
                      </TableCell>
                      <TableCell>
                        {feedback.comment ? (
                          <div className="flex items-center gap-1">
                            <MessageSquare className="h-4 w-4 text-muted-foreground" />
                            <span className="truncate max-w-[200px]" title={feedback.comment}>
                              {feedback.comment}
                            </span>
                          </div>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                      <TableCell>{formatDate(feedback.createdAt)}</TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Dialog>
                            <DialogTrigger asChild>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setSelectedFeedback(feedback)}
                              >
                                View
                              </Button>
                            </DialogTrigger>
                            <DialogContent className="max-w-2xl">
                              <DialogHeader>
                                <DialogTitle>Feedback Details</DialogTitle>
                              </DialogHeader>
                              {selectedFeedback && (
                                <div className="space-y-4">
                                  <div className="grid grid-cols-2 gap-4">
                                    <div>
                                      <label className="text-sm font-medium">Type</label>
                                      <div className="mt-1">
                                        <Badge className={getItemTypeColor(selectedFeedback.itemType)}>
                                          {selectedFeedback.itemType}
                                        </Badge>
                                      </div>
                                    </div>
                                    <div>
                                      <label className="text-sm font-medium">Date</label>
                                      <div className="mt-1">{formatDate(selectedFeedback.createdAt)}</div>
                                    </div>
                                  </div>

                                  <div>
                                    <label className="text-sm font-medium">Item ID</label>
                                    <div className="mt-1 font-mono text-sm bg-gray-50 p-2 rounded">
                                      {selectedFeedback.itemId}
                                    </div>
                                  </div>

                                  {selectedFeedback.traceId && (
                                    <div>
                                      <label className="text-sm font-medium">Trace ID</label>
                                      <div className="mt-1 font-mono text-sm bg-gray-50 p-2 rounded">
                                        {selectedFeedback.traceId}
                                      </div>
                                    </div>
                                  )}

                                  <div className="grid grid-cols-2 gap-4">
                                    <div>
                                      <label className="text-sm font-medium">Thumbs</label>
                                      <div className="mt-1 flex items-center gap-2">
                                        {renderThumbs(selectedFeedback.thumbsUp)}
                                        {selectedFeedback.thumbsUp === true && <span>Good</span>}
                                        {selectedFeedback.thumbsUp === false && <span>Poor</span>}
                                        {selectedFeedback.thumbsUp === null && <span>No opinion</span>}
                                      </div>
                                    </div>
                                    <div>
                                      <label className="text-sm font-medium">Rating</label>
                                      <div className="mt-1">
                                        {renderRating(selectedFeedback.rating) || <span className="text-muted-foreground">Not rated</span>}
                                      </div>
                                    </div>
                                  </div>

                                  {selectedFeedback.comment && (
                                    <div>
                                      <label className="text-sm font-medium">Comment</label>
                                      <div className="mt-1 p-3 bg-gray-50 rounded border">
                                        {selectedFeedback.comment}
                                      </div>
                                    </div>
                                  )}

                                  {selectedFeedback.metadata && Object.keys(selectedFeedback.metadata).length > 0 && (
                                    <div>
                                      <label className="text-sm font-medium">Metadata</label>
                                      <div className="mt-1">
                                        <pre className="text-xs bg-gray-50 p-2 rounded border overflow-x-auto">
                                          {JSON.stringify(selectedFeedback.metadata, null, 2)}
                                        </pre>
                                      </div>
                                    </div>
                                  )}
                                </div>
                              )}
                            </DialogContent>
                          </Dialog>

                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDeleteClick(feedback.id)}
                            disabled={deleteFeedback.isPending}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              {/* Pagination */}
              <div className="flex items-center justify-between mt-4">
                <div className="text-sm text-muted-foreground">
                  Showing {data.feedback.length} of {data.total} feedback items
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handlePageChange('prev')}
                    disabled={filters.offset === 0}
                  >
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handlePageChange('next')}
                    disabled={(filters.offset || 0) + (filters.limit || 50) >= (data.total || 0)}
                  >
                    Next
                  </Button>
                </div>
              </div>
            </>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              No feedback found matching your criteria
            </div>
          )}
        </CardContent>
      </Card>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Feedback</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete this feedback? This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDeleteConfirm}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteFeedback.isPending ? 'Deleting...' : 'Delete'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export default FeedbackPage;
