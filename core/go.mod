module github.com/clidey/whodb/core

go 1.26.1

require (
	cloud.google.com/go/alloydb v1.27.0
	cloud.google.com/go/memcache v1.16.0
	cloud.google.com/go/redis v1.23.0
	github.com/99designs/gqlgen v0.17.91
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.22.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.14.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3 v3.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers v1.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/redis/armredis v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions v1.3.0
	github.com/ClickHouse/clickhouse-go/v2 v2.46.0
	github.com/aws/aws-sdk-go-v2 v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.32.25
	github.com/aws/aws-sdk-go-v2/feature/rds/auth v1.6.29
	github.com/aws/aws-sdk-go-v2/service/docdb v1.49.5
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.54.3
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.70.8
	github.com/aws/aws-sdk-go-v2/service/rds v1.119.2
	github.com/aws/aws-sdk-go-v2/service/sts v1.43.3
	github.com/aws/smithy-go v1.27.2
	github.com/boundaryml/baml v0.222.0
	github.com/brianvoe/gofakeit/v7 v7.15.0
	github.com/deckarep/golang-set/v2 v2.9.0
	github.com/dromara/carbon/v2 v2.6.16
	github.com/duckdb/duckdb-go/v2 v2.10504.0
	github.com/elastic/elastic-transport-go/v8 v8.11.0
	github.com/elastic/go-elasticsearch/v8 v8.19.6
	github.com/go-chi/chi/v5 v5.3.0
	github.com/go-chi/cors v1.2.2
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-sql-driver/mysql v1.10.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-version v1.9.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/posthog/posthog-go v1.15.0
	github.com/shopspring/decimal v1.4.0
	github.com/sirupsen/logrus v1.9.4
	github.com/twpayne/go-geom v1.6.1
	github.com/vektah/gqlparser/v2 v2.5.35
	github.com/xuri/excelize/v2 v2.10.1
	github.com/zalando/go-keyring v0.2.8
	go.mongodb.org/mongo-driver/v2 v2.7.0
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sync v0.21.0
	google.golang.org/api v0.284.0
	google.golang.org/grpc v1.81.1
	gorm.io/driver/clickhouse v0.7.0
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.20.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/longrunning v1.0.0 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.12.0 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.7.2 // indirect
	github.com/ClickHouse/ch-go v0.71.0 // indirect
	github.com/agnivade/levenshtein v1.2.1 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/apache/arrow-go/v18 v18.5.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.24 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.29 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.29 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.2.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.31.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.36.6 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/danieljoos/wincred v1.2.3 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/duckdb/duckdb-go-bindings v0.10504.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10504.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10504.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10504.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10504.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10504.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.7.1 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.16 // indirect
	github.com/googleapis/gax-go/v2 v2.22.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.42 // indirect
	github.com/paulmach/orb v0.13.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/richardlehane/mscfb v1.0.6 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/sosodev/duration v1.4.0 // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/urfave/cli/v3 v3.9.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.mongodb.org/mongo-driver v1.17.7 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.67.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/mod v0.36.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/telemetry v0.0.0-20260508192327-42602be52be6 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.45.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
