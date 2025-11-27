"""Configuration for OTelGuard SDK."""

import os
from dataclasses import dataclass, field
from typing import Optional


@dataclass
class Config:
    """Configuration for OTelGuard client.

    Attributes:
        api_key: API key for authentication (required)
        project: Project identifier (required)
        base_url: Base URL for OTelGuard API (default: http://localhost:8080)
        timeout: Request timeout in seconds (default: 30)
        max_retries: Maximum number of retries for failed requests (default: 3)
        enable_local_validation: Enable local guardrail validation (default: True)
        batch_size: Batch size for trace ingestion (default: 100)
        flush_interval: Flush interval in seconds for batched traces (default: 5.0)
        debug: Enable debug logging (default: False)
    """

    api_key: str
    project: str
    base_url: str = "http://localhost:8080"
    timeout: int = 30
    max_retries: int = 3
    enable_local_validation: bool = True
    batch_size: int = 100
    flush_interval: float = 5.0
    debug: bool = False

    @classmethod
    def from_env(cls, **kwargs) -> "Config":
        """Create configuration from environment variables.

        Environment variables:
            OTELGUARD_API_KEY: API key
            OTELGUARD_PROJECT: Project identifier
            OTELGUARD_BASE_URL: Base URL (optional)
            OTELGUARD_DEBUG: Enable debug mode (optional)

        Args:
            **kwargs: Override specific configuration values

        Returns:
            Config instance

        Raises:
            ValueError: If required environment variables are missing
        """
        api_key = kwargs.get("api_key") or os.getenv("OTELGUARD_API_KEY")
        project = kwargs.get("project") or os.getenv("OTELGUARD_PROJECT")

        if not api_key:
            raise ValueError("api_key must be provided or set OTELGUARD_API_KEY environment variable")
        if not project:
            raise ValueError("project must be provided or set OTELGUARD_PROJECT environment variable")

        config = {
            "api_key": api_key,
            "project": project,
            "base_url": kwargs.get("base_url") or os.getenv("OTELGUARD_BASE_URL", "http://localhost:8080"),
            "debug": kwargs.get("debug") or os.getenv("OTELGUARD_DEBUG", "").lower() in ("true", "1", "yes"),
        }

        # Add any other kwargs
        for key, value in kwargs.items():
            if key not in config and hasattr(cls, key):
                config[key] = value

        return cls(**config)

    def __post_init__(self):
        """Validate configuration after initialization."""
        if not self.api_key:
            raise ValueError("api_key is required")
        if not self.project:
            raise ValueError("project is required")
        if self.timeout <= 0:
            raise ValueError("timeout must be positive")
        if self.max_retries < 0:
            raise ValueError("max_retries must be non-negative")
        if self.batch_size <= 0:
            raise ValueError("batch_size must be positive")
        if self.flush_interval <= 0:
            raise ValueError("flush_interval must be positive")
