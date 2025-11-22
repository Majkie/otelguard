# OTelGuard

**Open-Source LLM Observability Platform with Auto-Remediating Guardrails**

OTelGuard is an enterprise-grade LLM observability and engineering platform that provides comprehensive tracing, prompt management, evaluation, and intelligent guardrails for AI applications. Built on OpenTelemetry standards, it offers 100% language and framework agnostic instrumentation with unique features like auto-remediating guardrails and live multi-agent graph visualization.

---

## Key Features

### Core Observability

- **End-to-End Tracing** - Capture complete request lifecycles including LLM calls, embeddings, retrievals, tool executions, and agent reasoning steps
- **OpenTelemetry Native** - Built on OTEL standards for seamless integration with existing observability infrastructure
- **Multi-Model Support** - Works with OpenAI, Anthropic, Azure, AWS Bedrock, Google Vertex, Ollama, vLLM, and 100+ LLM providers
- **Session & User Tracking** - Track multi-turn conversations, user journeys, and usage patterns across sessions

### Prompt Management

- **Version Control** - Git-like versioning for prompts with branching, diffing, and rollback capabilities
- **Collaborative Editing** - Team-based prompt development with comments, reviews, and approval workflows
- **A/B Testing** - Deploy multiple prompt variants and measure performance differences
- **Prompt Playground** - Interactive testing environment with real-time evaluation
- **Template Engine** - Composable prompts with variables, conditionals, and includes

### Evaluation & Scoring

- **LLM-as-a-Judge** - Automated evaluation using configurable judge models
- **Human Annotation Queues** - Structured workflows for manual evaluation at scale
- **Custom Scoring** - Numeric, boolean, and categorical scores via API/SDK
- **User Feedback Integration** - Capture thumbs up/down, ratings, and free-form feedback
- **Score Analytics** - Statistical analysis with Pearson correlation, Cohen's Kappa, F1 scores
- **Dataset Experiments** - Benchmark prompts and models against curated test sets

### Analytics & Metrics

- **Real-Time Dashboards** - Monitor latency, cost, token usage, and error rates
- **Cost Attribution** - Per-user, per-session, per-model cost breakdowns
- **Quality Metrics** - Track evaluation scores, feedback trends, and regression detection
- **Custom Reports** - Build reports with flexible filtering and aggregation

---

## Differentiating Features

### 100% Language & Framework Agnostic

OTelGuard provides first-class support for any programming language through:

- **OpenTelemetry SDKs** - Native instrumentation for Python, JavaScript/TypeScript, Go, Java, Ruby, .NET, Rust, PHP, and more
- **HTTP API** - Direct REST API for languages without SDK support
- **Auto-Instrumentation** - Zero-code instrumentation for popular frameworks
- **Protocol Buffers** - Efficient binary serialization for high-throughput scenarios

### Auto-Remediating Guardrails

Intelligent guardrails that don't just detect issues but automatically fix them:

- **Prompt Injection Defense** - Detect and neutralize injection attempts with automatic sanitization
- **PII Redaction** - Automatically mask sensitive data in inputs and outputs
- **Content Moderation** - Filter toxic, harmful, or off-topic content with configurable responses
- **Output Validation** - Enforce schema compliance with automatic retry and correction
- **Rate Limiting** - Intelligent throttling with graceful degradation
- **Hallucination Detection** - Cross-reference outputs with source documents and trigger regeneration
- **Remediation Actions**:
  - `block` - Reject the request with a safe response
  - `sanitize` - Clean and continue processing
  - `retry` - Request regeneration with modified parameters
  - `fallback` - Route to alternative model or response
  - `alert` - Continue but notify operators

### Live Multi-Agent Graph Visualization

Real-time visualization of multi-agent systems:

- **Agent Topology** - Interactive graph showing agent relationships and communication patterns
- **Live State Tracking** - Watch agent states, tool calls, and message flows in real-time
- **Execution Timeline** - Waterfall view of parallel and sequential agent operations
- **Dependency Analysis** - Understand agent dependencies and bottlenecks
- **Replay Mode** - Step through historical executions for debugging
- **Performance Overlays** - Latency and cost heatmaps on agent graphs

### In-Code + No-Code Guardrails

Flexible guardrail configuration for both developers and non-technical users:

#### In-Code (SDK)

```python
from otelguard import Guard, validators

@Guard(
    input_validators=[
        validators.no_pii(),
        validators.prompt_injection_shield(),
    ],
    output_validators=[
        validators.json_schema(schema),
        validators.toxicity_filter(threshold=0.8),
    ],
    on_fail="retry",
    max_retries=3
)
async def chat_completion(prompt: str) -> str:
    return await llm.complete(prompt)
```

#### No-Code (UI Configuration)

```yaml
# Configured via web UI, stored as policy
guardrails:
  - name: "Customer Support Safety"
    triggers:
      - model: "gpt-4*"
        environment: "production"
    rules:
      - type: "pii_detection"
        action: "redact"
        fields: ["email", "phone", "ssn"]
      - type: "topic_filter"
        action: "block"
        topics: ["competitors", "pricing_internal"]
      - type: "response_length"
        action: "truncate"
        max_tokens: 500
```

---

## Architecture

```
                                   +------------------+
                                   |   Web Dashboard  |
                                   |  (React + ShadCN)|
                                   +--------+---------+
                                            |
                                            | HTTP/WebSocket
                                            |
+-------------------+              +--------+---------+              +------------------+
|                   |              |                  |              |                  |
|  Your LLM App     +------------->+   OTelGuard API  +------------->+   PostgreSQL     |
|  (Any Language)   |   OTLP/HTTP  |   (Go + Gin)     |              |   (Metadata)     |
|                   |              |                  |              |                  |
+-------------------+              +--------+---------+              +------------------+
                                            |
                                            |
                                   +--------+---------+
                                   |                  |
                                   |   ClickHouse     |
                                   |  (Trace Events)  |
                                   |                  |
                                   +------------------+
```

### Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Backend API** | Go + Gin | High-performance API server |
| **Frontend** | React + TypeScript + ShadCN | Modern, accessible UI |
| **Data Tables** | TanStack Table | Powerful data grid with sorting, filtering |
| **Data Fetching** | TanStack Query | Caching, synchronization, background updates |
| **Metadata Store** | PostgreSQL | Users, projects, prompts, configurations |
| **Event Store** | ClickHouse | High-volume trace events, spans, metrics |
| **Real-Time** | WebSockets | Live updates, agent visualization |
| **Protocol** | OpenTelemetry | Standard telemetry collection |

---

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 15+
- ClickHouse 24+
- Docker & Docker Compose (optional)

### Quick Start with Docker

```bash
# Clone the repository
git clone https://github.com/your-org/otelguard.git
cd otelguard

# Start all services
docker-compose up -d

# Access the dashboard
open http://localhost:3000
```

### Manual Installation

```bash
# Backend
cd backend
go mod download
go run cmd/server/main.go

# Frontend
cd frontend
npm install
npm run dev
```

### Instrument Your Application

#### Python

```bash
pip install otelguard-sdk
```

```python
from otelguard import OTelGuard

og = OTelGuard(
    api_key="your-api-key",
    project="my-project"
)

# Automatic instrumentation
og.instrument_openai()

# Or manual tracing
with og.trace("chat-request") as trace:
    response = openai.chat.completions.create(...)
    trace.set_output(response)
```

#### JavaScript/TypeScript

```bash
npm install @otelguard/sdk
```

```typescript
import { OTelGuard } from '@otelguard/sdk';

const og = new OTelGuard({
  apiKey: 'your-api-key',
  project: 'my-project'
});

// Automatic instrumentation
og.instrumentOpenAI();

// Or manual tracing
const trace = og.startTrace('chat-request');
const response = await openai.chat.completions.create(...);
trace.setOutput(response);
trace.end();
```

#### Go

```bash
go get github.com/your-org/otelguard-go
```

```go
import "github.com/your-org/otelguard-go"

og := otelguard.New(otelguard.Config{
    APIKey:  "your-api-key",
    Project: "my-project",
})

// Manual tracing
ctx, span := og.StartTrace(ctx, "chat-request")
defer span.End()

response, err := llm.Complete(ctx, prompt)
span.SetOutput(response)
```

---

## API Reference

### Tracing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/traces` | POST | Ingest trace data |
| `/v1/traces` | GET | List traces with filtering |
| `/v1/traces/{id}` | GET | Get trace details |
| `/v1/traces/{id}/spans` | GET | Get spans for a trace |

### Prompts

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/prompts` | GET | List prompts |
| `/v1/prompts` | POST | Create prompt |
| `/v1/prompts/{id}/versions` | GET | List versions |
| `/v1/prompts/{id}/compile` | POST | Compile with variables |

### Guardrails

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/guardrails` | GET | List guardrail policies |
| `/v1/guardrails` | POST | Create guardrail policy |
| `/v1/guardrails/evaluate` | POST | Evaluate content against policies |
| `/v1/guardrails/remediate` | POST | Apply remediation to content |

### Evaluations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/v1/scores` | POST | Submit evaluation scores |
| `/v1/datasets` | GET | List datasets |
| `/v1/experiments` | POST | Run dataset experiment |

---

## Configuration

### Environment Variables

```bash
# Server
OTELGUARD_PORT=8080
OTELGUARD_ENV=production

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=otelguard
POSTGRES_USER=otelguard
POSTGRES_PASSWORD=secret

# ClickHouse
CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_DB=otelguard

# Authentication
JWT_SECRET=your-jwt-secret
API_KEY_SALT=your-salt

# Optional: External Services
OPENAI_API_KEY=sk-...  # For LLM-as-Judge
```

---

## Roadmap

See [PLAN.md](./PLAN.md) for the detailed implementation roadmap.

### Phase 1: Foundation
- Core tracing infrastructure
- Basic UI dashboard
- PostgreSQL + ClickHouse setup

### Phase 2: Observability
- Full trace visualization
- Session management
- User tracking
- Cost analytics

### Phase 3: Prompt Engineering
- Prompt management
- Version control
- Playground
- A/B testing

### Phase 4: Evaluation
- Scoring system
- LLM-as-Judge
- Human annotations
- Dataset experiments

### Phase 5: Guardrails
- Policy engine
- Built-in validators
- Auto-remediation
- No-code configuration

### Phase 6: Advanced Features
- Multi-agent visualization
- Real-time collaboration
- Advanced analytics
- Enterprise features

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) for details.

---

## License

OTelGuard is open-source software licensed under the [MIT License](./LICENSE).

---

## Acknowledgments

OTelGuard is inspired by excellent projects in the LLM observability space:
- [Langfuse](https://langfuse.com/) - Open source LLM engineering platform
- [Guardrails AI](https://www.guardrailsai.com/) - LLM validation framework
- [OpenTelemetry](https://opentelemetry.io/) - Observability framework

---

## Support

- **Documentation**: [docs.otelguard.dev](https://docs.otelguard.dev)
- **Discord**: [Join our community](https://discord.gg/otelguard)
- **GitHub Issues**: [Report bugs or request features](https://github.com/your-org/otelguard/issues)
