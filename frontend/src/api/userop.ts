import { ethers } from 'ethers'
import {
  getEntryPointAddress,
  getSimpleAccountFactoryAddress,
  type Chain,
  type UserOpV07,
} from './bundler'
import { getProvider } from './rollup'

const ENTRY_POINT_ABI = [
  'function balanceOf(address) view returns (uint256)',
  'function depositTo(address) payable',
  'function getNonce(address sender, uint192 key) view returns (uint256)',
  'function getUserOpHash(tuple(address sender, uint256 nonce, bytes initCode, bytes callData, bytes32 accountGasLimits, uint256 preVerificationGas, bytes32 gasFees, bytes paymasterAndData, bytes signature) userOp) view returns (bytes32)',
  'event UserOperationEvent(bytes32 indexed userOpHash, address indexed sender, address indexed paymaster, uint256 nonce, bool success, uint256 actualGasCost, uint256 actualGasUsed)',
]

const FACTORY_ABI = [
  'function createAccount(address owner, uint256 salt) returns (address)',
  'function getAddress(address owner, uint256 salt) view returns (address)',
]

const SIMPLE_ACCOUNT_ABI = [
  'function execute(address dest, uint256 value, bytes calldata data)',
]

export interface SmartAccountInfo {
  address: string
  deployed: boolean
  deposit: bigint
}

// resolveSmartAccount derives the counterfactual SimpleAccount address for the
// given owner + salt and reports whether it's been deployed yet and what its
// EntryPoint deposit balance is.
export async function resolveSmartAccount(
  chain: Chain,
  owner: string,
  salt: bigint
): Promise<SmartAccountInfo> {
  const provider = getProvider(chain)
  const factory = new ethers.Contract(
    getSimpleAccountFactoryAddress(chain),
    FACTORY_ABI,
    provider
  )
  const entryPoint = new ethers.Contract(
    getEntryPointAddress(chain),
    ENTRY_POINT_ABI,
    provider
  )

  // BaseContract has its own getAddress(); use getFunction to call the ABI method.
  const address: string = await factory.getFunction('getAddress')(owner, salt)
  const code = await provider.getCode(address)
  const deposit: bigint = await entryPoint.balanceOf(address)

  return { address, deployed: code !== '0x', deposit }
}

// depositToEntryPoint sends ETH to EntryPoint.depositTo(account) from the
// given signer. Returns the broadcast tx hash.
export async function depositToEntryPoint(
  chain: Chain,
  signer: ethers.Wallet,
  account: string,
  amount: bigint
): Promise<string> {
  const entryPoint = new ethers.Contract(
    getEntryPointAddress(chain),
    ENTRY_POINT_ABI,
    signer
  )
  const tx: ethers.TransactionResponse = await entryPoint.depositTo(account, {
    value: amount,
  })
  await tx.wait()
  return tx.hash
}

interface BuildUserOpInput {
  chain: Chain
  ownerSigner: ethers.Wallet
  salt: bigint
  target: string
  value: bigint
  callData: string
  callGasLimit?: bigint
  verificationGasLimit?: bigint
  preVerificationGas?: bigint
  maxFeePerGas?: bigint
  maxPriorityFeePerGas?: bigint
}

// buildSignedUserOp constructs a v0.7 UserOperation, fetches the account nonce
// from EntryPoint, derives factory + factoryData when the SimpleAccount has
// not been deployed yet, and signs the userOpHash with the owner key. The
// returned object is the unpacked v0.7 wire format the bundler accepts.
export async function buildSignedUserOp(
  input: BuildUserOpInput
): Promise<UserOpV07> {
  const provider = getProvider(input.chain)
  const entryPointAddr = getEntryPointAddress(input.chain)
  const factoryAddr = getSimpleAccountFactoryAddress(input.chain)
  const account = await resolveSmartAccount(
    input.chain,
    input.ownerSigner.address,
    input.salt
  )

  const accountIface = new ethers.Interface(SIMPLE_ACCOUNT_ABI)
  const innerCallData = accountIface.encodeFunctionData('execute', [
    input.target,
    input.value,
    input.callData,
  ])

  const factoryIface = new ethers.Interface(FACTORY_ABI)
  const factoryData = account.deployed
    ? '0x'
    : factoryIface.encodeFunctionData('createAccount', [
        input.ownerSigner.address,
        input.salt,
      ])

  const entryPoint = new ethers.Contract(
    entryPointAddr,
    ENTRY_POINT_ABI,
    provider
  )
  const nonce: bigint = await entryPoint.getNonce(account.address, 0n)

  const callGasLimit = input.callGasLimit ?? 200_000n
  const verificationGasLimit = input.verificationGasLimit ?? 300_000n
  const preVerificationGas = input.preVerificationGas ?? 60_000n
  const maxFeePerGas = input.maxFeePerGas ?? 5_000_000_000n
  const maxPriorityFeePerGas = input.maxPriorityFeePerGas ?? 1_000_000_000n

  const unsignedOp: UserOpV07 = {
    sender: account.address,
    nonce: hex(nonce),
    factory: account.deployed ? undefined : factoryAddr,
    factoryData: account.deployed ? undefined : factoryData,
    callData: innerCallData,
    callGasLimit: hex(callGasLimit),
    verificationGasLimit: hex(verificationGasLimit),
    preVerificationGas: hex(preVerificationGas),
    maxFeePerGas: hex(maxFeePerGas),
    maxPriorityFeePerGas: hex(maxPriorityFeePerGas),
    signature: '0x',
  }

  const packed = packUserOp(unsignedOp)
  const userOpHash: string = await entryPoint.getUserOpHash(packed)
  const signature = await input.ownerSigner.signMessage(ethers.getBytes(userOpHash))

  return { ...unsignedOp, signature }
}

// packUserOp converts the unpacked v0.7 wire form into the on-chain packed
// tuple used by EntryPoint.getUserOpHash. Mirrors `crates/bundler/src/packing.rs`.
function packUserOp(op: UserOpV07): {
  sender: string
  nonce: bigint
  initCode: string
  callData: string
  accountGasLimits: string
  preVerificationGas: bigint
  gasFees: string
  paymasterAndData: string
  signature: string
} {
  const initCode =
    op.factory && op.factory !== ethers.ZeroAddress && op.factoryData
      ? ethers.concat([op.factory, op.factoryData])
      : '0x'

  const paymasterAndData =
    op.paymaster && op.paymaster !== ethers.ZeroAddress
      ? ethers.concat([
          op.paymaster,
          padU128(op.paymasterVerificationGasLimit ?? '0x0'),
          padU128(op.paymasterPostOpGasLimit ?? '0x0'),
          op.paymasterData ?? '0x',
        ])
      : '0x'

  const accountGasLimits = packPair(op.verificationGasLimit, op.callGasLimit)
  const gasFees = packPair(op.maxPriorityFeePerGas, op.maxFeePerGas)

  return {
    sender: op.sender,
    nonce: BigInt(op.nonce),
    initCode,
    callData: op.callData,
    accountGasLimits,
    preVerificationGas: BigInt(op.preVerificationGas),
    gasFees,
    paymasterAndData,
    signature: op.signature,
  }
}

function packPair(hi: string, lo: string): string {
  return ethers.concat([padU128(hi), padU128(lo)])
}

function padU128(value: string | bigint): string {
  const v = typeof value === 'bigint' ? value : BigInt(value)
  return ethers.zeroPadValue(ethers.toBeHex(v), 16)
}

function hex(value: bigint): string {
  return '0x' + value.toString(16)
}

export const USER_OPERATION_EVENT_TOPIC = ethers.id(
  'UserOperationEvent(bytes32,address,address,uint256,bool,uint256,uint256)'
)
