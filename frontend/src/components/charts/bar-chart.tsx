import { Bar, BarChart as RechartsBarChart, XAxis, YAxis, CartesianGrid, ResponsiveContainer } from 'recharts';
import { ChartContainer, ChartTooltip, ChartTooltipContent, ChartConfig } from '@/components/ui/chart';

interface BarChartProps {
  data: any[];
  xKey: string;
  bars: Array<{
    dataKey: string;
    color: string;
    label: string;
    stackId?: string;
  }>;
  height?: number;
  showGrid?: boolean;
  showTooltip?: boolean;
  formatXAxis?: (value: any) => string;
  formatYAxis?: (value: any) => string;
  formatTooltip?: (value: any) => string;
  layout?: 'horizontal' | 'vertical';
}

export function BarChart({
  data,
  xKey,
  bars,
  height = 350,
  showGrid = true,
  showTooltip = true,
  formatXAxis,
  formatYAxis,
  formatTooltip,
  layout = 'horizontal',
}: BarChartProps) {
  const chartConfig: ChartConfig = bars.reduce((acc, bar) => {
    acc[bar.dataKey] = {
      label: bar.label,
      color: bar.color,
    };
    return acc;
  }, {} as ChartConfig);

  return (
    <ChartContainer config={chartConfig} className="w-full" style={{ height }}>
      <RechartsBarChart
        data={data}
        margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
        layout={layout}
      >
        {showGrid && <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />}
        {layout === 'horizontal' ? (
          <>
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
          </>
        ) : (
          <>
            <XAxis
              type="number"
              tickLine={false}
              axisLine={false}
              tickFormatter={formatYAxis}
              className="text-xs"
            />
            <YAxis
              type="category"
              dataKey={xKey}
              tickLine={false}
              axisLine={false}
              tickFormatter={formatXAxis}
              className="text-xs"
            />
          </>
        )}
        {showTooltip && (
          <ChartTooltip
            content={<ChartTooltipContent />}
            formatter={formatTooltip}
          />
        )}
        {bars.map((bar) => (
          <Bar
            key={bar.dataKey}
            dataKey={bar.dataKey}
            fill={bar.color}
            stackId={bar.stackId}
            radius={[4, 4, 0, 0]}
          />
        ))}
      </RechartsBarChart>
    </ChartContainer>
  );
}
