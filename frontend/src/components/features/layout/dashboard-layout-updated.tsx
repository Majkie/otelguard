import { Outlet, NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Activity,
  Users,
  FileText,
  Shield,
  Settings,
  LogOut,
  Menu,
  MessageSquare,
  BarChart3,
  CheckSquare,
  Heart,
  Network,
  FlaskConical,
  Bell,
  PanelTop,
  ChevronDown,
  Eye,
  Target,
  Cog,
} from 'lucide-react';
import { useState, useEffect } from 'react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { ThemeToggle } from '@/components/ui/theme-toggle';
import { useAuth } from '@/hooks/use-auth';
import { ProjectSelector } from '@/components/features/projects/project-selector';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';

// Navigation structure with sections
const navigationSections = [
  {
    id: 'overview',
    name: 'Overview',
    items: [
      { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
      { name: 'Dashboards', href: '/dashboards', icon: PanelTop },
    ],
  },
  {
    id: 'observability',
    name: 'Observability',
    icon: Eye,
    collapsible: true,
    items: [
      { name: 'Traces', href: '/traces', icon: Activity },
      { name: 'Agent Graphs', href: '/agents', icon: Network },
      { name: 'Sessions', href: '/sessions', icon: MessageSquare },
      { name: 'Users', href: '/users', icon: Users },
    ],
  },
  {
    id: 'evaluation',
    name: 'Evaluation',
    icon: Target,
    collapsible: true,
    items: [
      { name: 'Scores', href: '/scores', icon: BarChart3 },
      { name: 'Evaluators', href: '/evaluators', icon: FlaskConical },
      { name: 'Feedback', href: '/feedback', icon: Heart },
      { name: 'Annotations', href: '/annotations', icon: CheckSquare },
    ],
  },
  {
    id: 'management',
    name: 'Management',
    icon: Cog,
    collapsible: true,
    items: [
      { name: 'Prompts', href: '/prompts', icon: FileText },
      { name: 'Guardrails', href: '/guardrails', icon: Shield },
      { name: 'Alerts', href: '/alerts/rules', icon: Bell },
    ],
  },
  {
    id: 'settings',
    name: 'Settings',
    items: [
      { name: 'Settings', href: '/settings', icon: Settings },
    ],
  },
];

export function DashboardLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const { user, logout } = useAuth();
  const location = useLocation();

  // Track which sections are open (default to open)
  const [openSections, setOpenSections] = useState<Record<string, boolean>>(() => {
    const initial: Record<string, boolean> = {};
    navigationSections.forEach(section => {
      if (section.collapsible) {
        initial[section.id] = true; // Default all sections to open
      }
    });
    return initial;
  });

  // Auto-expand section if current route is in it
  useEffect(() => {
    navigationSections.forEach(section => {
      if (section.collapsible) {
        const hasActiveItem = section.items.some(item =>
          location.pathname.startsWith(item.href)
        );
        if (hasActiveItem && !openSections[section.id]) {
          setOpenSections(prev => ({ ...prev, [section.id]: true }));
        }
      }
    });
  }, [location.pathname]);

  const toggleSection = (sectionId: string) => {
    setOpenSections(prev => ({ ...prev, [sectionId]: !prev[sectionId] }));
  };

  return (
    <div className="min-h-screen bg-background">
      {/* Mobile sidebar backdrop */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform bg-card border-r transition-transform duration-200 lg:translate-x-0',
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <div className="flex h-full flex-col">
          {/* Logo */}
          <div className="flex h-16 items-center gap-2 border-b px-6">
            <div className="h-8 w-8 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-primary-foreground font-bold">O</span>
            </div>
            <span className="text-xl font-bold">OTelGuard</span>
          </div>

          {/* Navigation */}
          <nav className="flex-1 overflow-y-auto px-3 py-4">
            <div className="space-y-4">
              {navigationSections.map((section) => (
                <div key={section.id}>
                  {section.collapsible ? (
                    <Collapsible
                      open={openSections[section.id]}
                      onOpenChange={() => toggleSection(section.id)}
                    >
                      <CollapsibleTrigger className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm font-semibold text-foreground hover:bg-muted transition-colors">
                        <div className="flex items-center gap-2">
                          {section.icon && <section.icon className="h-4 w-4" />}
                          <span>{section.name}</span>
                        </div>
                        <ChevronDown
                          className={cn(
                            'h-4 w-4 transition-transform',
                            openSections[section.id] ? 'rotate-180' : ''
                          )}
                        />
                      </CollapsibleTrigger>
                      <CollapsibleContent className="space-y-1 pt-1">
                        {section.items.map((item) => (
                          <NavLink
                            key={item.href}
                            to={item.href}
                            className={({ isActive }) =>
                              cn(
                                'flex items-center gap-3 rounded-lg px-3 py-2 pl-9 text-sm font-medium transition-colors',
                                isActive
                                  ? 'bg-primary text-primary-foreground'
                                  : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                              )
                            }
                          >
                            <item.icon className="h-4 w-4" />
                            {item.name}
                          </NavLink>
                        ))}
                      </CollapsibleContent>
                    </Collapsible>
                  ) : (
                    <div className="space-y-1">
                      {section.name !== 'Overview' && section.name !== 'Settings' && (
                        <div className="px-3 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider">
                          {section.name}
                        </div>
                      )}
                      {section.items.map((item) => (
                        <NavLink
                          key={item.href}
                          to={item.href}
                          className={({ isActive }) =>
                            cn(
                              'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
                              isActive
                                ? 'bg-primary text-primary-foreground'
                                : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                            )
                          }
                        >
                          <item.icon className="h-5 w-5" />
                          {item.name}
                        </NavLink>
                      ))}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </nav>

          {/* User section */}
          <div className="border-t p-4">
            <div className="flex items-center gap-3">
              <div className="h-9 w-9 rounded-full bg-muted flex items-center justify-center">
                <span className="text-sm font-medium">
                  {user?.name?.charAt(0).toUpperCase() || 'U'}
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{user?.name}</p>
                <p className="text-xs text-muted-foreground truncate">
                  {user?.email}
                </p>
              </div>
              <Button
                variant="ghost"
                size="icon"
                onClick={logout}
                title="Logout"
              >
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </aside>

      {/* Main content */}
      <div className="lg:pl-64">
        {/* Header */}
        <header className="sticky top-0 z-30 flex h-16 items-center gap-4 border-b bg-background px-4">
          {/* Mobile menu button */}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSidebarOpen(true)}
            className="lg:hidden"
          >
            <Menu className="h-5 w-5" />
          </Button>

          {/* Mobile logo */}
          <div className="flex items-center gap-2 lg:hidden">
            <div className="h-7 w-7 rounded-lg bg-primary flex items-center justify-center">
              <span className="text-primary-foreground font-bold text-sm">O</span>
            </div>
            <span className="font-bold">OTelGuard</span>
          </div>

          {/* Project Selector */}
          <div className="flex-1 flex items-center justify-center lg:justify-start">
            <ProjectSelector />
          </div>

          {/* Theme toggle */}
          <ThemeToggle />
        </header>

        {/* Page content */}
        <main className="p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
