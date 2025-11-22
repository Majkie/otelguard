import { useState, useMemo } from 'react';
import { diffLines, diffWords, Change } from 'diff';
import { cn } from '@/lib/utils';

interface DiffViewerProps {
  left: string;
  right: string;
  leftLabel?: string;
  rightLabel?: string;
  mode?: 'unified' | 'split';
  className?: string;
}

export function DiffViewer({
  left,
  right,
  leftLabel = 'Original',
  rightLabel = 'Changed',
  mode = 'split',
  className,
}: DiffViewerProps) {
  const [viewMode, setViewMode] = useState<'unified' | 'split'>(mode);

  const lineDiff = useMemo(() => diffLines(left, right), [left, right]);

  if (viewMode === 'unified') {
    return (
      <div className={cn('rounded-lg border overflow-hidden', className)}>
        <DiffHeader
          leftLabel={leftLabel}
          rightLabel={rightLabel}
          viewMode={viewMode}
          onViewModeChange={setViewMode}
        />
        <UnifiedDiffView diff={lineDiff} />
      </div>
    );
  }

  return (
    <div className={cn('rounded-lg border overflow-hidden', className)}>
      <DiffHeader
        leftLabel={leftLabel}
        rightLabel={rightLabel}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
      />
      <SplitDiffView left={left} right={right} diff={lineDiff} />
    </div>
  );
}

interface DiffHeaderProps {
  leftLabel: string;
  rightLabel: string;
  viewMode: 'unified' | 'split';
  onViewModeChange: (mode: 'unified' | 'split') => void;
}

function DiffHeader({
  leftLabel,
  rightLabel,
  viewMode,
  onViewModeChange,
}: DiffHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-2 bg-muted border-b">
      <div className="flex items-center gap-4 text-sm">
        <span className="font-medium text-red-600 dark:text-red-400">
          - {leftLabel}
        </span>
        <span className="font-medium text-green-600 dark:text-green-400">
          + {rightLabel}
        </span>
      </div>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onViewModeChange('split')}
          className={cn(
            'px-2 py-1 text-xs rounded',
            viewMode === 'split'
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-muted-foreground/10'
          )}
        >
          Split
        </button>
        <button
          onClick={() => onViewModeChange('unified')}
          className={cn(
            'px-2 py-1 text-xs rounded',
            viewMode === 'unified'
              ? 'bg-primary text-primary-foreground'
              : 'hover:bg-muted-foreground/10'
          )}
        >
          Unified
        </button>
      </div>
    </div>
  );
}

interface UnifiedDiffViewProps {
  diff: Change[];
}

function UnifiedDiffView({ diff }: UnifiedDiffViewProps) {
  let leftLineNum = 1;
  let rightLineNum = 1;

  return (
    <div className="overflow-auto max-h-[500px] font-mono text-sm">
      {diff.map((part, index) => {
        const lines = part.value.split('\n').filter((_, i, arr) =>
          i < arr.length - 1 || arr[i] !== ''
        );

        return lines.map((line, lineIndex) => {
          let leftNum = '';
          let rightNum = '';

          if (part.added) {
            rightNum = String(rightLineNum++);
          } else if (part.removed) {
            leftNum = String(leftLineNum++);
          } else {
            leftNum = String(leftLineNum++);
            rightNum = String(rightLineNum++);
          }

          return (
            <div
              key={`${index}-${lineIndex}`}
              className={cn(
                'flex',
                part.added && 'bg-green-50 dark:bg-green-950/30',
                part.removed && 'bg-red-50 dark:bg-red-950/30'
              )}
            >
              <span className="w-12 px-2 py-0.5 text-right text-muted-foreground border-r bg-muted/50 select-none">
                {leftNum}
              </span>
              <span className="w-12 px-2 py-0.5 text-right text-muted-foreground border-r bg-muted/50 select-none">
                {rightNum}
              </span>
              <span className="w-6 px-1 py-0.5 text-center select-none">
                {part.added ? (
                  <span className="text-green-600 dark:text-green-400">+</span>
                ) : part.removed ? (
                  <span className="text-red-600 dark:text-red-400">-</span>
                ) : (
                  ' '
                )}
              </span>
              <span className="flex-1 px-2 py-0.5 whitespace-pre-wrap break-all">
                {line || ' '}
              </span>
            </div>
          );
        });
      })}
    </div>
  );
}

interface SplitDiffViewProps {
  left: string;
  right: string;
  diff: Change[];
}

function SplitDiffView({ left, right, diff }: SplitDiffViewProps) {
  const leftLines = left.split('\n');
  const rightLines = right.split('\n');

  // Build aligned lines for split view
  const alignedLines: Array<{
    leftLine: string | null;
    rightLine: string | null;
    leftNum: number | null;
    rightNum: number | null;
    type: 'unchanged' | 'added' | 'removed' | 'modified';
  }> = [];

  let leftIdx = 0;
  let rightIdx = 0;

  for (const part of diff) {
    const lines = part.value.split('\n');
    // Remove trailing empty string from split
    if (lines[lines.length - 1] === '') {
      lines.pop();
    }

    if (part.removed) {
      for (const line of lines) {
        alignedLines.push({
          leftLine: line,
          rightLine: null,
          leftNum: leftIdx + 1,
          rightNum: null,
          type: 'removed',
        });
        leftIdx++;
      }
    } else if (part.added) {
      // Check if we can pair with previous removed lines
      let i = alignedLines.length - 1;
      let pairCount = 0;
      while (i >= 0 && alignedLines[i].type === 'removed' && alignedLines[i].rightLine === null) {
        pairCount++;
        i--;
      }

      for (const line of lines) {
        if (pairCount > 0) {
          // Pair with a removed line to show as modified
          const pairIdx = alignedLines.length - pairCount;
          alignedLines[pairIdx].rightLine = line;
          alignedLines[pairIdx].rightNum = rightIdx + 1;
          alignedLines[pairIdx].type = 'modified';
          pairCount--;
        } else {
          alignedLines.push({
            leftLine: null,
            rightLine: line,
            leftNum: null,
            rightNum: rightIdx + 1,
            type: 'added',
          });
        }
        rightIdx++;
      }
    } else {
      for (const line of lines) {
        alignedLines.push({
          leftLine: line,
          rightLine: line,
          leftNum: leftIdx + 1,
          rightNum: rightIdx + 1,
          type: 'unchanged',
        });
        leftIdx++;
        rightIdx++;
      }
    }
  }

  return (
    <div className="grid grid-cols-2 divide-x overflow-auto max-h-[500px]">
      {/* Left side */}
      <div className="font-mono text-sm">
        {alignedLines.map((line, index) => (
          <div
            key={`left-${index}`}
            className={cn(
              'flex',
              (line.type === 'removed' || line.type === 'modified') &&
                'bg-red-50 dark:bg-red-950/30'
            )}
          >
            <span className="w-10 px-2 py-0.5 text-right text-muted-foreground border-r bg-muted/50 select-none">
              {line.leftNum ?? ''}
            </span>
            <span className="flex-1 px-2 py-0.5 whitespace-pre-wrap break-all">
              {line.leftLine ?? ''}
            </span>
          </div>
        ))}
      </div>

      {/* Right side */}
      <div className="font-mono text-sm">
        {alignedLines.map((line, index) => (
          <div
            key={`right-${index}`}
            className={cn(
              'flex',
              (line.type === 'added' || line.type === 'modified') &&
                'bg-green-50 dark:bg-green-950/30'
            )}
          >
            <span className="w-10 px-2 py-0.5 text-right text-muted-foreground border-r bg-muted/50 select-none">
              {line.rightNum ?? ''}
            </span>
            <span className="flex-1 px-2 py-0.5 whitespace-pre-wrap break-all">
              {line.rightLine ?? ''}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// Simple inline diff for short text
export function InlineDiff({
  left,
  right,
  className,
}: {
  left: string;
  right: string;
  className?: string;
}) {
  const wordDiff = useMemo(() => diffWords(left, right), [left, right]);

  return (
    <span className={cn('font-mono text-sm', className)}>
      {wordDiff.map((part, index) => (
        <span
          key={index}
          className={cn(
            part.added && 'bg-green-200 dark:bg-green-800 text-green-900 dark:text-green-100',
            part.removed && 'bg-red-200 dark:bg-red-800 text-red-900 dark:text-red-100 line-through'
          )}
        >
          {part.value}
        </span>
      ))}
    </span>
  );
}
