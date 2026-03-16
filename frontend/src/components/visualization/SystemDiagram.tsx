import { useCallback, useMemo, useState, useEffect } from 'react'
import type { MouseEvent } from 'react'
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  NodeTypes,
  MarkerType,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { CurrentStatus, FlowStep } from '../../stores/transactionStore'
import type { FlowMode } from './TransactionFlowPanel'
import RollupNode from './nodes/RollupNode'
import SidecarNode from './nodes/SidecarNode'
import BuilderNode from './nodes/BuilderNode'
import BoostNode from './nodes/BoostNode'
import OpNodeNode from './nodes/OpNodeNode'
import GethNode from './nodes/GethNode'
import PublisherNode from './nodes/PublisherNode'
import UserNode from './nodes/UserNode'

interface SystemDiagramProps {
  currentStatus: CurrentStatus
  onSelectFlow?: (mode: FlowMode) => void
  selectedFlow?: FlowMode | null
}

const nodeTypes: NodeTypes = {
  rollup: RollupNode,
  sidecar: SidecarNode,
  builder: BuilderNode,
  boost: BoostNode,
  opnode: OpNodeNode,
  geth: GethNode,
  publisher: PublisherNode,
  user: UserNode,
}

function getEdgeStatus(step: FlowStep, edgeId: string): 'idle' | 'active' | 'complete' {
  const activeEdges: Record<FlowStep, string[]> = {
    idle: [],
    submitting: [
      'user-sidecar-a',
      'user-sidecar-b',
      'publisher-sidecar-a',
      'publisher-sidecar-b',
    ],
    minting_a: ['op-node-a-boost-a', 'boost-a-builder-a'],
    minting_b: ['op-node-b-boost-b', 'boost-b-builder-b'],
    minting_both: [
      'op-node-a-boost-a',
      'boost-a-builder-a',
      'op-node-b-boost-b',
      'boost-b-builder-b',
    ],
    forward_to_peer: ['sidecar-a-sidecar-b'],
    builder_poll_a: ['builder-a-sidecar-a'],
    builder_poll_b: ['builder-b-sidecar-b'],
    simulating_a: ['sidecar-a-simulate-builder-a'],
    simulating_b: ['sidecar-b-simulate-builder-b'],
    circ_exchange: ['sidecar-a-sidecar-b'],
    voting: ['sidecar-a-sidecar-b', 'sidecar-a-publisher', 'sidecar-b-publisher'],
    decided: ['publisher-sidecar-a', 'publisher-sidecar-b'],
    delivering: ['sidecar-a-builder-a', 'sidecar-b-builder-b'],
    complete: [],
  }

  if (activeEdges[step]?.includes(edgeId)) {
    return 'active'
  }
  return 'idle'
}

const EDGE_ACTIVE = '#00D4A8'
const EDGE_IDLE = '#505070'
const EDGE_MINT_ACTIVE = '#00D4A8'

export default function SystemDiagram({
  currentStatus,
  onSelectFlow,
  selectedFlow,
}: SystemDiagramProps) {
  const { step } = currentStatus
  const highlightNormal = selectedFlow === 'normal'
  const highlightXt = selectedFlow === 'xt'
  const [isFullscreen, setIsFullscreen] = useState(false)

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isFullscreen) {
        setIsFullscreen(false)
      }
    }

    if (isFullscreen) {
      document.addEventListener('keydown', handleEscape)
      document.body.style.overflow = 'hidden'
    }

    return () => {
      document.removeEventListener('keydown', handleEscape)
      document.body.style.overflow = ''
    }
  }, [isFullscreen])

  const nodes: Node[] = useMemo(
    () => [
      // User Node - Top
      {
        id: 'user',
        type: 'user',
        position: { x: 400, y: -120 },
        data: {
          label: 'User',
        },
      },
      // Layer 1: op-node (consensus clients)
      {
        id: 'op-node-a',
        type: 'opnode',
        position: { x: 50, y: 50 },
        data: {
          label: 'op-node A',
          port: 18547,
          active: true,
        },
      },
      {
        id: 'op-node-b',
        type: 'opnode',
        position: { x: 750, y: 50 },
        data: {
          label: 'op-node B',
          port: 28547,
          active: true,
        },
      },
      // Layer 2: rollup-boost
      {
        id: 'boost-a',
        type: 'boost',
        position: { x: 50, y: 170 },
        data: {
          label: 'rollup-boost A',
          port: 17551,
          active: true,
        },
      },
      {
        id: 'boost-b',
        type: 'boost',
        position: { x: 750, y: 170 },
        data: {
          label: 'rollup-boost B',
          port: 27551,
          active: true,
        },
      },
      // Layer 3: op-rbuilder (builders)
      {
        id: 'builder-a',
        type: 'builder',
        position: { x: 50, y: 290 },
        data: {
          label: 'op-rbuilder A',
          port: 17545,
          polling: step === 'builder_poll_a',
        },
      },
      {
        id: 'builder-b',
        type: 'builder',
        position: { x: 750, y: 290 },
        data: {
          label: 'op-rbuilder B',
          port: 27545,
          polling: step === 'builder_poll_b',
        },
      },
      // Layer 4: Sidecars
      {
        id: 'sidecar-a',
        type: 'sidecar',
        position: { x: 250, y: 410 },
        data: {
          label: 'Sidecar A',
          port: 17090,
          active: currentStatus.sidecarAActive,
          processing: ['simulating_a', 'voting'].includes(step),
        },
      },
      {
        id: 'sidecar-b',
        type: 'sidecar',
        position: { x: 550, y: 410 },
        data: {
          label: 'Sidecar B',
          port: 27090,
          active: currentStatus.sidecarBActive,
          processing: ['simulating_b', 'voting'].includes(step),
        },
      },
      // Publisher
      {
        id: 'publisher',
        type: 'publisher',
        position: { x: 400, y: 290 },
        data: {
          label: 'Publisher',
          port: 8080,
          active: true,
          coordinating: step === 'voting',
        },
      },
      // Layer 5: op-geth (execution clients)
      {
        id: 'geth-a',
        type: 'geth',
        position: { x: 50, y: 530 },
        data: {
          label: 'op-geth A',
          port: 18545,
          connected: currentStatus.chainAConnected,
        },
      },
      {
        id: 'geth-b',
        type: 'geth',
        position: { x: 750, y: 530 },
        data: {
          label: 'op-geth B',
          port: 28545,
          connected: currentStatus.chainBConnected,
        },
      },
    ],
    [currentStatus, step]
  )

  // Shared label background — transparent so no white box appears
  const LBG = { fill: 'transparent' } as const
  const LSTYLE = (_activeColor: string, idleColor = '#9090B0') =>
    ({ fontSize: 9, fontFamily: '"IBM Plex Mono", monospace', fill: idleColor } as const)
  const LSTYLE_ACTIVE = (color: string) =>
    ({ fontSize: 9, fontFamily: '"IBM Plex Mono", monospace', fill: color } as const)

  const edges: Edge[] = useMemo(
    () => [
      // User -> Sidecar A (XT)
      {
        id: 'user-sidecar-a',
        source: 'user',
        target: 'sidecar-a',
        animated: getEdgeStatus(step, 'user-sidecar-a') === 'active',
        label: 'submit XT',
        labelStyle: getEdgeStatus(step, 'user-sidecar-a') === 'active' || highlightXt
          ? LSTYLE_ACTIVE(EDGE_ACTIVE)
          : LSTYLE(EDGE_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'user-sidecar-a') === 'active' || highlightXt ? EDGE_ACTIVE : EDGE_IDLE,
          strokeWidth: 1.5,
          strokeDasharray: getEdgeStatus(step, 'user-sidecar-a') === 'active' || highlightXt ? '0' : '5,5',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'user-sidecar-a') === 'active' || highlightXt ? EDGE_ACTIVE : EDGE_IDLE,
        },
      },
      // User -> Sidecar B (XT)
      {
        id: 'user-sidecar-b',
        source: 'user',
        target: 'sidecar-b',
        animated: getEdgeStatus(step, 'user-sidecar-b') === 'active',
        label: 'submit XT',
        labelStyle: getEdgeStatus(step, 'user-sidecar-b') === 'active' || highlightXt
          ? LSTYLE_ACTIVE(EDGE_ACTIVE)
          : LSTYLE(EDGE_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'user-sidecar-b') === 'active' || highlightXt ? EDGE_ACTIVE : EDGE_IDLE,
          strokeWidth: 1.5,
          strokeDasharray: getEdgeStatus(step, 'user-sidecar-b') === 'active' || highlightXt ? '0' : '5,5',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'user-sidecar-b') === 'active' || highlightXt ? EDGE_ACTIVE : EDGE_IDLE,
        },
      },
      // User -> Builder A (Normal TX)
      {
        id: 'user-builder-a',
        source: 'user',
        target: 'builder-a',
        animated: false,
        label: 'normal tx',
        labelStyle: highlightNormal ? LSTYLE_ACTIVE(EDGE_ACTIVE) : LSTYLE(EDGE_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: highlightNormal ? EDGE_ACTIVE : EDGE_IDLE,
          strokeWidth: 1.5,
          strokeDasharray: highlightNormal ? '0' : '5,5',
        },
        markerEnd: { type: MarkerType.ArrowClosed, color: highlightNormal ? EDGE_ACTIVE : EDGE_IDLE },
      },
      // User -> Builder B (Normal TX)
      {
        id: 'user-builder-b',
        source: 'user',
        target: 'builder-b',
        animated: false,
        label: 'normal tx',
        labelStyle: highlightNormal ? LSTYLE_ACTIVE(EDGE_ACTIVE) : LSTYLE(EDGE_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: highlightNormal ? EDGE_ACTIVE : EDGE_IDLE,
          strokeWidth: 1.5,
          strokeDasharray: highlightNormal ? '0' : '5,5',
        },
        markerEnd: { type: MarkerType.ArrowClosed, color: highlightNormal ? EDGE_ACTIVE : EDGE_IDLE },
      },
      // op-node A -> rollup-boost A (engine API)
      {
        id: 'op-node-a-boost-a',
        source: 'op-node-a',
        target: 'boost-a',
        sourceHandle: 'to-boost',
        targetHandle: 'from-rollup',
        animated: getEdgeStatus(step, 'op-node-a-boost-a') === 'active',
        label: 'engine api',
        labelStyle: getEdgeStatus(step, 'op-node-a-boost-a') === 'active'
          ? LSTYLE_ACTIVE(EDGE_MINT_ACTIVE)
          : LSTYLE(EDGE_MINT_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'op-node-a-boost-a') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'op-node-a-boost-a') === 'active' ? 3 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'op-node-a-boost-a') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
      },
      // op-node B -> rollup-boost B (engine API)
      {
        id: 'op-node-b-boost-b',
        source: 'op-node-b',
        target: 'boost-b',
        sourceHandle: 'to-boost',
        targetHandle: 'from-rollup',
        animated: getEdgeStatus(step, 'op-node-b-boost-b') === 'active',
        label: 'engine api',
        labelStyle: getEdgeStatus(step, 'op-node-b-boost-b') === 'active'
          ? LSTYLE_ACTIVE(EDGE_MINT_ACTIVE)
          : LSTYLE(EDGE_MINT_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'op-node-b-boost-b') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'op-node-b-boost-b') === 'active' ? 3 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'op-node-b-boost-b') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
      },
      // rollup-boost A <-> Builder A (engine API)
      {
        id: 'boost-a-builder-a',
        source: 'boost-a',
        target: 'builder-a',
        sourceHandle: 'to-builder',
        targetHandle: 'from-rollup',
        animated: getEdgeStatus(step, 'boost-a-builder-a') === 'active',
        label: 'engine api',
        labelStyle: getEdgeStatus(step, 'boost-a-builder-a') === 'active'
          ? LSTYLE_ACTIVE(EDGE_MINT_ACTIVE)
          : LSTYLE(EDGE_MINT_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'boost-a-builder-a') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'boost-a-builder-a') === 'active' ? 3 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'boost-a-builder-a') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'boost-a-builder-a') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
      },
      // rollup-boost B <-> Builder B (engine API)
      {
        id: 'boost-b-builder-b',
        source: 'boost-b',
        target: 'builder-b',
        sourceHandle: 'to-builder',
        targetHandle: 'from-rollup',
        animated: getEdgeStatus(step, 'boost-b-builder-b') === 'active',
        label: 'engine api',
        labelStyle: getEdgeStatus(step, 'boost-b-builder-b') === 'active'
          ? LSTYLE_ACTIVE(EDGE_MINT_ACTIVE)
          : LSTYLE(EDGE_MINT_ACTIVE),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'boost-b-builder-b') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'boost-b-builder-b') === 'active' ? 3 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'boost-b-builder-b') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'boost-b-builder-b') === 'active' ? EDGE_MINT_ACTIVE : EDGE_IDLE,
        },
      },
      // rollup-boost A -> op-geth A (fallback, dashed)
      {
        id: 'boost-a-geth-a',
        source: 'boost-a',
        target: 'geth-a',
        animated: false,
        label: 'fallback',
        labelStyle: { fontSize: 9, fontFamily: '"IBM Plex Mono", monospace', fill: '#606075' },
        labelBgStyle: LBG,
        style: { stroke: '#404055', strokeWidth: 1.5, strokeDasharray: '4,4' },
        markerEnd: { type: MarkerType.ArrowClosed, color: '#404055' },
      },
      // rollup-boost B -> op-geth B (fallback, dashed)
      {
        id: 'boost-b-geth-b',
        source: 'boost-b',
        target: 'geth-b',
        animated: false,
        label: 'fallback',
        labelStyle: { fontSize: 9, fontFamily: '"IBM Plex Mono", monospace', fill: '#606075' },
        labelBgStyle: LBG,
        style: { stroke: '#404055', strokeWidth: 1.5, strokeDasharray: '4,4' },
        markerEnd: { type: MarkerType.ArrowClosed, color: '#404055' },
      },
      // Sidecar A -> Builder A (simulation)
      {
        id: 'sidecar-a-simulate-builder-a',
        source: 'sidecar-a',
        target: 'builder-a',
        type: 'smoothstep',
        pathOptions: { offset: 20 },
        animated: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active',
        label: 'simulate/trace',
        labelStyle: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active' || highlightXt
          ? LSTYLE_ACTIVE('#60A5FA')
          : LSTYLE('#60A5FA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active' || highlightXt ? '#60A5FA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active' ? 2.5 : 1.5,
          strokeDasharray: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active' || highlightXt ? '0' : '4,4',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-a-simulate-builder-a') === 'active' || highlightXt ? '#60A5FA' : EDGE_IDLE,
        },
      },
      // Sidecar B -> Builder B (simulation)
      {
        id: 'sidecar-b-simulate-builder-b',
        source: 'sidecar-b',
        target: 'builder-b',
        type: 'smoothstep',
        pathOptions: { offset: 20 },
        animated: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active',
        label: 'simulate/trace',
        labelStyle: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active' || highlightXt
          ? LSTYLE_ACTIVE('#60A5FA')
          : LSTYLE('#60A5FA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active' || highlightXt ? '#60A5FA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active' ? 2.5 : 1.5,
          strokeDasharray: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active' || highlightXt ? '0' : '4,4',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-b-simulate-builder-b') === 'active' || highlightXt ? '#60A5FA' : EDGE_IDLE,
        },
      },
      // Sidecar A <-> Sidecar B (cross-chain coordination — always prominent)
      {
        id: 'sidecar-a-sidecar-b',
        source: 'sidecar-a',
        target: 'sidecar-b',
        animated: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active',
        label: 'cross-chain coordination',
        labelStyle: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active'
          ? { fontSize: 10, fontFamily: '"IBM Plex Mono", monospace', fill: '#FF6B00', fontWeight: 600 }
          : { fontSize: 10, fontFamily: '"IBM Plex Mono", monospace', fill: '#9090B0' },
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active' ? '#FF6B00' : '#7070A0',
          strokeWidth: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active' ? 3.5 : 2,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active' ? '#FF6B00' : '#7070A0',
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-a-sidecar-b') === 'active' ? '#FF6B00' : '#7070A0',
        },
      },
      // Sidecar A -> Publisher
      {
        id: 'sidecar-a-publisher',
        source: 'sidecar-a',
        target: 'publisher',
        animated: getEdgeStatus(step, 'sidecar-a-publisher') === 'active',
        label: 'vote',
        labelStyle: getEdgeStatus(step, 'sidecar-a-publisher') === 'active'
          ? LSTYLE_ACTIVE('#A78BFA')
          : LSTYLE('#A78BFA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-a-publisher') === 'active' ? '#A78BFA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-a-publisher') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-a-publisher') === 'active' ? '#A78BFA' : EDGE_IDLE,
        },
      },
      // Sidecar B -> Publisher
      {
        id: 'sidecar-b-publisher',
        source: 'sidecar-b',
        target: 'publisher',
        animated: getEdgeStatus(step, 'sidecar-b-publisher') === 'active',
        label: 'vote',
        labelStyle: getEdgeStatus(step, 'sidecar-b-publisher') === 'active'
          ? LSTYLE_ACTIVE('#A78BFA')
          : LSTYLE('#A78BFA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-b-publisher') === 'active' ? '#A78BFA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-b-publisher') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-b-publisher') === 'active' ? '#A78BFA' : EDGE_IDLE,
        },
      },
      // Publisher -> Sidecar A (StartSC / Decided)
      {
        id: 'publisher-sidecar-a',
        source: 'publisher',
        target: 'sidecar-a',
        animated: getEdgeStatus(step, 'publisher-sidecar-a') === 'active',
        label: 'start/decide',
        labelStyle: getEdgeStatus(step, 'publisher-sidecar-a') === 'active' || highlightXt
          ? LSTYLE_ACTIVE('#A78BFA')
          : LSTYLE('#A78BFA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'publisher-sidecar-a') === 'active' || highlightXt ? '#A78BFA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'publisher-sidecar-a') === 'active' ? 2.5 : 1.5,
          strokeDasharray: getEdgeStatus(step, 'publisher-sidecar-a') === 'active' || highlightXt ? '0' : '4,4',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'publisher-sidecar-a') === 'active' || highlightXt ? '#A78BFA' : EDGE_IDLE,
        },
      },
      // Publisher -> Sidecar B (StartSC / Decided)
      {
        id: 'publisher-sidecar-b',
        source: 'publisher',
        target: 'sidecar-b',
        animated: getEdgeStatus(step, 'publisher-sidecar-b') === 'active',
        label: 'start/decide',
        labelStyle: getEdgeStatus(step, 'publisher-sidecar-b') === 'active' || highlightXt
          ? LSTYLE_ACTIVE('#A78BFA')
          : LSTYLE('#A78BFA'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'publisher-sidecar-b') === 'active' || highlightXt ? '#A78BFA' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'publisher-sidecar-b') === 'active' ? 2.5 : 1.5,
          strokeDasharray: getEdgeStatus(step, 'publisher-sidecar-b') === 'active' || highlightXt ? '0' : '4,4',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'publisher-sidecar-b') === 'active' || highlightXt ? '#A78BFA' : EDGE_IDLE,
        },
      },
      // Builder A -> Sidecar A (polling)
      {
        id: 'builder-a-sidecar-a',
        source: 'builder-a',
        target: 'sidecar-a',
        animated: getEdgeStatus(step, 'builder-a-sidecar-a') === 'active',
        label: 'polls /tx',
        labelStyle: getEdgeStatus(step, 'builder-a-sidecar-a') === 'active'
          ? LSTYLE_ACTIVE('#FBBF24')
          : LSTYLE('#FBBF24'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'builder-a-sidecar-a') === 'active' ? '#FBBF24' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'builder-a-sidecar-a') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'builder-a-sidecar-a') === 'active' ? '#FBBF24' : EDGE_IDLE,
        },
      },
      // Builder B -> Sidecar B (polling)
      {
        id: 'builder-b-sidecar-b',
        source: 'builder-b',
        target: 'sidecar-b',
        animated: getEdgeStatus(step, 'builder-b-sidecar-b') === 'active',
        label: 'polls /tx',
        labelStyle: getEdgeStatus(step, 'builder-b-sidecar-b') === 'active'
          ? LSTYLE_ACTIVE('#FBBF24')
          : LSTYLE('#FBBF24'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'builder-b-sidecar-b') === 'active' ? '#FBBF24' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'builder-b-sidecar-b') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'builder-b-sidecar-b') === 'active' ? '#FBBF24' : EDGE_IDLE,
        },
      },
      // Sidecar A -> Builder A (delivering)
      {
        id: 'sidecar-a-builder-a',
        source: 'sidecar-a',
        target: 'builder-a',
        type: 'smoothstep',
        pathOptions: { offset: -20 },
        animated: getEdgeStatus(step, 'sidecar-a-builder-a') === 'active',
        label: 'deliver tx',
        labelStyle: getEdgeStatus(step, 'sidecar-a-builder-a') === 'active'
          ? LSTYLE_ACTIVE('#C084FC')
          : LSTYLE('#C084FC'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-a-builder-a') === 'active' ? '#C084FC' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-a-builder-a') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-a-builder-a') === 'active' ? '#C084FC' : EDGE_IDLE,
        },
      },
      // Sidecar B -> Builder B (delivering)
      {
        id: 'sidecar-b-builder-b',
        source: 'sidecar-b',
        target: 'builder-b',
        type: 'smoothstep',
        pathOptions: { offset: -20 },
        animated: getEdgeStatus(step, 'sidecar-b-builder-b') === 'active',
        label: 'deliver tx',
        labelStyle: getEdgeStatus(step, 'sidecar-b-builder-b') === 'active'
          ? LSTYLE_ACTIVE('#C084FC')
          : LSTYLE('#C084FC'),
        labelBgStyle: LBG,
        style: {
          stroke: getEdgeStatus(step, 'sidecar-b-builder-b') === 'active' ? '#C084FC' : EDGE_IDLE,
          strokeWidth: getEdgeStatus(step, 'sidecar-b-builder-b') === 'active' ? 2.5 : 1.5,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getEdgeStatus(step, 'sidecar-b-builder-b') === 'active' ? '#C084FC' : EDGE_IDLE,
        },
      },
    ],
    [highlightNormal, highlightXt, step]
  )

  const onInit = useCallback(() => {
    // Fit view on init
  }, [])

  const onEdgeClick = useCallback(
    (_event: MouseEvent, edge: Edge) => {
      if (!onSelectFlow) {
        return
      }
      if (edge.id === 'user-builder-a' || edge.id === 'user-builder-b') {
        onSelectFlow('normal')
      }
      if (edge.id === 'user-sidecar-a' || edge.id === 'user-sidecar-b') {
        onSelectFlow('xt')
      }
    },
    [onSelectFlow]
  )

  if (isFullscreen) {
    return (
      <>
        {/* Backdrop */}
        <div
          className="fixed inset-0 z-[9999] bg-black/60 backdrop-blur-sm flex items-center justify-center p-8"
          onClick={() => setIsFullscreen(false)}
        >
          {/* Diagram Container */}
          <div
            className="w-full h-full max-w-[95vw] max-h-[95vh] shadow-2xl border border-border relative overflow-hidden"
            style={{ background: '#0A0A0C' }}
            onClick={(e) => e.stopPropagation()}
          >
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onEdgeClick={onEdgeClick}
              onInit={onInit}
              fitView
              attributionPosition="bottom-left"
              proOptions={{ hideAttribution: true }}
              style={{ background: '#0A0A0C' }}
            >
              <Background color="#1A1A24" gap={32} />
              <Controls showInteractive={false} />
            </ReactFlow>

            {/* Current step indicator */}
            <div className="absolute bottom-6 left-6 bg-bg-card/95 border border-border px-4 py-3 z-10">
              <p className="text-[9px] font-display tracking-widest uppercase text-text-dim">Step</p>
              <p className="text-xs font-mono text-text-secondary capitalize">
                {step.replace(/_/g, ' ')}
              </p>
            </div>

            {/* Close button */}
            <button
              onClick={() => setIsFullscreen(false)}
              className="absolute top-6 right-6 z-10 bg-bg-card/95 border border-border px-4 py-2.5 hover:border-amber hover:text-amber transition-colors text-[10px] font-display tracking-widest uppercase text-text-secondary flex items-center gap-2"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="square">
                <path d="M18 6 6 18M6 6l12 12" />
              </svg>
              Close
            </button>

            {/* ESC hint */}
            <div className="absolute top-6 left-6 z-10 bg-bg-card/95 border border-border px-3 py-2 text-[10px] font-mono text-text-dim">
              <kbd className="font-display tracking-widest">ESC</kbd> to exit
            </div>
          </div>
        </div>
      </>
    )
  }

  return (
    <div className="w-full h-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onEdgeClick={onEdgeClick}
        onInit={onInit}
        fitView
        attributionPosition="bottom-left"
        proOptions={{ hideAttribution: true }}
        style={{ background: '#0D0D14' }}
      >
        <Background color="#252535" gap={32} />
        <Controls showInteractive={false} />
      </ReactFlow>

      {/* Fullscreen button */}
      <button
        onClick={() => setIsFullscreen(true)}
        className="absolute top-4 right-4 bg-bg-card/95 border border-border px-3 py-2 hover:border-amber hover:text-amber transition-colors text-[10px] font-display tracking-widest uppercase text-text-secondary flex items-center gap-2"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        >
          <path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3" />
        </svg>
        Fullscreen
      </button>
    </div>
  )
}
