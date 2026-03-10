package blockscout

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/ethera-labs/local-testnet/configs"
)

//go:embed nginx.conf.tmpl
var nginxTemplateFS embed.FS

const nginxTemplate = "nginx.conf.tmpl"

type nginxTemplateData struct {
	BackendService  string
	FrontendService string
	BackendPort     int
	FrontendPort    int
}

func generateNginxConfigs(networksDir string, rollupConfigs []RollupConfig) error {
	tmplContent, err := nginxTemplateFS.ReadFile(nginxTemplate)
	if err != nil {
		return fmt.Errorf("failed to read embedded %s: %w", nginxTemplate, err)
	}

	tmpl, err := template.New("nginx").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse nginx template: %w", err)
	}

	for _, config := range rollupConfigs {
		var suffix string
		switch config.Name {
		case configs.L2ChainNameRollupA:
			suffix = "a"
		case configs.L2ChainNameRollupB:
			suffix = "b"
		default:
			return fmt.Errorf("unknown rollup name: %s", config.Name)
		}

		rollupDir := filepath.Join(networksDir, string(config.Name))
		if err := os.MkdirAll(rollupDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", rollupDir, err)
		}

		data := nginxTemplateData{
			BackendService:  fmt.Sprintf("%s-%s", backendServiceName, suffix),
			FrontendService: fmt.Sprintf("%s-%s", frontendServiceName, suffix),
			BackendPort:     backendPort,
			FrontendPort:    frontendPort,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template for %s: %w", config.Name, err)
		}

		nginxConfPath := filepath.Join(rollupDir, "blockscout-nginx.conf")
		if err := os.WriteFile(nginxConfPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", nginxConfPath, err)
		}
	}

	return nil
}
