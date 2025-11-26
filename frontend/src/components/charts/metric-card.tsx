import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { cn } from '@/lib/utils';
import { ArrowDown, ArrowUp, Minus } from 'lucide-react';

interface MetricCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: React.ReactNode;
  trend?: {
    value: number;
    label?: string;
    isPositiveGood?: boolean;
  };
  className?: string;
  valueClassName?: string;
  loading?: boolean;
}

export function MetricCard({
  title,
  value,
  subtitle,
  icon,
  trend,
  className,
  valueClassName,
  loading = false,
}: MetricCardProps) {
  const getTrendColor = () => {
    if (!trend) return '';
    if (trend.value === 0) return 'text-muted-foreground';

    const isPositive = trend.value > 0;
    const isGood = trend.isPositiveGood ?? true;

    if (isPositive && isGood) return 'text-green-600 dark:text-green-400';
    if (isPositive && !isGood) return 'text-red-600 dark:text-red-400';
    if (!isPositive && isGood) return 'text-red-600 dark:text-red-400';
    return 'text-green-600 dark:text-green-400';
  };

  const getTrendIcon = () => {
    if (!trend) return null;
    if (trend.value === 0) return <Minus className="h-4 w-4" />;
    if (trend.value > 0) return <ArrowUp className="h-4 w-4" />;
    return <ArrowDown className="h-4 w-4" />;
  };

  if (loading) {
    return (
      <Card className={cn('', className)}>
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {icon && <div className="text-muted-foreground">{icon}</div>}
        </CardHeader>
        <CardContent>
          <div className="h-8 w-24 animate-pulse rounded bg-muted" />
          {subtitle && <div className="mt-1 h-4 w-32 animate-pulse rounded bg-muted" />}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className={cn('', className)}>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        {icon && <div className="text-muted-foreground">{icon}</div>}
      </CardHeader>
      <CardContent>
        <div className={cn('text-2xl font-bold', valueClassName)}>
          {value}
        </div>
        {(subtitle || trend) && (
          <div className="flex items-center gap-2 mt-1">
            {trend && (
              <div className={cn('flex items-center gap-1 text-xs font-medium', getTrendColor())}>
                {getTrendIcon()}
                <span>{Math.abs(trend.value)}%</span>
                {trend.label && <span className="text-muted-foreground">{trend.label}</span>}
              </div>
            )}
            {subtitle && !trend && (
              <p className="text-xs text-muted-foreground">{subtitle}</p>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
