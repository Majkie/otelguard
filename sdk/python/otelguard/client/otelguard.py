"""Main OTelGuard client class."""

import atexit
import logging
import threading
import time
from typing import Optional, Dict, Any

from otelguard.client.config import Config
from otelguard.client.http_client import HTTPClient
from otelguard.tracing.tracer import Tracer
from otelguard.guardrails.client import GuardrailsClient
from otelguard.prompts.client import PromptsClient


logger = logging.getLogger(__name__)


class OTelGuard:
    """Main OTelGuard client for tracing, guardrails, and prompt management.

    Example:
        >>> og = OTelGuard(api_key="your-key", project="my-project")
        >>> with og.trace("chat-request") as trace:
        ...     response = openai.chat.completions.create(...)
        ...     trace.set_output(response)
    """

    def __init__(
        self,
        api_key: Optional[str] = None,
        project: Optional[str] = None,
        base_url: Optional[str] = None,
        config: Optional[Config] = None,
        **kwargs
    ):
        """Initialize OTelGuard client.

        Args:
            api_key: API key for authentication
            project: Project identifier
            base_url: Base URL for OTelGuard API
            config: Pre-configured Config object (overrides other args)
            **kwargs: Additional configuration options

        Raises:
            ValueError: If required parameters are missing
        """
        if config:
            self.config = config
        elif api_key and project:
            config_args = {"api_key": api_key, "project": project}
            if base_url:
                config_args["base_url"] = base_url
            config_args.update(kwargs)
            self.config = Config(**config_args)
        else:
            # Try to load from environment
            config_args = {}
            if api_key:
                config_args["api_key"] = api_key
            if project:
                config_args["project"] = project
            if base_url:
                config_args["base_url"] = base_url
            config_args.update(kwargs)
            self.config = Config.from_env(**config_args)

        # Set up logging
        if self.config.debug:
            logging.basicConfig(level=logging.DEBUG)
            logger.setLevel(logging.DEBUG)

        # Initialize HTTP client
        self._http_client = HTTPClient(self.config)

        # Initialize sub-clients
        self.tracer = Tracer(self._http_client, self.config)
        self.guardrails = GuardrailsClient(self._http_client, self.config)
        self.prompts = PromptsClient(self._http_client, self.config)

        # Start background flusher
        self._stop_flusher = threading.Event()
        self._flusher_thread = threading.Thread(
            target=self._background_flusher,
            daemon=True,
            name="otelguard-flusher"
        )
        self._flusher_thread.start()

        # Register cleanup on exit
        atexit.register(self.flush)

        logger.info(f"OTelGuard client initialized for project: {self.config.project}")

    def _background_flusher(self):
        """Background thread to flush traces periodically."""
        while not self._stop_flusher.is_set():
            time.sleep(self.config.flush_interval)
            try:
                self.tracer.flush()
            except Exception as e:
                logger.error(f"Error flushing traces: {e}")

    def trace(self, name: str, **attributes):
        """Create a new trace context.

        Args:
            name: Name of the trace/operation
            **attributes: Additional attributes to attach to the trace

        Returns:
            Trace context manager

        Example:
            >>> with og.trace("chat-completion") as trace:
            ...     result = openai.chat.completions.create(...)
            ...     trace.set_output(result)
        """
        return self.tracer.trace(name, **attributes)

    async def atrace(self, name: str, **attributes):
        """Create a new async trace context.

        Args:
            name: Name of the trace/operation
            **attributes: Additional attributes to attach to the trace

        Returns:
            Async trace context manager

        Example:
            >>> async with og.atrace("chat-completion") as trace:
            ...     result = await openai.chat.completions.create(...)
            ...     trace.set_output(result)
        """
        return self.tracer.atrace(name, **attributes)

    def flush(self):
        """Flush pending traces to the server."""
        try:
            self.tracer.flush()
            logger.debug("Flushed pending traces")
        except Exception as e:
            logger.error(f"Error flushing traces: {e}")

    async def aflush(self):
        """Async flush pending traces to the server."""
        try:
            await self.tracer.aflush()
            logger.debug("Flushed pending traces (async)")
        except Exception as e:
            logger.error(f"Error flushing traces: {e}")

    def close(self):
        """Close the client and flush pending data."""
        logger.info("Closing OTelGuard client")
        self._stop_flusher.set()
        self.flush()
        self._http_client.close()
        if self._flusher_thread.is_alive():
            self._flusher_thread.join(timeout=5.0)

    async def aclose(self):
        """Async close the client and flush pending data."""
        logger.info("Closing OTelGuard client (async)")
        self._stop_flusher.set()
        await self.aflush()
        await self._http_client.aclose()

    def instrument_openai(self):
        """Auto-instrument OpenAI SDK (future implementation)."""
        raise NotImplementedError("OpenAI instrumentation will be implemented in a future release")

    def instrument_anthropic(self):
        """Auto-instrument Anthropic SDK (future implementation)."""
        raise NotImplementedError("Anthropic instrumentation will be implemented in a future release")

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - flush and close."""
        self.close()
        return False

    async def __aenter__(self):
        """Async context manager entry."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit - flush and close."""
        await self.aclose()
        return False
