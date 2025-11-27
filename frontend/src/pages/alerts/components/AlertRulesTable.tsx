import { AlertRule } from '@/api/alerts';
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
import { Edit, Trash2 } from 'lucide-react';

interface AlertRulesTableProps {
  rules: AlertRule[];
  isLoading: boolean;
  onEdit: (ruleId: string) => void;
  onDelete: (ruleId: string) => void;
}

export function AlertRulesTable({
  rules,
  isLoading,
  onEdit,
  onDelete,
}: AlertRulesTableProps) {
  if (isLoading) {
    return <div className="text-center py-8">Loading...</div>;
  }

  if (rules.length === 0) {
    return (
      <div className="text-center py-12 border rounded-lg">
        <p className="text-muted-foreground">No alert rules configured</p>
        <p className="text-sm text-muted-foreground mt-2">
          Create your first alert rule to start monitoring metrics
        </p>
      </div>
    );
  }

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

  return (
    <div className="border rounded-lg">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead>Metric</TableHead>
            <TableHead>Condition</TableHead>
            <TableHead>Severity</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Channels</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rules.map((rule) => (
            <TableRow key={rule.id}>
              <TableCell className="font-medium">{rule.name}</TableCell>
              <TableCell>
                <span className="capitalize">{rule.metric_type}</span>
              </TableCell>
              <TableCell>
                {rule.operator.toUpperCase()} {rule.threshold_value}
              </TableCell>
              <TableCell>
                <Badge variant={getSeverityColor(rule.severity) as any}>
                  {rule.severity}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge variant={rule.enabled ? 'default' : 'secondary'}>
                  {rule.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </TableCell>
              <TableCell>
                <span className="text-sm text-muted-foreground">
                  {rule.notification_channels.length} channel(s)
                </span>
              </TableCell>
              <TableCell className="text-right">
                <div className="flex justify-end gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onEdit(rule.id)}
                  >
                    <Edit className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onDelete(rule.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
