"""Exceptions for OTelGuard SDK."""


class OTelGuardError(Exception):
    """Base exception for OTelGuard SDK."""
    pass


class AuthenticationError(OTelGuardError):
    """Raised when authentication fails."""
    pass


class ValidationError(OTelGuardError):
    """Raised when validation fails."""
    pass


class RateLimitError(OTelGuardError):
    """Raised when rate limit is exceeded."""
    pass


class ServerError(OTelGuardError):
    """Raised when server returns 5xx error."""
    pass


class GuardrailViolationError(OTelGuardError):
    """Raised when a guardrail is violated."""

    def __init__(self, message: str, violations: list = None, remediation: dict = None):
        super().__init__(message)
        self.violations = violations or []
        self.remediation = remediation or {}


class ConfigurationError(OTelGuardError):
    """Raised when configuration is invalid."""
    pass
