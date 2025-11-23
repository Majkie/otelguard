import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Play,
  AlertCircle,
  CheckCircle2,
  Variable,
  Loader2,
} from 'lucide-react';
import { useCompilePrompt, useExtractVariables } from '@/api/prompts';

interface PlaygroundProps {
  promptId: string;
  content: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function PromptPlayground({
  promptId,
  content,
  open,
  onOpenChange,
}: PlaygroundProps) {
  const [variables, setVariables] = useState<Record<string, string>>({});
  const [extractedVars, setExtractedVars] = useState<string[]>([]);
  const [compiledResult, setCompiledResult] = useState<string>('');
  const [errors, setErrors] = useState<string[]>([]);
  const [missing, setMissing] = useState<string[]>([]);

  const compilePrompt = useCompilePrompt();
  const extractVariables = useExtractVariables();

  // Extract variables when content changes
  useEffect(() => {
    if (content && open) {
      extractVariables.mutate(content, {
        onSuccess: (data) => {
          setExtractedVars(data.variables || []);
          // Initialize variables with empty strings
          const initialVars: Record<string, string> = {};
          (data.variables || []).forEach((v: string) => {
            initialVars[v] = variables[v] || '';
          });
          setVariables(initialVars);
        },
      });
    }
  }, [content, open]);

  const handleVariableChange = (name: string, value: string) => {
    setVariables((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleCompile = () => {
    // Convert string variables to appropriate types for conditionals
    const processedVars: Record<string, unknown> = {};
    Object.entries(variables).forEach(([key, value]) => {
      // Try to parse as JSON for arrays/objects
      try {
        const parsed = JSON.parse(value);
        processedVars[key] = parsed;
      } catch {
        // Check for boolean-like values
        if (value.toLowerCase() === 'true') {
          processedVars[key] = true;
        } else if (value.toLowerCase() === 'false') {
          processedVars[key] = false;
        } else if (!isNaN(Number(value)) && value !== '') {
          processedVars[key] = Number(value);
        } else {
          processedVars[key] = value;
        }
      }
    });

    compilePrompt.mutate(
      {
        promptId,
        data: {
          variables: processedVars,
        },
      },
      {
        onSuccess: (data) => {
          setCompiledResult(data.compiled);
          setErrors(data.errors || []);
          setMissing(data.missing || []);
        },
        onError: () => {
          setErrors(['Failed to compile prompt']);
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Play className="h-5 w-5" />
            Prompt Playground
          </DialogTitle>
          <DialogDescription>
            Test your prompt template with different variables and preview the
            compiled output.
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Variables Input */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="font-medium flex items-center gap-2">
                <Variable className="h-4 w-4" />
                Variables
              </h3>
              <Badge variant="outline">{extractedVars.length} found</Badge>
            </div>

            {extractedVars.length === 0 ? (
              <Card>
                <CardContent className="py-6 text-center text-muted-foreground">
                  <Variable className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p>No variables detected in template</p>
                  <p className="text-xs mt-1">
                    Use {`{{variable_name}}`} syntax to add variables
                  </p>
                </CardContent>
              </Card>
            ) : (
              <div className="space-y-3">
                {extractedVars.map((varName) => (
                  <div key={varName} className="space-y-1">
                    <Label
                      htmlFor={varName}
                      className="flex items-center gap-2"
                    >
                      {varName}
                      {missing.includes(varName) && (
                        <Badge variant="destructive" className="text-xs">
                          Missing
                        </Badge>
                      )}
                    </Label>
                    <Input
                      id={varName}
                      value={variables[varName] || ''}
                      onChange={(e) =>
                        handleVariableChange(varName, e.target.value)
                      }
                      placeholder={`Enter ${varName}...`}
                    />
                    <p className="text-xs text-muted-foreground">
                      For arrays: [&quot;item1&quot;, &quot;item2&quot;] | For
                      booleans: true/false
                    </p>
                  </div>
                ))}
              </div>
            )}

            <Button
              onClick={handleCompile}
              disabled={compilePrompt.isPending}
              className="w-full"
            >
              {compilePrompt.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Compiling...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  Compile & Preview
                </>
              )}
            </Button>
          </div>

          {/* Output Preview */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="font-medium">Compiled Output</h3>
              {errors.length > 0 ? (
                <Badge variant="destructive">
                  <AlertCircle className="h-3 w-3 mr-1" />
                  {errors.length} error(s)
                </Badge>
              ) : compiledResult ? (
                <Badge variant="secondary" className="bg-green-500/20 text-green-700">
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  Success
                </Badge>
              ) : null}
            </div>

            {errors.length > 0 && (
              <Card className="border-destructive/50 bg-destructive/10">
                <CardContent className="py-3">
                  <ul className="text-sm text-destructive space-y-1">
                    {errors.map((error, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
                        {error}
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            )}

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">Preview</CardTitle>
              </CardHeader>
              <CardContent>
                <Textarea
                  value={compiledResult || '(Click "Compile & Preview" to see output)'}
                  readOnly
                  className="min-h-[300px] font-mono text-sm bg-muted"
                />
              </CardContent>
            </Card>

            {missing.length > 0 && (
              <div className="text-sm text-muted-foreground">
                <span className="font-medium">Missing variables: </span>
                {missing.join(', ')}
              </div>
            )}
          </div>
        </div>

        {/* Template Reference */}
        <Card className="mt-4">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm">Template Syntax Reference</CardTitle>
          </CardHeader>
          <CardContent className="text-xs font-mono space-y-1 text-muted-foreground">
            <p>
              <span className="text-foreground">{`{{variable}}`}</span> - Insert
              variable value
            </p>
            <p>
              <span className="text-foreground">{`{{#if var}}...{{/if}}`}</span>{' '}
              - Conditional (truthy check)
            </p>
            <p>
              <span className="text-foreground">{`{{#unless var}}...{{/unless}}`}</span>{' '}
              - Inverse conditional
            </p>
            <p>
              <span className="text-foreground">{`{{#each arr}}{{this}}{{/each}}`}</span>{' '}
              - Loop over array
            </p>
            <p>
              <span className="text-foreground">{`{{else}}`}</span> - Else
              branch in conditionals
            </p>
          </CardContent>
        </Card>
      </DialogContent>
    </Dialog>
  );
}
