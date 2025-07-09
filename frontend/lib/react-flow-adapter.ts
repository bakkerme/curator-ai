import type { Node, Edge } from '@xyflow/react'
import type { WorkflowNode, WorkflowConnection, StageType, BlockType } from './workflow-types'

export interface WorkflowNodeData extends Record<string, unknown> {
  title: string
  subtitle: string
  status: 'active' | 'processing' | 'idle'
  stageType: StageType
  blockType: BlockType
}

export interface WorkflowEdgeData extends Record<string, unknown> {
  isRunning?: boolean
}

export type ReactFlowWorkflowNode = Node<WorkflowNodeData, 'workflow'>
export type ReactFlowWorkflowEdge = Edge<WorkflowEdgeData, 'workflow'>

export function transformNodesToReactFlow(
  workflowNodes: WorkflowNode[]
): ReactFlowWorkflowNode[] {
  return workflowNodes.map(node => ({
    id: node.id,
    type: 'workflow',
    position: node.position,
    data: {
      title: node.title,
      subtitle: node.subtitle,
      status: node.status,
      stageType: node.type,
      blockType: node.blockType
    },
    draggable: true,
    selectable: true
  }))
}

export function transformConnectionsToReactFlow(
  connections: WorkflowConnection[],
  isRunning: boolean = false
): ReactFlowWorkflowEdge[] {
  return connections.map(connection => ({
    id: `${connection.from}-${connection.to}`,
    source: connection.from,
    target: connection.to,
    type: 'workflow',
    data: {
      isRunning
    },
    animated: isRunning
  }))
}

export function createReactFlowData(
  workflowNodes: WorkflowNode[],
  connections: WorkflowConnection[],
  isRunning: boolean = false
) {
  return {
    nodes: transformNodesToReactFlow(workflowNodes),
    edges: transformConnectionsToReactFlow(connections, isRunning)
  }
}