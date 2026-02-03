"""
Logger provides structured logging functionality matching the Go and TypeScript implementations
"""

import json
import os
import sys
from typing import Any


class Logger:
    """Logger with debug, info, warn, and error levels"""

    def __init__(self) -> None:
        self._debug_enabled = os.environ.get("DEBUG_LOGGING", "").lower() == "true"

    def debugf(self, format_str: str, *args: Any) -> None:
        """Log debug message if debug logging is enabled"""
        if self._debug_enabled:
            message = self._format_message(format_str, *args)
            print(f"[DEBUG] {message}", file=sys.stdout)

    def infof(self, format_str: str, *args: Any) -> None:
        """Log info message"""
        message = self._format_message(format_str, *args)
        print(f"[INFO] {message}", file=sys.stdout)

    def warnf(self, format_str: str, *args: Any) -> None:
        """Log warning message"""
        message = self._format_message(format_str, *args)
        print(f"[WARN] {message}", file=sys.stderr)

    def errorf(self, format_str: str, *args: Any) -> None:
        """Log error message"""
        message = self._format_message(format_str, *args)
        print(f"[ERROR] {message}", file=sys.stderr)

    def _format_message(self, format_str: str, *args: Any) -> str:
        """Format message with Go-style format specifiers (%s, %v, %d, %+v)"""
        message = format_str
        for arg in args:
            if isinstance(arg, (dict, list)):
                try:
                    value = json.dumps(arg)
                except (TypeError, ValueError):
                    value = json.dumps(arg, default=str)
            elif isinstance(arg, (int, float)):
                value = str(arg)
            else:
                value = str(arg)

            # Replace first occurrence of format specifier
            for spec in ["%s", "%v", "%d", "%+v"]:
                if spec in message:
                    message = message.replace(spec, value, 1)
                    break

        return message


def new_logger() -> Logger:
    """Create a new Logger instance"""
    return Logger()
