"""Prompt management example."""

from otelguard import OTelGuard

def main():
    """Demonstrate prompt management."""
    og = OTelGuard(
        api_key="demo-api-key",
        project="demo-project",
        base_url="http://localhost:8080"
    )

    # Create a new prompt
    print("Creating prompt...")
    prompt = og.prompts.create(
        name="customer-greeting",
        description="Personalized greeting for customer support",
        tags=["support", "greeting", "customer"]
    )
    print(f"Created prompt: {prompt['id']}")

    # Create first version
    print("\nCreating version 1...")
    v1 = og.prompts.create_version(
        prompt_id=prompt["id"],
        content="Hello {{customer_name}}, how can I help you today?",
        config={
            "model": "gpt-4",
            "temperature": 0.7,
            "max_tokens": 150
        },
        labels=["draft"]
    )
    print(f"Created version {v1['version']}")

    # Compile version 1
    print("\nCompiling version 1...")
    compiled = og.prompts.compile(
        prompt_id=prompt["id"],
        version=v1["version"],
        variables={"customer_name": "Alice"}
    )
    print(f"Compiled: {compiled}")

    # Create improved version
    print("\nCreating version 2 (improved)...")
    v2 = og.prompts.create_version(
        prompt_id=prompt["id"],
        content="""Hello {{customer_name}}! 👋

I hope you're having a great day. I'm here to help with any questions or concerns you may have.

What can I assist you with today?""",
        config={
            "model": "gpt-4",
            "temperature": 0.7,
            "max_tokens": 150
        },
        labels=["production"]
    )
    print(f"Created version {v2['version']}")

    # Compile version 2
    print("\nCompiling version 2...")
    compiled_v2 = og.prompts.compile(
        prompt_id=prompt["id"],
        version=v2["version"],
        variables={"customer_name": "Bob"}
    )
    print(f"Compiled: {compiled_v2}")

    # List all versions
    print("\nListing all versions...")
    versions = og.prompts.list_versions(prompt_id=prompt["id"])
    for version in versions:
        print(f"  - Version {version['version']}: {version['labels']}")

    # List all prompts
    print("\nListing all prompts...")
    all_prompts = og.prompts.list(tags=["support"])
    print(f"Found {all_prompts['total']} support prompts")

    print("\n✓ Prompt management example completed!")

if __name__ == "__main__":
    main()
