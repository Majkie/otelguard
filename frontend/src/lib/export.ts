import type { Trace } from '@/api/traces';

/**
 * Export traces to JSON format and trigger download
 */
export function exportTracesToJson(traces: Trace[], filename: string = 'traces') {
  const jsonData = JSON.stringify(traces, null, 2);
  const blob = new Blob([jsonData], { type: 'application/json' });
  downloadBlob(blob, `${filename}.json`);
}

/**
 * Export traces to CSV format and trigger download
 */
export function exportTracesToCsv(traces: Trace[], filename: string = 'traces') {
  const headers = [
    'ID',
    'Name',
    'Model',
    'Status',
    'Latency (ms)',
    'Total Tokens',
    'Prompt Tokens',
    'Completion Tokens',
    'Cost',
    'Session ID',
    'User ID',
    'Start Time',
    'End Time',
    'Tags',
    'Error Message',
  ];

  const rows = traces.map((trace) => [
    trace.id,
    escapeCSV(trace.name),
    escapeCSV(trace.model || ''),
    trace.status,
    trace.latencyMs.toString(),
    trace.totalTokens.toString(),
    trace.promptTokens.toString(),
    trace.completionTokens.toString(),
    trace.cost.toFixed(6),
    escapeCSV(trace.sessionId || ''),
    escapeCSV(trace.userId || ''),
    trace.startTime,
    trace.endTime,
    escapeCSV((trace.tags || []).join(', ')),
    escapeCSV(trace.errorMessage || ''),
  ]);

  const csvContent = [
    headers.join(','),
    ...rows.map((row) => row.join(',')),
  ].join('\n');

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  downloadBlob(blob, `${filename}.csv`);
}

/**
 * Export trace detail with spans to JSON
 */
export function exportTraceDetailToJson(trace: Trace, spans: unknown[], filename?: string) {
  const exportData = {
    trace,
    spans,
    exportedAt: new Date().toISOString(),
  };
  const jsonData = JSON.stringify(exportData, null, 2);
  const blob = new Blob([jsonData], { type: 'application/json' });
  downloadBlob(blob, filename || `trace-${trace.id}.json`);
}

/**
 * Escape CSV special characters
 */
function escapeCSV(value: string): string {
  if (value.includes(',') || value.includes('"') || value.includes('\n')) {
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

/**
 * Trigger blob download
 */
function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
