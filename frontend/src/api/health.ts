import { HEALTH_API_URL } from '../config/chains'

// Keep in sync with the catalogue in cmd/localnet-health/main.go.
export type ServiceKind =
  | 'publisher'
  | 'op-reth'
  | 'op-node'
  | 'op-batcher'
  | 'op-proposer'
  | 'rollup-boost'
  | 'op-rbuilder'
  | 'sidecar'
  | 'op-alt-da'
  | 'op-succinct'
  | 'op-succinct-postgres'
  | 'frontend'

// "missing" = optional feature disabled (container absent); "down" =
// container exists but is not running / unhealthy.
export type ServiceStatus = 'up' | 'starting' | 'down' | 'missing'

export interface Service {
  id: string
  name: string
  kind: ServiceKind
  host_port?: number
  status: ServiceStatus
}

interface ServicesResponse {
  services: Service[]
}

export async function fetchServices(
  signal?: AbortSignal,
  baseUrl: string = HEALTH_API_URL,
): Promise<Service[]> {
  const response = await fetch(`${baseUrl}/api/services`, {
    method: 'GET',
    headers: { Accept: 'application/json' },
    signal,
  })
  if (!response.ok) {
    throw new Error(`health api ${response.status}`)
  }
  const data = (await response.json()) as ServicesResponse
  return data.services
}

export function indexById(services: Service[]): Record<string, Service> {
  const out: Record<string, Service> = {}
  for (const s of services) {
    out[s.id] = s
  }
  return out
}
