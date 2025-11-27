import { useAlertHistory, useAcknowledgeAlert, type AlertHistory } from '@/api/alerts';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Badge } from '@/components/ui/badge';
import { Check } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

interface AlertHistoryPageProps {
  projectId: string;
  userId: string;
}

export function AlertHistoryPage({ projectId, userId }: AlertHistoryPageProps) {
  const { data, isLoading } = useAlertHistory(projectId);
  const acknowledgeMutation = useAcknowledgeAlert(projectId);

  const handleAcknowledge = async (alertId: string) => {
    await acknowledgeMutation.mutateAsync({ alertId, userId });
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'destructive';
      case 'error':
        return 'destructive';
      case 'warning':
        return 'default';
      case 'info':
        return 'secondary';
      default:
        return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'firing':
        return 'destructive';
      case 'acknowledged':
        return 'default';
      case 'resolved':
        return 'secondary';
      default:
        return 'default';
    }
  };

  if (isLoading) {
    return <div className="text-center py-8">Loading...</div>;
  }

  const alerts = data?.data || [];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Alert History</h1>
        <p className="text-muted-foreground mt-2">
          View and manage triggered alerts
        </p>
      </div>

      {alerts.length === 0 ? (
        <div className="text-center py-12 border rounded-lg">
          <p className="text-muted-foreground">No alerts triggered yet</p>
        </div>
      ) : (
        <div className="border rounded-lg">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Severity</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Value</TableHead>
                <TableHead>Fired</TableHead>
                <TableHead>Message</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {alerts.map((alert) => (
                <TableRow key={alert.id}>
                  <TableCell>
                    <Badge variant={getSeverityColor(alert.severity) as any}>
                      {alert.severity}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={getStatusColor(alert.status) as any}>
                      {alert.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {alert.metric_value !== undefined && alert.threshold_value !== undefined ? (
                      <span className="text-sm">
                        {alert.metric_value.toFixed(2)} / {alert.threshold_value.toFixed(2)}
                      </span>
                    ) : (
                      <span className="text-sm text-muted-foreground">N/A</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">
                      {formatDistanceToNow(new Date(alert.fired_at), {
                        addSuffix: true,
                      })}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-sm">{alert.message || 'No message'}</span>
                  </TableCell>
                  <TableCell className="text-right">
                    {alert.status === 'firing' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleAcknowledge(alert.id)}
                        disabled={acknowledgeMutation.isPending}
                      >
                        <Check className="h-4 w-4 mr-2" />
                        Acknowledge
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
