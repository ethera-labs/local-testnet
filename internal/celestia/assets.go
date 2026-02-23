package celestia

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const (
	composeTemplatePath       = "assets/docker-compose.yml.tmpl"
	opAltDATemplatePath       = "assets/op-alt-da.local.toml.tmpl"
	celeniumEnvTemplatePath   = "assets/celenium.local.env.tmpl"
	bootstrapCelestiaPath     = "assets/bootstrap-local-celestia.sh"
	bootstrapBridgePath       = "assets/bootstrap-bridge.sh"
	composeFileName           = "docker-compose.yml"
	opAltDAConfigFileName     = "op-alt-da.local.toml"
	celeniumEnvConfigFileName = "celenium.local.env"
)

//go:embed assets/docker-compose.yml.tmpl assets/op-alt-da.local.toml.tmpl assets/celenium.local.env.tmpl assets/bootstrap-local-celestia.sh assets/bootstrap-bridge.sh
var embeddedAssetsFS embed.FS

type runtimeLayout struct {
	RuntimeDir string
	ConfigsDir string
	ScriptsDir string
	DataDir    string
}

type composeTemplateData struct {
	ProjectName           string
	ChainID               string
	AttachToL2Network     bool
	CeleniumEnabled       bool
	Images                composeImages
	DataDir               string
	ConfigsDir            string
	ScriptsDir            string
	CeleniumIndexerPath   string
	CeleniumInterfacePath string
}

type composeImages struct {
	CelestiaApp  string
	CelestiaNode string
	OpAltDA      string
	CeleniumDB   string
}

type opAltDAConfigData struct {
	Namespace string
	ChainID   string
}

type celeniumEnvData struct {
	IndexerStartLevel uint64
	ChainID           string
}

func ensureRuntimeAssets(layout runtimeLayout, composeData composeTemplateData, opAltDAData opAltDAConfigData, celeniumData celeniumEnvData) (string, error) {
	if err := os.MkdirAll(layout.RuntimeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create runtime directory: %w", err)
	}
	if err := os.MkdirAll(layout.ConfigsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create configs directory: %w", err)
	}
	if err := os.MkdirAll(layout.ScriptsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create scripts directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(layout.DataDir, "celestia-app"), 0755); err != nil {
		return "", fmt.Errorf("failed to create Celestia app data directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(layout.DataDir, "celestia-node"), 0755); err != nil {
		return "", fmt.Errorf("failed to create Celestia node data directory: %w", err)
	}
	if composeData.CeleniumEnabled {
		if err := os.MkdirAll(filepath.Join(layout.DataDir, "celenium-db"), 0755); err != nil {
			return "", fmt.Errorf("failed to create Celenium DB data directory: %w", err)
		}
	}

	if err := writeAssetFile(bootstrapCelestiaPath, filepath.Join(layout.ScriptsDir, "bootstrap-local-celestia.sh"), 0755); err != nil {
		return "", err
	}
	if err := writeAssetFile(bootstrapBridgePath, filepath.Join(layout.ScriptsDir, "bootstrap-bridge.sh"), 0755); err != nil {
		return "", err
	}

	if err := renderTemplateToFile(opAltDATemplatePath, opAltDAData, filepath.Join(layout.ConfigsDir, opAltDAConfigFileName)); err != nil {
		return "", err
	}
	if err := renderTemplateToFile(celeniumEnvTemplatePath, celeniumData, filepath.Join(layout.ConfigsDir, celeniumEnvConfigFileName)); err != nil {
		return "", err
	}
	composePath := filepath.Join(layout.RuntimeDir, composeFileName)
	if err := renderTemplateToFile(composeTemplatePath, composeData, composePath); err != nil {
		return "", err
	}

	return composePath, nil
}

func writeAssetFile(assetPath, destination string, perm os.FileMode) error {
	content, err := embeddedAssetsFS.ReadFile(assetPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded asset %q: %w", assetPath, err)
	}
	if err := os.WriteFile(destination, content, perm); err != nil {
		return fmt.Errorf("failed to write asset %q: %w", destination, err)
	}
	return nil
}

func renderTemplateToFile(assetPath string, data any, destination string) error {
	rendered, err := renderTemplate(assetPath, data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destination, rendered, 0644); err != nil {
		return fmt.Errorf("failed to write rendered file %q: %w", destination, err)
	}
	return nil
}

func renderTemplate(assetPath string, data any) ([]byte, error) {
	tmplRaw, err := embeddedAssetsFS.ReadFile(assetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded template %q: %w", assetPath, err)
	}

	tmpl, err := template.New(filepath.Base(assetPath)).Parse(string(tmplRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", assetPath, err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("failed to render template %q: %w", assetPath, err)
	}

	return out.Bytes(), nil
}
