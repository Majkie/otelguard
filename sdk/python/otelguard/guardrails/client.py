"""Guardrails client for remote validation."""

import logging
from typing import Any, Dict, List, Optional

from otelguard.client.config import Config
from otelguard.client.http_client import HTTPClient
from otelguard.exceptions import GuardrailViolationError


logger = logging.getLogger(__name__)


class GuardrailsClient:
    """Client for interacting with guardrails API."""

    def __init__(self, http_client: HTTPClient, config: Config):
        self.http_client = http_client
        self.config = config

    def evaluate(
        self,
        input_text: Optional[str] = None,
        output_text: Optional[str] = None,
        policy_ids: Optional[List[str]] = None,
        context: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Evaluate content against guardrail policies.

        Args:
            input_text: Input text to validate
            output_text: Output text to validate
            policy_ids: Specific policy IDs to evaluate (optional)
            context: Additional context for evaluation

        Returns:
            Evaluation result with violations and remediation

        Raises:
            GuardrailViolationError: If guardrails are violated and blocking is enabled
        """
        data = {
            "input_text": input_text,
            "output_text": output_text,
            "context": context or {},
        }
        if policy_ids:
            data["policy_ids"] = policy_ids

        try:
            result = self.http_client.post("/v1/guardrails/evaluate", data=data)
            return result
        except Exception as e:
            logger.error(f"Guardrail evaluation failed: {e}")
            # In case of error, allow by default unless strict mode
            return {
                "triggered": False,
                "violations": [],
                "remediation": {},
                "error": str(e),
            }

    async def aevaluate(
        self,
        input_text: Optional[str] = None,
        output_text: Optional[str] = None,
        policy_ids: Optional[List[str]] = None,
        context: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Async evaluate content against guardrail policies.

        Args:
            input_text: Input text to validate
            output_text: Output text to validate
            policy_ids: Specific policy IDs to evaluate (optional)
            context: Additional context for evaluation

        Returns:
            Evaluation result with violations and remediation
        """
        data = {
            "input_text": input_text,
            "output_text": output_text,
            "context": context or {},
        }
        if policy_ids:
            data["policy_ids"] = policy_ids

        try:
            result = await self.http_client.apost("/v1/guardrails/evaluate", data=data)
            return result
        except Exception as e:
            logger.error(f"Guardrail evaluation failed (async): {e}")
            return {
                "triggered": False,
                "violations": [],
                "remediation": {},
                "error": str(e),
            }

    def remediate(
        self,
        text: str,
        violations: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """Apply remediation to text based on violations.

        Args:
            text: Text to remediate
            violations: List of violations to remediate

        Returns:
            Remediation result with cleaned text
        """
        data = {
            "text": text,
            "violations": violations,
        }

        try:
            result = self.http_client.post("/v1/guardrails/remediate", data=data)
            return result
        except Exception as e:
            logger.error(f"Remediation failed: {e}")
            return {
                "text": text,
                "applied": False,
                "error": str(e),
            }

    async def aremediate(
        self,
        text: str,
        violations: List[Dict[str, Any]],
    ) -> Dict[str, Any]:
        """Async apply remediation to text based on violations."""
        data = {
            "text": text,
            "violations": violations,
        }

        try:
            result = await self.http_client.apost("/v1/guardrails/remediate", data=data)
            return result
        except Exception as e:
            logger.error(f"Remediation failed (async): {e}")
            return {
                "text": text,
                "applied": False,
                "error": str(e),
            }

    def list_policies(self, enabled_only: bool = True) -> List[Dict[str, Any]]:
        """List available guardrail policies.

        Args:
            enabled_only: Only return enabled policies

        Returns:
            List of policies
        """
        params = {"enabled": enabled_only} if enabled_only else {}
        try:
            result = self.http_client.get("/v1/guardrails", params=params)
            return result.get("data", [])
        except Exception as e:
            logger.error(f"Failed to list policies: {e}")
            return []

    async def alist_policies(self, enabled_only: bool = True) -> List[Dict[str, Any]]:
        """Async list available guardrail policies."""
        params = {"enabled": enabled_only} if enabled_only else {}
        try:
            result = await self.http_client.aget("/v1/guardrails", params=params)
            return result.get("data", [])
        except Exception as e:
            logger.error(f"Failed to list policies (async): {e}")
            return []
