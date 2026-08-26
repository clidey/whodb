"""Official Python SDK for the WhoDB hosted platform.

```python
from whodb import WhoDB

whodb = WhoDB(api_key=os.environ["WHODB_API_KEY"])
user = whodb.ontology("User").get("u_123")
```
"""

from ._auth import CliCredentials, StaticCredentials
from ._errors import (
    AuthError,
    CliCredentialsError,
    NotFoundError,
    PlatformError,
    TransportCapabilityError,
    ValidationError,
    WhoDBError,
    WhoDBVersionError,
)
from ._pagination import ListCall, Page
from ._version import SDK_VERSION
from .client import DEFAULT_HOST, AsyncWhoDB, WhoDB
from .dataset import DatasetHandle
from .ontology import OntologyHandle
from .source import SourceHandle

__version__ = SDK_VERSION

__all__ = [
    "WhoDB",
    "AsyncWhoDB",
    "DEFAULT_HOST",
    "OntologyHandle",
    "DatasetHandle",
    "SourceHandle",
    "ListCall",
    "Page",
    "WhoDBError",
    "AuthError",
    "NotFoundError",
    "ValidationError",
    "WhoDBVersionError",
    "CliCredentialsError",
    "TransportCapabilityError",
    "PlatformError",
    "StaticCredentials",
    "CliCredentials",
]
