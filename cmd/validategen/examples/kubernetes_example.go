package examples

// KubernetesResourceSpec demonstrates CPU and memory validation for Kubernetes resources.
// Example usage:
//
//	spec := KubernetesResourceSpec{
//		CPURequest:    "500m",    // Valid: 500 millicores
//		CPULimit:      "1000m",   // Valid: 1000 millicores
//		MemoryRequest: "512Mi",   // Valid: 512 megabytes
//		MemoryLimit:   "2Gi",     // Valid: 2 gigabytes
//	}
//	if err := spec.Validate(); err != nil {
//		log.Fatal(err)
//	}
//
// The @cpu and @memory annotations validate Kubernetes resource quantities:
// - @cpu validates CPU quantities (valid units: m, cores, or scientific notation)
// - @memory validates memory quantities (valid units: Ki, Mi, Gi, Ti, Pi, Ei, or bytes)
// validategen:@validate
type KubernetesResourceSpec struct {
	// validategen:@required
	// validategen:@cpu
	CPURequest string

	// validategen:@cpu
	CPULimit string

	// validategen:@required
	// validategen:@memory
	MemoryRequest string

	// validategen:@memory
	MemoryLimit string
}

// PodContainerSpec demonstrates a complete container specification with resource validation.
// Example usage:
//
//	container := PodContainerSpec{
//		Name:              "nginx",
//		Image:             "nginx:latest",
//		CPURequest:        "100m",    // Required: 100 millicores
//		MemoryRequest:     "128Mi",   // Required: 128 megabytes
//		CPULimit:          "500m",    // Optional: 500 millicores
//		MemoryLimit:       "512Mi",   // Optional: 512 megabytes
//	}
//	if err := container.Validate(); err != nil {
//		log.Fatal(err)
//	}
//
// validategen:@validate
type PodContainerSpec struct {
	// validategen:@required
	// validategen:@min(1)
	Name string

	// validategen:@required
	Image string

	// Container resource requests (required)
	// validategen:@required
	// validategen:@cpu
	CPURequest string

	// validategen:@required
	// validategen:@memory
	MemoryRequest string

	// Container resource limits (optional but validated if provided)
	// validategen:@cpu
	CPULimit string

	// validategen:@memory
	MemoryLimit string
}
