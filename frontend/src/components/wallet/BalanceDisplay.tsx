import { useState, useEffect } from 'react'
import { getTokenBalance, getSigner, getTokenAddress, formatBalance } from '../../api/rollup'

interface BalanceDisplayProps {
  chain: 'A' | 'B'
}

export default function BalanceDisplay({ chain }: BalanceDisplayProps) {
  const [balance, setBalance] = useState<string>('0.0')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadBalance = async () => {
    try {
      setLoading(true)
      setError(null)
      const signer = getSigner(chain)
      const walletAddress = await signer.getAddress()
      const tokenAddress = getTokenAddress(chain)
      const balanceWei = await getTokenBalance(tokenAddress, walletAddress, chain)
      setBalance(formatBalance(balanceWei))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'err')
      setBalance('0.0')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadBalance()
    const interval = setInterval(loadBalance, 5000)
    return () => clearInterval(interval)
  }, [chain])

  if (loading && balance === '0.0') {
    return (
      <div className="flex items-center gap-1.5">
        <span className="w-1 h-1 rounded-full bg-border-bright indicator-active" />
        <span className="text-[10px] text-text-dim font-mono">—</span>
      </div>
    )
  }

  if (error) {
    return (
      <span className="text-[10px] text-error/70 font-mono">err</span>
    )
  }

  return (
    <div className="flex items-center gap-1.5">
      <span className="text-[11px] font-mono text-text-secondary">
        {parseFloat(balance).toFixed(4)}
      </span>
      <span className="text-[9px] text-text-dim font-display tracking-widest uppercase">TKN</span>
    </div>
  )
}
