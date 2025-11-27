import { Line, LineChart as RechartsLineChart, XAxis, YAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartConfig } from '@/components/ui/chart';

interface LineChartProps {
  data: any[];
  xKey: string;
  lines: Array<{
    dataKey: string;
    color: string;
    label: string;
    strokeWidth?: number;
  }>;
  height?: number;
  showGrid?: boolean;
  showTooltip?: boolean;
  formatXAxis?: (value: any) => string;
  formatYAxis?: (value: any) => string;
  formatTooltip?: (value: any) => string;
}

export function LineChart({
  data,
  xKey,
  lines,
  height = 350,
  showGrid = true,
  showTooltip = true,
  formatXAxis,
  formatYAxis,
  formatTooltip,
}: LineChartProps) {
  const chartConfig: ChartConfig = lines.reduce((acc, line) => {
    acc[line.dataKey] = {
      label: line.label,
      color: line.color,
    };
    return acc;
  }, {} as ChartConfig);

  return (
    <ChartContainer config={chartConfig} className="w-full" style={{ height }}>
      <RechartsLineChart data={data} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
        {showGrid && <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />}
        <XAxis
          dataKey={xKey}
          tickLine={false}
          axisLine={false}
          tickFormatter={formatXAxis}
          className="text-xs"
        />
        <YAxis
          tickLine={false}
          axisLine={false}
          tickFormatter={formatYAxis}
          className="text-xs"
        />
        {showTooltip && (
          <ChartTooltip
            content={<ChartTooltipContent />}
            formatter={formatTooltip}
          />
        )}
        {lines.map((line) => (
          <Line
            key={line.dataKey}
            type="monotone"
            dataKey={line.dataKey}
            stroke={line.color}
            strokeWidth={line.strokeWidth || 2}
            dot={false}
            activeDot={{ r: 4 }}
          />
        ))}
      </RechartsLineChart>
    </ChartContainer>
  );
}
