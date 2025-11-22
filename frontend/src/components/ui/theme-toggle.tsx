import { Moon, Sun, Monitor } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useTheme } from '@/hooks/use-theme';

export function ThemeToggle() {
  const { theme, setTheme, resolvedTheme } = useTheme();

  const cycleTheme = () => {
    if (theme === 'light') {
      setTheme('dark');
    } else if (theme === 'dark') {
      setTheme('system');
    } else {
      setTheme('light');
    }
  };

  return (
    <Button variant="ghost" size="icon" onClick={cycleTheme} title={`Theme: ${theme}`}>
      {theme === 'system' ? (
        <Monitor className="h-5 w-5" />
      ) : resolvedTheme === 'dark' ? (
        <Moon className="h-5 w-5" />
      ) : (
        <Sun className="h-5 w-5" />
      )}
      <span className="sr-only">Toggle theme</span>
    </Button>
  );
}

export function ThemeDropdown() {
  const { theme, setTheme } = useTheme();

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm text-muted-foreground">Theme:</span>
      <div className="flex rounded-md border">
        <Button
          variant={theme === 'light' ? 'secondary' : 'ghost'}
          size="sm"
          className="rounded-r-none"
          onClick={() => setTheme('light')}
        >
          <Sun className="h-4 w-4" />
        </Button>
        <Button
          variant={theme === 'dark' ? 'secondary' : 'ghost'}
          size="sm"
          className="rounded-none border-x"
          onClick={() => setTheme('dark')}
        >
          <Moon className="h-4 w-4" />
        </Button>
        <Button
          variant={theme === 'system' ? 'secondary' : 'ghost'}
          size="sm"
          className="rounded-l-none"
          onClick={() => setTheme('system')}
        >
          <Monitor className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
