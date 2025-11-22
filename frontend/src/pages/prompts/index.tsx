import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Plus, FileText } from 'lucide-react';

export function PromptsPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Prompts</h1>
          <p className="text-muted-foreground">
            Manage and version your prompt templates
          </p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          New Prompt
        </Button>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <FileText className="h-12 w-12 text-muted-foreground mb-4" />
            <h3 className="text-lg font-medium">No prompts yet</h3>
            <p className="text-muted-foreground max-w-sm mt-2">
              Create your first prompt template to start managing and versioning
              your prompts.
            </p>
            <Button className="mt-4">
              <Plus className="h-4 w-4 mr-2" />
              Create Prompt
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
