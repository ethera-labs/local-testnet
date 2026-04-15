package docker

// Shared-image service pairs use the same Docker build definition and image tag.
// Build one representative target to avoid racing docker compose --parallel
// against the same local image tag.
var sharedImageBuildTargets = map[string]string{
	"op-geth-b":     "op-geth-a",
	"op-alt-da-b":   "op-alt-da-a",
	"op-succinct-b": "op-succinct-a",
	"op-rbuilder-b": "op-rbuilder-a",
	"sidecar-b":     "sidecar-a",
}

func composeBuildArgs(services []string) []string {
	buildServices := normalizeComposeBuildServices(services)

	args := []string{"build"}
	if len(buildServices) > 1 {
		args = append(args, "--parallel")
	}

	return append(args, buildServices...)
}

func normalizeComposeBuildServices(services []string) []string {
	seen := make(map[string]struct{}, len(services))
	buildServices := make([]string, 0, len(services))

	for _, service := range services {
		if canonical, ok := sharedImageBuildTargets[service]; ok {
			service = canonical
		}
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		buildServices = append(buildServices, service)
	}

	return buildServices
}
