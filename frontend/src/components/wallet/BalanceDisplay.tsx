import { useState, useEffect } from 'react'
import {
  formatBalance,
  getChainId,
  getSigner,
  getTokenAddress,
  getTokenBalance,
  getWrappedTokenAddress,
} from '../../api/rollup'

interface BalanceDisplayProps {
  chain: 'A' | 'B'
  remoteChain?: 'A' | 'B'
}

interface Balances {
  native: string
  wrapped?: string
  wrappedUnavailable?: boolean
}

export default function BalanceDisplay({ chain, remoteChain }: BalanceDisplayProps) {
  const [balances, setBalances] = useState<Balances>({ native: '0.0' })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadBalance = async () => {
    try {
      setLoading(true)
      setError(null)
      const signer = getSigner(chain)
      const walletAddress = await signer.getAddress()
      const tokenAddress = getTokenAddress(chain)
      const nativeBalanceWei = await getTokenBalance(tokenAddress, walletAddress, chain)

      let wrappedBalance: string | undefined
      let wrappedUnavailable = false
      if (remoteChain) {
        try {
          const wrappedTokenAddress = await getWrappedTokenAddress(
            getTokenAddress(remoteChain),
            getChainId(remoteChain),
            chain
          )
          const wrappedBalanceWei = await getTokenBalance(wrappedTokenAddress, walletAddress, chain)
          wrappedBalance = formatBalance(wrappedBalanceWei)
        } catch (err) {
          console.warn('Unable to load bridged token balance:', err)
          wrappedUnavailable = true
        }
      }

      setBalances({
        native: formatBalance(nativeBalanceWei),
        wrapped: wrappedBalance,
        wrappedUnavailable,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'err')
      setBalances({ native: '0.0' })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadBalance()
    const interval = setInterval(loadBalance, 5000)
    return () => clearInterval(interval)
  }, [chain, remoteChain])

  const nativeDisplay = error
    ? <span className="text-[11px] font-mono text-error/70">err</span>
    : loading && balances.native === '0.0'
    ? <span className="text-[11px] font-mono text-text-dim">-</span>
    : <span className="text-[11px] font-mono text-text-secondary">
        {parseFloat(balances.native).toFixed(4)}
      </span>

  const wrappedValue = balances.wrapped !== undefined
    ? parseFloat(balances.wrapped).toFixed(4)
    : '-'
  const wrappedColor = balances.wrapped !== undefined ? 'text-amber' : 'text-text-dim'

  return (
    <div className="flex flex-col items-end gap-0.5">
      <div className="flex items-center gap-1.5">
        {nativeDisplay}
        <span className="text-[9px] text-text-dim font-display tracking-widest uppercase">native</span>
      </div>
      {remoteChain && (
        <div className="flex items-center gap-1.5">
          <span className={`text-[11px] font-mono ${wrappedColor}`}>{wrappedValue}</span>
          <span className="text-[9px] text-text-dim font-display tracking-widest uppercase">bridged</span>
        </div>
      )}
    </div>
  )
}
