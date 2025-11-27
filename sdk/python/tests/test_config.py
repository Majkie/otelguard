"""Tests for configuration module."""

import os
import pytest
from otelguard.client.config import Config


class TestConfig:
    """Test Config class."""

    def test_config_creation(self):
        """Test basic config creation."""
        config = Config(
            api_key="test-key",
            project="test-project"
        )
        assert config.api_key == "test-key"
        assert config.project == "test-project"
        assert config.base_url == "http://localhost:8080"
        assert config.timeout == 30

    def test_config_custom_values(self):
        """Test config with custom values."""
        config = Config(
            api_key="test-key",
            project="test-project",
            base_url="https://api.otelguard.dev",
            timeout=60,
            max_retries=5
        )
        assert config.base_url == "https://api.otelguard.dev"
        assert config.timeout == 60
        assert config.max_retries == 5

    def test_config_validation_missing_api_key(self):
        """Test that missing API key raises ValueError."""
        with pytest.raises(ValueError, match="api_key"):
            Config(api_key="", project="test-project")

    def test_config_validation_missing_project(self):
        """Test that missing project raises ValueError."""
        with pytest.raises(ValueError, match="project"):
            Config(api_key="test-key", project="")

    def test_config_validation_invalid_timeout(self):
        """Test that invalid timeout raises ValueError."""
        with pytest.raises(ValueError, match="timeout"):
            Config(api_key="test-key", project="test-project", timeout=-1)

    def test_config_validation_invalid_batch_size(self):
        """Test that invalid batch size raises ValueError."""
        with pytest.raises(ValueError, match="batch_size"):
            Config(api_key="test-key", project="test-project", batch_size=0)

    def test_config_from_env(self, monkeypatch):
        """Test config creation from environment variables."""
        monkeypatch.setenv("OTELGUARD_API_KEY", "env-key")
        monkeypatch.setenv("OTELGUARD_PROJECT", "env-project")
        monkeypatch.setenv("OTELGUARD_BASE_URL", "https://custom.api")
        monkeypatch.setenv("OTELGUARD_DEBUG", "true")

        config = Config.from_env()
        assert config.api_key == "env-key"
        assert config.project == "env-project"
        assert config.base_url == "https://custom.api"
        assert config.debug is True

    def test_config_from_env_missing(self, monkeypatch):
        """Test that from_env raises ValueError when env vars are missing."""
        monkeypatch.delenv("OTELGUARD_API_KEY", raising=False)
        monkeypatch.delenv("OTELGUARD_PROJECT", raising=False)

        with pytest.raises(ValueError, match="api_key"):
            Config.from_env()

    def test_config_from_env_with_override(self, monkeypatch):
        """Test that from_env allows overrides."""
        monkeypatch.setenv("OTELGUARD_API_KEY", "env-key")
        monkeypatch.setenv("OTELGUARD_PROJECT", "env-project")

        config = Config.from_env(timeout=120)
        assert config.api_key == "env-key"
        assert config.project == "env-project"
        assert config.timeout == 120


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
