import { useParams, Link } from 'react-router-dom';
import { useScore } from '@/api/scores';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { formatDate } from '@/lib/utils';
import {
  ArrowLeft,
  Calendar,
  Hash,
  Tag,
  MessageSquare,
  ExternalLink,
} from 'lucide-react';

function ScoreDetailPage() {
  const { scoreId } = useParams<{ scoreId: string }>();
  const { data: score, isLoading, error } = useScore(scoreId!);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-2"></div>
          <p className="text-muted-foreground">Loading score...</p>
        </div>
      </div>
    );
  }

  if (error || !score) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <h3 className="text-lg font-semibold text-destructive">Score not found</h3>
          <p className="text-muted-foreground">The score you're looking for doesn't exist</p>
        </div>
      </div>
    );
  }

  const formatScoreValue = () => {
    if (score.dataType === 'boolean') {
      return score.value === 1 ? 'True' : 'False';
    }
    if (score.dataType === 'categorical') {
      return score.stringValue || score.value.toString();
    }
    return score.value.toFixed(4);
  };

  const getSourceBadgeVariant = (source: string) => {
    switch (source) {
      case 'api': return 'default';
      case 'llm_judge': return 'secondary';
      case 'human': return 'outline';
      case 'user_feedback': return 'destructive';
      default: return 'default';
    }
  };

  const getDataTypeBadgeVariant = (dataType: string) => {
    switch (dataType) {
      case 'numeric': return 'default';
      case 'boolean': return 'secondary';
      case 'categorical': return 'outline';
      default: return 'default';
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/scores">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Scores
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">{score.name}</h1>
          <p className="text-muted-foreground">Score Details</p>
        </div>
      </div>

      {/* Score Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Value</CardTitle>
            <Hash className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold font-mono">
              {formatScoreValue()}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Source</CardTitle>
            <Tag className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <Badge variant={getSourceBadgeVariant(score.source)} className="text-sm">
              {score.source.replace('_', ' ')}
            </Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Data Type</CardTitle>
          </CardHeader>
          <CardContent>
            <Badge variant={getDataTypeBadgeVariant(score.dataType)} className="text-sm">
              {score.dataType}
            </Badge>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Created</CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-sm">
              {formatDate(score.createdAt)}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Score Details */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Basic Information */}
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm font-medium text-muted-foreground">Score ID</label>
              <p className="font-mono text-sm">{score.id}</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Project ID</label>
              <p className="font-mono text-sm">{score.projectId}</p>
            </div>

            <div>
              <label className="text-sm font-medium text-muted-foreground">Trace ID</label>
              <div className="flex items-center gap-2">
                <p className="font-mono text-sm">{score.traceId}</p>
                <Link to={`/traces/${score.traceId}`}>
                  <Button variant="ghost" size="sm">
                    <ExternalLink className="h-4 w-4" />
                  </Button>
                </Link>
              </div>
            </div>

            {score.spanId && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Span ID</label>
                <p className="font-mono text-sm">{score.spanId}</p>
              </div>
            )}

            {score.configId && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Config ID</label>
                <p className="font-mono text-sm">{score.configId}</p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Additional Information */}
        <Card>
          <CardHeader>
            <CardTitle>Additional Information</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {score.comment && (
              <div>
                <label className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                  <MessageSquare className="h-4 w-4" />
                  Comment
                </label>
                <p className="text-sm bg-muted p-3 rounded-md">{score.comment}</p>
              </div>
            )}

            {score.dataType === 'categorical' && score.stringValue && (
              <div>
                <label className="text-sm font-medium text-muted-foreground">Category</label>
                <Badge variant="outline" className="mt-1">
                  {score.stringValue}
                </Badge>
              </div>
            )}

            <Separator />

            <div className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <label className="text-muted-foreground">Raw Value</label>
                <p className="font-mono">{score.value}</p>
              </div>

              {score.stringValue && (
                <div>
                  <label className="text-muted-foreground">String Value</label>
                  <p className="font-mono break-all">{score.stringValue}</p>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

export default ScoreDetailPage;
