import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
  useAlertRule,
  useCreateAlertRule,
  useUpdateAlertRule,
  type CreateAlertRuleRequest,
} from '@/api/alerts';

interface AlertRuleDialogProps {
  projectId: string;
  ruleId?: string | null;
  open: boolean;
  onClose: () => void;
}

export function AlertRuleDialog({
  projectId,
  ruleId,
  open,
  onClose,
}: AlertRuleDialogProps) {
  const { data: existingRule } = useAlertRule(projectId, ruleId || '');
  const createMutation = useCreateAlertRule(projectId);
  const updateMutation = useUpdateAlertRule(projectId, ruleId || '');

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors },
  } = useForm<CreateAlertRuleRequest>({
    defaultValues: {
      enabled: true,
      metric_type: 'latency',
      condition_type: 'threshold',
      operator: 'gt',
      severity: 'warning',
      notification_channels: [],
      tags: [],
    },
  });

  useEffect(() => {
    if (existingRule) {
      reset({
        name: existingRule.name,
        description: existingRule.description,
        enabled: existingRule.enabled,
        metric_type: existingRule.metric_type,
        metric_field: existingRule.metric_field,
        condition_type: existingRule.condition_type,
        operator: existingRule.operator,
        threshold_value: existingRule.threshold_value,
        window_duration: existingRule.window_duration,
        evaluation_frequency: existingRule.evaluation_frequency,
        notification_channels: existingRule.notification_channels,
        notification_message: existingRule.notification_message,
        severity: existingRule.severity,
      });
    } else {
      reset({
        enabled: true,
        metric_type: 'latency',
        condition_type: 'threshold',
        operator: 'gt',
        severity: 'warning',
        notification_channels: [],
        tags: [],
      });
    }
  }, [existingRule, reset]);

  const onSubmit = async (data: CreateAlertRuleRequest) => {
    try {
      if (ruleId) {
        await updateMutation.mutateAsync(data);
      } else {
        await createMutation.mutateAsync(data);
      }
      onClose();
    } catch (error) {
      console.error('Failed to save alert rule:', error);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {ruleId ? 'Edit Alert Rule' : 'Create Alert Rule'}
          </DialogTitle>
          <DialogDescription>
            Configure an alert rule to monitor your application metrics
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="space-y-4">
            {/* Basic Info */}
            <div className="space-y-2">
              <Label htmlFor="name">Name *</Label>
              <Input
                id="name"
                {...register('name', { required: 'Name is required' })}
                placeholder="High Latency Alert"
              />
              {errors.name && (
                <p className="text-sm text-destructive">{errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                {...register('description')}
                placeholder="Triggers when latency exceeds threshold..."
              />
            </div>

            <div className="flex items-center space-x-2">
              <Switch
                id="enabled"
                checked={watch('enabled')}
                onCheckedChange={(checked) => setValue('enabled', checked)}
              />
              <Label htmlFor="enabled">Enable this rule</Label>
            </div>

            {/* Metric Selection */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="metric_type">Metric Type *</Label>
                <Select
                  value={watch('metric_type')}
                  onValueChange={(value) => setValue('metric_type', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="latency">Latency</SelectItem>
                    <SelectItem value="cost">Cost</SelectItem>
                    <SelectItem value="error_rate">Error Rate</SelectItem>
                    <SelectItem value="token_count">Token Count</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="severity">Severity *</Label>
                <Select
                  value={watch('severity')}
                  onValueChange={(value) => setValue('severity', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warning">Warning</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                    <SelectItem value="critical">Critical</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Condition */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="operator">Operator *</Label>
                <Select
                  value={watch('operator')}
                  onValueChange={(value) => setValue('operator', value)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="gt">Greater Than</SelectItem>
                    <SelectItem value="gte">Greater Than or Equal</SelectItem>
                    <SelectItem value="lt">Less Than</SelectItem>
                    <SelectItem value="lte">Less Than or Equal</SelectItem>
                    <SelectItem value="eq">Equal</SelectItem>
                    <SelectItem value="ne">Not Equal</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="threshold_value">Threshold Value *</Label>
                <Input
                  id="threshold_value"
                  type="number"
                  step="any"
                  {...register('threshold_value', {
                    required: 'Threshold is required',
                    valueAsNumber: true,
                  })}
                  placeholder="1000"
                />
                {errors.threshold_value && (
                  <p className="text-sm text-destructive">
                    {errors.threshold_value.message}
                  </p>
                )}
              </div>
            </div>

            {/* Time Window */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="window_duration">
                  Window Duration (seconds)
                </Label>
                <Input
                  id="window_duration"
                  type="number"
                  {...register('window_duration', { valueAsNumber: true })}
                  placeholder="300"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="evaluation_frequency">
                  Evaluation Frequency (seconds)
                </Label>
                <Input
                  id="evaluation_frequency"
                  type="number"
                  {...register('evaluation_frequency', { valueAsNumber: true })}
                  placeholder="60"
                />
              </div>
            </div>

            {/* Notification */}
            <div className="space-y-2">
              <Label htmlFor="notification_message">Notification Message</Label>
              <Textarea
                id="notification_message"
                {...register('notification_message')}
                placeholder="Custom alert message..."
              />
              <p className="text-sm text-muted-foreground">
                Leave empty to use default message
              </p>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={createMutation.isPending || updateMutation.isPending}
            >
              {createMutation.isPending || updateMutation.isPending
                ? 'Saving...'
                : ruleId
                ? 'Update'
                : 'Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
