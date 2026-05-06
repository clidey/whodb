package sealos

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNormalizeNamespaceMatchesDbproviderFallback(t *testing.T) {
	t.Run("prefer explicit context namespace", func(t *testing.T) {
		namespace, err := NamespaceFromKubeconfig(testKubeconfig("workspace-a"))
		if err != nil {
			t.Fatalf("namespace from kubeconfig: %v", err)
		}
		if namespace != "workspace-a" {
			t.Fatalf("expected explicit namespace, got %q", namespace)
		}
	})

	t.Run("fallback to ns-user", func(t *testing.T) {
		namespace, err := NamespaceFromKubeconfig(testKubeconfig(""))
		if err != nil {
			t.Fatalf("namespace from kubeconfig: %v", err)
		}
		if namespace != "ns-demo-user" {
			t.Fatalf("expected dbprovider-compatible fallback, got %q", namespace)
		}
	})
}

func TestNormalizeSecretHostMatchesDbprovider(t *testing.T) {
	if got := NormalizeSecretHost("db-host", "workspace-a"); got != "db-host.workspace-a.svc" {
		t.Fatalf("expected namespace svc host, got %q", got)
	}
	if got := NormalizeSecretHost("db-host.workspace-a.svc", "workspace-a"); got != "db-host.workspace-a.svc" {
		t.Fatalf("expected .svc host to remain unchanged, got %q", got)
	}
	if got := NormalizeSecretHost("db-host.workspace-a.svc.cluster.local", "workspace-a"); got != "db-host.workspace-a.svc.cluster.local" {
		t.Fatalf("expected cluster-local host to remain unchanged, got %q", got)
	}
}

func TestSecretNameMatchesDbproviderConvention(t *testing.T) {
	if got := SecretName("my-db"); got != "my-db-conn-credential" {
		t.Fatalf("expected dbprovider secret naming, got %q", got)
	}
}

func TestNormalizeCredentialsAllowsClickHouseDefaults(t *testing.T) {
	spec := dbTypeSpecs["clickhouse"]

	normalized, err := normalizeResolvedCredentials(spec, "", "", "clickhouse.ns.svc", "8123")
	if err != nil {
		t.Fatalf("expected clickhouse credentials to allow empty auth, got %v", err)
	}
	if normalized.username != "default" {
		t.Fatalf("expected clickhouse username to default to 'default', got %q", normalized.username)
	}
	if normalized.password != "" {
		t.Fatalf("expected clickhouse password to remain empty, got %q", normalized.password)
	}
}

func TestResolveBootstrapForRedisUsesAccountSecretAndService(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-stacks-redis-account-default",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username": []byte("default"),
				"password": []byte("redis-password"),
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-stacks-redis-redis",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-stacks",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.1",
				Ports: []corev1.ServicePort{
					{Name: "redis", Port: 6379},
				},
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "redis",
		ResourceName: "test-stacks",
	})
	if err != nil {
		t.Fatalf("expected redis bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-stacks-redis-redis.ns-admin.svc" || result.Port != "6379" {
		t.Fatalf("expected redis host/port from service, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "default" || result.Credentials.Password != "redis-password" {
		t.Fatalf("expected redis credentials from account secret, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForRedisFallsBackToConnCredential(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-redis-conn-credential",
				Namespace: "ns-q4jurwf5",
			},
			Data: map[string][]byte{
				"username": []byte("default"),
				"password": []byte("redis-password"),
				"host":     []byte("test-db-redis-redis"),
				"port":     []byte("6379"),
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-q4jurwf5"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "redis",
		ResourceName: "test-db-redis",
	})
	if err != nil {
		t.Fatalf("expected redis generic conn-credential bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-db-redis-redis.ns-q4jurwf5.svc" || result.Port != "6379" {
		t.Fatalf("expected redis host/port from conn-credential, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "default" || result.Credentials.Password != "redis-password" {
		t.Fatalf("expected redis credentials from conn-credential, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForMongoDBUsesAccountSecretAndService(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb-account-root",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username": []byte("root"),
				"password": []byte("mongo-password"),
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb-mongodb",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-db",
					"kubeblocks.io/role":         "primary",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.3",
				Ports: []corev1.ServicePort{
					{Name: "mongodb", Port: 27017},
				},
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "mongodb",
		ResourceName: "test-db",
	})
	if err != nil {
		t.Fatalf("expected mongodb bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-db-mongodb-mongodb.ns-admin.svc" || result.Port != "27017" {
		t.Fatalf("expected mongodb host/port from service, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "root" || result.Credentials.Password != "mongo-password" {
		t.Fatalf("expected mongodb credentials from account secret, got %#v", result.Credentials)
	}
	if result.DatabaseName != "admin" || result.Credentials.Database != "admin" {
		t.Fatalf("expected mongodb default database admin, got result=%q credentials=%q", result.DatabaseName, result.Credentials.Database)
	}
}

func TestResolveBootstrapForMongoDBKeepsConnCredentialCompatibility(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-conn-credential",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username": []byte("root"),
				"password": []byte("mongo-password"),
				"host":     []byte("test-db-mongodb"),
				"port":     []byte("27017"),
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "mongodb",
		ResourceName: "test-db",
	})
	if err != nil {
		t.Fatalf("expected mongodb legacy conn-credential bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-db-mongodb.ns-admin.svc" || result.Port != "27017" {
		t.Fatalf("expected mongodb host/port from conn-credential, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "root" || result.Credentials.Password != "mongo-password" {
		t.Fatalf("expected mongodb credentials from conn-credential, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForMongoDBPrefersPrimaryService(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb-account-root",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username": []byte("root"),
				"password": []byte("mongo-password"),
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-db",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.4",
				Ports: []corev1.ServicePort{
					{Name: "mongodb", Port: 27017},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb-mongodb-ro",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-db",
					"kubeblocks.io/role":         "secondary",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.5",
				Ports: []corev1.ServicePort{
					{Name: "mongodb", Port: 27017},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db-mongodb-mongodb",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-db",
					"kubeblocks.io/role":         "primary",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.6",
				Ports: []corev1.ServicePort{
					{Name: "mongodb", Port: 27017},
				},
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "mongodb",
		ResourceName: "test-db",
	})
	if err != nil {
		t.Fatalf("expected mongodb bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-db-mongodb-mongodb.ns-admin.svc" || result.Port != "27017" {
		t.Fatalf("expected mongodb primary service endpoint, got %q:%q", result.Host, result.Port)
	}
}

func TestResolveBootstrapForClickHouseUsesAdminPasswordAndTcpEndpoint(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-conn-credential",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username":       []byte("admin"),
				"admin-password": []byte("house-password"),
				"tcpEndpoint":    []byte("test-house-clickhouse:9000"),
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "clickhouse",
		ResourceName: "test-house",
	})
	if err != nil {
		t.Fatalf("expected clickhouse bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-house-clickhouse.ns-admin.svc" || result.Port != "9000" {
		t.Fatalf("expected clickhouse tcp endpoint to be normalized, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "admin" || result.Credentials.Password != "house-password" {
		t.Fatalf("expected clickhouse auth from secret, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForClickHouseUsesNativeServiceWhenTcpEndpointMissing(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-conn-credential",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username":       []byte("admin"),
				"admin-password": []byte("house-password"),
				"endpoint":       []byte("test-house-clickhouse:8123"),
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-clickhouse",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-house",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.2",
				Ports: []corev1.ServicePort{
					{Name: "http", Port: 8123},
					{Name: "tcp", Port: 9000},
				},
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "clickhouse",
		ResourceName: "test-house",
	})
	if err != nil {
		t.Fatalf("expected clickhouse bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-house-clickhouse.ns-admin.svc" || result.Port != "9000" {
		t.Fatalf("expected clickhouse native service endpoint, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "admin" || result.Credentials.Password != "house-password" {
		t.Fatalf("expected clickhouse auth from secret, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForClickHouseUsesNativeServiceWhenTcpEndpointIsTemplated(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-conn-credential",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username":       []byte("admin"),
				"admin-password": []byte("house-password"),
				"tcpEndpoint":    []byte("test-house-zookeeper:$(SVC_PORT_tcp)"),
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-clickhouse",
				Namespace: "ns-admin",
				Labels: map[string]string{
					"app.kubernetes.io/instance": "test-house",
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "10.0.0.2",
				Ports: []corev1.ServicePort{
					{Name: "http", Port: 8123},
					{Name: "tcp", Port: 9000},
				},
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "clickhouse",
		ResourceName: "test-house",
	})
	if err != nil {
		t.Fatalf("expected clickhouse bootstrap to succeed, got %v", err)
	}
	if result.Host != "test-house-clickhouse.ns-admin.svc" || result.Port != "9000" {
		t.Fatalf("expected clickhouse native service endpoint, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "admin" || result.Credentials.Password != "house-password" {
		t.Fatalf("expected clickhouse auth from secret, got %#v", result.Credentials)
	}
}

func TestResolveBootstrapForClickHouseFallsBackToGenericFields(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-house-conn-credential",
				Namespace: "ns-admin",
			},
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("house-password"),
				"host":     []byte("test-house-clickhouse"),
				"port":     []byte("9000"),
			},
		},
	)

	resolver := &resolver{
		kubeconfig: testKubeconfig("ns-admin"),
		clientset:  clientset,
	}

	result, err := resolver.ResolveBootstrap(context.Background(), BootstrapInput{
		DBType:       "clickhouse",
		ResourceName: "test-house",
	})
	if err != nil {
		t.Fatalf("expected clickhouse generic fallback to succeed, got %v", err)
	}
	if result.Host != "test-house-clickhouse.ns-admin.svc" || result.Port != "9000" {
		t.Fatalf("expected clickhouse host/port from generic fields, got %q:%q", result.Host, result.Port)
	}
	if result.Credentials.Username != "admin" || result.Credentials.Password != "house-password" {
		t.Fatalf("expected clickhouse auth from generic fields, got %#v", result.Credentials)
	}
}

func testKubeconfig(namespace string) string {
	if namespace == "" {
		return `apiVersion: v1
kind: Config
clusters:
- name: demo-cluster
  cluster:
    server: https://example.invalid
users:
- name: demo-user
  user:
    token: test
contexts:
- name: demo-user@demo-cluster
  context:
    cluster: demo-cluster
    user: demo-user
current-context: demo-user@demo-cluster
`
	}

	return `apiVersion: v1
kind: Config
clusters:
- name: demo-cluster
  cluster:
    server: https://example.invalid
users:
- name: demo-user
  user:
    token: test
contexts:
- name: demo-user@demo-cluster
  context:
    cluster: demo-cluster
    user: demo-user
    namespace: ` + namespace + `
current-context: demo-user@demo-cluster
`
}
