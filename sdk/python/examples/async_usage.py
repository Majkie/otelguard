"""Async usage example."""

import asyncio
from otelguard import OTelGuard, Guard, validators

async def main():
    """Demonstrate async usage."""
    # Initialize client
    og = OTelGuard(
        api_key="demo-api-key",
        project="demo-project",
        base_url="http://localhost:8080"
    )

    # Async tracing
    async with og.atrace("async-chat-completion") as trace:
        user_input = "What is async/await in Python?"
        trace.set_input(user_input)

        # Simulate async LLM call
        await asyncio.sleep(0.5)
        response = "Async/await is a syntax for writing asynchronous code in Python."

        trace.set_output(response)
        trace.set_llm_metadata(
            model="gpt-4",
            total_tokens=30,
            cost=0.0009
        )

    # Async guardrails
    @Guard(
        input_validators=[validators.no_pii()],
        output_validators=[validators.toxicity_filter()],
        on_fail="retry",
        max_retries=2
    )
    async def async_chat(message: str) -> str:
        """Async chat function with guardrails."""
        await asyncio.sleep(0.2)
        return f"Response to: {message}"

    # Use async function with guardrails
    result = await async_chat("Hello, how are you?")
    print(f"Async result: {result}")

    # Async prompt management
    prompts = await og.prompts.alist(limit=10)
    print(f"Found {len(prompts.get('data', []))} prompts")

    # Async guardrails evaluation
    eval_result = await og.guardrails.aevaluate(
        input_text="User message",
        output_text="LLM response"
    )
    print(f"Guardrail result: {eval_result}")

    # Flush and close
    await og.aflush()
    await og.aclose()

    print("✓ Async examples completed!")

if __name__ == "__main__":
    asyncio.run(main())
