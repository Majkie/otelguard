"""Tracer implementation for OTelGuard SDK."""

import json
import logging
import threading
import time
import uuid
from contextlib import contextmanager, asynccontextmanager
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union

from otelguard.client.config import Config
from otelguard.client.http_client import HTTPClient


logger = logging.getLogger(__name__)


class Span:
    """Represents a span within a trace."""

    def __init__(
        self,
        span_id: str,
        trace_id: str,
        name: str,
        parent_span_id: Optional[str] = None,
        **attributes
    ):
        self.span_id = span_id
        self.trace_id = trace_id
        self.name = name
        self.parent_span_id = parent_span_id
        self.attributes = attributes
        self.start_time = datetime.now(timezone.utc)
        self.end_time: Optional[datetime] = None
        self.input: Optional[str] = None
        self.output: Optional[str] = None
        self.metadata: Dict[str, Any] = {}
        self.status = "success"
        self.error_message: Optional[str] = None

    def set_input(self, input_data: Any):
        """Set input data for the span."""
        if isinstance(input_data, str):
            self.input = input_data
        else:
            self.input = json.dumps(input_data, default=str)

    def set_output(self, output_data: Any):
        """Set output data for the span."""
        if isinstance(output_data, str):
            self.output = output_data
        else:
            self.output = json.dumps(output_data, default=str)

    def set_attribute(self, key: str, value: Any):
        """Set an attribute on the span."""
        self.attributes[key] = value

    def set_metadata(self, key: str, value: Any):
        """Set metadata on the span."""
        self.metadata[key] = value

    def set_error(self, error: Union[str, Exception]):
        """Mark span as error and set error message."""
        self.status = "error"
        if isinstance(error, Exception):
            self.error_message = f"{type(error).__name__}: {str(error)}"
        else:
            self.error_message = str(error)

    def end(self):
        """End the span."""
        if not self.end_time:
            self.end_time = datetime.now(timezone.utc)

    def to_dict(self) -> Dict[str, Any]:
        """Convert span to dictionary for API."""
        if not self.end_time:
            self.end()

        latency_ms = int((self.end_time - self.start_time).total_seconds() * 1000)

        return {
            "id": self.span_id,
            "trace_id": self.trace_id,
            "parent_span_id": self.parent_span_id,
            "name": self.name,
            "type": self.attributes.get("type", "custom"),
            "input": self.input or "",
            "output": self.output or "",
            "metadata": json.dumps({**self.metadata, **self.attributes}),
            "start_time": self.start_time.isoformat(),
            "end_time": self.end_time.isoformat(),
            "latency_ms": latency_ms,
            "status": self.status,
            "error_message": self.error_message,
        }


class Trace:
    """Represents a complete trace with nested spans."""

    def __init__(
        self,
        trace_id: str,
        project_id: str,
        name: str,
        session_id: Optional[str] = None,
        user_id: Optional[str] = None,
        **attributes
    ):
        self.trace_id = trace_id
        self.project_id = project_id
        self.name = name
        self.session_id = session_id
        self.user_id = user_id
        self.attributes = attributes
        self.start_time = datetime.now(timezone.utc)
        self.end_time: Optional[datetime] = None
        self.input: Optional[str] = None
        self.output: Optional[str] = None
        self.metadata: Dict[str, Any] = {}
        self.tags: List[str] = []
        self.spans: List[Span] = []
        self.status = "success"
        self.error_message: Optional[str] = None

        # LLM-specific attributes
        self.model: Optional[str] = None
        self.total_tokens: int = 0
        self.prompt_tokens: int = 0
        self.completion_tokens: int = 0
        self.cost: float = 0.0

    def set_input(self, input_data: Any):
        """Set input data for the trace."""
        if isinstance(input_data, str):
            self.input = input_data
        else:
            self.input = json.dumps(input_data, default=str)

    def set_output(self, output_data: Any):
        """Set output data for the trace."""
        if isinstance(output_data, str):
            self.output = output_data
        else:
            self.output = json.dumps(output_data, default=str)

    def set_attribute(self, key: str, value: Any):
        """Set an attribute on the trace."""
        self.attributes[key] = value

    def set_metadata(self, key: str, value: Any):
        """Set metadata on the trace."""
        self.metadata[key] = value

    def add_tag(self, tag: str):
        """Add a tag to the trace."""
        if tag not in self.tags:
            self.tags.append(tag)

    def set_llm_metadata(
        self,
        model: Optional[str] = None,
        total_tokens: Optional[int] = None,
        prompt_tokens: Optional[int] = None,
        completion_tokens: Optional[int] = None,
        cost: Optional[float] = None,
    ):
        """Set LLM-specific metadata."""
        if model:
            self.model = model
        if total_tokens is not None:
            self.total_tokens = total_tokens
        if prompt_tokens is not None:
            self.prompt_tokens = prompt_tokens
        if completion_tokens is not None:
            self.completion_tokens = completion_tokens
        if cost is not None:
            self.cost = cost

    def set_error(self, error: Union[str, Exception]):
        """Mark trace as error and set error message."""
        self.status = "error"
        if isinstance(error, Exception):
            self.error_message = f"{type(error).__name__}: {str(error)}"
        else:
            self.error_message = str(error)

    def create_span(self, name: str, parent_span_id: Optional[str] = None, **attributes) -> Span:
        """Create a new span within this trace."""
        span = Span(
            span_id=str(uuid.uuid4()),
            trace_id=self.trace_id,
            name=name,
            parent_span_id=parent_span_id,
            **attributes
        )
        self.spans.append(span)
        return span

    def end(self):
        """End the trace."""
        if not self.end_time:
            self.end_time = datetime.now(timezone.utc)
            # End any open spans
            for span in self.spans:
                if not span.end_time:
                    span.end()

    def to_dict(self) -> Dict[str, Any]:
        """Convert trace to dictionary for API."""
        if not self.end_time:
            self.end()

        latency_ms = int((self.end_time - self.start_time).total_seconds() * 1000)

        return {
            "id": self.trace_id,
            "project_id": self.project_id,
            "session_id": self.session_id,
            "user_id": self.user_id,
            "name": self.name,
            "input": self.input or "",
            "output": self.output or "",
            "metadata": json.dumps({**self.metadata, **self.attributes}),
            "start_time": self.start_time.isoformat(),
            "end_time": self.end_time.isoformat(),
            "latency_ms": latency_ms,
            "total_tokens": self.total_tokens,
            "prompt_tokens": self.prompt_tokens,
            "completion_tokens": self.completion_tokens,
            "cost": self.cost,
            "model": self.model or "",
            "tags": self.tags,
            "status": self.status,
            "error_message": self.error_message,
        }


class Tracer:
    """Tracer for creating and managing traces."""

    def __init__(self, http_client: HTTPClient, config: Config):
        self.http_client = http_client
        self.config = config
        self._trace_buffer: List[Trace] = []
        self._buffer_lock = threading.Lock()

    @contextmanager
    def trace(self, name: str, session_id: Optional[str] = None, user_id: Optional[str] = None, **attributes):
        """Create a trace context manager.

        Args:
            name: Name of the trace
            session_id: Optional session identifier
            user_id: Optional user identifier
            **attributes: Additional attributes

        Yields:
            Trace object

        Example:
            >>> with tracer.trace("chat-completion") as trace:
            ...     trace.set_input("Hello")
            ...     result = llm.complete("Hello")
            ...     trace.set_output(result)
        """
        trace_obj = Trace(
            trace_id=str(uuid.uuid4()),
            project_id=self.config.project,
            name=name,
            session_id=session_id,
            user_id=user_id,
            **attributes
        )

        try:
            yield trace_obj
        except Exception as e:
            trace_obj.set_error(e)
            raise
        finally:
            trace_obj.end()
            self._buffer_trace(trace_obj)

    @asynccontextmanager
    async def atrace(self, name: str, session_id: Optional[str] = None, user_id: Optional[str] = None, **attributes):
        """Create an async trace context manager.

        Args:
            name: Name of the trace
            session_id: Optional session identifier
            user_id: Optional user identifier
            **attributes: Additional attributes

        Yields:
            Trace object

        Example:
            >>> async with tracer.atrace("chat-completion") as trace:
            ...     trace.set_input("Hello")
            ...     result = await llm.complete("Hello")
            ...     trace.set_output(result)
        """
        trace_obj = Trace(
            trace_id=str(uuid.uuid4()),
            project_id=self.config.project,
            name=name,
            session_id=session_id,
            user_id=user_id,
            **attributes
        )

        try:
            yield trace_obj
        except Exception as e:
            trace_obj.set_error(e)
            raise
        finally:
            trace_obj.end()
            self._buffer_trace(trace_obj)

    def _buffer_trace(self, trace: Trace):
        """Add trace to buffer."""
        with self._buffer_lock:
            self._trace_buffer.append(trace)
            if len(self._trace_buffer) >= self.config.batch_size:
                self._flush_buffer()

    def _flush_buffer(self):
        """Flush buffered traces to API (called with lock held)."""
        if not self._trace_buffer:
            return

        traces_to_send = self._trace_buffer[:]
        self._trace_buffer.clear()

        # Send without holding lock
        try:
            self._send_traces(traces_to_send)
        except Exception as e:
            logger.error(f"Failed to send traces: {e}")
            # Re-add to buffer for retry (basic retry logic)
            with self._buffer_lock:
                self._trace_buffer.extend(traces_to_send)

    def _send_traces(self, traces: List[Trace]):
        """Send traces to API."""
        if not traces:
            return

        # Send trace by trace for now (can be optimized to batch endpoint)
        for trace in traces:
            try:
                self.http_client.post("/v1/traces", data=trace.to_dict())
                logger.debug(f"Sent trace: {trace.trace_id}")
            except Exception as e:
                logger.error(f"Failed to send trace {trace.trace_id}: {e}")

    async def _asend_traces(self, traces: List[Trace]):
        """Async send traces to API."""
        if not traces:
            return

        for trace in traces:
            try:
                await self.http_client.apost("/v1/traces", data=trace.to_dict())
                logger.debug(f"Sent trace (async): {trace.trace_id}")
            except Exception as e:
                logger.error(f"Failed to send trace {trace.trace_id} (async): {e}")

    def flush(self):
        """Flush all buffered traces."""
        with self._buffer_lock:
            self._flush_buffer()

    async def aflush(self):
        """Async flush all buffered traces."""
        with self._buffer_lock:
            if not self._trace_buffer:
                return
            traces_to_send = self._trace_buffer[:]
            self._trace_buffer.clear()

        try:
            await self._asend_traces(traces_to_send)
        except Exception as e:
            logger.error(f"Failed to flush traces (async): {e}")
