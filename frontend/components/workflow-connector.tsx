import type React from "react"
import type { WorkflowNode, WorkflowConnection } from "@/lib/workflow-types"

interface WorkflowConnectorProps {
  connection: WorkflowConnection
  nodes: WorkflowNode[]
  isRunning: boolean
}

export const WorkflowConnector: React.FC<WorkflowConnectorProps> = ({
  connection,
  nodes,
  isRunning
}) => {
  const fromNode = nodes.find(n => n.id === connection.from)
  const toNode = nodes.find(n => n.id === connection.to)

  if (!fromNode || !toNode) return null

  // Calculate connection points
  const x1 = fromNode.position.x + 160 // Card width (160px) for right edge
  const y1 = fromNode.position.y + 40  // Half card height for center
  const x2 = toNode.position.x         // Left edge of target card
  const y2 = toNode.position.y + 40    // Half card height for center

  // Calculate control points for smooth curves
  const controlX1 = x1 + Math.min(80, (x2 - x1) * 0.5)
  const controlX2 = x2 - Math.min(80, (x2 - x1) * 0.5)

  const pathData = `M ${x1} ${y1} C ${controlX1} ${y1}, ${controlX2} ${y2}, ${x2} ${y2}`

  return (
    <g>
      {/* Connection line with smooth curve */}
      <path
        d={pathData}
        stroke="rgba(255,255,255,0.4)"
        strokeWidth="2"
        fill="none"
        strokeDasharray={isRunning ? "5,5" : "none"}
        className={isRunning ? "animate-pulse" : ""}
        filter="drop-shadow(0 0 2px rgba(255,255,255,0.3))"
      />
      
      {/* Arrow head */}
      <polygon
        points={`${x2-8},${y2-4} ${x2},${y2} ${x2-8},${y2+4}`}
        fill="rgba(255,255,255,0.6)"
        filter="drop-shadow(0 0 2px rgba(255,255,255,0.3))"
      />
    </g>
  )
}