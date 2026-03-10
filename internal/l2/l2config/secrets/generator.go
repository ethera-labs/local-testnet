package secrets

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const (
	jwtSecretLength  = 32 // 32 bytes = 64 hex characters
	JWTFileName      = "jwt.txt"
	passwordFileName = "password.txt"
)

// Generator generates secrets for L2 chains
type Generator struct {
	writer filesystem.Writer
	logger *slog.Logger
}

// NewGenerator creates a new secrets generator
func NewGenerator(writer filesystem.Writer) *Generator {
	return &Generator{
		writer: writer,
		logger: logger.Named("jwt_generator"),
	}
}

// GenerateJWT generates a random JWT secret
func (g *Generator) GenerateJWT(path string) error {
	g.logger.Info("generating JWT secret")

	secret := make([]byte, jwtSecretLength)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Return as hex string without 0x prefix
	hexSecret := []byte(hex.EncodeToString(secret))

	jwtPath := filepath.Join(path, JWTFileName)

	g.logger.
		With("file_path", jwtPath).
		Info("JWT secret generated. Writing file")
	if err := g.writer.WriteBytes(jwtPath, hexSecret); err != nil {
		return fmt.Errorf("failed to write '%s': %w", JWTFileName, err)
	}

	return nil
}

func (g *Generator) GeneratePassword(path string) error {
	randomPassword := []byte("")

	passwordPath := filepath.Join(path, passwordFileName)
	if err := g.writer.WriteBytes(passwordPath, randomPassword); err != nil {
		return fmt.Errorf("failed to write '%s': %w", passwordFileName, err)
	}

	return nil
}
