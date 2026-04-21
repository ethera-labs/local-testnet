package l1

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	coreenclaves "github.com/kurtosis-tech/kurtosis/api/golang/core/lib/enclaves"
	"github.com/kurtosis-tech/kurtosis/api/golang/engine/lib/kurtosis_context"
)

const LocalChainID = 3151908

type Endpoints struct {
	ExecutionURL string
	ConsensusURL string
	ChainID      int
}

type serviceSnapshot struct {
	name        string
	publicIP    string
	publicPorts map[string]uint16
}

// EnsureRunning starts the local Kurtosis L1 if it is not already running.
func EnsureRunning(ctx context.Context) error {
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		return fmt.Errorf("failed to create Kurtosis context: %w", err)
	}

	running, err := hasRunningEnclave(ctx, kurtosisCtx, enclaveName)
	if err != nil {
		return err
	}
	if running {
		if _, err := resolvePublicEndpoints(ctx, kurtosisCtx); err == nil {
			return nil
		}

		slog.Warn("local L1 enclave exists but is not usable, recreating it", "enclave", enclaveName)
		if err := kurtosisCtx.DestroyEnclave(ctx, enclaveName); err != nil {
			return fmt.Errorf("failed to destroy unusable local L1 enclave %q: %w", enclaveName, err)
		}
	}

	return start(ctx)
}

// ResolvePublicEndpoints discovers the host-accessible EL/CL endpoints for the local Kurtosis L1.
func ResolvePublicEndpoints(ctx context.Context) (Endpoints, error) {
	kurtosisCtx, err := kurtosis_context.NewKurtosisContextFromLocalEngine()
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to create Kurtosis context: %w", err)
	}

	return resolvePublicEndpoints(ctx, kurtosisCtx)
}

func resolvePublicEndpoints(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext) (Endpoints, error) {
	running, err := hasRunningEnclave(ctx, kurtosisCtx, enclaveName)
	if err != nil {
		return Endpoints{}, err
	}
	if !running {
		return Endpoints{}, fmt.Errorf("local L1 enclave %q is not running", enclaveName)
	}

	enclaveCtx, err := kurtosisCtx.GetEnclaveContext(ctx, enclaveName)
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to get Kurtosis enclave context: %w", err)
	}

	snapshots, err := getServiceSnapshots(enclaveCtx)
	if err != nil {
		return Endpoints{}, err
	}

	executionURL, err := selectPublicURL(snapshots, 8545, []string{"execution", "geth"})
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to resolve L1 execution RPC endpoint: %w", err)
	}

	consensusURL, err := selectPublicURL(snapshots, 5052, []string{"beacon", "lighthouse", "consensus"})
	if err != nil {
		return Endpoints{}, fmt.Errorf("failed to resolve L1 consensus REST endpoint: %w", err)
	}

	chainID, err := waitForChainID(ctx, executionURL)
	if err != nil {
		return Endpoints{}, err
	}

	return Endpoints{
		ExecutionURL: executionURL,
		ConsensusURL: consensusURL,
		ChainID:      chainID,
	}, nil
}

func hasRunningEnclave(ctx context.Context, kurtosisCtx *kurtosis_context.KurtosisContext, name string) (bool, error) {
	enclaves, err := kurtosisCtx.GetEnclaves(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list Kurtosis enclaves: %w", err)
	}

	return len(enclaves.GetEnclavesByName()[name]) > 0, nil
}

func getServiceSnapshots(enclaveCtx *coreenclaves.EnclaveContext) ([]serviceSnapshot, error) {
	serviceIDs, err := enclaveCtx.GetServices()
	if err != nil {
		return nil, fmt.Errorf("failed to list services in local L1 enclave: %w", err)
	}

	serviceIdentifiers := make(map[string]bool, len(serviceIDs))
	for serviceName := range serviceIDs {
		serviceIdentifiers[string(serviceName)] = true
	}

	serviceContexts, err := enclaveCtx.GetServiceContexts(serviceIdentifiers)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect services in local L1 enclave: %w", err)
	}

	snapshots := make([]serviceSnapshot, 0, len(serviceContexts))
	for serviceName, serviceCtx := range serviceContexts {
		publicPorts := make(map[string]uint16, len(serviceCtx.GetPublicPorts()))
		for portName, spec := range serviceCtx.GetPublicPorts() {
			publicPorts[portName] = spec.GetNumber()
		}

		snapshots = append(snapshots, serviceSnapshot{
			name:        string(serviceName),
			publicIP:    normalizePublicHost(serviceCtx.GetMaybePublicIPAddress()),
			publicPorts: publicPorts,
		})
	}

	return snapshots, nil
}

func normalizePublicHost(host string) string {
	switch host {
	case "":
		return ""
	case "0.0.0.0":
		return "127.0.0.1"
	default:
		return host
	}
}

func selectPublicURL(services []serviceSnapshot, portNumber uint16, preferredTokens []string) (string, error) {
	type candidate struct {
		service serviceSnapshot
		score   int
	}

	candidates := make([]candidate, 0)
	for _, service := range services {
		if service.publicIP == "" {
			continue
		}

		for _, publicPort := range service.publicPorts {
			if publicPort != portNumber {
				continue
			}

			candidates = append(candidates, candidate{
				service: service,
				score:   scoreServiceName(service.name, preferredTokens),
			})
			break
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no public service exposes port %d", portNumber)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].service.name < candidates[j].service.name
	})

	return buildHTTPURL(candidates[0].service.publicIP, portNumber), nil
}

func scoreServiceName(name string, preferredTokens []string) int {
	lowerName := strings.ToLower(name)
	score := 0
	for idx, token := range preferredTokens {
		if strings.Contains(lowerName, token) {
			score += len(preferredTokens) - idx
		}
	}
	return score
}

func buildHTTPURL(host string, port uint16) string {
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, fmt.Sprintf("%d", port)),
	}
	return u.String()
}

func waitForChainID(ctx context.Context, executionURL string) (int, error) {
	deadline := time.Now().Add(2 * time.Minute)

	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		client, err := ethclient.DialContext(attemptCtx, executionURL)
		if err == nil {
			chainID, chainErr := client.ChainID(attemptCtx)
			client.Close()
			cancel()
			if chainErr == nil {
				return int(chainID.Int64()), nil
			}
		} else {
			cancel()
		}

		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timed out waiting for local L1 execution RPC at %s", executionURL)
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}
