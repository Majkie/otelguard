import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Plus, Shield } from 'lucide-react';

export function GuardrailsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Guardrails</h1>
          <p className="text-muted-foreground">
            Configure policies to protect your LLM applications
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          New Policy
        </Button>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <Shield className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No guardrail policies</h3>
            <p className="text-muted-foreground max-w-sm mt-2">
              Create guardrail policies to detect and remediate issues like prompt
              injection, PII exposure, and toxic content.
            </p>
            <Button className="mt-4">
              <Plus className="h-4 w-4 mr-2" />
              Create Policy
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Feature overview */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Input Validation</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Detect prompt injection, jailbreak attempts, and PII before they reach your LLM.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Output Validation</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Check for toxicity, hallucinations, and ensure responses match expected formats.
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Auto-Remediation</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            Automatically block, sanitize, retry, or fallback when issues are detected.
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
