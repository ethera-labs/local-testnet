package deployer

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
)

type stubWriter struct {
	writes map[string][]byte
}

func (w *stubWriter) WriteJSON(path string, data any) error {
	return nil
}

func (w *stubWriter) WriteBytes(path string, data []byte) error {
	if w.writes == nil {
		w.writes = make(map[string][]byte)
	}
	w.writes[path] = append([]byte(nil), data...)
	return nil
}

func TestWriteIntentOmitsAltDAWhenSkipL1DeployIsTrue(t *testing.T) {
	t.Parallel()

	writer := &stubWriter{}
	stateDir := t.TempDir()
	intentWriter := NewIntentWriter(stateDir, writer)

	err := intentWriter.WriteIntent(
		"0x1111111111111111111111111111111111111111",
		"0x2222222222222222222222222222222222222222",
		560048,
		map[configs.L2ChainName]configs.Chain{
			configs.L2ChainNameRollupA: {ID: 77777},
		},
		configs.AltDAConfig{
			Enabled:          true,
			SkipL1Deploy:     true,
			DACommitmentType: configs.AltDACommitmentTypeKeccak,
		},
	)
	if err != nil {
		t.Fatalf("WriteIntent() error = %v", err)
	}

	intentPath := filepath.Join(stateDir, intentFileName)
	got := string(writer.writes[intentPath])

	for _, unwanted := range []string{
		"[chains.dangerousAltDAConfig]",
		"useAltDA = true",
		`daCommitmentType = "KeccakCommitment"`,
	} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("intent unexpectedly contains %q\n%s", unwanted, got)
		}
	}
}
