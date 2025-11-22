import { useState, useMemo } from 'react';
import { ChevronRight, ChevronDown, Copy, Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';

interface JsonViewerProps {
  data: string | object;
  defaultExpanded?: boolean;
  maxDepth?: number;
  className?: string;
}

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

export function JsonViewer({
  data,
  defaultExpanded = true,
  maxDepth = 5,
  className,
}: JsonViewerProps) {
  const [copied, setCopied] = useState(false);

  const parsedData = useMemo(() => {
    if (typeof data === 'string') {
      try {
        return JSON.parse(data);
      } catch {
        return data;
      }
    }
    return data;
  }, [data]);

  const handleCopy = async () => {
    const text = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // If it's a plain string (not JSON), render as-is
  if (typeof parsedData === 'string') {
    return (
      <div className={cn('relative group', className)}>
        <Button
          variant="ghost"
          size="icon"
          className="absolute right-2 top-2 h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity"
          onClick={handleCopy}
        >
          {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
        </Button>
        <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto max-h-[400px] whitespace-pre-wrap font-mono">
          {parsedData}
        </pre>
      </div>
    );
  }

  return (
    <div className={cn('relative group', className)}>
      <Button
        variant="ghost"
        size="icon"
        className="absolute right-2 top-2 h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity z-10"
        onClick={handleCopy}
      >
        {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
      </Button>
      <div className="bg-muted p-4 rounded-lg overflow-auto max-h-[400px] font-mono text-sm">
        <JsonNode
          value={parsedData}
          depth={0}
          defaultExpanded={defaultExpanded}
          maxDepth={maxDepth}
        />
      </div>
    </div>
  );
}

interface JsonNodeProps {
  value: JsonValue;
  depth: number;
  keyName?: string;
  defaultExpanded: boolean;
  maxDepth: number;
  isLast?: boolean;
}

function JsonNode({
  value,
  depth,
  keyName,
  defaultExpanded,
  maxDepth,
  isLast = true,
}: JsonNodeProps) {
  const [isExpanded, setIsExpanded] = useState(depth < maxDepth && defaultExpanded);

  const isObject = value !== null && typeof value === 'object';
  const isArray = Array.isArray(value);

  const renderValue = () => {
    if (value === null) {
      return <span className="text-orange-600 dark:text-orange-400">null</span>;
    }

    switch (typeof value) {
      case 'string':
        return (
          <span className="text-green-600 dark:text-green-400">
            "{value.length > 100 ? value.slice(0, 100) + '...' : value}"
          </span>
        );
      case 'number':
        return <span className="text-blue-600 dark:text-blue-400">{value}</span>;
      case 'boolean':
        return (
          <span className="text-purple-600 dark:text-purple-400">
            {value.toString()}
          </span>
        );
      default:
        return null;
    }
  };

  const renderKey = () => {
    if (keyName === undefined) return null;
    return (
      <span className="text-foreground">
        "{keyName}":
      </span>
    );
  };

  if (!isObject) {
    return (
      <div className="leading-relaxed" style={{ paddingLeft: `${depth * 16}px` }}>
        {renderKey()}
        {renderValue()}
        {!isLast && <span className="text-muted-foreground">,</span>}
      </div>
    );
  }

  const entries = isArray
    ? (value as JsonValue[]).map((v, i) => [i.toString(), v] as const)
    : Object.entries(value as Record<string, JsonValue>);

  const bracketOpen = isArray ? '[' : '{';
  const bracketClose = isArray ? ']' : '}';

  if (entries.length === 0) {
    return (
      <div className="leading-relaxed" style={{ paddingLeft: `${depth * 16}px` }}>
        {renderKey()}
        <span className="text-muted-foreground">
          {bracketOpen}{bracketClose}
        </span>
        {!isLast && <span className="text-muted-foreground">,</span>}
      </div>
    );
  }

  return (
    <div style={{ paddingLeft: depth > 0 ? `${depth * 16}px` : 0 }}>
      <div
        className="flex items-center cursor-pointer hover:bg-muted-foreground/10 rounded -ml-4 pl-4"
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isExpanded ? (
          <ChevronDown className="h-3 w-3 mr-1 text-muted-foreground shrink-0" />
        ) : (
          <ChevronRight className="h-3 w-3 mr-1 text-muted-foreground shrink-0" />
        )}
        {renderKey()}
        <span className="text-muted-foreground">{bracketOpen}</span>
        {!isExpanded && (
          <>
            <span className="text-muted-foreground mx-1">
              {isArray ? `${entries.length} items` : `${entries.length} keys`}
            </span>
            <span className="text-muted-foreground">{bracketClose}</span>
            {!isLast && <span className="text-muted-foreground">,</span>}
          </>
        )}
      </div>

      {isExpanded && (
        <>
          {entries.map(([key, val], index) => (
            <JsonNode
              key={key}
              keyName={isArray ? undefined : key}
              value={val as JsonValue}
              depth={depth + 1}
              defaultExpanded={defaultExpanded}
              maxDepth={maxDepth}
              isLast={index === entries.length - 1}
            />
          ))}
          <div style={{ paddingLeft: `${depth * 16}px` }}>
            <span className="text-muted-foreground">{bracketClose}</span>
            {!isLast && <span className="text-muted-foreground">,</span>}
          </div>
        </>
      )}
    </div>
  );
}

// Simple raw viewer for non-JSON content
export function RawViewer({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={cn('relative group', className)}>
      <Button
        variant="ghost"
        size="icon"
        className="absolute right-2 top-2 h-7 w-7 opacity-0 group-hover:opacity-100 transition-opacity"
        onClick={handleCopy}
      >
        {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
      </Button>
      <pre className="text-sm bg-muted p-4 rounded-lg overflow-auto max-h-[400px] whitespace-pre-wrap font-mono">
        {content || 'No content'}
      </pre>
    </div>
  );
}
