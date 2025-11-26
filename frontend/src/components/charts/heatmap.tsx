import { cn } from '@/lib/utils';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface HeatmapCell {
  x: string | number;
  y: string | number;
  value: number;
  label?: string;
}

interface HeatmapProps {
  data: HeatmapCell[];
  xLabels: (string | number)[];
  yLabels: (string | number)[];
  colorScale?: {
    min: string;
    mid?: string;
    max: string;
  };
  height?: number;
  cellSize?: number;
  showValues?: boolean;
  formatValue?: (value: number) => string;
  formatTooltip?: (cell: HeatmapCell) => string;
}

export function Heatmap({
  data,
  xLabels,
  yLabels,
  colorScale = {
    min: 'hsl(var(--chart-1))',
    mid: 'hsl(var(--chart-3))',
    max: 'hsl(var(--chart-5))',
  },
  height = 400,
  cellSize = 60,
  showValues = false,
  formatValue = (v) => v.toString(),
  formatTooltip,
}: HeatmapProps) {
  const maxValue = Math.max(...data.map((d) => d.value));
  const minValue = Math.min(...data.map((d) => d.value));

  const getColor = (value: number): string => {
    if (maxValue === minValue) return colorScale.min;

    const normalized = (value - minValue) / (maxValue - minValue);

    if (colorScale.mid) {
      if (normalized < 0.5) {
        return interpolateColor(colorScale.min, colorScale.mid, normalized * 2);
      } else {
        return interpolateColor(colorScale.mid, colorScale.max, (normalized - 0.5) * 2);
      }
    }

    return interpolateColor(colorScale.min, colorScale.max, normalized);
  };

  const getCellData = (x: string | number, y: string | number): HeatmapCell | undefined => {
    return data.find((cell) => cell.x === x && cell.y === y);
  };

  return (
    <div className="w-full overflow-x-auto" style={{ height }}>
      <div className="inline-block min-w-full">
        <div className="flex">
          {/* Y-axis labels */}
          <div className="flex flex-col justify-around pr-2">
            <div style={{ height: cellSize }} /> {/* Header spacer */}
            {yLabels.map((label, i) => (
              <div
                key={i}
                className="flex items-center justify-end text-xs text-muted-foreground"
                style={{ height: cellSize }}
              >
                {label}
              </div>
            ))}
          </div>

          {/* Grid */}
          <div>
            {/* X-axis labels */}
            <div className="flex mb-2">
              {xLabels.map((label, i) => (
                <div
                  key={i}
                  className="flex items-center justify-center text-xs text-muted-foreground"
                  style={{ width: cellSize }}
                >
                  {label}
                </div>
              ))}
            </div>

            {/* Cells */}
            {yLabels.map((yLabel, yi) => (
              <div key={yi} className="flex">
                {xLabels.map((xLabel, xi) => {
                  const cellData = getCellData(xLabel, yLabel);
                  const value = cellData?.value ?? 0;
                  const color = getColor(value);

                  return (
                    <TooltipProvider key={xi}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <div
                            className={cn(
                              'border border-border flex items-center justify-center text-xs font-medium cursor-default',
                              'transition-all hover:ring-2 hover:ring-primary hover:z-10'
                            )}
                            style={{
                              width: cellSize,
                              height: cellSize,
                              backgroundColor: color,
                              color: value > (maxValue - minValue) / 2 + minValue ? 'white' : 'black',
                            }}
                          >
                            {showValues && cellData && formatValue(value)}
                          </div>
                        </TooltipTrigger>
                        <TooltipContent>
                          {formatTooltip && cellData ? (
                            formatTooltip(cellData)
                          ) : (
                            <div>
                              <div className="font-medium">{`${yLabel} × ${xLabel}`}</div>
                              <div className="text-xs text-muted-foreground">
                                {cellData ? formatValue(value) : 'No data'}
                              </div>
                            </div>
                          )}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  );
                })}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// Helper function to interpolate between two HSL colors
function interpolateColor(color1: string, color2: string, factor: number): string {
  // Simple interpolation for HSL values
  // In production, you might want to use a proper color library
  const hsl1 = parseHSL(color1);
  const hsl2 = parseHSL(color2);

  if (!hsl1 || !hsl2) return color1;

  const h = hsl1.h + (hsl2.h - hsl1.h) * factor;
  const s = hsl1.s + (hsl2.s - hsl1.s) * factor;
  const l = hsl1.l + (hsl2.l - hsl1.l) * factor;

  return `hsl(${h}, ${s}%, ${l}%)`;
}

function parseHSL(hsl: string): { h: number; s: number; l: number } | null {
  const match = hsl.match(/hsl\(var\(--chart-(\d+)\)\)/);
  if (match) {
    const chartNum = parseInt(match[1]);
    // Approximate values - in production, read from CSS variables
    return {
      h: chartNum * 60,
      s: 70,
      l: 50,
    };
  }

  const directMatch = hsl.match(/hsl\((\d+),\s*(\d+)%,\s*(\d+)%\)/);
  if (directMatch) {
    return {
      h: parseInt(directMatch[1]),
      s: parseInt(directMatch[2]),
      l: parseInt(directMatch[3]),
    };
  }

  return null;
}
