package docker

import (
	"strings"
	"testing"
)

func TestEmbeddedComposeIncludesConditionalSequencerKeyHandling(t *testing.T) {
	t.Parallel()

	content, err := getComposeFileContent()
	if err != nil {
		t.Fatalf("getComposeFileContent() error = %v", err)
	}

	for _, want := range []string{
		"dockerfile: ${ROOT_DIR}/internal/l2/infra/docker/publisher.Dockerfile",
		"ssh:",
		"- default",
		"SEQUENCER_KEY=$$(printf '%s' \"$${SEQUENCER_PRIVATE_KEY:-}\" | sed 's/^0x//')",
		"grep -q -- '--sequencer.key' \"$$GETH_HELP_FILE\"",
		"\"--sequencer.key=$${SEQUENCER_KEY}\"",
		"SEQUENCER_PRIVATE_KEY is required by this geth build",
		"L1_COMPOSE_NETWORK_NAME: \"${L1_COMPOSE_NETWORK_NAME}\"",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("embedded compose missing %q", want)
		}
	}
}
