import { parse } from 'yaml'
import type { WorkflowConfig, WorkflowStage, WorkflowNode, WorkflowConnection, StageType, BlockType } from './workflow-types'

export class WorkflowParser {
  private config: WorkflowConfig
  private nodes: Map<string, WorkflowNode> = new Map()
  private connections: WorkflowConnection[] = []

  constructor(yamlContent: string) {
    this.config = parse(yamlContent).workflow
    this.parseWorkflow()
  }

  private parseWorkflow() {
    // First pass: create all nodes
    Object.entries(this.config.stages).forEach(([stageId, stage]) => {
      const node = this.createNode(stageId, stage)
      this.nodes.set(stageId, node)
    })

    // Second pass: create connections and calculate layout
    this.createConnections()
    this.calculateLayout()
  }

  private createNode(id: string, stage: WorkflowStage): WorkflowNode {
    return {
      id,
      type: stage.type,
      title: this.formatTitle(id),
      subtitle: this.getSubtitle(stage),
      status: id === 'redditSrc' ? 'active' : 'idle',
      position: { x: 0, y: 0 }, // Will be calculated later
      blockType: this.getPrimaryBlockType(stage),
      config: stage.config,
      connections: stage.next || []
    }
  }

  private createConnections() {
    Object.entries(this.config.stages).forEach(([fromId, stage]) => {
      // Handle direct next connections
      if (stage.next) {
        stage.next.forEach(toId => {
          this.connections.push({ from: fromId, to: toId })
        })
      }

      // Handle router connections
      if (stage.routes) {
        stage.routes.forEach(route => {
          if (route.to && route.to !== 'drop') {
            this.connections.push({ from: fromId, to: route.to })
          }
        })
      }
    })
  }

  private calculateLayout() {
    const levels = new Map<string, number>()
    const columns = new Map<number, string[]>()

    // Find root nodes (sources)
    const rootNodes = Array.from(this.nodes.values()).filter(node => 
      node.type === 'source' || !this.connections.some(conn => conn.to === node.id)
    )

    // Initialize all nodes with level 0
    Array.from(this.nodes.keys()).forEach(nodeId => {
      levels.set(nodeId, 0)
    })

    // Calculate maximum depth for each node using iterative approach
    let changed = true
    while (changed) {
      changed = false
      
      this.connections.forEach(conn => {
        const fromLevel = levels.get(conn.from) || 0
        const currentToLevel = levels.get(conn.to) || 0
        const newToLevel = fromLevel + 1
        
        if (newToLevel > currentToLevel) {
          levels.set(conn.to, newToLevel)
          changed = true
        }
      })
    }

    // Group nodes by their final levels
    levels.forEach((level, nodeId) => {
      if (!columns.has(level)) {
        columns.set(level, [])
      }
      columns.get(level)!.push(nodeId)
    })

    // Position nodes
    const COLUMN_WIDTH = 280
    const ROW_HEIGHT = 120
    const START_X = 80
    const START_Y = 80

    columns.forEach((nodeIds, level) => {
      const x = START_X + (level * COLUMN_WIDTH)
      
      // Sort nodes in column by type and dependencies for better visual flow
      const sortedNodes = this.sortNodesInColumn(nodeIds)
      
      sortedNodes.forEach((nodeId, index) => {
        const node = this.nodes.get(nodeId)!
        node.position = {
          x,
          y: START_Y + (index * ROW_HEIGHT)
        }
      })
    })
  }

  private sortNodesInColumn(nodeIds: string[]): string[] {
    // Sort by type priority and then by number of incoming connections
    const typeOrder: Record<StageType, number> = {
      'source': 0,
      'router': 1,
      'processor': 2,
      'formatter': 3,
      'batch_join': 4,
      'destination': 5
    }

    return nodeIds.sort((a, b) => {
      const nodeA = this.nodes.get(a)!
      const nodeB = this.nodes.get(b)!
      
      const typeOrderA = typeOrder[nodeA.type] || 999
      const typeOrderB = typeOrder[nodeB.type] || 999
      
      if (typeOrderA !== typeOrderB) {
        return typeOrderA - typeOrderB
      }

      // If same type, sort by incoming connections count
      const incomingA = this.connections.filter(c => c.to === a).length
      const incomingB = this.connections.filter(c => c.to === b).length
      
      return incomingA - incomingB
    })
  }

  private formatTitle(id: string): string {
    // Convert camelCase to Title Case
    return id
      .replace(/([A-Z])/g, ' $1')
      .replace(/^./, str => str.toUpperCase())
      .trim()
  }

  private getSubtitle(stage: WorkflowStage): string {
    if (stage.config?.subreddit) {
      return `r/${stage.config.subreddit} posts`
    }
    if (stage.use) {
      return stage.use.replace(/_/g, ' ')
    }
    if (stage.type === 'router') {
      return 'Route Elements'
    }
    return stage.type.charAt(0).toUpperCase() + stage.type.slice(1)
  }

  private getPrimaryBlockType(stage: WorkflowStage): BlockType {
    if (stage.produces && stage.produces.length > 0) {
      return stage.produces[0]
    }
    if (stage.operates_on && stage.operates_on.length > 0) {
      return stage.operates_on[0]
    }
    return 'ObjectBlock'
  }

  public getNodes(): WorkflowNode[] {
    return Array.from(this.nodes.values())
  }

  public getConnections(): WorkflowConnection[] {
    return this.connections
  }

  public getWorkflowName(): string {
    return this.config.name
  }
}