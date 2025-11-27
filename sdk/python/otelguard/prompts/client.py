"""Prompts client for managing prompts and versions."""

import logging
from typing import Any, Dict, List, Optional

from otelguard.client.config import Config
from otelguard.client.http_client import HTTPClient


logger = logging.getLogger(__name__)


class PromptsClient:
    """Client for interacting with prompts API."""

    def __init__(self, http_client: HTTPClient, config: Config):
        self.http_client = http_client
        self.config = config

    def list(
        self,
        limit: int = 50,
        offset: int = 0,
        tags: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """List prompts.

        Args:
            limit: Maximum number of prompts to return
            offset: Pagination offset
            tags: Filter by tags

        Returns:
            List of prompts with pagination info
        """
        params = {"limit": limit, "offset": offset}
        if tags:
            params["tags"] = ",".join(tags)

        try:
            result = self.http_client.get("/v1/prompts", params=params)
            return result
        except Exception as e:
            logger.error(f"Failed to list prompts: {e}")
            return {"data": [], "total": 0}

    async def alist(
        self,
        limit: int = 50,
        offset: int = 0,
        tags: Optional[List[str]] = None,
    ) -> Dict[str, Any]:
        """Async list prompts."""
        params = {"limit": limit, "offset": offset}
        if tags:
            params["tags"] = ",".join(tags)

        try:
            result = await self.http_client.aget("/v1/prompts", params=params)
            return result
        except Exception as e:
            logger.error(f"Failed to list prompts (async): {e}")
            return {"data": [], "total": 0}

    def get(self, prompt_id: str) -> Optional[Dict[str, Any]]:
        """Get prompt by ID.

        Args:
            prompt_id: Prompt ID

        Returns:
            Prompt data or None if not found
        """
        try:
            result = self.http_client.get(f"/v1/prompts/{prompt_id}")
            return result
        except Exception as e:
            logger.error(f"Failed to get prompt {prompt_id}: {e}")
            return None

    async def aget(self, prompt_id: str) -> Optional[Dict[str, Any]]:
        """Async get prompt by ID."""
        try:
            result = await self.http_client.aget(f"/v1/prompts/{prompt_id}")
            return result
        except Exception as e:
            logger.error(f"Failed to get prompt {prompt_id} (async): {e}")
            return None

    def create(
        self,
        name: str,
        description: Optional[str] = None,
        tags: Optional[List[str]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Create a new prompt.

        Args:
            name: Prompt name
            description: Prompt description
            tags: List of tags

        Returns:
            Created prompt data
        """
        data = {
            "name": name,
            "description": description,
            "tags": tags or [],
        }

        try:
            result = self.http_client.post("/v1/prompts", data=data)
            return result
        except Exception as e:
            logger.error(f"Failed to create prompt: {e}")
            return None

    async def acreate(
        self,
        name: str,
        description: Optional[str] = None,
        tags: Optional[List[str]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Async create a new prompt."""
        data = {
            "name": name,
            "description": description,
            "tags": tags or [],
        }

        try:
            result = await self.http_client.apost("/v1/prompts", data=data)
            return result
        except Exception as e:
            logger.error(f"Failed to create prompt (async): {e}")
            return None

    def get_version(self, prompt_id: str, version: int) -> Optional[Dict[str, Any]]:
        """Get specific prompt version.

        Args:
            prompt_id: Prompt ID
            version: Version number

        Returns:
            Prompt version data
        """
        try:
            result = self.http_client.get(f"/v1/prompts/{prompt_id}/versions/{version}")
            return result
        except Exception as e:
            logger.error(f"Failed to get prompt version: {e}")
            return None

    async def aget_version(self, prompt_id: str, version: int) -> Optional[Dict[str, Any]]:
        """Async get specific prompt version."""
        try:
            result = await self.http_client.aget(f"/v1/prompts/{prompt_id}/versions/{version}")
            return result
        except Exception as e:
            logger.error(f"Failed to get prompt version (async): {e}")
            return None

    def create_version(
        self,
        prompt_id: str,
        content: str,
        config: Optional[Dict[str, Any]] = None,
        labels: Optional[List[str]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Create a new prompt version.

        Args:
            prompt_id: Prompt ID
            content: Prompt content/template
            config: Configuration (model, temperature, etc.)
            labels: Version labels (e.g., ["production", "staging"])

        Returns:
            Created version data
        """
        data = {
            "content": content,
            "config": config or {},
            "labels": labels or [],
        }

        try:
            result = self.http_client.post(f"/v1/prompts/{prompt_id}/versions", data=data)
            return result
        except Exception as e:
            logger.error(f"Failed to create prompt version: {e}")
            return None

    async def acreate_version(
        self,
        prompt_id: str,
        content: str,
        config: Optional[Dict[str, Any]] = None,
        labels: Optional[List[str]] = None,
    ) -> Optional[Dict[str, Any]]:
        """Async create a new prompt version."""
        data = {
            "content": content,
            "config": config or {},
            "labels": labels or [],
        }

        try:
            result = await self.http_client.apost(f"/v1/prompts/{prompt_id}/versions", data=data)
            return result
        except Exception as e:
            logger.error(f"Failed to create prompt version (async): {e}")
            return None

    def compile(
        self,
        prompt_id: str,
        version: Optional[int] = None,
        variables: Optional[Dict[str, Any]] = None,
    ) -> Optional[str]:
        """Compile prompt template with variables.

        Args:
            prompt_id: Prompt ID
            version: Version number (uses latest if not specified)
            variables: Variables to substitute in template

        Returns:
            Compiled prompt text
        """
        data = {
            "variables": variables or {},
        }

        endpoint = f"/v1/prompts/{prompt_id}/compile"
        if version is not None:
            endpoint = f"/v1/prompts/{prompt_id}/versions/{version}/compile"

        try:
            result = self.http_client.post(endpoint, data=data)
            return result.get("compiled")
        except Exception as e:
            logger.error(f"Failed to compile prompt: {e}")
            return None

    async def acompile(
        self,
        prompt_id: str,
        version: Optional[int] = None,
        variables: Optional[Dict[str, Any]] = None,
    ) -> Optional[str]:
        """Async compile prompt template with variables."""
        data = {
            "variables": variables or {},
        }

        endpoint = f"/v1/prompts/{prompt_id}/compile"
        if version is not None:
            endpoint = f"/v1/prompts/{prompt_id}/versions/{version}/compile"

        try:
            result = await self.http_client.apost(endpoint, data=data)
            return result.get("compiled")
        except Exception as e:
            logger.error(f"Failed to compile prompt (async): {e}")
            return None

    def list_versions(self, prompt_id: str) -> List[Dict[str, Any]]:
        """List all versions of a prompt.

        Args:
            prompt_id: Prompt ID

        Returns:
            List of prompt versions
        """
        try:
            result = self.http_client.get(f"/v1/prompts/{prompt_id}/versions")
            return result.get("data", [])
        except Exception as e:
            logger.error(f"Failed to list prompt versions: {e}")
            return []

    async def alist_versions(self, prompt_id: str) -> List[Dict[str, Any]]:
        """Async list all versions of a prompt."""
        try:
            result = await self.http_client.aget(f"/v1/prompts/{prompt_id}/versions")
            return result.get("data", [])
        except Exception as e:
            logger.error(f"Failed to list prompt versions (async): {e}")
            return []
