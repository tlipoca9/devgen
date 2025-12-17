package examples

// DNS1123Example demonstrates @dns1123_label annotation for DNS label name validation.
// DNS label is based on RFC 1123 and commonly used in Kubernetes and cloud-native applications.
// validategen:@validate
type DNS1123Example struct {
	// validategen:@required
	// validategen:@dns1123_label
	Hostname string

	// validategen:@required
	// validategen:@dns1123_label
	ServiceName string

	// validategen:@required
	// validategen:@dns1123_label
	PodName string
}

// KubernetesName demonstrates DNS label validation for Kubernetes naming conventions.
// validategen:@validate
type KubernetesName struct {
	// validategen:@required
	// validategen:@dns1123_label
	// Kubernetes namespace name
	Namespace string

	// validategen:@required
	// validategen:@dns1123_label
	// Kubernetes pod name
	Pod string

	// validategen:@required
	// validategen:@dns1123_label
	// Kubernetes service name
	Service string

	// validategen:@required
	// validategen:@dns1123_label
	// StatefulSet name
	StatefulSetName string
}

// CloudNativeService demonstrates DNS label validation for cloud-native service naming.
// validategen:@validate
type CloudNativeService struct {
	// validategen:@required
	// validategen:@dns1123_label
	// Service domain name (e.g., "api.service.cloud.example.com")
	DomainName string

	// validategen:@required
	// validategen:@dns1123_label
	// Service instance name (e.g., "api-service-01")
	InstanceName string

	// validategen:@required
	// validategen:@dns1123_label
	// Container registry hostname (e.g., "registry.example.com")
	RegistryHost string
}
