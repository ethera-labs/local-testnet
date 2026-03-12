package l2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
)

const (
	localDockerHostAlias        = "host.docker.internal"
	kurtosisELRPCContainerPort  = "8545/tcp"
	kurtosisCLHTTPContainerPort = "4000/tcp"
	l1BlockProgressTimeout      = 15 * time.Second
)

var (
	kurtosisELContainerPrefixes = []string{
		"el-1-geth-lighthouse--",
		"el-2-geth-lighthouse--",
	}
	kurtosisCLContainerPrefixes = []string{
		"cl-1-lighthouse-geth--",
		"cl-2-lighthouse-geth--",
	}
)

func resolveManagedL1Endpoints(ctx context.Context, cfg configs.L2, logger *slog.Logger) (configs.L2, error) {
	if !isManagedLocalURL(cfg.L1ElURL) && !isManagedLocalURL(cfg.L1ClURL) {
		return cfg, nil
	}

	dockerClient, err := docker.New()
	if err != nil {
		return cfg, fmt.Errorf("failed to create docker client for L1 endpoint resolution: %w", err)
	}
	defer dockerClient.Close()

	resolved := cfg

	elURL, elChanged, err := resolveManagedLocalURL(
		ctx,
		dockerClient,
		cfg.L1ElURL,
		kurtosisELContainerPrefixes,
		kurtosisELRPCContainerPort,
		probeELRPC,
	)
	if err != nil {
		return cfg, fmt.Errorf("failed to resolve L1 execution RPC URL %q: %w", cfg.L1ElURL, err)
	}
	if elChanged {
		logger.With("previous", cfg.L1ElURL, "resolved", elURL).Info("resolved live Kurtosis L1 execution RPC URL")
	}
	resolved.L1ElURL = elURL

	if err := ensureManagedL1BlockProgress(ctx, resolved.L1ElURL); err != nil {
		return cfg, err
	}

	clURL, clChanged, err := resolveManagedLocalURL(
		ctx,
		dockerClient,
		cfg.L1ClURL,
		kurtosisCLContainerPrefixes,
		kurtosisCLHTTPContainerPort,
		probeBeaconRPC,
	)
	if err != nil {
		logger.With("configured_url", cfg.L1ClURL, "error", err).Warn("could not refresh local L1 beacon URL; keeping configured value")
		return resolved, nil
	}
	if clChanged {
		logger.With("previous", cfg.L1ClURL, "resolved", clURL).Info("resolved live Kurtosis L1 beacon URL")
	}
	resolved.L1ClURL = clURL

	return resolved, nil
}

func resolveManagedLocalURL(
	ctx context.Context,
	dockerClient *docker.Client,
	rawURL string,
	containerPrefixes []string,
	containerPort string,
	probe func(string) error,
) (string, bool, error) {
	if !isManagedLocalURL(rawURL) {
		return rawURL, false, nil
	}
	if err := probe(rawURL); err == nil {
		return rawURL, false, nil
	}

	containerName, err := dockerClient.FindRunningContainerByPrefixes(ctx, containerPrefixes...)
	if err != nil {
		return "", false, err
	}
	hostPort, err := dockerClient.ContainerPublishedHostPort(ctx, containerName, containerPort)
	if err != nil {
		return "", false, err
	}

	resolvedURL, err := withManagedDockerHost(rawURL, hostPort)
	if err != nil {
		return "", false, err
	}
	if err := probe(resolvedURL); err != nil {
		return "", false, fmt.Errorf("resolved URL %q failed probe: %w", resolvedURL, err)
	}

	return resolvedURL, true, nil
}

func ensureManagedL1BlockProgress(ctx context.Context, rawURL string) error {
	if !isManagedLocalURL(rawURL) {
		return nil
	}

	startBlock, err := latestBlockNumber(rawURL)
	if err != nil {
		return fmt.Errorf("failed to query current L1 block number from %q: %w", rawURL, err)
	}

	timer := time.NewTimer(l1BlockProgressTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	endBlock, err := latestBlockNumber(rawURL)
	if err != nil {
		return fmt.Errorf("failed to query later L1 block number from %q: %w", rawURL, err)
	}
	if endBlock <= startBlock {
		return fmt.Errorf("local L1 is not producing blocks on %q (stayed at block %d for %s)", rawURL, startBlock, l1BlockProgressTimeout)
	}

	return nil
}

func latestBlockNumber(rawURL string) (uint64, error) {
	resp, err := doRPCRequest(rawURL, "eth_blockNumber")
	if err != nil {
		return 0, err
	}

	var result struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to decode block-number response: %w", err)
	}
	if result.Error != nil {
		return 0, fmt.Errorf("rpc error: %s", result.Error.Message)
	}
	if strings.TrimSpace(result.Result) == "" {
		return 0, fmt.Errorf("rpc response missing block number")
	}

	return strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
}

func probeELRPC(rawURL string) error {
	_, err := doRPCRequest(rawURL, "eth_chainId")
	return err
}

func probeBeaconRPC(rawURL string) error {
	requestURL, err := probeURL(rawURL, "/eth/v1/node/health")
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("beacon health endpoint returned 404")
	}

	return nil
}

func doRPCRequest(rawURL, method string) ([]byte, error) {
	requestURL, err := probeURL(rawURL, "")
	if err != nil {
		return nil, err
	}

	body := []byte(fmt.Sprintf(`{"jsonrpc":"2.0","method":"%s","params":[],"id":1}`, method))
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected http status %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func probeURL(rawURL, pathSuffix string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	if parsed.Hostname() == localDockerHostAlias {
		if port := parsed.Port(); port != "" {
			parsed.Host = net.JoinHostPort("127.0.0.1", port)
		} else {
			parsed.Host = "127.0.0.1"
		}
	}
	if pathSuffix != "" {
		parsed.Path = pathSuffix
	}

	return parsed.String(), nil
}

func withManagedDockerHost(rawURL, hostPort string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	parsed.Host = net.JoinHostPort(localDockerHostAlias, hostPort)
	return parsed.String(), nil
}

func isManagedLocalURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	switch parsed.Hostname() {
	case localDockerHostAlias, "127.0.0.1", "localhost":
		return true
	default:
		return false
	}
}
