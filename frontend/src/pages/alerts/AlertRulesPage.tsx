import { useState } from 'react';
import { Plus } from 'lucide-react';
import { useAlertRules, useDeleteAlertRule } from '@/api/alerts';
import { Button } from '@/components/ui/button';
import { AlertRuleDialog } from './components/AlertRuleDialog';
import { AlertRulesTable } from './components/AlertRulesTable';

interface AlertRulesPageProps {
  projectId: string;
}

export function AlertRulesPage({ projectId }: AlertRulesPageProps) {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);

  const { data, isLoading } = useAlertRules(projectId);
  const deleteMutation = useDeleteAlertRule(projectId);

  const handleEdit = (ruleId: string) => {
    setSelectedRuleId(ruleId);
    setIsDialogOpen(true);
  };

  const handleDelete = async (ruleId: string) => {
    if (confirm('Are you sure you want to delete this alert rule?')) {
      await deleteMutation.mutateAsync(ruleId);
    }
  };

  const handleCloseDialog = () => {
    setIsDialogOpen(false);
    setSelectedRuleId(null);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Alert Rules</h1>
          <p className="text-muted-foreground mt-2">
            Configure alert rules to monitor your application metrics
          </p>
        </div>
        <Button onClick={() => setIsDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          Create Alert Rule
        </Button>
      </div>

      <AlertRulesTable
        rules={data?.data || []}
        isLoading={isLoading}
        onEdit={handleEdit}
        onDelete={handleDelete}
      />

      <AlertRuleDialog
        projectId={projectId}
        ruleId={selectedRuleId}
        open={isDialogOpen}
        onClose={handleCloseDialog}
      />
    </div>
  );
}
