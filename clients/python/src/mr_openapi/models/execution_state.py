"""Model Registry REST API.

REST API for Model Registry to create and manage ML model metadata

The version of the OpenAPI document: v1alpha3
Generated by OpenAPI Generator (https://openapi-generator.tech)

Do not edit the class manually.
"""  # noqa: E501

from __future__ import annotations

import json
from enum import Enum

from typing_extensions import Self


class ExecutionState(str, Enum):
    """The state of the Execution. The state transitions are   NEW -> RUNNING -> COMPLETE | CACHED | FAILED | CANCELED CACHED means the execution is skipped due to cached results. CANCELED means the execution is skipped due to precondition not met. It is different from CACHED in that a CANCELED execution will not have any event associated with it. It is different from FAILED in that there is no unexpected error happened and it is regarded as a normal state.  See also: ml-metadata Execution.State."""

    """
    allowed enum values
    """
    UNKNOWN = "UNKNOWN"
    NEW = "NEW"
    RUNNING = "RUNNING"
    COMPLETE = "COMPLETE"
    FAILED = "FAILED"
    CACHED = "CACHED"
    CANCELED = "CANCELED"

    @classmethod
    def from_json(cls, json_str: str) -> Self:
        """Create an instance of ExecutionState from a JSON string."""
        return cls(json.loads(json_str))