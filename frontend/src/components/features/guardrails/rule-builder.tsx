import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Checkbox } from '@/components/ui/checkbox';
import { X, Plus, Info } from 'lucide-react';
import { Separator } from '@/components/ui/separator';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip';

export interface GuardrailRuleConfig {
  type: string;
  config: Record<string, any>;
  action: string;
  actionConfig: Record<string, any>;
  orderIndex: number;
}

interface RuleBuilderProps {
  rules: GuardrailRuleConfig[];
  onChange: (rules: GuardrailRuleConfig[]) => void;
}

const RULE_TYPES = {
  input: [
    { value: 'pii_detection', label: 'PII Detection', description: 'Detect personally identifiable information' },
    { value: 'prompt_injection', label: 'Prompt Injection', description: 'Detect prompt injection attempts' },
    { value: 'secrets_detection', label: 'Secrets Detection', description: 'Detect API keys, passwords, etc.' },
    { value: 'length_limit', label: 'Length Limit', description: 'Enforce length constraints' },
    { value: 'regex_pattern', label: 'Regex Pattern', description: 'Custom regex matching' },
    { value: 'keyword_blocker', label: 'Keyword Blocker', description: 'Block specific keywords' },
    { value: 'language_detection', label: 'Language Detection', description: 'Detect text language' },
  ],
  output: [
    { value: 'toxicity', label: 'Toxicity Detection', description: 'Detect toxic or harmful content' },
    { value: 'json_schema', label: 'JSON Schema', description: 'Validate JSON structure' },
    { value: 'format_validator', label: 'Format Validator', description: 'Validate text format' },
    { value: 'completeness', label: 'Completeness Check', description: 'Check if response is complete' },
    { value: 'relevance', label: 'Relevance Check', description: 'Check if response is relevant' },
  ],
};

const ACTIONS = [
  { value: 'block', label: 'Block', description: 'Reject the request' },
  { value: 'sanitize', label: 'Sanitize', description: 'Remove sensitive content' },
  { value: 'alert', label: 'Alert', description: 'Log and continue' },
  { value: 'retry', label: 'Retry', description: 'Retry with modifications' },
  { value: 'fallback', label: 'Fallback', description: 'Use fallback response' },
  { value: 'transform', label: 'Transform', description: 'Transform the output' },
];

const PII_TYPES = ['email', 'phone', 'ssn', 'credit_card', 'ip_address'];
const SECRET_TYPES = ['api_key', 'password', 'token', 'bearer_token', 'aws_key', 'private_key'];
const FORMAT_TYPES = ['email', 'url', 'uuid', 'date', 'ipv4', 'phone'];

export function RuleBuilder({ rules, onChange }: RuleBuilderProps) {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const addRule = () => {
    const newRule: GuardrailRuleConfig = {
      type: 'pii_detection',
      config: {},
      action: 'block',
      actionConfig: {},
      orderIndex: rules.length,
    };
    onChange([...rules, newRule]);
    setEditingIndex(rules.length);
  };

  const removeRule = (index: number) => {
    const newRules = rules.filter((_, i) => i !== index);
    // Update order indices
    newRules.forEach((rule, i) => {
      rule.orderIndex = i;
    });
    onChange(newRules);
    if (editingIndex === index) {
      setEditingIndex(null);
    }
  };

  const updateRule = (index: number, updates: Partial<GuardrailRuleConfig>) => {
    const newRules = [...rules];
    newRules[index] = { ...newRules[index], ...updates };
    onChange(newRules);
  };

  const moveRule = (index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= rules.length) return;

    const newRules = [...rules];
    [newRules[index], newRules[newIndex]] = [newRules[newIndex], newRules[index]];
    newRules.forEach((rule, i) => {
      rule.orderIndex = i;
    });
    onChange(newRules);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium">Rules</h3>
          <p className="text-sm text-muted-foreground">
            Define validation rules and remediation actions
          </p>
        </div>
        <Button onClick={addRule} size="sm">
          <Plus className="h-4 w-4 mr-2" />
          Add Rule
        </Button>
      </div>

      {rules.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">No rules configured</p>
            <Button onClick={addRule} variant="outline" className="mt-4">
              <Plus className="h-4 w-4 mr-2" />
              Add Your First Rule
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {rules.map((rule, index) => (
            <Card key={index}>
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <Badge variant="outline">Rule {index + 1}</Badge>
                    <div>
                      <CardTitle className="text-sm font-medium">
                        {RULE_TYPES.input.find((r) => r.value === rule.type)?.label ||
                          RULE_TYPES.output.find((r) => r.value === rule.type)?.label ||
                          rule.type}
                      </CardTitle>
                      <p className="text-xs text-muted-foreground">
                        Action: {rule.action}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => moveRule(index, 'up')}
                      disabled={index === 0}
                    >
                      ↑
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => moveRule(index, 'down')}
                      disabled={index === rules.length - 1}
                    >
                      ↓
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() =>
                        setEditingIndex(editingIndex === index ? null : index)
                      }
                    >
                      {editingIndex === index ? 'Collapse' : 'Edit'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => removeRule(index)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </CardHeader>

              {editingIndex === index && (
                <CardContent className="space-y-4">
                  {/* Rule Type */}
                  <div className="space-y-2">
                    <Label>Rule Type</Label>
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-2">
                        <p className="text-xs font-medium text-muted-foreground">
                          Input Validators
                        </p>
                        <Select
                          value={rule.type}
                          onValueChange={(value) =>
                            updateRule(index, { type: value, config: {} })
                          }
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {RULE_TYPES.input.map((type) => (
                              <SelectItem key={type.value} value={type.value}>
                                {type.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <p className="text-xs font-medium text-muted-foreground">
                          Output Validators
                        </p>
                        <Select
                          value={rule.type}
                          onValueChange={(value) =>
                            updateRule(index, { type: value, config: {} })
                          }
                        >
                          <SelectTrigger>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {RULE_TYPES.output.map((type) => (
                              <SelectItem key={type.value} value={type.value}>
                                {type.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  </div>

                  <Separator />

                  {/* Rule Configuration */}
                  <RuleConfigEditor
                    ruleType={rule.type}
                    config={rule.config}
                    onChange={(config) => updateRule(index, { config })}
                  />

                  <Separator />

                  {/* Action Configuration */}
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label>Remediation Action</Label>
                      <Select
                        value={rule.action}
                        onValueChange={(value) =>
                          updateRule(index, { action: value, actionConfig: {} })
                        }
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {ACTIONS.map((action) => (
                            <SelectItem key={action.value} value={action.value}>
                              <div>
                                <p>{action.label}</p>
                                <p className="text-xs text-muted-foreground">
                                  {action.description}
                                </p>
                              </div>
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <ActionConfigEditor
                      action={rule.action}
                      config={rule.actionConfig}
                      onChange={(actionConfig) =>
                        updateRule(index, { actionConfig })
                      }
                    />
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

// Rule configuration editor based on rule type
function RuleConfigEditor({
  ruleType,
  config,
  onChange,
}: {
  ruleType: string;
  config: Record<string, any>;
  onChange: (config: Record<string, any>) => void;
}) {
  const updateConfig = (key: string, value: any) => {
    onChange({ ...config, [key]: value });
  };

  switch (ruleType) {
    case 'pii_detection':
      return (
        <div className="space-y-3">
          <Label>PII Types to Detect</Label>
          <div className="grid grid-cols-3 gap-2">
            {PII_TYPES.map((type) => (
              <div key={type} className="flex items-center space-x-2">
                <Checkbox
                  id={`pii-${type}`}
                  checked={config.pii_types?.includes(type) ?? false}
                  onCheckedChange={(checked) => {
                    const current = config.pii_types || [];
                    const updated = checked
                      ? [...current, type]
                      : current.filter((t: string) => t !== type);
                    updateConfig('pii_types', updated);
                  }}
                />
                <label htmlFor={`pii-${type}`} className="text-sm">
                  {type}
                </label>
              </div>
            ))}
          </div>
        </div>
      );

    case 'secrets_detection':
      return (
        <div className="space-y-3">
          <Label>Secret Types to Detect</Label>
          <div className="grid grid-cols-3 gap-2">
            {SECRET_TYPES.map((type) => (
              <div key={type} className="flex items-center space-x-2">
                <Checkbox
                  id={`secret-${type}`}
                  checked={config.secret_types?.includes(type) ?? false}
                  onCheckedChange={(checked) => {
                    const current = config.secret_types || [];
                    const updated = checked
                      ? [...current, type]
                      : current.filter((t: string) => t !== type);
                    updateConfig('secret_types', updated);
                  }}
                />
                <label htmlFor={`secret-${type}`} className="text-sm">
                  {type}
                </label>
              </div>
            ))}
          </div>
        </div>
      );

    case 'length_limit':
      return (
        <div className="grid grid-cols-3 gap-4">
          <div className="space-y-2">
            <Label>Min Length</Label>
            <Input
              type="number"
              placeholder="Optional"
              value={config.min_length || ''}
              onChange={(e) =>
                updateConfig('min_length', parseInt(e.target.value) || 0)
              }
            />
          </div>
          <div className="space-y-2">
            <Label>Max Length</Label>
            <Input
              type="number"
              placeholder="Optional"
              value={config.max_length || ''}
              onChange={(e) =>
                updateConfig('max_length', parseInt(e.target.value) || 0)
              }
            />
          </div>
          <div className="space-y-2">
            <Label>Max Tokens</Label>
            <Input
              type="number"
              placeholder="Optional"
              value={config.max_tokens || ''}
              onChange={(e) =>
                updateConfig('max_tokens', parseInt(e.target.value) || 0)
              }
            />
          </div>
        </div>
      );

    case 'regex_pattern':
      return (
        <div className="space-y-2">
          <Label>Regex Pattern</Label>
          <Input
            placeholder="e.g., ^\d{3}-\d{2}-\d{4}$"
            value={config.pattern || ''}
            onChange={(e) => updateConfig('pattern', e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Text matching this pattern will trigger the rule
          </p>
        </div>
      );

    case 'keyword_blocker':
      return (
        <div className="space-y-3">
          <div className="space-y-2">
            <Label>Keywords (one per line)</Label>
            <Textarea
              placeholder="e.g.,&#10;competitor name&#10;internal info"
              value={config.keywords?.join('\n') || ''}
              onChange={(e) =>
                updateConfig(
                  'keywords',
                  e.target.value.split('\n').filter((k) => k.trim())
                )
              }
              rows={4}
            />
          </div>
          <div className="flex items-center space-x-2">
            <Checkbox
              id="case-sensitive"
              checked={config.case_sensitive ?? false}
              onCheckedChange={(checked) =>
                updateConfig('case_sensitive', checked)
              }
            />
            <label htmlFor="case-sensitive" className="text-sm">
              Case sensitive
            </label>
          </div>
        </div>
      );

    case 'toxicity':
      return (
        <div className="space-y-2">
          <Label>Toxicity Threshold (0-1)</Label>
          <Input
            type="number"
            step="0.1"
            min="0"
            max="1"
            placeholder="0.7"
            value={config.threshold || ''}
            onChange={(e) =>
              updateConfig('threshold', parseFloat(e.target.value) || 0.7)
            }
          />
          <p className="text-xs text-muted-foreground">
            Higher values are more strict
          </p>
        </div>
      );

    case 'json_schema':
      return (
        <div className="space-y-2">
          <Label>JSON Schema</Label>
          <Textarea
            placeholder='{&#10;  "type": "object",&#10;  "properties": { ... }&#10;}'
            value={
              config.schema ? JSON.stringify(config.schema, null, 2) : ''
            }
            onChange={(e) => {
              try {
                const schema = JSON.parse(e.target.value);
                updateConfig('schema', schema);
              } catch {
                // Invalid JSON, ignore
              }
            }}
            rows={6}
            className="font-mono text-sm"
          />
        </div>
      );

    case 'format_validator':
      return (
        <div className="space-y-2">
          <Label>Expected Format</Label>
          <Select
            value={config.format || ''}
            onValueChange={(value) => updateConfig('format', value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select format" />
            </SelectTrigger>
            <SelectContent>
              {FORMAT_TYPES.map((format) => (
                <SelectItem key={format} value={format}>
                  {format}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      );

    case 'relevance':
      return (
        <div className="space-y-2">
          <Label>Relevance Threshold (0-1)</Label>
          <Input
            type="number"
            step="0.1"
            min="0"
            max="1"
            placeholder="0.1"
            value={config.threshold || ''}
            onChange={(e) =>
              updateConfig('threshold', parseFloat(e.target.value) || 0.1)
            }
          />
          <p className="text-xs text-muted-foreground">
            Minimum word overlap ratio
          </p>
        </div>
      );

    default:
      return null;
  }
}

// Action configuration editor based on action type
function ActionConfigEditor({
  action,
  config,
  onChange,
}: {
  action: string;
  config: Record<string, any>;
  onChange: (config: Record<string, any>) => void;
}) {
  const updateConfig = (key: string, value: any) => {
    onChange({ ...config, [key]: value });
  };

  switch (action) {
    case 'block':
      return (
        <div className="space-y-2">
          <Label>Custom Block Response (optional)</Label>
          <Textarea
            placeholder="Default: I cannot process this request as it violates our content policy."
            value={config.block_response || ''}
            onChange={(e) => updateConfig('block_response', e.target.value)}
            rows={2}
          />
        </div>
      );

    case 'sanitize':
      return (
        <div className="space-y-3">
          <div className="space-y-2">
            <Label>Redaction Text</Label>
            <Input
              placeholder="[REDACTED]"
              value={config.redact_text || ''}
              onChange={(e) => updateConfig('redact_text', e.target.value)}
            />
          </div>
        </div>
      );

    case 'retry':
      return (
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>Max Retries</Label>
            <Input
              type="number"
              placeholder="3"
              value={config.retry_count || ''}
              onChange={(e) =>
                updateConfig('retry_count', parseInt(e.target.value) || 3)
              }
            />
          </div>
          <div className="space-y-2">
            <Label>Retry Delay (ms)</Label>
            <Input
              type="number"
              placeholder="1000"
              value={config.retry_delay || ''}
              onChange={(e) =>
                updateConfig('retry_delay', parseInt(e.target.value) || 1000)
              }
            />
          </div>
        </div>
      );

    case 'fallback':
      return (
        <div className="space-y-3">
          <div className="space-y-2">
            <Label>Fallback Response</Label>
            <Textarea
              placeholder="I apologize, but I cannot provide a complete response at this time."
              value={config.fallback_response || ''}
              onChange={(e) => updateConfig('fallback_response', e.target.value)}
              rows={2}
            />
          </div>
          <div className="space-y-2">
            <Label>Fallback Model (optional)</Label>
            <Input
              placeholder="e.g., gpt-3.5-turbo"
              value={config.fallback_model || ''}
              onChange={(e) => updateConfig('fallback_model', e.target.value)}
            />
          </div>
        </div>
      );

    case 'transform':
      return (
        <div className="space-y-2">
          <Label>Transform Type</Label>
          <Select
            value={config.transform_type || ''}
            onValueChange={(value) => updateConfig('transform_type', value)}
          >
            <SelectTrigger>
              <SelectValue placeholder="Select transform" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="truncate">Truncate</SelectItem>
              <SelectItem value="format">Format</SelectItem>
              <SelectItem value="extract">Extract</SelectItem>
              <SelectItem value="lowercase">Lowercase</SelectItem>
              <SelectItem value="uppercase">Uppercase</SelectItem>
            </SelectContent>
          </Select>

          {config.transform_type === 'truncate' && (
            <div className="space-y-2 pt-2">
              <Label>Max Length</Label>
              <Input
                type="number"
                placeholder="500"
                value={config.transform_config?.max_length || ''}
                onChange={(e) =>
                  updateConfig('transform_config', {
                    ...config.transform_config,
                    max_length: parseInt(e.target.value) || 500,
                  })
                }
              />
            </div>
          )}
        </div>
      );

    case 'alert':
      return (
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label>Alert Channel</Label>
            <Select
              value={config.alert_channel || ''}
              onValueChange={(value) => updateConfig('alert_channel', value)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select channel" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="email">Email</SelectItem>
                <SelectItem value="slack">Slack</SelectItem>
                <SelectItem value="webhook">Webhook</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Recipients (comma separated)</Label>
            <Input
              placeholder="alerts@example.com"
              value={config.alert_recipients?.join(', ') || ''}
              onChange={(e) =>
                updateConfig(
                  'alert_recipients',
                  e.target.value.split(',').map((r) => r.trim())
                )
              }
            />
          </div>
        </div>
      );

    default:
      return null;
  }
}
