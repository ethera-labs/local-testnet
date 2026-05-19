import { create } from 'zustand'
import type { Service, ServiceStatus } from '../api/health'

export type TransactionStatus =
  | 'pending'
  | 'simulating'
  | 'waiting_circ'
  | 'voted'
  | 'committed'
  | 'aborted'

export type FlowStep =
  | 'idle'
  | 'submitting'
  | 'minting_a'
  | 'minting_b'
  | 'minting_both'
  | 'forward_to_peer'
  | 'lock_builder_a'
  | 'lock_builder_b'
  | 'simulating_a'
  | 'simulating_b'
  | 'circ_exchange'
  | 'voting'
  | 'decided'
  | 'delivering'
  | 'confirming'
  | 'complete'

export interface Transaction {
  instanceId: string
  type: 'mint' | 'bridge' | 'xt-mint' | 'native' | 'scenario'
  status: TransactionStatus
  chainId: number
  chainATx?: string
  chainBTx?: string
  createdAt: Date
  decidedAt?: Date
  decision?: boolean
}

// services map is keyed by service id from cmd/localnet-health. An absent
// key means the first poll has not landed yet; status === "missing" means
// the catalogue entry exists but the container does not (optional feature
// disabled).
export interface CurrentStatus {
  step: FlowStep
  services: Record<string, Service>
}

interface TransactionStore {
  transactions: Transaction[]
  currentStatus: CurrentStatus

  addTransaction: (tx: Transaction) => void
  updateTransaction: (instanceId: string, updates: Partial<Transaction>) => void
  clearTransactions: () => void
  setServices: (services: Record<string, Service>) => void
  setFlowStep: (step: FlowStep) => void
  reset: () => void
}

const initialStatus: CurrentStatus = {
  step: 'idle',
  services: {},
}

export const useTransactionStore = create<TransactionStore>((set) => ({
  transactions: [],
  currentStatus: initialStatus,

  addTransaction: (tx) =>
    set((state) => ({
      transactions: [tx, ...state.transactions].slice(0, 50),
    })),

  updateTransaction: (instanceId, updates) =>
    set((state) => ({
      transactions: state.transactions.map((tx) =>
        tx.instanceId === instanceId ? { ...tx, ...updates } : tx,
      ),
    })),

  clearTransactions: () => set({ transactions: [] }),

  setServices: (services) =>
    set((state) => ({
      currentStatus: { ...state.currentStatus, services },
    })),

  setFlowStep: (step) =>
    set((state) => ({
      currentStatus: { ...state.currentStatus, step },
    })),

  reset: () =>
    set({
      transactions: [],
      currentStatus: initialStatus,
    }),
}))

export function statusOf(
  services: Record<string, Service>,
  id: string,
): ServiceStatus {
  return services[id]?.status ?? 'missing'
}
