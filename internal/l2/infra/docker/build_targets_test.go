package docker

import (
	"reflect"
	"testing"
)

func TestNormalizeComposeBuildServicesCollapsesSharedImagePairs(t *testing.T) {
	t.Parallel()

	services := []string{
		"publisher",
		"op-alt-da-a",
		"op-alt-da-b",
		"op-succinct-a",
		"op-succinct-b",
		"op-rbuilder-a",
		"op-rbuilder-b",
		"sidecar-a",
		"sidecar-b",
	}

	got := normalizeComposeBuildServices(services)
	want := []string{
		"publisher",
		"op-alt-da-a",
		"op-succinct-a",
		"op-rbuilder-a",
		"sidecar-a",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeComposeBuildServices() = %v, want %v", got, want)
	}
}

func TestNormalizeComposeBuildServicesMapsBOnlyRequestsToSharedBuilder(t *testing.T) {
	t.Parallel()

	services := []string{
		"op-alt-da-b",
		"op-succinct-b",
		"op-rbuilder-b",
		"sidecar-b",
	}

	got := normalizeComposeBuildServices(services)
	want := []string{
		"op-alt-da-a",
		"op-succinct-a",
		"op-rbuilder-a",
		"sidecar-a",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeComposeBuildServices() = %v, want %v", got, want)
	}
}

func TestComposeBuildArgsOnlyAddsParallelForMultipleUniqueTargets(t *testing.T) {
	t.Parallel()

	many := composeBuildArgs([]string{"publisher", "op-rbuilder-a", "op-rbuilder-b"})
	if want := []string{"build", "--parallel", "publisher", "op-rbuilder-a"}; !reflect.DeepEqual(many, want) {
		t.Fatalf("composeBuildArgs(many) = %v, want %v", many, want)
	}

	one := composeBuildArgs([]string{"op-alt-da-b"})
	if want := []string{"build", "op-alt-da-a"}; !reflect.DeepEqual(one, want) {
		t.Fatalf("composeBuildArgs(one) = %v, want %v", one, want)
	}
}
