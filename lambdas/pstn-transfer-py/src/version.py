"""
Version information for the Lambda function
This is injected at build time via environment variable or setup script
Falls back to environment variable or "unknown" if not set
"""

import os

# BUILD_TIME_VERSION is a placeholder that build tools can replace at build time
# (e.g., sed -i 's/__VERSION__/1.2.3/g' version.py)
BUILD_TIME_VERSION: str = "__VERSION__"


# VERSION resolution order:
# 1. LAMBDA_VERSION environment variable (runtime override)
# 2. BUILD_TIME_VERSION if it was replaced at build time
# 3. "unknown" as final fallback
def _resolve_version() -> str:
    env_version = os.environ.get("LAMBDA_VERSION")
    if env_version:
        return env_version
    if BUILD_TIME_VERSION != "__VERSION__":
        return BUILD_TIME_VERSION
    return "unknown"


VERSION: str = _resolve_version()
