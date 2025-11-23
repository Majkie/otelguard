import { useState } from 'react';
import { ChevronDown, Plus, Settings, Building2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useProjectContext } from '@/contexts/project-context';
import { useCreateProject, useProjects } from '@/api/projects';
import { useOrganizations } from '@/api/projects';
import { Skeleton } from '@/components/ui/skeleton';

export function ProjectSelector() {
  const { selectedProject, setSelectedProject, projects, isLoading, hasProjects } = useProjectContext();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newProjectName, setNewProjectName] = useState('');
  const [selectedOrgId, setSelectedOrgId] = useState('');
  
  const { data: orgsData } = useOrganizations({ limit: 50 });
  const createProject = useCreateProject();

  const organizations = orgsData?.data || [];

  const handleCreateProject = async () => {
    if (!newProjectName.trim() || !selectedOrgId) return;

    try {
      await createProject.mutateAsync({
        organizationId: selectedOrgId,
        name: newProjectName,
      });
      setCreateDialogOpen(false);
      setNewProjectName('');
      setSelectedOrgId('');
    } catch (error) {
      console.error('Failed to create project:', error);
    }
  };

  if (isLoading) {
    return <Skeleton className="h-10 w-48" />;
  }

  if (!hasProjects) {
    return (
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          onClick={() => setCreateDialogOpen(true)}
          className="text-sm"
        >
          <Plus className="h-4 w-4 mr-2" />
          Create First Project
        </Button>
      </div>
    );
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" className="w-48 justify-between">
            <div className="flex items-center gap-2 overflow-hidden">
              <Building2 className="h-4 w-4 shrink-0" />
              <span className="truncate">
                {selectedProject?.name || 'Select Project'}
              </span>
            </div>
            <ChevronDown className="h-4 w-4 shrink-0 opacity-50" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-56">
          <DropdownMenuLabel>Projects</DropdownMenuLabel>
          <DropdownMenuSeparator />
          {projects.map((project) => (
            <DropdownMenuItem
              key={project.id}
              onClick={() => setSelectedProject(project)}
              className="cursor-pointer"
            >
              <div className="flex flex-col">
                <span className="font-medium">{project.name}</span>
                <span className="text-xs text-muted-foreground">
                  {project.slug}
                </span>
              </div>
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => setCreateDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Create New Project
          </DropdownMenuItem>
          <DropdownMenuItem asChild>
            <a href="/settings/projects" className="cursor-pointer">
              <Settings className="h-4 w-4 mr-2" />
              Manage Projects
            </a>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Project</DialogTitle>
            <DialogDescription>
              Create a new project to organize your prompts, traces, and other resources.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="project-name">Project Name</Label>
              <Input
                id="project-name"
                placeholder="e.g., My Application"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="organization">Organization</Label>
              <select
                id="organization"
                value={selectedOrgId}
                onChange={(e) => setSelectedOrgId(e.target.value)}
                className="w-full p-2 border rounded-md"
              >
                <option value="">Select organization...</option>
                {organizations.map((org) => (
                  <option key={org.id} value={org.id}>
                    {org.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateDialogOpen(false)}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateProject}
              disabled={!newProjectName.trim() || !selectedOrgId || createProject.isPending}
            >
              {createProject.isPending ? 'Creating...' : 'Create Project'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
