import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useProjects } from '@/api/projects';
import { Project } from '@/api/projects';

interface ProjectContextType {
  selectedProject: Project | null;
  setSelectedProject: (project: Project | null) => void;
  projects: Project[];
  isLoading: boolean;
  error: Error | null;
  hasProjects: boolean;
}

const ProjectContext = createContext<ProjectContextType | undefined>(undefined);

interface ProjectProviderProps {
  children: ReactNode;
}

export function ProjectProvider({ children }: ProjectProviderProps) {
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const { data, isLoading, error } = useProjects({ limit: 100 });

  const projects = data?.data || [];
  const hasProjects = projects.length > 0;

  // Auto-select first project if none is selected
  useEffect(() => {
    if (!selectedProject && projects.length > 0) {
      setSelectedProject(projects[0]);
    }
  }, [projects, selectedProject]);

  // Persist selected project in localStorage
  useEffect(() => {
    const savedProjectId = localStorage.getItem('selectedProjectId');
    if (savedProjectId && projects.length > 0) {
      const foundProject = projects.find(p => p.id === savedProjectId);
      if (foundProject) {
        setSelectedProject(foundProject);
      }
    }
  }, [projects]);

  useEffect(() => {
    if (selectedProject) {
      localStorage.setItem('selectedProjectId', selectedProject.id);
    } else {
      localStorage.removeItem('selectedProjectId');
    }
  }, [selectedProject]);

  return (
    <ProjectContext.Provider
      value={{
        selectedProject,
        setSelectedProject,
        projects,
        isLoading,
        error: error as Error | null,
        hasProjects,
      }}
    >
      {children}
    </ProjectContext.Provider>
  );
}

export function useProjectContext() {
  const context = useContext(ProjectContext);
  if (context === undefined) {
    throw new Error('useProjectContext must be used within a ProjectProvider');
  }
  return context;
}
