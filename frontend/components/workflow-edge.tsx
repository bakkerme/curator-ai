import type React from "react"
import type { EdgeProps } from '@xyflow/react'
import { BaseEdge, getBezierPath } from '@xyflow/react'
import type { WorkflowEdgeData } from '@/lib/react-flow-adapter'

function WorkflowEdgeComponent({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data
}: EdgeProps) {
  const workflowData = data as WorkflowEdgeData
  
  const [edgePath] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  return (
    <>
      <defs>
        <marker
          id="workflow-arrow"
          markerWidth="10"
          markerHeight="10"
          refX="8"
          refY="3"
          orient="auto"
          markerUnits="strokeWidth"
        >
          <polygon
            points="0,0 0,6 9,3"
            fill="rgba(255,255,255,0.6)"
            style={{filter: 'drop-shadow(0 0 2px rgba(255,255,255,0.3))'}}
          />
        </marker>
      </defs>
      
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          stroke: 'rgba(255,255,255,0.4)',
          strokeWidth: 2,
          filter: 'drop-shadow(0 0 2px rgba(255,255,255,0.3))'
        }}
        markerEnd="url(#workflow-arrow)"
        className={workflowData?.isRunning ? 'animate-pulse' : ''}
        strokeDasharray={workflowData?.isRunning ? '5,5' : undefined}
      />
    </>
  )
}

export default WorkflowEdgeComponent