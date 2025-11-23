import { useState, useEffect } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
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
  DollarSign,
  Hash,
  Save,
  History,
  GitCompare,
  Zap,
} from 'lucide-react';
import {
  useCompilePrompt,
  useExtractVariables,
  useLLMModels,
  useExecutePrompt,
  useCountTokens,
  useEstimateCost,
  useCreatePrompt,
  LLMRequest,
  LLMModel,
} from '@/api/prompts';
import { useProjectContext } from '@/contexts/project-context';

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

  // LLM-related state
  const [selectedModel, setSelectedModel] = useState<LLMModel | null>(null);
  const [llmResponse, setLlmResponse] = useState<string>('');
  const [llmErrors, setLlmErrors] = useState<string[]>([]);
  const [tokenCount, setTokenCount] = useState<number | null>(null);
  const [estimatedCost, setEstimatedCost] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const [executionHistory, setExecutionHistory] = useState<Array<{
    timestamp: Date;
    model: string;
    prompt: string;
    response: string;
    tokens: number;
    cost: string;
  }>>([]);
  const [activeTab, setActiveTab] = useState('variables');
  const [comparisonMode, setComparisonMode] = useState(false);
  const [comparisonLeft, setComparisonLeft] = useState<number | null>(null);
  const [comparisonRight, setComparisonRight] = useState<number | null>(null);

  const compilePrompt = useCompilePrompt();
  const extractVariables = useExtractVariables();
  const { data: llmModels } = useLLMModels();
  const executePrompt = useExecutePrompt();
  const countTokens = useCountTokens();
  const estimateCost = useEstimateCost();
  const createPrompt = useCreatePrompt();
  const { selectedProject } = useProjectContext();

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

  // Set default model when models are loaded
  useEffect(() => {
    if (llmModels && llmModels.length > 0 && !selectedModel) {
      setSelectedModel(llmModels[0]);
    }
  }, [llmModels, selectedModel]);

  // Count tokens when content changes
  useEffect(() => {
    if (content && selectedModel) {
      countTokens.mutate({
        text: content,
        model: selectedModel.modelId,
      }, {
        onSuccess: (data) => {
          setTokenCount(data.tokens);
        },
      });
    }
  }, [content, selectedModel]);

  // Estimate cost when model or content changes
  useEffect(() => {
    if (content && selectedModel) {
      estimateCost.mutate({
        provider: selectedModel.provider,
        model: selectedModel.modelId,
        prompt: content,
        maxTokens: 1000,
      }, {
        onSuccess: (data) => {
          setEstimatedCost(data.formattedCost);
        },
      });
    }
  }, [content, selectedModel]);

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
          setActiveTab('output');
        },
        onError: () => {
          setErrors(['Failed to compile prompt']);
        },
      }
    );
  };

  const handleExecuteLLM = async () => {
    if (!selectedModel || !compiledResult) return;

    setIsStreaming(true);
    setLlmResponse('');
    setLlmErrors([]);

    const request: LLMRequest = {
      provider: selectedModel.provider,
      model: selectedModel.modelId,
      prompt: compiledResult,
      maxTokens: 1000,
      temperature: 0.7,
    };

    try {
      const response = await executePrompt.mutateAsync(request);
      setLlmResponse(response.text);

      // Add to execution history
      setExecutionHistory(prev => [{
        timestamp: new Date(),
        model: `${selectedModel.provider}/${selectedModel.name}`,
        prompt: compiledResult,
        response: response.text,
        tokens: response.usage.totalTokens,
        cost: estimatedCost || 'N/A',
      }, ...prev.slice(0, 9)]); // Keep last 10 executions

      setActiveTab('llm');
    } catch (error) {
      setLlmErrors(['Failed to execute LLM request']);
    } finally {
      setIsStreaming(false);
    }
  };

  const handleSaveToPrompt = async () => {
    if (!compiledResult || !selectedProject) return;

    try {
      await createPrompt.mutateAsync({
        name: `Playground Prompt ${new Date().toLocaleString()}`,
        description: `Created from playground with model: ${selectedModel?.name || 'Unknown'}`,
        content: content, // Save original template
        tags: ['playground', selectedModel?.provider || 'llm'],
        projectId: selectedProject.id,
      });

      // Close the dialog after successful save
      onOpenChange(false);
    } catch (error) {
      setLlmErrors(['Failed to save prompt']);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center justify-between">
            <div>
              <DialogTitle className="flex items-center gap-2">
                <Play className="h-5 w-5" />
                Prompt Playground
              </DialogTitle>
              <DialogDescription>
                Test your prompt template with variables, preview compilation, and execute against LLM models.
              </DialogDescription>
            </div>
            <Button
              onClick={handleSaveToPrompt}
              disabled={!compiledResult || createPrompt.isPending}
              variant="outline"
              size="sm"
            >
              {createPrompt.isPending ? (
                <>
                  <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                  Saving...
                </>
              ) : (
                <>
                  <Save className="h-3 w-3 mr-2" />
                  Save as Prompt
                </>
              )}
            </Button>
          </div>
        </DialogHeader>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Left Panel - Model Selection and Stats */}
          <div className="space-y-4">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Zap className="h-4 w-4" />
                  Model Selection
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="space-y-2">
                  <Label>Provider & Model</Label>
                  <Select
                    value={selectedModel?.id || ''}
                    onValueChange={(value) => {
                      const model = llmModels?.find(m => m.id === value);
                      setSelectedModel(model || null);
                    }}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select a model" />
                    </SelectTrigger>
                    <SelectContent>
                      {llmModels?.map((model) => (
                        <SelectItem key={model.id} value={model.id}>
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs">
                              {model.provider}
                            </Badge>
                            {model.name}
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                {selectedModel && (
                  <div className="space-y-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Context:</span>
                      <span>{selectedModel.contextSize.toLocaleString()} tokens</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Input:</span>
                      <span>${selectedModel.pricing.inputTokens}/1K</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Output:</span>
                      <span>${selectedModel.pricing.outputTokens}/1K</span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Hash className="h-4 w-4" />
                  Token Analysis
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Tokens:</span>
                  <span>{tokenCount?.toLocaleString() || 'N/A'}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Est. Cost:</span>
                  <span className="flex items-center gap-1">
                    <DollarSign className="h-3 w-3" />
                    {estimatedCost || 'N/A'}
                  </span>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Main Content */}
          <div className="lg:col-span-2">
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className="grid w-full grid-cols-5">
                <TabsTrigger value="variables">
                  <Variable className="h-3 w-3 mr-1" />
                  Variables
                </TabsTrigger>
                <TabsTrigger value="output">
                  <CheckCircle2 className="h-3 w-3 mr-1" />
                  Output
                </TabsTrigger>
                <TabsTrigger value="llm">
                  <Zap className="h-3 w-3 mr-1" />
                  LLM
                </TabsTrigger>
                <TabsTrigger value="comparison">
                  <GitCompare className="h-3 w-3 mr-1" />
                  Compare
                </TabsTrigger>
                <TabsTrigger value="history">
                  <History className="h-3 w-3 mr-1" />
                  History
                </TabsTrigger>
              </TabsList>

              <TabsContent value="variables" className="space-y-4 mt-4">
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
                          For arrays: ["item1", "item2"] | For booleans: true/false
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
              </TabsContent>

              <TabsContent value="output" className="space-y-4 mt-4">
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
              </TabsContent>

              <TabsContent value="llm" className="space-y-4 mt-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium flex items-center gap-2">
                    <Zap className="h-4 w-4" />
                    LLM Execution
                  </h3>
                  <Button
                    onClick={handleExecuteLLM}
                    disabled={!selectedModel || !compiledResult || isStreaming || executePrompt.isPending}
                    size="sm"
                  >
                    {isStreaming || executePrompt.isPending ? (
                      <>
                        <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                        Executing...
                      </>
                    ) : (
                      <>
                        <Play className="h-3 w-3 mr-2" />
                        Execute
                      </>
                    )}
                  </Button>
                </div>

                {llmErrors.length > 0 && (
                  <Card className="border-destructive/50 bg-destructive/10">
                    <CardContent className="py-3">
                      <ul className="text-sm text-destructive space-y-1">
                        {llmErrors.map((error, i) => (
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
                    <CardTitle className="text-sm">LLM Response</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Textarea
                      value={llmResponse || '(Execute to see LLM response)'}
                      readOnly
                      className="min-h-[300px] font-mono text-sm bg-muted"
                    />
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="comparison" className="space-y-4 mt-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium flex items-center gap-2">
                    <GitCompare className="h-4 w-4" />
                    Side-by-Side Comparison
                  </h3>
                  <Button
                    onClick={() => setComparisonMode(!comparisonMode)}
                    variant={comparisonMode ? "default" : "outline"}
                    size="sm"
                  >
                    {comparisonMode ? "Exit Compare" : "Enter Compare Mode"}
                  </Button>
                </div>

                {comparisonMode ? (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label className="text-sm font-medium">Left Side</Label>
                        <Select
                          value={comparisonLeft?.toString() || ''}
                          onValueChange={(value) => setComparisonLeft(parseInt(value))}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select execution" />
                          </SelectTrigger>
                          <SelectContent>
                            {executionHistory.map((_, index) => (
                              <SelectItem key={index} value={index.toString()}>
                                Execution {index + 1} - {executionHistory[index].model}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div>
                        <Label className="text-sm font-medium">Right Side</Label>
                        <Select
                          value={comparisonRight?.toString() || ''}
                          onValueChange={(value) => setComparisonRight(parseInt(value))}
                        >
                          <SelectTrigger>
                            <SelectValue placeholder="Select execution" />
                          </SelectTrigger>
                          <SelectContent>
                            {executionHistory.map((_, index) => (
                              <SelectItem key={index} value={index.toString()}>
                                Execution {index + 1} - {executionHistory[index].model}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>

                    {comparisonLeft !== null && comparisonRight !== null ? (
                      <div className="grid grid-cols-2 gap-4">
                        <Card>
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">
                              Execution {comparisonLeft + 1}
                            </CardTitle>
                            <div className="text-xs text-muted-foreground">
                              {executionHistory[comparisonLeft].model} • {executionHistory[comparisonLeft].tokens} tokens • {executionHistory[comparisonLeft].cost}
                            </div>
                          </CardHeader>
                          <CardContent>
                            <Textarea
                              value={executionHistory[comparisonLeft].response}
                              readOnly
                              className="min-h-[300px] font-mono text-sm bg-muted"
                            />
                          </CardContent>
                        </Card>

                        <Card>
                          <CardHeader className="pb-2">
                            <CardTitle className="text-sm">
                              Execution {comparisonRight + 1}
                            </CardTitle>
                            <div className="text-xs text-muted-foreground">
                              {executionHistory[comparisonRight].model} • {executionHistory[comparisonRight].tokens} tokens • {executionHistory[comparisonRight].cost}
                            </div>
                          </CardHeader>
                          <CardContent>
                            <Textarea
                              value={executionHistory[comparisonRight].response}
                              readOnly
                              className="min-h-[300px] font-mono text-sm bg-muted"
                            />
                          </CardContent>
                        </Card>
                      </div>
                    ) : (
                      <Card>
                        <CardContent className="py-8 text-center text-muted-foreground">
                          <GitCompare className="h-8 w-8 mx-auto mb-2 opacity-50" />
                          <p>Select two executions to compare</p>
                          <p className="text-xs mt-1">
                            Execute prompts multiple times to build comparison options
                          </p>
                        </CardContent>
                      </Card>
                    )}
                  </div>
                ) : (
                  <Card>
                    <CardContent className="py-8 text-center text-muted-foreground">
                      <GitCompare className="h-8 w-8 mx-auto mb-2 opacity-50" />
                      <p>Comparison mode allows side-by-side comparison of LLM executions</p>
                      <p className="text-xs mt-1">
                        Click "Enter Compare Mode" to select and compare different executions
                      </p>
                    </CardContent>
                  </Card>
                )}
              </TabsContent>

              <TabsContent value="history" className="space-y-4 mt-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-medium flex items-center gap-2">
                    <History className="h-4 w-4" />
                    Execution History
                  </h3>
                  <Badge variant="outline">{executionHistory.length} executions</Badge>
                </div>

                {executionHistory.length === 0 ? (
                  <Card>
                    <CardContent className="py-8 text-center text-muted-foreground">
                      <History className="h-8 w-8 mx-auto mb-2 opacity-50" />
                      <p>No execution history yet</p>
                      <p className="text-xs mt-1">
                        Execute prompts to see history here
                      </p>
                    </CardContent>
                  </Card>
                ) : (
                  <div className="space-y-3 max-h-[400px] overflow-y-auto">
                    {executionHistory.map((exec, index) => (
                      <Card key={index}>
                        <CardContent className="pt-4">
                          <div className="flex items-start justify-between mb-2">
                            <div className="flex items-center gap-2">
                              <Badge variant="outline" className="text-xs">
                                {exec.model}
                              </Badge>
                              <span className="text-xs text-muted-foreground">
                                {exec.timestamp.toLocaleTimeString()}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                              <span>{exec.tokens} tokens</span>
                              <span>{exec.cost}</span>
                            </div>
                          </div>
                          <div className="space-y-2">
                            <div>
                              <p className="text-xs font-medium text-muted-foreground mb-1">Prompt:</p>
                              <p className="text-xs bg-muted p-2 rounded font-mono truncate">
                                {exec.prompt}
                              </p>
                            </div>
                            <div>
                              <p className="text-xs font-medium text-muted-foreground mb-1">Response:</p>
                              <p className="text-xs bg-muted p-2 rounded font-mono truncate">
                                {exec.response}
                              </p>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                )}
              </TabsContent>
            </Tabs>
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
