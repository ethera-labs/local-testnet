package l1resolver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l1"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const (
	bundledHoodiELURL   = "http://141.95.35.120:31061"
	bundledHoodiCLURL   = "http://141.95.35.120:31125"
	bundledHoodiChainID = 560048

	dockerHostName = "host.docker.internal"
)

type localBootstrapper interface {
	EnsureRunning(context.Context) error
	ResolvePublicEndpoints(context.Context) (l1.Endpoints, error)
}

type Resolver struct {
	bootstrapper   localBootstrapper
	executionProbe func(context.Context, string) bool
	consensusProbe func(context.Context, string) bool
}

type kurtosisBootstrapper struct{}

func New() *Resolver {
	return &Resolver{
		bootstrapper:   kurtosisBootstrapper{},
		executionProbe: executionReachable,
		consensusProbe: consensusReachable,
	}
}

func (kurtosisBootstrapper) EnsureRunning(ctx context.Context) error {
	return l1.EnsureRunning(ctx)
}

func (kurtosisBootstrapper) ResolvePublicEndpoints(ctx context.Context) (l1.Endpoints, error) {
	return l1.ResolvePublicEndpoints(ctx)
}

// Resolve replaces the stale bundled hoodi RPC with a live local Kurtosis L1 when needed.
func (r *Resolver) Resolve(ctx context.Context, cfg configs.L2) (configs.L2, error) {
	if !shouldUseBundledHoodiLocalFallback(cfg) {
		return cfg, nil
	}

	executionOK := r.executionProbe(ctx, cfg.L1ElURL)
	consensusOK := r.consensusProbe(ctx, cfg.L1ClURL)
	if executionOK && consensusOK {
		return cfg, nil
	}

	log := logger.Named("l1_resolver")
	log.Warn(
		"bundled hoodi L1 endpoint is unavailable; switching to local Kurtosis L1",
		"l1_el_url", cfg.L1ElURL,
		"l1_cl_url", cfg.L1ClURL,
		"execution_reachable", executionOK,
		"consensus_reachable", consensusOK,
	)

	if err := r.bootstrapper.EnsureRunning(ctx); err != nil {
		return cfg, fmt.Errorf("failed to start local L1 fallback: %w", err)
	}

	endpoints, err := r.bootstrapper.ResolvePublicEndpoints(ctx)
	if err != nil {
		return cfg, fmt.Errorf("failed to discover local L1 fallback endpoints: %w", err)
	}

	cfg.L1ElURL = endpoints.ExecutionURL
	cfg.L1ClURL = endpoints.ConsensusURL
	cfg.L1ChainID = endpoints.ChainID

	log.Info(
		"using local Kurtosis L1 fallback",
		"l1_el_url", cfg.L1ElURL,
		"l1_cl_url", cfg.L1ClURL,
		"l1_chain_id", cfg.L1ChainID,
	)

	return cfg, nil
}

func shouldUseBundledHoodiLocalFallback(cfg configs.L2) bool {
	return strings.TrimSpace(cfg.L1ElURL) == bundledHoodiELURL &&
		strings.TrimSpace(cfg.L1ClURL) == bundledHoodiCLURL &&
		cfg.L1ChainID == bundledHoodiChainID
}

func executionReachable(ctx context.Context, rawURL string) bool {
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(probeCtx, rawURL)
	if err != nil {
		return false
	}
	defer client.Close()

	_, err = client.ChainID(probeCtx)
	return err == nil
}

func consensusReachable(ctx context.Context, rawURL string) bool {
	specURL, err := beaconSpecURL(rawURL)
	if err != nil {
		return false
	}

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, specURL, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func beaconSpecURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsed.Path = "/eth/v1/config/spec"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

// DockerAccessibleURL rewrites loopback URLs so Docker containers can reach services running on the host.
func DockerAccessibleURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	hostname := parsed.Hostname()
	if !isLoopbackHostname(hostname) {
		return rawURL
	}

	port := parsed.Port()
	if port == "" {
		parsed.Host = dockerHostName
		return parsed.String()
	}

	parsed.Host = net.JoinHostPort(dockerHostName, port)
	return parsed.String()
}

func isLoopbackHostname(host string) bool {
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
