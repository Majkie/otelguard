# CLAUDE.md - OTelGuard Development Guide

This document contains best practices, coding standards, and development guidelines for the OTelGuard project.

---

## Project Structure

```
otelguard/
├── backend/                 # Go backend
│   ├── cmd/
│   │   └── server/         # Main application entry point
│   │       └── main.go
│   ├── internal/           # Private application code
│   │   ├── api/           # HTTP handlers and routes
│   │   │   ├── handlers/  # Request handlers
│   │   │   ├── middleware/# HTTP middleware
│   │   │   └── routes.go  # Route definitions
│   │   ├── config/        # Configuration management
│   │   ├── domain/        # Business logic and entities
│   │   │   ├── trace/     # Trace domain
│   │   │   ├── prompt/    # Prompt domain
│   │   │   ├── guardrail/ # Guardrail domain
│   │   │   └── eval/      # Evaluation domain
│   │   ├── repository/    # Data access layer
│   │   │   ├── postgres/  # PostgreSQL repositories
│   │   │   └── clickhouse/# ClickHouse repositories
│   │   └── service/       # Application services
│   ├── pkg/               # Public libraries
│   │   ├── otel/         # OpenTelemetry utilities
│   │   └── validator/    # Validation utilities
│   ├── migrations/        # Database migrations
│   ├── go.mod
│   └── go.sum
├── frontend/              # React frontend
│   ├── src/
│   │   ├── components/   # Reusable UI components
│   │   │   ├── ui/      # ShadCN components
│   │   │   └── features/# Feature-specific components
│   │   ├── pages/       # Page components
│   │   ├── hooks/       # Custom React hooks
│   │   ├── lib/         # Utility functions
│   │   ├── api/         # API client and queries
│   │   ├── stores/      # State management
│   │   ├── types/       # TypeScript types
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── sdk/                   # Client SDKs
│   ├── python/
│   ├── typescript/
│   └── go/
├── deploy/               # Deployment configurations
│   ├── docker/
│   └── kubernetes/
├── docs/                 # Documentation
├── README.md
├── PLAN.md
├── CLAUDE.md
└── docker-compose.yml
```

---

## Go Backend Best Practices

### Project Configuration

```go
// internal/config/config.go
package config

import (
    "github.com/kelseyhightower/envconfig"
)

type Config struct {
    Server   ServerConfig
    Postgres PostgresConfig
    ClickHouse ClickHouseConfig
    Auth     AuthConfig
}

type ServerConfig struct {
    Port         int    `envconfig:"PORT" default:"8080"`
    Environment  string `envconfig:"ENV" default:"development"`
    ReadTimeout  int    `envconfig:"READ_TIMEOUT" default:"30"`
    WriteTimeout int    `envconfig:"WRITE_TIMEOUT" default:"30"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := envconfig.Process("OTELGUARD", &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### Gin Router Setup

```go
// internal/api/routes.go
package api

import (
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "github.com/gin-contrib/requestid"
)

func SetupRouter(handlers *Handlers, cfg *config.Config) *gin.Engine {
    if cfg.Server.Environment == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    r := gin.New()

    // Middleware
    r.Use(gin.Recovery())
    r.Use(requestid.New())
    r.Use(LoggerMiddleware())
    r.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:3000"},
        AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        AllowCredentials: true,
    }))

    // Health check
    r.GET("/health", handlers.Health)
    r.GET("/ready", handlers.Ready)

    // API v1
    v1 := r.Group("/v1")
    {
        // Public routes
        v1.POST("/auth/login", handlers.Login)
        v1.POST("/auth/register", handlers.Register)

        // Protected routes
        protected := v1.Group("")
        protected.Use(AuthMiddleware())
        {
            // Traces
            traces := protected.Group("/traces")
            traces.POST("", handlers.IngestTrace)
            traces.GET("", handlers.ListTraces)
            traces.GET("/:id", handlers.GetTrace)

            // Prompts
            prompts := protected.Group("/prompts")
            prompts.GET("", handlers.ListPrompts)
            prompts.POST("", handlers.CreatePrompt)
            prompts.GET("/:id", handlers.GetPrompt)
            prompts.PUT("/:id", handlers.UpdatePrompt)

            // Guardrails
            guardrails := protected.Group("/guardrails")
            guardrails.GET("", handlers.ListGuardrails)
            guardrails.POST("", handlers.CreateGuardrail)
            guardrails.POST("/evaluate", handlers.EvaluateGuardrail)
        }
    }

    return r
}
```

### Handler Pattern

```go
// internal/api/handlers/trace_handler.go
package handlers

import (
    "net/http"
    "github.com/gin-gonic/gin"
)

type TraceHandler struct {
    traceService *service.TraceService
    logger       *zap.Logger
}

func NewTraceHandler(ts *service.TraceService, logger *zap.Logger) *TraceHandler {
    return &TraceHandler{
        traceService: ts,
        logger:       logger,
    }
}

// IngestTrace handles trace ingestion
// @Summary Ingest trace data
// @Tags traces
// @Accept json
// @Produce json
// @Param trace body IngestTraceRequest true "Trace data"
// @Success 201 {object} IngestTraceResponse
// @Failure 400 {object} ErrorResponse
// @Router /v1/traces [post]
func (h *TraceHandler) IngestTrace(c *gin.Context) {
    var req IngestTraceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error:   "invalid_request",
            Message: err.Error(),
        })
        return
    }

    // Get project from context (set by auth middleware)
    projectID := c.GetString("project_id")

    trace, err := h.traceService.Ingest(c.Request.Context(), projectID, &req)
    if err != nil {
        h.logger.Error("failed to ingest trace",
            zap.Error(err),
            zap.String("project_id", projectID),
        )
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error:   "internal_error",
            Message: "failed to process trace",
        })
        return
    }

    c.JSON(http.StatusCreated, IngestTraceResponse{
        TraceID: trace.ID,
    })
}

// ListTraces returns paginated traces
func (h *TraceHandler) ListTraces(c *gin.Context) {
    var query ListTracesQuery
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(http.StatusBadRequest, ErrorResponse{
            Error:   "invalid_query",
            Message: err.Error(),
        })
        return
    }

    // Set defaults
    if query.Limit == 0 {
        query.Limit = 50
    }
    if query.Limit > 100 {
        query.Limit = 100
    }

    projectID := c.GetString("project_id")

    traces, total, err := h.traceService.List(c.Request.Context(), projectID, &query)
    if err != nil {
        h.logger.Error("failed to list traces", zap.Error(err))
        c.JSON(http.StatusInternalServerError, ErrorResponse{
            Error:   "internal_error",
            Message: "failed to retrieve traces",
        })
        return
    }

    c.JSON(http.StatusOK, ListTracesResponse{
        Data:   traces,
        Total:  total,
        Limit:  query.Limit,
        Offset: query.Offset,
    })
}
```

### Repository Pattern

```go
// internal/repository/postgres/prompt_repo.go
package postgres

import (
    "context"
    "database/sql"
    "github.com/jmoiron/sqlx"
)

type PromptRepository struct {
    db *sqlx.DB
}

func NewPromptRepository(db *sqlx.DB) *PromptRepository {
    return &PromptRepository{db: db}
}

func (r *PromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
    query := `
        INSERT INTO prompts (id, project_id, name, description, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
    _, err := r.db.ExecContext(ctx, query,
        prompt.ID,
        prompt.ProjectID,
        prompt.Name,
        prompt.Description,
        prompt.CreatedAt,
        prompt.UpdatedAt,
    )
    return err
}

func (r *PromptRepository) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
    var prompt domain.Prompt
    query := `
        SELECT id, project_id, name, description, created_at, updated_at
        FROM prompts
        WHERE id = $1 AND deleted_at IS NULL
    `
    err := r.db.GetContext(ctx, &prompt, query, id)
    if err == sql.ErrNoRows {
        return nil, domain.ErrNotFound
    }
    return &prompt, err
}

func (r *PromptRepository) List(ctx context.Context, projectID string, opts *ListOptions) ([]*domain.Prompt, int, error) {
    var prompts []*domain.Prompt
    var total int

    // Count query
    countQuery := `SELECT COUNT(*) FROM prompts WHERE project_id = $1 AND deleted_at IS NULL`
    if err := r.db.GetContext(ctx, &total, countQuery, projectID); err != nil {
        return nil, 0, err
    }

    // List query with pagination
    listQuery := `
        SELECT id, project_id, name, description, created_at, updated_at
        FROM prompts
        WHERE project_id = $1 AND deleted_at IS NULL
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `
    if err := r.db.SelectContext(ctx, &prompts, listQuery, projectID, opts.Limit, opts.Offset); err != nil {
        return nil, 0, err
    }

    return prompts, total, nil
}
```

### ClickHouse Repository

```go
// internal/repository/clickhouse/trace_repo.go
package clickhouse

import (
    "context"
    "github.com/ClickHouse/clickhouse-go/v2"
)

type TraceRepository struct {
    conn clickhouse.Conn
}

func NewTraceRepository(conn clickhouse.Conn) *TraceRepository {
    return &TraceRepository{conn: conn}
}

func (r *TraceRepository) Insert(ctx context.Context, traces []*domain.Trace) error {
    batch, err := r.conn.PrepareBatch(ctx, `
        INSERT INTO traces (
            id, project_id, session_id, user_id, name,
            input, output, metadata, start_time, end_time,
            latency_ms, total_tokens, prompt_tokens, completion_tokens,
            cost, model, tags
        )
    `)
    if err != nil {
        return err
    }

    for _, trace := range traces {
        err := batch.Append(
            trace.ID,
            trace.ProjectID,
            trace.SessionID,
            trace.UserID,
            trace.Name,
            trace.Input,
            trace.Output,
            trace.Metadata,
            trace.StartTime,
            trace.EndTime,
            trace.LatencyMs,
            trace.TotalTokens,
            trace.PromptTokens,
            trace.CompletionTokens,
            trace.Cost,
            trace.Model,
            trace.Tags,
        )
        if err != nil {
            return err
        }
    }

    return batch.Send()
}

func (r *TraceRepository) Query(ctx context.Context, projectID string, opts *QueryOptions) ([]*domain.Trace, error) {
    query := `
        SELECT
            id, project_id, session_id, user_id, name,
            input, output, metadata, start_time, end_time,
            latency_ms, total_tokens, prompt_tokens, completion_tokens,
            cost, model, tags
        FROM traces
        WHERE project_id = ?
          AND start_time >= ?
          AND start_time <= ?
        ORDER BY start_time DESC
        LIMIT ?
        OFFSET ?
    `

    rows, err := r.conn.Query(ctx, query,
        projectID,
        opts.StartTime,
        opts.EndTime,
        opts.Limit,
        opts.Offset,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var traces []*domain.Trace
    for rows.Next() {
        var t domain.Trace
        if err := rows.ScanStruct(&t); err != nil {
            return nil, err
        }
        traces = append(traces, &t)
    }

    return traces, nil
}
```

### Error Handling

```go
// internal/domain/errors.go
package domain

import "errors"

var (
    ErrNotFound      = errors.New("resource not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrValidation    = errors.New("validation error")
    ErrConflict      = errors.New("resource conflict")
    ErrInternal      = errors.New("internal error")
)

// ValidationError contains field-level validation errors
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
    if len(ve) == 0 {
        return "validation failed"
    }
    return ve[0].Message
}
```

### Dependency Injection

```go
// cmd/server/main.go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("failed to load config:", err)
    }

    // Initialize logger
    logger, err := initLogger(cfg.Server.Environment)
    if err != nil {
        log.Fatal("failed to init logger:", err)
    }

    // Initialize databases
    pgDB, err := initPostgres(cfg.Postgres)
    if err != nil {
        logger.Fatal("failed to connect to postgres", zap.Error(err))
    }
    defer pgDB.Close()

    chConn, err := initClickHouse(cfg.ClickHouse)
    if err != nil {
        logger.Fatal("failed to connect to clickhouse", zap.Error(err))
    }
    defer chConn.Close()

    // Initialize repositories
    promptRepo := postgres.NewPromptRepository(pgDB)
    traceRepo := clickhouse.NewTraceRepository(chConn)

    // Initialize services
    promptService := service.NewPromptService(promptRepo, logger)
    traceService := service.NewTraceService(traceRepo, logger)

    // Initialize handlers
    handlers := &api.Handlers{
        Prompt: handlers.NewPromptHandler(promptService, logger),
        Trace:  handlers.NewTraceHandler(traceService, logger),
    }

    // Setup router
    router := api.SetupRouter(handlers, cfg)

    // Create server
    srv := &http.Server{
        Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
        Handler:      router,
        ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
        WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
    }

    // Graceful shutdown
    go func() {
        logger.Info("starting server", zap.Int("port", cfg.Server.Port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("server error", zap.Error(err))
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("server forced shutdown", zap.Error(err))
    }

    logger.Info("server stopped")
}
```

---

## TypeScript/React Frontend Best Practices

### TanStack Query Setup

```typescript
// src/lib/query-client.ts
import { QueryClient } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      gcTime: 1000 * 60 * 5, // 5 minutes (formerly cacheTime)
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
});
```

### API Client

```typescript
// src/api/client.ts
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const { params, ...fetchOptions } = options;

    let url = `${this.baseUrl}${endpoint}`;

    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          searchParams.append(key, String(value));
        }
      });
      const queryString = searchParams.toString();
      if (queryString) {
        url += `?${queryString}`;
      }
    }

    const token = localStorage.getItem('token');

    const response = await fetch(url, {
      ...fetchOptions,
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...fetchOptions.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new ApiError(response.status, error.message || 'Request failed');
    }

    return response.json();
  }

  get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'GET' });
  }

  post<T>(endpoint: string, data?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  put<T>(endpoint: string, data?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  delete<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: 'DELETE' });
  }
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

export const api = new ApiClient(API_BASE_URL);
```

### Query Hooks Pattern

```typescript
// src/api/traces.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './client';

// Types
export interface Trace {
  id: string;
  projectId: string;
  sessionId?: string;
  userId?: string;
  name: string;
  input: string;
  output: string;
  startTime: string;
  endTime: string;
  latencyMs: number;
  totalTokens: number;
  cost: number;
  model: string;
  tags: string[];
}

export interface ListTracesParams {
  limit?: number;
  offset?: number;
  sessionId?: string;
  userId?: string;
  startTime?: string;
  endTime?: string;
}

export interface ListTracesResponse {
  data: Trace[];
  total: number;
  limit: number;
  offset: number;
}

// Query keys factory
export const traceKeys = {
  all: ['traces'] as const,
  lists: () => [...traceKeys.all, 'list'] as const,
  list: (params: ListTracesParams) => [...traceKeys.lists(), params] as const,
  details: () => [...traceKeys.all, 'detail'] as const,
  detail: (id: string) => [...traceKeys.details(), id] as const,
};

// Hooks
export function useTraces(params: ListTracesParams = {}) {
  return useQuery({
    queryKey: traceKeys.list(params),
    queryFn: () =>
      api.get<ListTracesResponse>('/v1/traces', { params }),
  });
}

export function useTrace(id: string) {
  return useQuery({
    queryKey: traceKeys.detail(id),
    queryFn: () => api.get<Trace>(`/v1/traces/${id}`),
    enabled: !!id,
  });
}

export function useDeleteTrace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/traces/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: traceKeys.lists() });
    },
  });
}
```

### TanStack Table Setup

```typescript
// src/components/features/traces/traces-table.tsx
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
  type ColumnFiltersState,
} from '@tanstack/react-table';
import { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import type { Trace } from '@/api/traces';

const columns: ColumnDef<Trace>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <span className="font-medium">{row.getValue('name')}</span>
    ),
  },
  {
    accessorKey: 'model',
    header: 'Model',
  },
  {
    accessorKey: 'latencyMs',
    header: 'Latency',
    cell: ({ row }) => `${row.getValue('latencyMs')}ms`,
  },
  {
    accessorKey: 'totalTokens',
    header: 'Tokens',
  },
  {
    accessorKey: 'cost',
    header: 'Cost',
    cell: ({ row }) => {
      const cost = row.getValue('cost') as number;
      return `$${cost.toFixed(4)}`;
    },
  },
  {
    accessorKey: 'startTime',
    header: 'Time',
    cell: ({ row }) => {
      const date = new Date(row.getValue('startTime'));
      return date.toLocaleString();
    },
  },
];

interface TracesTableProps {
  data: Trace[];
  isLoading?: boolean;
}

export function TracesTable({ data, isLoading }: TracesTableProps) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);
  const [globalFilter, setGlobalFilter] = useState('');

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    state: {
      sorting,
      columnFilters,
      globalFilter,
    },
  });

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <div className="space-y-4">
      <Input
        placeholder="Search traces..."
        value={globalFilter}
        onChange={(e) => setGlobalFilter(e.target.value)}
        className="max-w-sm"
      />

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(
                          header.column.columnDef.header,
                          header.getContext()
                        )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(
                        cell.column.columnDef.cell,
                        cell.getContext()
                      )}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="text-center">
                  No traces found.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      <div className="flex items-center justify-end space-x-2">
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.previousPage()}
          disabled={!table.getCanPreviousPage()}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => table.nextPage()}
          disabled={!table.getCanNextPage()}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
```

### Component Structure

```typescript
// src/pages/traces/index.tsx
import { useState } from 'react';
import { useTraces, type ListTracesParams } from '@/api/traces';
import { TracesTable } from '@/components/features/traces/traces-table';
import { TracesFilters } from '@/components/features/traces/traces-filters';
import { PageHeader } from '@/components/ui/page-header';

export function TracesPage() {
  const [filters, setFilters] = useState<ListTracesParams>({
    limit: 50,
    offset: 0,
  });

  const { data, isLoading, error } = useTraces(filters);

  if (error) {
    return <div>Error loading traces: {error.message}</div>;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Traces"
        description="View and analyze your LLM application traces"
      />

      <TracesFilters filters={filters} onFiltersChange={setFilters} />

      <TracesTable data={data?.data ?? []} isLoading={isLoading} />

      {data && (
        <div className="text-sm text-muted-foreground">
          Showing {data.data.length} of {data.total} traces
        </div>
      )}
    </div>
  );
}
```

### Custom Hooks

```typescript
// src/hooks/use-debounce.ts
import { useState, useEffect } from 'react';

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(handler);
    };
  }, [value, delay]);

  return debouncedValue;
}

// src/hooks/use-local-storage.ts
import { useState, useEffect } from 'react';

export function useLocalStorage<T>(key: string, initialValue: T) {
  const [storedValue, setStoredValue] = useState<T>(() => {
    try {
      const item = window.localStorage.getItem(key);
      return item ? JSON.parse(item) : initialValue;
    } catch {
      return initialValue;
    }
  });

  useEffect(() => {
    try {
      window.localStorage.setItem(key, JSON.stringify(storedValue));
    } catch (error) {
      console.error('Error saving to localStorage:', error);
    }
  }, [key, storedValue]);

  return [storedValue, setStoredValue] as const;
}
```

---

## Database Best Practices

### PostgreSQL Schema

```sql
-- migrations/001_initial_schema.sql

-- PostgreSQL 18+ has native UUID v7 support via uuidv7()
-- No extension needed - UUID v7 is time-ordered for better index performance

-- Organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, slug)
);

CREATE INDEX idx_projects_organization_id ON projects(organization_id);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    avatar_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Organization memberships
CREATE TABLE organization_members (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(10) NOT NULL,
    scopes TEXT[] DEFAULT '{}',
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_api_keys_project_id ON api_keys(project_id);
CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- Prompts
CREATE TABLE prompts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_prompts_project_id ON prompts(project_id);

-- Prompt versions
CREATE TABLE prompt_versions (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    prompt_id UUID NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    config JSONB DEFAULT '{}',
    labels TEXT[] DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(prompt_id, version)
);

CREATE INDEX idx_prompt_versions_prompt_id ON prompt_versions(prompt_id);

-- Guardrail policies
CREATE TABLE guardrail_policies (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    triggers JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_guardrail_policies_project_id ON guardrail_policies(project_id);

-- Guardrail rules
CREATE TABLE guardrail_rules (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    policy_id UUID NOT NULL REFERENCES guardrail_policies(id) ON DELETE CASCADE,
    type VARCHAR(100) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    action VARCHAR(50) NOT NULL DEFAULT 'block',
    action_config JSONB DEFAULT '{}',
    order_index INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_guardrail_rules_policy_id ON guardrail_rules(policy_id);

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to tables with updated_at
CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_prompts_updated_at BEFORE UPDATE ON prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_guardrail_policies_updated_at BEFORE UPDATE ON guardrail_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

### ClickHouse Schema

```sql
-- clickhouse/001_traces.sql

-- Traces table
CREATE TABLE IF NOT EXISTS traces (
    id UUID,
    project_id UUID,
    session_id Nullable(String),
    user_id Nullable(String),
    name String,
    input String,
    output String,
    metadata String DEFAULT '{}',
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    total_tokens UInt32 DEFAULT 0,
    prompt_tokens UInt32 DEFAULT 0,
    completion_tokens UInt32 DEFAULT 0,
    cost Decimal64(8) DEFAULT 0,
    model String DEFAULT '',
    tags Array(String) DEFAULT [],
    status String DEFAULT 'success',
    error_message Nullable(String),

    INDEX idx_session_id session_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_user_id user_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_model model TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, start_time, id)
TTL start_time + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Spans table
CREATE TABLE IF NOT EXISTS spans (
    id UUID,
    trace_id UUID,
    parent_span_id Nullable(UUID),
    project_id UUID,
    name String,
    type String,  -- 'llm', 'retrieval', 'tool', 'agent', 'embedding', 'custom'
    input String,
    output String,
    metadata String DEFAULT '{}',
    start_time DateTime64(3),
    end_time DateTime64(3),
    latency_ms UInt32,
    tokens UInt32 DEFAULT 0,
    cost Decimal64(8) DEFAULT 0,
    model Nullable(String),
    status String DEFAULT 'success',
    error_message Nullable(String),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_type type TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(start_time)
ORDER BY (project_id, trace_id, start_time, id)
TTL start_time + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Scores table
CREATE TABLE IF NOT EXISTS scores (
    id UUID,
    project_id UUID,
    trace_id UUID,
    span_id Nullable(UUID),
    name String,
    value Float64,
    string_value Nullable(String),
    data_type String,  -- 'numeric', 'boolean', 'categorical'
    source String,     -- 'api', 'llm_judge', 'human', 'user_feedback'
    config_id Nullable(UUID),
    comment Nullable(String),
    created_at DateTime64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_name name TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (project_id, trace_id, created_at, id)
TTL created_at + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Guardrail events table
CREATE TABLE IF NOT EXISTS guardrail_events (
    id UUID,
    project_id UUID,
    trace_id Nullable(UUID),
    span_id Nullable(UUID),
    policy_id UUID,
    rule_id UUID,
    rule_type String,
    triggered Bool,
    action String,
    action_taken Bool,
    input_text String,
    output_text Nullable(String),
    detection_result String,
    latency_ms UInt32,
    created_at DateTime64(3),

    INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4,
    INDEX idx_policy_id policy_id TYPE bloom_filter GRANULARITY 4
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (project_id, created_at, id)
TTL created_at + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Materialized view for trace aggregations
CREATE MATERIALIZED VIEW IF NOT EXISTS trace_daily_stats
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (project_id, date, model)
AS SELECT
    project_id,
    toDate(start_time) AS date,
    model,
    count() AS trace_count,
    sum(latency_ms) AS total_latency_ms,
    sum(total_tokens) AS total_tokens,
    sum(cost) AS total_cost,
    countIf(status = 'error') AS error_count
FROM traces
GROUP BY project_id, date, model;
```

---

## Testing Guidelines

### Go Unit Tests

```go
// internal/service/prompt_service_test.go
package service_test

import (
    "context"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockPromptRepository struct {
    mock.Mock
}

func (m *MockPromptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
    args := m.Called(ctx, prompt)
    return args.Error(0)
}

func (m *MockPromptRepository) GetByID(ctx context.Context, id string) (*domain.Prompt, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.Prompt), args.Error(1)
}

func TestPromptService_Create(t *testing.T) {
    mockRepo := new(MockPromptRepository)
    logger := zap.NewNop()
    service := NewPromptService(mockRepo, logger)

    ctx := context.Background()
    input := &CreatePromptInput{
        ProjectID:   "project-123",
        Name:        "Test Prompt",
        Description: "A test prompt",
    }

    mockRepo.On("Create", ctx, mock.AnythingOfType("*domain.Prompt")).Return(nil)

    prompt, err := service.Create(ctx, input)

    assert.NoError(t, err)
    assert.NotEmpty(t, prompt.ID)
    assert.Equal(t, input.Name, prompt.Name)
    mockRepo.AssertExpectations(t)
}

func TestPromptService_GetByID_NotFound(t *testing.T) {
    mockRepo := new(MockPromptRepository)
    logger := zap.NewNop()
    service := NewPromptService(mockRepo, logger)

    ctx := context.Background()
    id := "non-existent-id"

    mockRepo.On("GetByID", ctx, id).Return(nil, domain.ErrNotFound)

    prompt, err := service.GetByID(ctx, id)

    assert.Nil(t, prompt)
    assert.ErrorIs(t, err, domain.ErrNotFound)
    mockRepo.AssertExpectations(t)
}
```

### React Component Tests

```typescript
// src/components/features/traces/__tests__/traces-table.test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TracesTable } from '../traces-table';

const mockTraces = [
  {
    id: '1',
    name: 'chat-completion',
    model: 'gpt-4',
    latencyMs: 1500,
    totalTokens: 500,
    cost: 0.015,
    startTime: '2024-01-01T12:00:00Z',
  },
  {
    id: '2',
    name: 'embedding',
    model: 'text-embedding-ada-002',
    latencyMs: 200,
    totalTokens: 100,
    cost: 0.0001,
    startTime: '2024-01-01T12:01:00Z',
  },
];

describe('TracesTable', () => {
  it('renders traces correctly', () => {
    render(<TracesTable data={mockTraces} />);

    expect(screen.getByText('chat-completion')).toBeInTheDocument();
    expect(screen.getByText('gpt-4')).toBeInTheDocument();
    expect(screen.getByText('1500ms')).toBeInTheDocument();
  });

  it('shows empty state when no traces', () => {
    render(<TracesTable data={[]} />);

    expect(screen.getByText('No traces found.')).toBeInTheDocument();
  });

  it('filters traces by search', async () => {
    const user = userEvent.setup();
    render(<TracesTable data={mockTraces} />);

    const searchInput = screen.getByPlaceholderText('Search traces...');
    await user.type(searchInput, 'embedding');

    expect(screen.getByText('embedding')).toBeInTheDocument();
    expect(screen.queryByText('chat-completion')).not.toBeInTheDocument();
  });
});
```

---

## Code Style Guidelines

### Go

- Use `gofmt` and `goimports` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use meaningful variable names (avoid single letters except in loops)
- Group imports: standard library, external, internal
- Write table-driven tests when applicable
- Use context for cancellation and timeouts
- Handle errors explicitly, avoid `_` for error returns
- Prefer composition over inheritance
- Use interfaces for dependencies (enables testing)

### TypeScript/React

- Use functional components with hooks
- Prefer named exports over default exports
- Use TypeScript strict mode
- Define types/interfaces for all props and API responses
- Use `const` by default, `let` only when reassignment needed
- Avoid `any` type - use `unknown` if type is truly unknown
- Colocate tests with components
- Use absolute imports with path aliases
- Keep components focused (single responsibility)
- Extract custom hooks for reusable logic

### SQL

- Use lowercase for keywords (consistency with ClickHouse)
- Use snake_case for table and column names
- Always include `created_at` and `updated_at` timestamps
- Use UUIDs for primary keys
- Add appropriate indexes for query patterns
- Use foreign key constraints in PostgreSQL
- Document complex queries with comments

---

## Performance Guidelines

### Backend

- Use connection pooling for databases
- Implement request batching for high-throughput endpoints
- Use async processing for non-critical operations
- Cache frequently accessed data (Redis)
- Profile endpoints with pprof
- Set appropriate timeouts for external calls
- Use prepared statements for repeated queries

### Frontend

- Implement virtualization for long lists
- Use React.memo for expensive components
- Lazy load routes and heavy components
- Optimize images and assets
- Use TanStack Query's caching effectively
- Debounce search inputs
- Avoid unnecessary re-renders (check with React DevTools)

### Database

- PostgreSQL: Use EXPLAIN ANALYZE for query optimization
- ClickHouse: Use EXPLAIN to understand query execution
- Add indexes based on actual query patterns
- Partition large tables appropriately
- Monitor query performance regularly
- Use materialized views for complex aggregations

---

## Security Checklist

- [ ] Validate all user inputs
- [ ] Use parameterized queries (prevent SQL injection)
- [ ] Implement rate limiting
- [ ] Use HTTPS everywhere
- [ ] Store passwords with bcrypt
- [ ] Implement proper CORS configuration
- [ ] Sanitize outputs to prevent XSS
- [ ] Use secure session management
- [ ] Implement audit logging
- [ ] Keep dependencies updated
- [ ] Use environment variables for secrets
- [ ] Implement proper access control (RBAC)
