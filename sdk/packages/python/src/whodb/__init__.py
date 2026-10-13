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
from .functions import FunctionHandle, FunctionsHandle
from .lineage import LineageProvidersHandle
from .ontology import OntologyHandle
from .service_accounts import ServiceAccountsHandle
from .sharing import SharingHandle
from .source import SourceHandle
from .transforms import TransformHandle, TransformsHandle

__version__ = SDK_VERSION

__all__ = [
    "WhoDB",
    "AsyncWhoDB",
    "DEFAULT_HOST",
    "OntologyHandle",
    "DatasetHandle",
    "SourceHandle",
    "TransformsHandle",
    "TransformHandle",
    "FunctionsHandle",
    "FunctionHandle",
    "ServiceAccountsHandle",
    "LineageProvidersHandle",
    "SharingHandle",
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
