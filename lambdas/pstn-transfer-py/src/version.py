"""
Version information for the Lambda function
This is injected at build time via environment variable or setup script
Falls back to environment variable or "unknown" if not set
"""

import os

# VERSION_VALUE will be replaced at build time
# Falls back to environment variable or "unknown"
VERSION: str = os.environ.get("LAMBDA_VERSION", "unknown")
