"""Output validators for guardrails."""

import json
import re
from typing import Any, Callable, Dict, List, Optional


def toxicity_filter(threshold: float = 0.8) -> Callable:
    """Validator to filter toxic content.

    Args:
        threshold: Toxicity threshold (0-1)

    Returns:
        Validator function

    Note:
        This is a basic implementation using keyword matching.
        For production use, integrate with a toxicity detection API
        like Perspective API or Detoxify.

    Example:
        >>> validator = toxicity_filter(threshold=0.8)
        >>> result = validator("This is inappropriate content")
    """
    # Basic toxic keywords (minimal set for demonstration)
    toxic_keywords = [
        "hate", "kill", "die", "stupid", "idiot", "dumb",
        # In production, use a comprehensive list or ML model
    ]

    def validator(text: str) -> Dict:
        text_lower = text.lower()
        found_keywords = [kw for kw in toxic_keywords if kw in text_lower]

        # Simple scoring: percentage of toxic keywords found
        score = min(len(found_keywords) * 0.3, 1.0)

        if score >= threshold:
            return {
                "violated": True,
                "violations": [{
                    "type": "toxicity",
                    "score": score,
                    "keywords": found_keywords,
                    "message": f"Toxic content detected (score: {score:.2f})"
                }],
                "action": "block",
            }

        return {"violated": False, "violations": []}

    return validator


def json_schema(schema: Dict[str, Any], strict: bool = True) -> Callable:
    """Validator to enforce JSON schema compliance.

    Args:
        schema: JSON schema to validate against
        strict: If True, fail on schema violations; if False, just warn

    Returns:
        Validator function

    Example:
        >>> schema = {"type": "object", "properties": {"name": {"type": "string"}}}
        >>> validator = json_schema(schema)
        >>> result = validator('{"name": "John"}')
        >>> assert result["violated"] == False
    """
    try:
        import jsonschema
        has_jsonschema = True
    except ImportError:
        has_jsonschema = False

    def validator(text: str) -> Dict:
        if not has_jsonschema:
            return {
                "violated": False,
                "violations": [],
                "message": "jsonschema package not installed, skipping validation"
            }

        try:
            # Parse JSON
            data = json.loads(text)

            # Validate schema
            jsonschema.validate(instance=data, schema=schema)

            return {"violated": False, "violations": []}

        except json.JSONDecodeError as e:
            if strict:
                return {
                    "violated": True,
                    "violations": [{
                        "type": "json_schema",
                        "error": "invalid_json",
                        "message": f"Invalid JSON: {str(e)}"
                    }],
                    "action": "block",
                }
            return {"violated": False, "violations": []}

        except jsonschema.ValidationError as e:
            if strict:
                return {
                    "violated": True,
                    "violations": [{
                        "type": "json_schema",
                        "error": "schema_violation",
                        "message": f"Schema violation: {str(e.message)}",
                        "path": list(e.path),
                    }],
                    "action": "retry",
                }
            return {"violated": False, "violations": []}

    return validator


def format_validator(format_type: str) -> Callable:
    """Validator to check output format (email, URL, etc.).

    Args:
        format_type: Type of format to validate ("email", "url", "phone", "date")

    Returns:
        Validator function

    Example:
        >>> validator = format_validator("email")
        >>> result = validator("user@example.com")
        >>> assert result["violated"] == False
    """
    patterns = {
        "email": r'^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$',
        "url": r'^https?://[^\s<>"]+|www\.[^\s<>"]+$',
        "phone": r'^\+?\d{1,3}?[-.\s]?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}$',
        "date": r'^\d{4}-\d{2}-\d{2}$',  # YYYY-MM-DD
        "time": r'^\d{2}:\d{2}(:\d{2})?$',  # HH:MM or HH:MM:SS
    }

    if format_type not in patterns:
        raise ValueError(f"Unknown format type: {format_type}")

    pattern = re.compile(patterns[format_type])

    def validator(text: str) -> Dict:
        text = text.strip()

        if not pattern.match(text):
            return {
                "violated": True,
                "violations": [{
                    "type": "format_validation",
                    "format": format_type,
                    "message": f"Invalid {format_type} format"
                }],
                "action": "retry",
            }

        return {"violated": False, "violations": []}

    return validator


def relevance_check(keywords: Optional[List[str]] = None, min_score: float = 0.5) -> Callable:
    """Validator to check output relevance to input/context.

    Args:
        keywords: Keywords that should appear in output
        min_score: Minimum relevance score (0-1)

    Returns:
        Validator function

    Note:
        This is a basic implementation. For production use, integrate
        with semantic similarity models.

    Example:
        >>> validator = relevance_check(keywords=["python", "programming"])
        >>> result = validator("This is about Python programming")
        >>> assert result["violated"] == False
    """
    def validator(text: str) -> Dict:
        if not keywords:
            return {"violated": False, "violations": []}

        text_lower = text.lower()
        matched_keywords = [kw for kw in keywords if kw.lower() in text_lower]

        # Simple relevance score
        score = len(matched_keywords) / len(keywords) if keywords else 0

        if score < min_score:
            return {
                "violated": True,
                "violations": [{
                    "type": "relevance",
                    "score": score,
                    "matched": matched_keywords,
                    "total": len(keywords),
                    "message": f"Output not relevant enough (score: {score:.2f})"
                }],
                "action": "retry",
            }

        return {"violated": False, "violations": []}

    return validator


def completeness_check(required_fields: List[str]) -> Callable:
    """Validator to check if output contains required fields/information.

    Args:
        required_fields: List of required field names or keywords

    Returns:
        Validator function

    Example:
        >>> validator = completeness_check(["name", "email", "phone"])
        >>> result = validator('{"name": "John", "email": "john@example.com", "phone": "555-1234"}')
        >>> assert result["violated"] == False
    """
    def validator(text: str) -> Dict:
        # Try to parse as JSON first
        try:
            data = json.loads(text)
            if isinstance(data, dict):
                missing_fields = [field for field in required_fields if field not in data]
            else:
                # For non-dict JSON, check text content
                text_lower = text.lower()
                missing_fields = [field for field in required_fields if field.lower() not in text_lower]
        except json.JSONDecodeError:
            # Plain text - check for keywords
            text_lower = text.lower()
            missing_fields = [field for field in required_fields if field.lower() not in text_lower]

        if missing_fields:
            return {
                "violated": True,
                "violations": [{
                    "type": "completeness",
                    "missing_fields": missing_fields,
                    "message": f"Output missing required fields: {', '.join(missing_fields)}"
                }],
                "action": "retry",
            }

        return {"violated": False, "violations": []}

    return validator
