# OTelGuard Python SDK

Enterprise-grade LLM observability and guardrails platform SDK for Python.

## Features

- **Tracing**: Automatic and manual tracing for LLM applications
- **Guardrails**: Auto-remediating guardrails with built-in validators
- **Prompt Management**: Version control and templating for prompts
- **Async Support**: Full async/await support for all operations
- **Type Safe**: Fully typed with mypy support

## Installation

```bash
pip install otelguard-sdk
```

### Optional Dependencies

```bash
# For async support (recommended)
pip install otelguard-sdk[async]

# For advanced validators
pip install otelguard-sdk[validators]

# For development
pip install otelguard-sdk[dev]
```

## Quick Start

### Initialize Client

```python
from otelguard import OTelGuard

# Initialize with explicit credentials
og = OTelGuard(
    api_key="your-api-key",
    project="my-project",
    base_url="http://localhost:8080"  # Optional
)

# Or use environment variables
# OTELGUARD_API_KEY=your-api-key
# OTELGUARD_PROJECT=my-project
og = OTelGuard.from_env()
```

### Tracing

```python
# Context manager for tracing
with og.trace("chat-completion") as trace:
    # Your LLM call
    response = openai.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": "Hello!"}]
    )

    # Set trace metadata
    trace.set_input("Hello!")
    trace.set_output(response.choices[0].message.content)
    trace.set_llm_metadata(
        model="gpt-4",
        total_tokens=response.usage.total_tokens,
        prompt_tokens=response.usage.prompt_tokens,
        completion_tokens=response.usage.completion_tokens,
        cost=0.03  # Calculate based on pricing
    )

# Async tracing
async with og.atrace("chat-completion") as trace:
    response = await openai.chat.completions.create(...)
    trace.set_output(response.choices[0].message.content)
```

### Guardrails with Decorators

```python
from otelguard import Guard, validators

@Guard(
    input_validators=[
        validators.no_pii(),
        validators.prompt_injection_shield(),
        validators.length_limit(max_chars=1000),
    ],
    output_validators=[
        validators.toxicity_filter(threshold=0.8),
        validators.json_schema(schema),
    ],
    on_fail="retry",
    max_retries=3
)
async def chat_completion(prompt: str) -> str:
    response = await openai.chat.completions.create(
        model="gpt-4",
        messages=[{"role": "user", "content": prompt}]
    )
    return response.choices[0].message.content

# Use the protected function
result = await chat_completion("Hello, how are you?")
```

### Guardrails Strategies

The `on_fail` parameter determines what happens when a guardrail is violated:

- **`raise`** (default): Raise `GuardrailViolationError`
- **`retry`**: Automatically retry the operation (up to `max_retries`)
- **`block`**: Return a safe placeholder response
- **`sanitize`**: Clean the content and continue

```python
# Block on violation
@Guard(
    input_validators=[validators.no_pii()],
    on_fail="block"
)
def process_input(text: str) -> str:
    return llm.complete(text)

# Sanitize and continue
@Guard(
    input_validators=[validators.no_pii()],
    on_fail="sanitize"
)
def process_input(text: str) -> str:
    return llm.complete(text)  # PII will be redacted
```

### Built-in Validators

#### Input Validators

```python
from otelguard import validators

# PII detection
validators.no_pii()

# Prompt injection detection
validators.prompt_injection_shield()

# Secrets detection (API keys, tokens)
validators.no_secrets()

# Language check
validators.language_check(allowed_languages=["en", "es"])

# Length limits
validators.length_limit(max_chars=1000, max_tokens=500)

# Regex matching
validators.regex_matcher(r"\d{3}-\d{3}-\d{4}", block_on_match=True)

# Keyword blocking
validators.keyword_blocker(["banned", "forbidden"], case_sensitive=False)
```

#### Output Validators

```python
# Toxicity filtering
validators.toxicity_filter(threshold=0.8)

# JSON schema validation
schema = {
    "type": "object",
    "properties": {
        "name": {"type": "string"},
        "age": {"type": "number"}
    },
    "required": ["name"]
}
validators.json_schema(schema, strict=True)

# Format validation
validators.format_validator("email")
validators.format_validator("url")
validators.format_validator("phone")

# Relevance checking
validators.relevance_check(keywords=["python", "programming"], min_score=0.5)

# Completeness checking
validators.completeness_check(required_fields=["name", "email", "phone"])
```

### Prompt Management

```python
# Create a prompt
prompt = og.prompts.create(
    name="customer-support-greeting",
    description="Greeting message for customer support",
    tags=["support", "greeting"]
)

# Create a version with template
version = og.prompts.create_version(
    prompt_id=prompt["id"],
    content="Hello {{customer_name}}, how can I help you today?",
    config={"model": "gpt-4", "temperature": 0.7},
    labels=["production"]
)

# Compile template with variables
compiled = og.prompts.compile(
    prompt_id=prompt["id"],
    variables={"customer_name": "John"}
)
print(compiled)  # "Hello John, how can I help you today?"

# List all prompts
prompts = og.prompts.list(tags=["support"])

# Get prompt versions
versions = og.prompts.list_versions(prompt_id=prompt["id"])
```

### Remote Guardrails

```python
# Evaluate content against remote policies
result = og.guardrails.evaluate(
    input_text="User message here",
    output_text="LLM response here",
    policy_ids=["policy-123"],  # Optional: specific policies
    context={"user_id": "user-456"}  # Additional context
)

if result["triggered"]:
    print("Violations:", result["violations"])

    # Apply remediation
    remediated = og.guardrails.remediate(
        text=output_text,
        violations=result["violations"]
    )
    print("Cleaned text:", remediated["text"])

# List available policies
policies = og.guardrails.list_policies(enabled_only=True)
```

## Advanced Usage

### Nested Spans

```python
with og.trace("chat-request") as trace:
    trace.set_input(user_message)

    # Create child span for retrieval
    with trace.create_span("retrieval") as span:
        docs = retriever.search(user_message)
        span.set_output(docs)

    # Create child span for LLM call
    with trace.create_span("llm-completion") as span:
        response = llm.complete(user_message, context=docs)
        span.set_output(response)

    trace.set_output(response)
```

### Session and User Tracking

```python
# Track sessions
with og.trace("chat", session_id="session-123", user_id="user-456") as trace:
    trace.set_input(message)
    response = llm.complete(message)
    trace.set_output(response)
```

### Context Manager Usage

```python
# Automatic cleanup on exit
with OTelGuard(api_key="...", project="...") as og:
    with og.trace("operation") as trace:
        # Your code here
        pass
    # Traces are automatically flushed on exit

# Async context manager
async with OTelGuard(api_key="...", project="...") as og:
    async with og.atrace("operation") as trace:
        # Your async code here
        pass
```

### Manual Flush

```python
# Synchronous flush
og.flush()

# Async flush
await og.aflush()
```

## Configuration

### Environment Variables

```bash
OTELGUARD_API_KEY=your-api-key
OTELGUARD_PROJECT=my-project
OTELGUARD_BASE_URL=http://localhost:8080  # Optional
OTELGUARD_DEBUG=true  # Optional
```

### Config Object

```python
from otelguard import Config, OTelGuard

config = Config(
    api_key="your-api-key",
    project="my-project",
    base_url="http://localhost:8080",
    timeout=30,
    max_retries=3,
    enable_local_validation=True,
    batch_size=100,
    flush_interval=5.0,
    debug=False
)

og = OTelGuard(config=config)
```

## Error Handling

```python
from otelguard.exceptions import (
    OTelGuardError,
    AuthenticationError,
    ValidationError,
    GuardrailViolationError,
)

try:
    with og.trace("operation") as trace:
        # Your code
        pass
except AuthenticationError:
    print("Invalid API key")
except GuardrailViolationError as e:
    print("Guardrail violated:", e.violations)
except OTelGuardError as e:
    print("General error:", e)
```

## Examples

See the `examples/` directory for more comprehensive examples:

- `examples/basic_tracing.py` - Basic tracing usage
- `examples/guardrails.py` - Guardrails with validators
- `examples/async_usage.py` - Async tracing and guardrails
- `examples/prompt_management.py` - Prompt versioning and compilation
- `examples/openai_integration.py` - Integration with OpenAI SDK

## Development

```bash
# Clone the repository
git clone https://github.com/your-org/otelguard.git
cd otelguard/sdk/python

# Install development dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run tests with coverage
pytest --cov=otelguard --cov-report=html

# Format code
black otelguard/
isort otelguard/

# Type checking
mypy otelguard/
```

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: https://docs.otelguard.dev
- GitHub Issues: https://github.com/your-org/otelguard/issues
- Discord: https://discord.gg/otelguard
