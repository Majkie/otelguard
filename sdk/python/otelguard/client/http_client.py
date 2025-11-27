"""HTTP client for OTelGuard API."""

import time
import logging
from typing import Any, Dict, Optional, Union
from urllib.parse import urljoin

try:
    import httpx
    HTTPX_AVAILABLE = True
except ImportError:
    HTTPX_AVAILABLE = False
    import requests

from otelguard.client.config import Config
from otelguard.exceptions import (
    OTelGuardError,
    AuthenticationError,
    ValidationError,
    RateLimitError,
    ServerError,
)


logger = logging.getLogger(__name__)


class HTTPClient:
    """HTTP client for making requests to OTelGuard API."""

    def __init__(self, config: Config):
        """Initialize HTTP client.

        Args:
            config: OTelGuard configuration
        """
        self.config = config
        self._setup_client()

    def _setup_client(self):
        """Set up HTTP client with proper headers and configuration."""
        headers = {
            "Authorization": f"Bearer {self.config.api_key}",
            "Content-Type": "application/json",
            "User-Agent": "otelguard-python/0.1.0",
            "X-Project-ID": self.config.project,
        }

        if HTTPX_AVAILABLE:
            self.client = httpx.Client(
                base_url=self.config.base_url,
                headers=headers,
                timeout=self.config.timeout,
            )
            self.async_client = httpx.AsyncClient(
                base_url=self.config.base_url,
                headers=headers,
                timeout=self.config.timeout,
            )
        else:
            self.session = requests.Session()
            self.session.headers.update(headers)
            self.async_client = None

    def _build_url(self, endpoint: str) -> str:
        """Build full URL for endpoint.

        Args:
            endpoint: API endpoint path

        Returns:
            Full URL
        """
        if HTTPX_AVAILABLE:
            return endpoint  # httpx client handles base_url
        return urljoin(self.config.base_url, endpoint)

    def _handle_response(self, response) -> Dict[str, Any]:
        """Handle HTTP response and raise appropriate errors.

        Args:
            response: HTTP response object

        Returns:
            Response JSON data

        Raises:
            OTelGuardError: For various error conditions
        """
        try:
            if response.status_code == 200 or response.status_code == 201:
                return response.json()
            elif response.status_code == 400:
                error_data = response.json()
                raise ValidationError(error_data.get("message", "Validation error"))
            elif response.status_code == 401:
                raise AuthenticationError("Invalid API key or unauthorized access")
            elif response.status_code == 429:
                raise RateLimitError("Rate limit exceeded")
            elif response.status_code >= 500:
                raise ServerError(f"Server error: {response.status_code}")
            else:
                raise OTelGuardError(f"Unexpected status code: {response.status_code}")
        except Exception as e:
            if isinstance(e, OTelGuardError):
                raise
            logger.error(f"Error handling response: {e}")
            raise OTelGuardError(f"Failed to parse response: {e}")

    def request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make HTTP request with retries.

        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            endpoint: API endpoint path
            data: Request body data
            params: Query parameters

        Returns:
            Response data

        Raises:
            OTelGuardError: On request failure
        """
        url = self._build_url(endpoint)

        for attempt in range(self.config.max_retries + 1):
            try:
                if HTTPX_AVAILABLE:
                    response = self.client.request(
                        method=method,
                        url=url,
                        json=data,
                        params=params,
                    )
                else:
                    response = self.session.request(
                        method=method,
                        url=url,
                        json=data,
                        params=params,
                        timeout=self.config.timeout,
                    )

                return self._handle_response(response)

            except (AuthenticationError, ValidationError) as e:
                # Don't retry auth or validation errors
                raise
            except Exception as e:
                if attempt == self.config.max_retries:
                    logger.error(f"Request failed after {self.config.max_retries} retries: {e}")
                    raise OTelGuardError(f"Request failed: {e}")

                # Exponential backoff
                wait_time = 2 ** attempt
                logger.warning(f"Request failed (attempt {attempt + 1}/{self.config.max_retries + 1}), retrying in {wait_time}s: {e}")
                time.sleep(wait_time)

    async def arequest(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Make async HTTP request with retries.

        Args:
            method: HTTP method (GET, POST, PUT, DELETE)
            endpoint: API endpoint path
            data: Request body data
            params: Query parameters

        Returns:
            Response data

        Raises:
            OTelGuardError: On request failure
            RuntimeError: If httpx is not installed
        """
        if not HTTPX_AVAILABLE:
            raise RuntimeError("httpx is required for async operations. Install with: pip install httpx")

        url = self._build_url(endpoint)

        for attempt in range(self.config.max_retries + 1):
            try:
                response = await self.async_client.request(
                    method=method,
                    url=url,
                    json=data,
                    params=params,
                )

                return self._handle_response(response)

            except (AuthenticationError, ValidationError) as e:
                # Don't retry auth or validation errors
                raise
            except Exception as e:
                if attempt == self.config.max_retries:
                    logger.error(f"Async request failed after {self.config.max_retries} retries: {e}")
                    raise OTelGuardError(f"Async request failed: {e}")

                # Exponential backoff
                import asyncio
                wait_time = 2 ** attempt
                logger.warning(f"Async request failed (attempt {attempt + 1}/{self.config.max_retries + 1}), retrying in {wait_time}s: {e}")
                await asyncio.sleep(wait_time)

    def get(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make GET request."""
        return self.request("GET", endpoint, params=params)

    def post(self, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make POST request."""
        return self.request("POST", endpoint, data=data)

    def put(self, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make PUT request."""
        return self.request("PUT", endpoint, data=data)

    def delete(self, endpoint: str) -> Dict[str, Any]:
        """Make DELETE request."""
        return self.request("DELETE", endpoint)

    async def aget(self, endpoint: str, params: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make async GET request."""
        return await self.arequest("GET", endpoint, params=params)

    async def apost(self, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make async POST request."""
        return await self.arequest("POST", endpoint, data=data)

    async def aput(self, endpoint: str, data: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        """Make async PUT request."""
        return await self.arequest("PUT", endpoint, data=data)

    async def adelete(self, endpoint: str) -> Dict[str, Any]:
        """Make async DELETE request."""
        return await self.arequest("DELETE", endpoint)

    def close(self):
        """Close HTTP client."""
        if HTTPX_AVAILABLE:
            self.client.close()
            if self.async_client:
                # Note: async client should be closed with await
                pass
        else:
            self.session.close()

    async def aclose(self):
        """Close async HTTP client."""
        if HTTPX_AVAILABLE and self.async_client:
            await self.async_client.aclose()
