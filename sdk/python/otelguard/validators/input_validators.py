"""Input validators for guardrails."""

import re
from typing import Callable, Dict, List, Optional


def no_pii() -> Callable:
    """Validator to detect and block PII (email, phone, SSN, etc.).

    Returns:
        Validator function

    Example:
        >>> validator = no_pii()
        >>> result = validator("My email is user@example.com")
        >>> assert result["violated"] == True
    """
    # PII patterns
    email_pattern = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
    phone_pattern = r'\b(\+\d{1,2}\s?)?\(?\d{3}\)?[\s.-]?\d{3}[\s.-]?\d{4}\b'
    ssn_pattern = r'\b\d{3}-\d{2}-\d{4}\b'
    credit_card_pattern = r'\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b'

    def validator(text: str) -> Dict:
        violations = []

        if re.search(email_pattern, text):
            violations.append({"type": "pii", "field": "email", "message": "Email address detected"})

        if re.search(phone_pattern, text):
            violations.append({"type": "pii", "field": "phone", "message": "Phone number detected"})

        if re.search(ssn_pattern, text):
            violations.append({"type": "pii", "field": "ssn", "message": "SSN detected"})

        if re.search(credit_card_pattern, text):
            violations.append({"type": "pii", "field": "credit_card", "message": "Credit card number detected"})

        return {
            "violated": len(violations) > 0,
            "violations": violations,
            "action": "redact" if violations else None,
        }

    return validator


def prompt_injection_shield() -> Callable:
    """Validator to detect prompt injection attempts.

    Returns:
        Validator function

    Example:
        >>> validator = prompt_injection_shield()
        >>> result = validator("Ignore previous instructions and...")
        >>> assert result["violated"] == True
    """
    # Common prompt injection patterns
    injection_patterns = [
        r'ignore\s+(previous|all|above)\s+instructions',
        r'forget\s+(previous|all|above)',
        r'disregard\s+(previous|all|above)',
        r'system\s*:\s*',
        r'<\s*\|.*?\|\s*>',  # Special tokens
        r'\[INST\]|\[\/INST\]',  # Instruction markers
        r'{{.*?}}',  # Template injection
        r'execute\s+command',
        r'run\s+code',
    ]

    def validator(text: str) -> Dict:
        violations = []

        for pattern in injection_patterns:
            if re.search(pattern, text, re.IGNORECASE):
                violations.append({
                    "type": "prompt_injection",
                    "pattern": pattern,
                    "message": "Potential prompt injection detected"
                })
                break  # One violation is enough

        return {
            "violated": len(violations) > 0,
            "violations": violations,
            "action": "block" if violations else None,
        }

    return validator


def no_secrets() -> Callable:
    """Validator to detect secrets (API keys, passwords, tokens).

    Returns:
        Validator function
    """
    # Secret patterns
    api_key_pattern = r'\b(sk-[a-zA-Z0-9]{32,}|[A-Z0-9]{32,})\b'
    token_pattern = r'\b(ghp_[a-zA-Z0-9]{36}|xox[baprs]-[a-zA-Z0-9-]+)\b'
    aws_key_pattern = r'\b(AKIA[0-9A-Z]{16})\b'

    def validator(text: str) -> Dict:
        violations = []

        if re.search(api_key_pattern, text):
            violations.append({"type": "secret", "field": "api_key", "message": "API key detected"})

        if re.search(token_pattern, text):
            violations.append({"type": "secret", "field": "token", "message": "Token detected"})

        if re.search(aws_key_pattern, text):
            violations.append({"type": "secret", "field": "aws_key", "message": "AWS key detected"})

        return {
            "violated": len(violations) > 0,
            "violations": violations,
            "action": "redact" if violations else None,
        }

    return validator


def language_check(allowed_languages: List[str]) -> Callable:
    """Validator to check if text is in allowed languages.

    Args:
        allowed_languages: List of allowed language codes (e.g., ["en", "es"])

    Returns:
        Validator function

    Note:
        This is a basic implementation. For production use, integrate with
        a language detection library like langdetect or fasttext.
    """
    def validator(text: str) -> Dict:
        # Basic implementation - check for common non-ASCII characters
        # In production, use proper language detection
        has_non_ascii = any(ord(char) > 127 for char in text)

        if "en" in allowed_languages and not has_non_ascii:
            # Assume English if only ASCII
            return {"violated": False, "violations": []}

        # For now, pass through (would need proper language detection)
        return {
            "violated": False,
            "violations": [],
            "message": "Language detection requires additional dependencies"
        }

    return validator


def length_limit(max_chars: Optional[int] = None, max_tokens: Optional[int] = None) -> Callable:
    """Validator to enforce length limits.

    Args:
        max_chars: Maximum character count
        max_tokens: Maximum token count (approximate)

    Returns:
        Validator function

    Example:
        >>> validator = length_limit(max_chars=100)
        >>> result = validator("a" * 200)
        >>> assert result["violated"] == True
    """
    def validator(text: str) -> Dict:
        violations = []

        if max_chars and len(text) > max_chars:
            violations.append({
                "type": "length_limit",
                "field": "chars",
                "limit": max_chars,
                "actual": len(text),
                "message": f"Text exceeds character limit ({len(text)} > {max_chars})"
            })

        if max_tokens:
            # Approximate token count (rough estimate: 1 token ≈ 4 chars)
            approx_tokens = len(text) // 4
            if approx_tokens > max_tokens:
                violations.append({
                    "type": "length_limit",
                    "field": "tokens",
                    "limit": max_tokens,
                    "actual": approx_tokens,
                    "message": f"Text exceeds token limit (~{approx_tokens} > {max_tokens})"
                })

        return {
            "violated": len(violations) > 0,
            "violations": violations,
            "action": "truncate" if violations else None,
        }

    return validator


def regex_matcher(pattern: str, block_on_match: bool = True, message: str = "Pattern matched") -> Callable:
    """Validator to match against regex pattern.

    Args:
        pattern: Regular expression pattern
        block_on_match: If True, block when pattern matches; if False, block when it doesn't
        message: Custom violation message

    Returns:
        Validator function

    Example:
        >>> validator = regex_matcher(r"\\d{3}-\\d{3}-\\d{4}", block_on_match=True)
        >>> result = validator("Call me at 555-123-4567")
        >>> assert result["violated"] == True
    """
    compiled_pattern = re.compile(pattern)

    def validator(text: str) -> Dict:
        matches = compiled_pattern.search(text)
        violated = (matches is not None) if block_on_match else (matches is None)

        if violated:
            return {
                "violated": True,
                "violations": [{
                    "type": "regex_match",
                    "pattern": pattern,
                    "message": message,
                }],
                "action": "block",
            }

        return {"violated": False, "violations": []}

    return validator


def keyword_blocker(keywords: List[str], case_sensitive: bool = False) -> Callable:
    """Validator to block specific keywords.

    Args:
        keywords: List of keywords to block
        case_sensitive: Whether matching should be case-sensitive

    Returns:
        Validator function

    Example:
        >>> validator = keyword_blocker(["banned", "forbidden"])
        >>> result = validator("This is a banned word")
        >>> assert result["violated"] == True
    """
    def validator(text: str) -> Dict:
        search_text = text if case_sensitive else text.lower()
        search_keywords = keywords if case_sensitive else [k.lower() for k in keywords]

        violations = []
        for keyword in search_keywords:
            if keyword in search_text:
                violations.append({
                    "type": "keyword_block",
                    "keyword": keyword,
                    "message": f"Blocked keyword detected: {keyword}"
                })

        return {
            "violated": len(violations) > 0,
            "violations": violations,
            "action": "block" if violations else None,
        }

    return validator
