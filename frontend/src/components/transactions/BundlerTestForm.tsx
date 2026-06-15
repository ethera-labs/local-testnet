import { useEffect, useState } from 'react'
import { ethers } from 'ethers'
import {
  buildSignedUserOpsTx,
  BundlerError,
  type Chain,
} from '../../api/bundler'
import {
  buildSignedUserOp,
  depositToEntryPoint,
  resolveSmartAccount,
  USER_OPERATION_EVENT_TOPIC,
  type SmartAccountInfo,
} from '../../api/userop'
import {
  BUNDLER_A_URL,
  BUNDLER_B_URL,
  ENTRYPOINT_A,
  ENTRYPOINT_B,
} from '../../config/chains'
import { getProvider, getSigner } from '../../api/rollup'

const SALT = 0n
const DEPOSIT_AMOUNT = ethers.parseEther('0.05')
const MIN_DEPOSIT_TARGET = ethers.parseEther('0.01')

type Status =
  | 'idle'
  | 'depositing'
  | 'building'
  | 'broadcasting'
  | 'success'
  | 'error'

interface RunResult {
  bundlerTxHash: string
  userOpHash: string
  success: boolean
  gasUsed: bigint
  actualGasCost: bigint
}

export default function BundlerTestForm() {
  const [chain, setChain] = useState<Chain>('A')
  const [account, setAccount] = useState<SmartAccountInfo | null>(null)
  const [accountError, setAccountError] = useState<string | null>(null)
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<RunResult | null>(null)
  const [refreshTick, setRefreshTick] = useState(0)

  const bundlerUrl = chain === 'A' ? BUNDLER_A_URL : BUNDLER_B_URL
  const entryPoint = chain === 'A' ? ENTRYPOINT_A : ENTRYPOINT_B

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        setAccountError(null)
        const signer = getSigner(chain)
        const info = await resolveSmartAccount(chain, signer.address, SALT)
        if (!cancelled) setAccount(info)
      } catch (err) {
        if (!cancelled) {
          setAccount(null)
          setAccountError(err instanceof Error ? err.message : String(err))
        }
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [chain, refreshTick])

  const send = async () => {
    setStatus('building')
    setError(null)
    setResult(null)

    try {
      const signer = getSigner(chain)
      const provider = getProvider(chain)
      const { chainId } = await provider.getNetwork()

      const fresh = await resolveSmartAccount(chain, signer.address, SALT)
      if (fresh.deposit < MIN_DEPOSIT_TARGET) {
        setStatus('depositing')
        await depositToEntryPoint(chain, signer, fresh.address, DEPOSIT_AMOUNT)
      }

      setStatus('building')
      const userOp = await buildSignedUserOp({
        chain,
        ownerSigner: signer,
        salt: SALT,
        target: signer.address,
        value: 0n,
        callData: '0x',
      })

      const signed = await buildSignedUserOpsTx(chain, [userOp], Number(chainId))

      setStatus('broadcasting')
      const tx = await provider.broadcastTransaction(signed.raw)
      const receipt = await tx.wait()
      if (!receipt) throw new Error('No receipt returned')

      const eventLog = receipt.logs.find(
        (log) => log.topics[0] === USER_OPERATION_EVENT_TOPIC
      )
      let success = receipt.status === 1
      let actualGasCost = 0n
      let actualGasUsed = receipt.gasUsed
      if (eventLog) {
        // UserOperationEvent indexed topics: userOpHash, sender, paymaster.
        // Non-indexed data: nonce, success, actualGasCost, actualGasUsed.
        const decoded = ethers.AbiCoder.defaultAbiCoder().decode(
          ['uint256', 'bool', 'uint256', 'uint256'],
          eventLog.data
        )
        success = decoded[1] as boolean
        actualGasCost = decoded[2] as bigint
        actualGasUsed = decoded[3] as bigint
      }

      setResult({
        bundlerTxHash: tx.hash,
        userOpHash: signed.userOpHashes[0],
        success,
        actualGasCost,
        gasUsed: actualGasUsed,
      })
      setStatus('success')

      // Optimistically mark deployed so the panel reflects the new state
      // immediately; flashblocks read-state can lag the receipt by a tick.
      setAccount((prev) => (prev ? { ...prev, deployed: true } : prev))
      setRefreshTick((t) => t + 1)
    } catch (err) {
      const message =
        err instanceof BundlerError
          ? `${err.message}${err.data ? ` (${JSON.stringify(err.data)})` : ''}`
          : err instanceof Error
            ? err.message
            : String(err)
      setError(message)
      setStatus('error')
    }
  }

  const inProgress =
    status === 'depositing' ||
    status === 'building' ||
    status === 'broadcasting'
  const buttonDisabled = inProgress || !account

  return (
    <div className="space-y-5">
      {/* Header */}
      <div>
        <p className="font-display text-[11px] tracking-[0.3em] uppercase text-text-primary">
          Bundler Test
        </p>
        <p className="mt-1.5 text-[10px] font-mono text-text-dim leading-relaxed">
          Builds a v0.7 UserOperation against a SimpleAccount derived from your
          wallet key, calls{' '}
          <span className="text-text-secondary">ethera_buildSignedUserOpsTx</span>,
          and broadcasts the returned handleOps transaction.
        </p>
      </div>

      {/* Chain selector */}
      <div>
        <label className="font-display text-[9px] tracking-[0.25em] uppercase text-text-dim block mb-1.5">
          Chain
        </label>
        <div className="grid grid-cols-2 gap-2">
          {(['A', 'B'] as Chain[]).map((c) => (
            <button
              key={c}
              type="button"
              disabled={inProgress}
              onClick={() => setChain(c)}
              className={`px-3 py-2 font-display text-[10px] tracking-[0.25em] uppercase border transition-all ${
                chain === c
                  ? 'border-cyan text-cyan bg-cyan/5'
                  : 'border-border text-text-secondary hover:text-text-primary'
              } ${inProgress ? 'opacity-50 cursor-not-allowed' : ''}`}
            >
              Chain {c}
            </button>
          ))}
        </div>
      </div>

      {/* Info panel */}
      <div className="border border-border bg-bg-elevated">
        <InfoRow
          label="Bundler"
          value={bundlerUrl || ''}
          mono
        />
        <InfoRow
          label="EntryPoint"
          value={entryPoint || ''}
          mono
        />
        <InfoRow
          label="Smart account"
          value={account ? account.address : accountError ? '' : 'loading…'}
          mono
        />
        <InfoRow
          label="Deployed"
          value={
            account
              ? account.deployed
                ? 'yes'
                : 'no  first userop will deploy'
              : ''
          }
          accent={account?.deployed ? 'cyan' : undefined}
        />
        <InfoRow
          label="Deposit"
          value={
            account ? `${ethers.formatEther(account.deposit)} ETH` : ''
          }
        />
      </div>

      {accountError && (
        <div className="border border-error/40 bg-error/5 px-3 py-2.5 text-error text-xs font-mono">
          <span className="text-error/60 mr-2">!</span>
          {accountError}
        </div>
      )}

      <button
        type="button"
        onClick={send}
        disabled={buttonDisabled}
        className={`w-full py-3 font-display text-[11px] tracking-[0.3em] uppercase border transition-all ${
          buttonDisabled
            ? 'border-border text-text-dim cursor-not-allowed'
            : 'border-cyan text-cyan hover:bg-cyan hover:text-bg glow-cyan'
        }`}
      >
        {inProgress ? (
          <span className="flex items-center justify-center gap-2">
            <span className="w-1 h-1 bg-current rounded-full indicator-active" />
            <span
              className="w-1 h-1 bg-current rounded-full indicator-active"
              style={{ animationDelay: '0.3s' }}
            />
            <span
              className="w-1 h-1 bg-current rounded-full indicator-active"
              style={{ animationDelay: '0.6s' }}
            />
            <span className="ml-2">{statusLabel(status)}</span>
          </span>
        ) : (
          'Send Test UserOp'
        )}
      </button>

      {error && (
        <div className="border border-error/40 bg-error/5 px-3 py-2.5 text-error text-xs font-mono whitespace-pre-wrap break-all">
          <span className="text-error/60 mr-2">!</span>
          {error}
        </div>
      )}

      {result && (
        <div className="border border-border bg-bg-elevated">
          <div
            className={`px-3 py-2 border-b border-border font-display text-[10px] tracking-[0.25em] uppercase ${
              result.success ? 'text-cyan' : 'text-error'
            }`}
          >
            {result.success ? 'UserOp Succeeded' : 'UserOp Reverted'}
          </div>
          <InfoRow label="handleOps tx" value={result.bundlerTxHash} mono />
          <InfoRow label="userOp hash" value={result.userOpHash} mono />
          <InfoRow label="Gas used" value={result.gasUsed.toString()} mono />
          <InfoRow
            label="Gas cost"
            value={`${ethers.formatEther(result.actualGasCost)} ETH`}
            mono
          />
        </div>
      )}
    </div>
  )
}

interface InfoRowProps {
  label: string
  value: string
  mono?: boolean
  accent?: 'cyan' | 'amber' | 'error'
}

function InfoRow({ label, value, mono, accent }: InfoRowProps) {
  const accentClass =
    accent === 'cyan'
      ? 'text-cyan'
      : accent === 'amber'
        ? 'text-amber'
        : accent === 'error'
          ? 'text-error'
          : 'text-text-primary'
  return (
    <div className="flex items-center justify-between gap-3 px-3 py-2 border-b border-border last:border-b-0">
      <span className="font-display text-[9px] tracking-[0.25em] uppercase text-text-dim">
        {label}
      </span>
      <span
        className={`${mono ? 'font-mono' : ''} text-[10px] truncate ${accentClass}`}
        title={value}
      >
        {value}
      </span>
    </div>
  )
}

function statusLabel(status: Status): string {
  switch (status) {
    case 'depositing':
      return 'Funding deposit'
    case 'building':
      return 'Building userop'
    case 'broadcasting':
      return 'Broadcasting'
    default:
      return 'Working'
  }
}
