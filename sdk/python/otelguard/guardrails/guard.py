"""Guard decorator for applying guardrails to functions."""

import asyncio
import functools
import inspect
import logging
from typing import Any, Callable, List, Optional, Union

from otelguard.exceptions import GuardrailViolationError


logger = logging.getLogger(__name__)


class Guard:
    """Decorator for applying guardrails to functions.

    Example:
        >>> @Guard(
        ...     input_validators=[validators.no_pii(), validators.prompt_injection_shield()],
        ...     output_validators=[validators.json_schema(schema), validators.toxicity_filter()],
        ...     on_fail="retry",
        ...     max_retries=3
        ... )
        ... async def chat_completion(prompt: str) -> str:
        ...     return await llm.complete(prompt)
    """

    def __init__(
        self,
        input_validators: Optional[List[Callable]] = None,
        output_validators: Optional[List[Callable]] = None,
        on_fail: str = "raise",  # "raise", "retry", "block", "sanitize"
        max_retries: int = 3,
        policy_ids: Optional[List[str]] = None,
        enable_remote: bool = True,
        enable_local: bool = True,
        context: Optional[dict] = None,
    ):
        """Initialize Guard decorator.

        Args:
            input_validators: List of validators to run on inputs
            output_validators: List of validators to run on outputs
            on_fail: Action on validation failure ("raise", "retry", "block", "sanitize")
            max_retries: Maximum retry attempts
            policy_ids: Specific policy IDs to evaluate
            enable_remote: Enable remote validation via API
            enable_local: Enable local validation
            context: Additional context for validation
        """
        self.input_validators = input_validators or []
        self.output_validators = output_validators or []
        self.on_fail = on_fail
        self.max_retries = max_retries
        self.policy_ids = policy_ids
        self.enable_remote = enable_remote
        self.enable_local = enable_local
        self.context = context or {}

    def __call__(self, func: Callable) -> Callable:
        """Wrap function with guardrails."""
        if asyncio.iscoroutinefunction(func):
            return self._wrap_async(func)
        else:
            return self._wrap_sync(func)

    def _wrap_sync(self, func: Callable) -> Callable:
        """Wrap synchronous function."""
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            # Extract input from args/kwargs
            input_text = self._extract_input(func, args, kwargs)

            # Validate input
            if self.input_validators or self.enable_remote:
                input_result = self._validate_input(input_text)
                if input_result.get("triggered"):
                    input_text = self._handle_violation(
                        input_text,
                        input_result,
                        "input"
                    )
                    # Update args/kwargs with sanitized input if applicable
                    if input_text != self._extract_input(func, args, kwargs):
                        args, kwargs = self._update_input(func, args, kwargs, input_text)

            # Execute function with retry logic
            attempt = 0
            last_error = None

            while attempt <= self.max_retries:
                try:
                    result = func(*args, **kwargs)

                    # Validate output
                    if self.output_validators or self.enable_remote:
                        output_text = self._extract_output(result)
                        output_result = self._validate_output(output_text)

                        if output_result.get("triggered"):
                            if self.on_fail == "retry" and attempt < self.max_retries:
                                attempt += 1
                                logger.warning(f"Output validation failed, retrying (attempt {attempt}/{self.max_retries})")
                                continue
                            else:
                                result = self._handle_violation(
                                    output_text,
                                    output_result,
                                    "output"
                                )

                    return result

                except Exception as e:
                    last_error = e
                    if self.on_fail != "retry" or attempt >= self.max_retries:
                        raise
                    attempt += 1
                    logger.warning(f"Function execution failed, retrying (attempt {attempt}/{self.max_retries}): {e}")

            if last_error:
                raise last_error

        return wrapper

    def _wrap_async(self, func: Callable) -> Callable:
        """Wrap asynchronous function."""
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            # Extract input from args/kwargs
            input_text = self._extract_input(func, args, kwargs)

            # Validate input
            if self.input_validators or self.enable_remote:
                input_result = await self._avalidate_input(input_text)
                if input_result.get("triggered"):
                    input_text = await self._ahandle_violation(
                        input_text,
                        input_result,
                        "input"
                    )
                    # Update args/kwargs with sanitized input if applicable
                    if input_text != self._extract_input(func, args, kwargs):
                        args, kwargs = self._update_input(func, args, kwargs, input_text)

            # Execute function with retry logic
            attempt = 0
            last_error = None

            while attempt <= self.max_retries:
                try:
                    result = await func(*args, **kwargs)

                    # Validate output
                    if self.output_validators or self.enable_remote:
                        output_text = self._extract_output(result)
                        output_result = await self._avalidate_output(output_text)

                        if output_result.get("triggered"):
                            if self.on_fail == "retry" and attempt < self.max_retries:
                                attempt += 1
                                logger.warning(f"Output validation failed, retrying (attempt {attempt}/{self.max_retries})")
                                continue
                            else:
                                result = await self._ahandle_violation(
                                    output_text,
                                    output_result,
                                    "output"
                                )

                    return result

                except Exception as e:
                    last_error = e
                    if self.on_fail != "retry" or attempt >= self.max_retries:
                        raise
                    attempt += 1
                    logger.warning(f"Function execution failed, retrying (attempt {attempt}/{self.max_retries}): {e}")

            if last_error:
                raise last_error

        return wrapper

    def _extract_input(self, func: Callable, args: tuple, kwargs: dict) -> str:
        """Extract input text from function arguments."""
        # Get function signature
        sig = inspect.signature(func)
        params = list(sig.parameters.keys())

        # Try to find a parameter that looks like input
        for i, arg in enumerate(args):
            if isinstance(arg, str) and i < len(params):
                return arg

        for key, value in kwargs.items():
            if isinstance(value, str) and key in ["prompt", "input", "text", "message", "query"]:
                return value

        # Return first string argument or empty string
        for arg in args:
            if isinstance(arg, str):
                return arg

        return ""

    def _update_input(self, func: Callable, args: tuple, kwargs: dict, new_input: str) -> tuple:
        """Update function arguments with new input."""
        # Try to update the first string argument
        args_list = list(args)
        for i, arg in enumerate(args_list):
            if isinstance(arg, str):
                args_list[i] = new_input
                return tuple(args_list), kwargs

        # Try to update kwargs
        for key in ["prompt", "input", "text", "message", "query"]:
            if key in kwargs and isinstance(kwargs[key], str):
                kwargs[key] = new_input
                return args, kwargs

        return args, kwargs

    def _extract_output(self, result: Any) -> str:
        """Extract output text from function result."""
        if isinstance(result, str):
            return result
        elif isinstance(result, dict):
            # Try common keys
            for key in ["text", "content", "message", "output", "response"]:
                if key in result and isinstance(result[key], str):
                    return result[key]
        return str(result)

    def _validate_input(self, input_text: str) -> dict:
        """Validate input with local validators."""
        violations = []

        # Run local validators
        if self.enable_local:
            for validator in self.input_validators:
                try:
                    result = validator(input_text)
                    if result and result.get("violated"):
                        violations.append(result)
                except Exception as e:
                    logger.error(f"Input validator failed: {e}")

        # TODO: Run remote validation via API
        # This would require access to OTelGuard client instance

        return {
            "triggered": len(violations) > 0,
            "violations": violations,
        }

    async def _avalidate_input(self, input_text: str) -> dict:
        """Async validate input with local validators."""
        violations = []

        # Run local validators
        if self.enable_local:
            for validator in self.input_validators:
                try:
                    result = validator(input_text)
                    if asyncio.iscoroutine(result):
                        result = await result
                    if result and result.get("violated"):
                        violations.append(result)
                except Exception as e:
                    logger.error(f"Input validator failed: {e}")

        return {
            "triggered": len(violations) > 0,
            "violations": violations,
        }

    def _validate_output(self, output_text: str) -> dict:
        """Validate output with local validators."""
        violations = []

        # Run local validators
        if self.enable_local:
            for validator in self.output_validators:
                try:
                    result = validator(output_text)
                    if result and result.get("violated"):
                        violations.append(result)
                except Exception as e:
                    logger.error(f"Output validator failed: {e}")

        return {
            "triggered": len(violations) > 0,
            "violations": violations,
        }

    async def _avalidate_output(self, output_text: str) -> dict:
        """Async validate output with local validators."""
        violations = []

        # Run local validators
        if self.enable_local:
            for validator in self.output_validators:
                try:
                    result = validator(output_text)
                    if asyncio.iscoroutine(result):
                        result = await result
                    if result and result.get("violated"):
                        violations.append(result)
                except Exception as e:
                    logger.error(f"Output validator failed: {e}")

        return {
            "triggered": len(violations) > 0,
            "violations": violations,
        }

    def _handle_violation(self, text: str, result: dict, phase: str) -> Any:
        """Handle validation violation based on on_fail strategy."""
        violations = result.get("violations", [])

        if self.on_fail == "raise":
            raise GuardrailViolationError(
                f"Guardrail violation in {phase}",
                violations=violations
            )
        elif self.on_fail == "block":
            return "[Content blocked by guardrails]"
        elif self.on_fail == "sanitize":
            # Apply basic sanitization
            sanitized = text
            for violation in violations:
                if violation.get("action") == "redact":
                    # Simple redaction - replace with placeholder
                    sanitized = "[REDACTED]"
            return sanitized
        else:
            # For retry, return original and let retry logic handle it
            return text

    async def _ahandle_violation(self, text: str, result: dict, phase: str) -> Any:
        """Async handle validation violation."""
        return self._handle_violation(text, result, phase)
