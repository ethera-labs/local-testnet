package docker

import "runtime"

// CanUseOpSuccinctPrebuiltBinary reports whether a host-built validity binary
// can be copied directly into the Linux runtime image. On macOS, cargo builds
// a Mach-O binary that cannot be executed inside the container.
func CanUseOpSuccinctPrebuiltBinary() bool {
	return runtime.GOOS == "linux"
}
