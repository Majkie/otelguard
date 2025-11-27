import { Line, LineChart, ResponsiveContainer } from 'recharts';
import { cn } from '@/lib/utils';

interface SparklineProps {
  data: number[];
  color?: string;
  height?: number;
  width?: number;
  className?: string;
  showDots?: boolean;
}

export function Sparkline({
  data,
  color = 'hsl(var(--primary))',
  height = 40,
  width = 100,
  className,
  showDots = false,
}: SparklineProps) {
  const chartData = data.map((value, index) => ({
    index,
    value,
  }));

  return (
    <div className={cn('inline-block', className)} style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 2, right: 2, left: 2, bottom: 2 }}>
          <Line
            type="monotone"
            dataKey="value"
            stroke={color}
            strokeWidth={1.5}
            dot={showDots ? { r: 2 } : false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
