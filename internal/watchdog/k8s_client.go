package watchdog

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// Standard Kubernetes service account paths
	serviceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	tokenPath          = serviceAccountPath + "/token"
	caCertPath         = serviceAccountPath + "/ca.crt"
	namespacePath      = serviceAccountPath + "/namespace"

	// httpClientTimeout is the timeout for HTTP requests to the Kubernetes API.
	// 30 seconds is sufficient for pod deletion and event creation operations.
	// This matches the default timeout used by client-go.
	httpClientTimeout = 30 * time.Second

	// maxResponseBodySize limits response body reads to prevent memory exhaustion
	// from unexpectedly large API error responses (1MB should be more than sufficient
	// for any Kubernetes API response).
	maxResponseBodySize = 1 << 20 // 1MB
)

// K8sClient provides an abstraction for Kubernetes API interactions.
// It handles authentication, pod deletion, and event creation using the
// standard library HTTP client (no client-go dependency).
type K8sClient struct {
	httpClient   *http.Client
	apiServerURL string
	token        string
	namespace    string
	logger       *slog.Logger
}

// IsInCluster returns true if running inside a Kubernetes cluster.
// It checks for the presence of the service account token file.
func IsInCluster() bool {
	_, err := os.Stat(tokenPath)
	return err == nil
}

// LoadInClusterConfig reads the service account token, CA cert, and namespace
// from the standard Kubernetes paths.
func LoadInClusterConfig() (token string, caCert []byte, namespace string, err error) {
	// Read token
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", nil, "", fmt.Errorf("reading token: %w", err)
	}
	token = strings.TrimSpace(string(tokenBytes))

	// Read CA certificate
	caCert, err = os.ReadFile(caCertPath)
	if err != nil {
		return "", nil, "", fmt.Errorf("reading CA cert: %w", err)
	}

	// Read namespace
	namespaceBytes, err := os.ReadFile(namespacePath)
	if err != nil {
		return "", nil, "", fmt.Errorf("reading namespace: %w", err)
	}
	namespace = strings.TrimSpace(string(namespaceBytes))

	return token, caCert, namespace, nil
}

// NewK8sClient creates a new Kubernetes API client using in-cluster authentication.
// Returns an error if not running in a Kubernetes cluster or if credentials cannot be loaded.
func NewK8sClient(logger *slog.Logger) (*K8sClient, error) {
	if !IsInCluster() {
		return nil, fmt.Errorf("not running in kubernetes cluster")
	}

	// Load in-cluster config
	token, caCert, namespace, err := LoadInClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("loading in-cluster config: %w", err)
	}

	// Build API server URL from environment variables
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("KUBERNETES_SERVICE_HOST or KUBERNETES_SERVICE_PORT not set")
	}
	apiServerURL := fmt.Sprintf("https://%s:%s", host, port)

	// Create HTTP client with TLS
	httpClient, err := createHTTPClient(caCert)
	if err != nil {
		return nil, fmt.Errorf("creating http client: %w", err)
	}

	return &K8sClient{
		httpClient:   httpClient,
		apiServerURL: apiServerURL,
		token:        token,
		namespace:    namespace,
		logger:       logger,
	}, nil
}

// createHTTPClient creates an HTTP client configured with TLS using the cluster CA.
// Connection pooling is configured for efficient reuse of connections to the K8s API.
func createHTTPClient(caCert []byte) (*http.Client, error) {
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		// Connection pooling settings for efficient K8s API access
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   httpClientTimeout,
	}, nil
}

// DeletePod deletes the specified pod via the Kubernetes API.
// Returns nil on success (including 404/409 which are idempotent successes).
// Returns error for authentication failures or transient errors.
func (c *K8sClient) DeletePod(ctx context.Context, name string) error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", c.apiServerURL, c.namespace, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details (limited to prevent memory exhaustion)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		c.logger.Info("pod deletion initiated",
			"pod", name,
			"namespace", c.namespace,
			"status", resp.StatusCode)
		return nil

	case http.StatusNotFound:
		// Pod already deleted - idempotent success
		c.logger.Info("pod not found (already deleted)",
			"pod", name,
			"namespace", c.namespace)
		return nil

	case http.StatusConflict:
		// Pod already terminating - idempotent success
		c.logger.Info("pod already terminating",
			"pod", name,
			"namespace", c.namespace)
		return nil

	case http.StatusUnauthorized:
		return &PermanentError{Message: "unauthorized: invalid or expired token"}

	case http.StatusForbidden:
		return &PermanentError{Message: fmt.Sprintf("forbidden: missing RBAC permission to delete pods in namespace %s", c.namespace)}

	default:
		// Treat other errors as transient (retriable)
		return &TransientError{
			Message:    fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(body)),
			StatusCode: resp.StatusCode,
		}
	}
}

// IsPodTerminating checks if the specified pod has a deletionTimestamp set.
// Returns true if the pod is already being deleted.
func (c *K8sClient) IsPodTerminating(ctx context.Context, name string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", c.apiServerURL, c.namespace, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Pod doesn't exist - treat as already terminated
		return true, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		return false, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check deletionTimestamp
	var pod struct {
		Metadata struct {
			DeletionTimestamp *string `json:"deletionTimestamp"`
		} `json:"metadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return false, fmt.Errorf("decoding response: %w", err)
	}

	return pod.Metadata.DeletionTimestamp != nil, nil
}

// CreateEvent creates a Kubernetes Event resource for the specified pod.
func (c *K8sClient) CreateEvent(ctx context.Context, event *RestartEvent) error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/events", c.apiServerURL, c.namespace)

	now := time.Now().UTC().Format(time.RFC3339)
	eventName := fmt.Sprintf("mount-monitor.%d", time.Now().UnixNano())

	eventBody := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Event",
		"metadata": map[string]interface{}{
			"name":      eventName,
			"namespace": c.namespace,
		},
		"involvedObject": map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"name":       event.PodName,
			"namespace":  c.namespace,
		},
		"reason":         "WatchdogRestart",
		"message":        event.Reason,
		"type":           "Warning",
		"firstTimestamp": now,
		"lastTimestamp":  now,
		"count":          1,
		"source": map[string]interface{}{
			"component": "mount-monitor-watchdog",
		},
	}

	body, err := json.Marshal(eventBody)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	c.logger.Info("kubernetes event created",
		"event", eventName,
		"pod", event.PodName,
		"reason", "WatchdogRestart")

	return nil
}

// CanDeletePods validates that the service account has permission to delete pods.
// This uses the SelfSubjectAccessReview API.
func (c *K8sClient) CanDeletePods(ctx context.Context) (bool, error) {
	url := fmt.Sprintf("%s/apis/authorization.k8s.io/v1/selfsubjectaccessreviews", c.apiServerURL)

	reviewBody := map[string]interface{}{
		"apiVersion": "authorization.k8s.io/v1",
		"kind":       "SelfSubjectAccessReview",
		"spec": map[string]interface{}{
			"resourceAttributes": map[string]interface{}{
				"verb":      "delete",
				"resource":  "pods",
				"namespace": c.namespace,
			},
		},
	}

	body, err := json.Marshal(reviewBody)
	if err != nil {
		return false, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return false, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
		return false, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var review struct {
		Status struct {
			Allowed bool   `json:"allowed"`
			Reason  string `json:"reason"`
		} `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&review); err != nil {
		return false, fmt.Errorf("decoding response: %w", err)
	}

	return review.Status.Allowed, nil
}

// Namespace returns the Kubernetes namespace the client is configured for.
func (c *K8sClient) Namespace() string {
	return c.namespace
}

// PermanentError represents an error that should not be retried.
type PermanentError struct {
	Message string
}

func (e *PermanentError) Error() string {
	return e.Message
}

// IsPermanent returns true, indicating this error should not be retried.
func (e *PermanentError) IsPermanent() bool {
	return true
}

// TransientError represents an error that may succeed on retry.
type TransientError struct {
	Message    string
	StatusCode int
}

func (e *TransientError) Error() string {
	return e.Message
}

// IsTransient returns true, indicating this error may succeed on retry.
func (e *TransientError) IsTransient() bool {
	return true
}
