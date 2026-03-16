import { SIDECAR_A_URL } from '../config/chains'

export interface XTSubmitRequest {
  transactions: Record<string, string[]> // chainId -> raw tx hex list
}

export interface XTResponse {
  instance_id: string
  status: string
}

export interface XTStatus {
  instance_id: string
  status: string
  decision?: boolean
}

export async function submitXT(
  transactions: Record<number, string[]>
): Promise<XTResponse> {
  const txs: Record<string, string[]> = {}
  for (const [chainId, txList] of Object.entries(transactions)) {
    txs[chainId] = txList
  }

  const response = await fetch(`${SIDECAR_A_URL}/xt`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ transactions: txs }),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Failed to submit XT: ${response.status} ${text}`)
  }

  return response.json()
}

export async function getXTStatus(instanceId: string): Promise<XTStatus> {
  const response = await fetch(`${SIDECAR_A_URL}/xt/${instanceId}`)

  if (!response.ok) {
    throw new Error(`Failed to get XT status: ${response.status}`)
  }

  return response.json()
}

export async function waitForDecision(
  instanceId: string,
  timeoutMs: number = 30000
): Promise<boolean> {
  const deadline = Date.now() + timeoutMs
  const pollInterval = 100

  while (Date.now() < deadline) {
    try {
      const status = await getXTStatus(instanceId)

      if (status.decision !== undefined) {
        return status.decision
      }

      if (status.status === 'committed') {
        return true
      }
      if (status.status === 'aborted') {
        return false
      }
    } catch {
      // Ignore errors and retry
    }

    await new Promise((resolve) => setTimeout(resolve, pollInterval))
  }

  throw new Error('Timeout waiting for decision')
}

export async function checkHealth(): Promise<boolean> {
  try {
    const response = await fetch(`${SIDECAR_A_URL}/health`)
    return response.ok
  } catch {
    return false
  }
}
