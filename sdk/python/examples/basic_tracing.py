"""Basic tracing example."""

import time
from otelguard import OTelGuard

# Initialize client
og = OTelGuard(
    api_key="demo-api-key",
    project="demo-project",
    base_url="http://localhost:8080"
)

def main():
    """Demonstrate basic tracing."""
    # Simple trace
    with og.trace("chat-completion") as trace:
        # Simulate LLM call
        user_input = "What is the capital of France?"
        trace.set_input(user_input)

        # Simulate processing
        time.sleep(0.5)
        response = "The capital of France is Paris."

        # Set output and metadata
        trace.set_output(response)
        trace.set_llm_metadata(
            model="gpt-4",
            total_tokens=25,
            prompt_tokens=10,
            completion_tokens=15,
            cost=0.00075
        )
        trace.add_tag("geography")
        trace.add_tag("factual")

    # Trace with session and user tracking
    with og.trace(
        "chat-completion",
        session_id="session-123",
        user_id="user-456"
    ) as trace:
        user_input = "How about Germany?"
        trace.set_input(user_input)

        time.sleep(0.3)
        response = "The capital of Germany is Berlin."

        trace.set_output(response)
        trace.set_llm_metadata(
            model="gpt-4",
            total_tokens=20,
            cost=0.0006
        )

    # Trace with error handling
    try:
        with og.trace("failing-operation") as trace:
            trace.set_input("This will fail")
            raise ValueError("Simulated error")
    except ValueError:
        print("Error was traced and captured")

    # Flush traces to ensure they're sent
    og.flush()
    print("Traces sent successfully!")

if __name__ == "__main__":
    main()
