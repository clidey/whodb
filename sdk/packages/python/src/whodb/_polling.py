# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Polling helpers shared by the transform and function run facades."""

from __future__ import annotations

import json
import time
from collections.abc import Callable
from typing import Any, TypeVar

from ._errors import PlatformError

DEFAULT_WAIT_TIMEOUT_MS = 60_000
DEFAULT_WAIT_POLL_MS = 2_000

_TERMINAL = {
    "success",
    "succeeded",
    "completed",
    "complete",
    "failed",
    "failure",
    "cancelled",
    "canceled",
    "error",
}

T = TypeVar("T")


def is_terminal_run_status(status: Any) -> bool:
    """Run statuses after which a function or transform run will not change."""
    return str(status or "").strip().lower() in _TERMINAL


def encode_input(value: Any) -> str:
    """Encode a function input as the compact JSON string the platform expects."""
    if isinstance(value, str):
        return value
    return json.dumps({} if value is None else value, separators=(",", ":"))


def poll_until(
    read: Callable[[], T],
    done: Callable[[T], bool],
    describe: str,
    timeout_ms: int | None = None,
    poll_ms: int | None = None,
) -> T:
    """Poll ``read`` until ``done`` or the timeout elapses (PlatformError RUN_TIMEOUT)."""
    timeout = DEFAULT_WAIT_TIMEOUT_MS if timeout_ms is None else timeout_ms
    poll = DEFAULT_WAIT_POLL_MS if poll_ms is None else poll_ms
    deadline = time.monotonic() + timeout / 1000
    while True:
        value = read()
        if done(value):
            return value
        if time.monotonic() >= deadline:
            raise PlatformError(f"{describe} did not finish within {timeout}ms", "RUN_TIMEOUT")
        if poll > 0:
            time.sleep(poll / 1000)
