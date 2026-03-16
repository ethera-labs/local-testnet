import { create } from 'zustand'

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
  | 'builder_poll_a'
  | 'builder_poll_b'
  | 'simulating_a'
  | 'simulating_b'
  | 'circ_exchange'
  | 'voting'
  | 'decided'
  | 'delivering'
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

export interface CurrentStatus {
  step: FlowStep
  chainAConnected: boolean
  chainBConnected: boolean
  sidecarAActive: boolean
  sidecarBActive: boolean
}

interface TransactionStore {
  transactions: Transaction[]
  currentStatus: CurrentStatus

  addTransaction: (tx: Transaction) => void
  updateTransaction: (instanceId: string, updates: Partial<Transaction>) => void
  setCurrentStatus: (status: Partial<CurrentStatus>) => void
  setFlowStep: (step: FlowStep) => void
  reset: () => void
}

const initialStatus: CurrentStatus = {
  step: 'idle',
  chainAConnected: true,
  chainBConnected: true,
  sidecarAActive: true,
  sidecarBActive: true,
}

export const useTransactionStore = create<TransactionStore>((set) => ({
  transactions: [],
  currentStatus: initialStatus,

  addTransaction: (tx) =>
    set((state) => ({
      transactions: [tx, ...state.transactions].slice(0, 50), // Keep last 50
    })),

  updateTransaction: (instanceId, updates) =>
    set((state) => ({
      transactions: state.transactions.map((tx) =>
        tx.instanceId === instanceId ? { ...tx, ...updates } : tx
      ),
    })),

  setCurrentStatus: (status) =>
    set((state) => ({
      currentStatus: { ...state.currentStatus, ...status },
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
