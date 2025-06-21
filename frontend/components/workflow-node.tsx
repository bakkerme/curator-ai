import type React from "react"
import type { NodeProps } from '@xyflow/react'
import { Handle, Position } from '@xyflow/react'
import { Card } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { getStageIcon, getStageTypeColor } from '@/lib/workflow-icons'
import type { WorkflowNodeData } from '@/lib/react-flow-adapter'

function WorkflowNodeComponent({ data, selected }: NodeProps) {
  const workflowData = data as WorkflowNodeData
  const NodeIcon = getStageIcon(workflowData.stageType)
  
  const getStatusColor = (status: string) => {
    switch (status) {
      case "active":
        return "bg-yellow-500/20 border-yellow-500/50 text-yellow-400"
      case "processing":
        return "bg-blue-500/20 border-blue-500/50 text-blue-400"
      case "idle":
        return "bg-gray-500/20 border-gray-500/50 text-gray-400"
      default:
        return "bg-gray-500/20 border-gray-500/50 text-gray-400"
    }
  }

  return (
    <>
      {/* Connection handles - invisible but functional */}
      <Handle 
        type="target" 
        position={Position.Left} 
        className="opacity-0" 
        style={{ background: 'transparent' }}
      />
      <Handle 
        type="source" 
        position={Position.Right} 
        className="opacity-0"
        style={{ background: 'transparent' }}
      />
      
      <Card
        className={`w-40 bg-black/80 backdrop-blur-sm border-2 ${getStageTypeColor(workflowData.stageType)} 
          hover:border-yellow-500/50 transition-all shadow-lg workflow-card
          ${selected ? 'border-yellow-500/70' : ''}`}
        style={{
          boxShadow: selected 
            ? "0 0 20px rgba(255, 215, 0, 0.3)" 
            : "0 4px 20px rgba(0, 0, 0, 0.5)"
        }}
      >
        <div className="p-3">
          <div className="flex items-center justify-between mb-2">
            <NodeIcon className="w-5 h-5 text-yellow-400" />
            <div className="flex flex-col items-end space-y-1">
              <Badge className={`text-xs font-mono ${getStatusColor(workflowData.status)}`}>
                {workflowData.status.toUpperCase()}
              </Badge>
              <Badge variant="outline" className="text-xs text-gray-400 border-gray-600">
                {workflowData.blockType}
              </Badge>
            </div>
          </div>
          <h3 className="font-bold text-xs mb-1">{workflowData.title}</h3>
          <p className="text-xs text-gray-400">{workflowData.subtitle}</p>
        </div>
      </Card>
    </>
  )
}

export default WorkflowNodeComponent