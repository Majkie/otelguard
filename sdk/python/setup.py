"""Setup configuration for OTelGuard Python SDK."""

from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="otelguard-sdk",
    version="0.1.0",
    author="OTelGuard Team",
    author_email="support@otelguard.dev",
    description="Enterprise-grade LLM observability and guardrails platform SDK",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/your-org/otelguard",
    packages=find_packages(exclude=["tests", "tests.*", "examples"]),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "Topic :: Scientific/Engineering :: Artificial Intelligence",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
    python_requires=">=3.8",
    install_requires=[
        "requests>=2.28.0",
    ],
    extras_require={
        "async": [
            "httpx>=0.24.0",
        ],
        "validators": [
            "jsonschema>=4.0.0",
        ],
        "dev": [
            "pytest>=7.0.0",
            "pytest-asyncio>=0.21.0",
            "pytest-cov>=4.0.0",
            "black>=23.0.0",
            "flake8>=6.0.0",
            "mypy>=1.0.0",
            "httpx>=0.24.0",
            "jsonschema>=4.0.0",
        ],
    },
    entry_points={
        "console_scripts": [
            "otelguard=otelguard.cli:main",
        ],
    },
)
