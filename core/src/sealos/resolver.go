package sealos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	yaml "go.yaml.in/yaml/v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// BootstrapInput describes the public Sealos bootstrap request.
type BootstrapInput struct {
	Kubeconfig   string
	DBType       string
	ResourceName string
	DatabaseName string
	Host         string
	Port         string
	Namespace    string
}

// ResolvedBootstrap is the dbprovider-compatible bootstrap result returned to GraphQL.
type ResolvedBootstrap struct {
	Namespace    string
	ResourceName string
	DBType       string
	Host         string
	Port         string
	DatabaseName string
	K8sUsername  string
	Credentials  *engine.Credentials
}

// BootstrapResolver resolves Sealos bootstrap metadata into database credentials.
type BootstrapResolver interface {
	ResolveBootstrap(context.Context, BootstrapInput) (*ResolvedBootstrap, error)
}

// BootstrapResolverFactory creates a resolver from a Sealos kubeconfig.
type BootstrapResolverFactory func(kubeconfig string) (BootstrapResolver, error)

// DefaultBootstrapResolverFactory creates the production Sealos bootstrap resolver.
var DefaultBootstrapResolverFactory BootstrapResolverFactory = NewBootstrapResolver

type kubeconfig struct {
	CurrentContext string `yaml:"current-context"`
	Contexts       []struct {
		Name    string `yaml:"name"`
		Context struct {
			Namespace string `yaml:"namespace"`
			User      string `yaml:"user"`
		} `yaml:"context"`
	} `yaml:"contexts"`
	Users []struct {
		Name string `yaml:"name"`
	} `yaml:"users"`
}

// SecretName returns the dbprovider-compatible secret name for a database resource.
func SecretName(resourceName string) string {
	return resourceName + "-conn-credential"
}

// NormalizeSecretHost returns the dbprovider-compatible host value from a secret.
func NormalizeSecretHost(host string, namespace string) string {
	if strings.Contains(host, ".svc") {
		return host
	}
	return host + "." + namespace + ".svc"
}

// NamespaceFromKubeconfig resolves the effective namespace from a Sealos kubeconfig.
func NamespaceFromKubeconfig(raw string) (string, error) {
	var cfg kubeconfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return "", fmt.Errorf("parse kubeconfig: %w", err)
	}

	for _, ctx := range cfg.Contexts {
		if ctx.Name == cfg.CurrentContext {
			if strings.TrimSpace(ctx.Context.Namespace) != "" {
				return strings.TrimSpace(ctx.Context.Namespace), nil
			}
			if strings.TrimSpace(ctx.Context.User) != "" {
				return "ns-" + strings.TrimSpace(ctx.Context.User), nil
			}
		}
	}

	if len(cfg.Contexts) > 0 {
		namespace := strings.TrimSpace(cfg.Contexts[0].Context.Namespace)
		if namespace != "" {
			return namespace, nil
		}
		user := strings.TrimSpace(cfg.Contexts[0].Context.User)
		if user != "" {
			return "ns-" + user, nil
		}
	}

	if len(cfg.Users) > 0 && strings.TrimSpace(cfg.Users[0].Name) != "" {
		return "ns-" + strings.TrimSpace(cfg.Users[0].Name), nil
	}

	return "", errors.New("namespace not found in kubeconfig")
}

type resolver struct {
	kubeconfig string
	clientset  kubernetes.Interface
}

// NewBootstrapResolver creates the production Sealos bootstrap resolver.
func NewBootstrapResolver(kubeconfig string) (BootstrapResolver, error) {
	if strings.TrimSpace(kubeconfig) == "" {
		return nil, errors.New("kubeconfig is required")
	}

	config, err := clientConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	return &resolver{
		kubeconfig: kubeconfig,
		clientset:  clientset,
	}, nil
}

// ResolveBootstrap resolves bootstrap metadata into database credentials.
func (r *resolver) ResolveBootstrap(ctx context.Context, input BootstrapInput) (*ResolvedBootstrap, error) {
	spec, ok := dbTypeSpecs[input.DBType]
	if !ok {
		return nil, fmt.Errorf("unsupported dbType %q", input.DBType)
	}

	resourceName := strings.TrimSpace(input.ResourceName)
	if resourceName == "" {
		return nil, errors.New("resourceName is required")
	}

	namespace := strings.TrimSpace(input.Namespace)
	if namespace == "" {
		var err error
		namespace, err = NamespaceFromKubeconfig(r.kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	resolvedData, err := r.resolveConnectionData(ctx, spec, namespace, resourceName)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeResolvedCredentials(spec, resolvedData.username, resolvedData.password, resolvedData.host, resolvedData.port)
	if err != nil {
		return nil, err
	}

	if requestedHost := strings.TrimSpace(input.Host); requestedHost != "" && requestedHost != normalized.host {
		return nil, fmt.Errorf("host mismatch: requested %q resolved %q", requestedHost, normalized.host)
	}
	if requestedPort := strings.TrimSpace(input.Port); requestedPort != "" && requestedPort != normalized.port {
		return nil, fmt.Errorf("port mismatch: requested %q resolved %q", requestedPort, normalized.port)
	}

	databaseName := strings.TrimSpace(input.DatabaseName)
	if databaseName == "" {
		databaseName = spec.DefaultDatabase
	}

	credentials := &engine.Credentials{
		Type:     spec.EngineType,
		Hostname: normalized.host,
		Username: normalized.username,
		Password: normalized.password,
		Database: databaseName,
	}
	if normalized.port != "" {
		credentials.Advanced = []engine.Record{{Key: "Port", Value: normalized.port}}
	}

	return &ResolvedBootstrap{
		Namespace:    namespace,
		ResourceName: resourceName,
		DBType:       spec.EngineType,
		Host:         normalized.host,
		Port:         normalized.port,
		DatabaseName: databaseName,
		K8sUsername:  currentUserName(r.kubeconfig),
		Credentials:  credentials,
	}, nil
}

type dbTypeSpec struct {
	EngineType      string
	UsernameKey     string
	PasswordKey     string
	HostKey         string
	PortKey         string
	DefaultDatabase string
	AllowEmptyAuth  bool
	DefaultUsername string
}

var dbTypeSpecs = map[string]dbTypeSpec{
	"postgresql": {
		EngineType:      string(engine.DatabaseType_Postgres),
		UsernameKey:     "username",
		PasswordKey:     "password",
		HostKey:         "host",
		PortKey:         "port",
		DefaultDatabase: "postgres",
	},
	"apecloud-mysql": {
		EngineType:      string(engine.DatabaseType_MySQL),
		UsernameKey:     "username",
		PasswordKey:     "password",
		HostKey:         "host",
		PortKey:         "port",
		DefaultDatabase: "",
	},
	"mongodb": {
		EngineType:      string(engine.DatabaseType_MongoDB),
		UsernameKey:     "username",
		PasswordKey:     "password",
		HostKey:         "host",
		PortKey:         "port",
		DefaultDatabase: "admin",
	},
	"redis": {
		EngineType:      string(engine.DatabaseType_Redis),
		UsernameKey:     "username",
		PasswordKey:     "password",
		HostKey:         "host",
		PortKey:         "port",
		DefaultDatabase: "",
		AllowEmptyAuth:  true,
	},
	"clickhouse": {
		EngineType:      string(engine.DatabaseType_ClickHouse),
		UsernameKey:     "username",
		PasswordKey:     "password",
		HostKey:         "host",
		PortKey:         "port",
		DefaultDatabase: "default",
		AllowEmptyAuth:  true,
		DefaultUsername: "default",
	},
}

func decodeSecretField(value []byte) string {
	return strings.TrimSpace(string(value))
}

func genericResolveFromSecret(spec dbTypeSpec, secretData map[string][]byte, namespace string) *connectionData {
	return &connectionData{
		username: decodeSecretField(secretData[spec.UsernameKey]),
		password: decodeSecretField(secretData[spec.PasswordKey]),
		host:     NormalizeSecretHost(decodeSecretField(secretData[spec.HostKey]), namespace),
		port:     decodeSecretField(secretData[spec.PortKey]),
	}
}

type normalizedCredentials struct {
	username string
	password string
	host     string
	port     string
}

func normalizeResolvedCredentials(spec dbTypeSpec, username, password, host, port string) (*normalizedCredentials, error) {
	if host == "" || port == "" {
		return nil, errors.New("secret missing required fields")
	}

	if spec.AllowEmptyAuth {
		if username == "" && spec.DefaultUsername != "" {
			username = spec.DefaultUsername
		}
		return &normalizedCredentials{
			username: username,
			password: password,
			host:     host,
			port:     port,
		}, nil
	}

	if username == "" || password == "" {
		return nil, errors.New("secret missing required fields")
	}

	return &normalizedCredentials{
		username: username,
		password: password,
		host:     host,
		port:     port,
	}, nil
}

type connectionData struct {
	username string
	password string
	host     string
	port     string
}

func (r *resolver) resolveConnectionData(
	ctx context.Context,
	spec dbTypeSpec,
	namespace string,
	resourceName string,
) (*connectionData, error) {
	switch spec.EngineType {
	case string(engine.DatabaseType_Redis):
		return r.resolveRedisConnectionData(ctx, spec, namespace, resourceName)
	case string(engine.DatabaseType_MongoDB):
		return r.resolveMongoDBConnectionData(ctx, spec, namespace, resourceName)
	case string(engine.DatabaseType_ClickHouse):
		secretData, err := r.readConnCredentialSecret(ctx, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return r.resolveClickHouseConnectionData(ctx, spec, namespace, resourceName, secretData)
	default:
		secretData, err := r.readConnCredentialSecret(ctx, namespace, resourceName)
		if err != nil {
			return nil, err
		}
		return genericResolveFromSecret(spec, secretData, namespace), nil
	}
}

func (r *resolver) readConnCredentialSecret(ctx context.Context, namespace, resourceName string) (map[string][]byte, error) {
	secret, err := r.clientset.CoreV1().Secrets(namespace).Get(ctx, SecretName(resourceName), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("read secret: %w", err)
	}
	if secret.Data == nil {
		return nil, errors.New("secret is empty")
	}
	return secret.Data, nil
}

func (r *resolver) resolveMongoDBConnectionData(
	ctx context.Context,
	spec dbTypeSpec,
	namespace string,
	resourceName string,
) (*connectionData, error) {
	connSecret, err := r.clientset.CoreV1().Secrets(namespace).Get(ctx, SecretName(resourceName), metav1.GetOptions{})
	if err == nil {
		if connSecret.Data == nil {
			return nil, errors.New("secret is empty")
		}
		return genericResolveFromSecret(spec, connSecret.Data, namespace), nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("read secret: %w", err)
	}

	secretName := resourceName + "-mongodb-account-root"
	accountSecret, err := r.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("read mongodb account secret: %w", err)
	}

	service, port, err := r.findPrimaryServiceByPort(ctx, namespace, resourceName, 27017)
	if err != nil {
		return nil, err
	}

	return &connectionData{
		username: decodeSecretField(accountSecret.Data["username"]),
		password: decodeSecretField(accountSecret.Data["password"]),
		host:     NormalizeSecretHost(service.Name, namespace),
		port:     strconv.Itoa(port),
	}, nil
}

func (r *resolver) resolveRedisConnectionData(
	ctx context.Context,
	spec dbTypeSpec,
	namespace string,
	resourceName string,
) (*connectionData, error) {
	secretName := resourceName + "-redis-account-default"
	accountSecret, err := r.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			secretData, fallbackErr := r.readConnCredentialSecret(ctx, namespace, resourceName)
			if fallbackErr != nil {
				return nil, fallbackErr
			}
			return genericResolveFromSecret(spec, secretData, namespace), nil
		}
		return nil, fmt.Errorf("read redis account secret: %w", err)
	}

	service, port, err := r.findServiceByPort(ctx, namespace, resourceName, 6379)
	if err != nil {
		return nil, err
	}

	return &connectionData{
		username: decodeSecretField(accountSecret.Data["username"]),
		password: decodeSecretField(accountSecret.Data["password"]),
		host:     NormalizeSecretHost(service.Name, namespace),
		port:     strconv.Itoa(port),
	}, nil
}

func (r *resolver) resolveClickHouseConnectionData(
	ctx context.Context,
	spec dbTypeSpec,
	namespace string,
	resourceName string,
	secretData map[string][]byte,
) (*connectionData, error) {
	if _, ok := secretData["admin-password"]; !ok {
		return genericResolveFromSecret(spec, secretData, namespace), nil
	}

	username := decodeSecretField(secretData["username"])
	password := decodeSecretField(secretData["admin-password"])
	tcpEndpoint := decodeSecretField(secretData["tcpEndpoint"])

	if tcpEndpoint != "" {
		host, port, err := parseEndpoint(tcpEndpoint)
		if err != nil {
			return nil, err
		}
		if port == "" {
			port = "9000"
		}
		if host != "" {
			if _, err := strconv.Atoi(port); err == nil {
				return &connectionData{
					username: username,
					password: password,
					host:     NormalizeSecretHost(host, namespace),
					port:     port,
				}, nil
			}
		}
	}

	service, port, err := r.findServiceByPort(ctx, namespace, resourceName, 9000)
	if err != nil {
		return nil, err
	}
	return &connectionData{
		username: username,
		password: password,
		host:     NormalizeSecretHost(service.Name, namespace),
		port:     strconv.Itoa(port),
	}, nil
}

func (r *resolver) findServiceByPort(ctx context.Context, namespace, resourceName string, expectedPort int32) (*corev1.Service, int, error) {
	return r.findServiceByPortWithPreferredRole(ctx, namespace, resourceName, expectedPort, "")
}

func (r *resolver) findPrimaryServiceByPort(ctx context.Context, namespace, resourceName string, expectedPort int32) (*corev1.Service, int, error) {
	return r.findServiceByPortWithPreferredRole(ctx, namespace, resourceName, expectedPort, "primary")
}

func (r *resolver) findServiceByPortWithPreferredRole(ctx context.Context, namespace, resourceName string, expectedPort int32, preferredRole string) (*corev1.Service, int, error) {
	services, err := r.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=" + resourceName,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list services: %w", err)
	}

	var fallback *corev1.Service
	var fallbackPort int
	for _, service := range services.Items {
		if service.Spec.ClusterIP == "None" {
			continue
		}
		for _, port := range service.Spec.Ports {
			if port.Port == expectedPort {
				if preferredRole != "" && service.Labels["kubeblocks.io/role"] == preferredRole {
					matchedService := service
					return &matchedService, int(port.Port), nil
				}
				if fallback == nil {
					fallbackService := service
					fallback = &fallbackService
					fallbackPort = int(port.Port)
				}
			}
		}
	}

	if fallback != nil {
		return fallback, fallbackPort, nil
	}

	return nil, 0, errors.New("secret missing required fields")
}

func parseEndpoint(endpoint string) (string, string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", "", errors.New("secret missing required fields")
	}

	if strings.Contains(endpoint, "://") {
		parts := strings.SplitN(endpoint, "://", 2)
		endpoint = parts[1]
	}

	host, port, ok := strings.Cut(endpoint, ":")
	if !ok {
		return endpoint, "", nil
	}

	return host, port, nil
}

func currentUserName(raw string) string {
	var cfg kubeconfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return ""
	}
	for _, ctx := range cfg.Contexts {
		if ctx.Name == cfg.CurrentContext {
			return strings.TrimSpace(ctx.Context.User)
		}
	}
	if len(cfg.Contexts) > 0 {
		return strings.TrimSpace(cfg.Contexts[0].Context.User)
	}
	if len(cfg.Users) > 0 {
		return strings.TrimSpace(cfg.Users[0].Name)
	}
	return ""
}

func clientConfigFromKubeconfig(raw string) (*rest.Config, error) {
	config, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	cluster := currentCluster(config)
	if cluster != nil && !env.IsDevelopment {
		cluster.Server = effectiveAPIServer()
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}
	return restConfig, nil
}

func currentCluster(config *api.Config) *api.Cluster {
	if config == nil {
		return nil
	}
	if ctx, ok := config.Contexts[config.CurrentContext]; ok {
		return config.Clusters[ctx.Cluster]
	}
	if len(config.Contexts) > 0 {
		for _, ctx := range config.Contexts {
			return config.Clusters[ctx.Cluster]
		}
	}
	return nil
}

func effectiveAPIServer() string {
	host := strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_HOST"))
	port := strings.TrimSpace(os.Getenv("KUBERNETES_SERVICE_PORT"))
	if host != "" && port != "" {
		return "https://" + host + ":" + port
	}
	return "https://apiserver.cluster.local:6443"
}
