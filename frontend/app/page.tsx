"use client"

import type React from "react"

import Image from "next/image"
import { useCallback, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { StarscapeBackground } from "@/components/starscape-background"
import { useWorkflow } from "@/lib/use-workflow"
import WorkflowNodeComponent from "@/components/workflow-node"
import WorkflowEdgeComponent from "@/components/workflow-edge"
import {
  ReactFlow,
  Controls,
  Background,
  BackgroundVariant,
  useNodesState,
  useEdgesState,
  addEdge,
  type Connection,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import {
  Play,
  Pause,
  Settings,
  Plus,
} from "lucide-react"


// Define node and edge types for React Flow
const nodeTypes = {
  workflow: WorkflowNodeComponent,
}

const edgeTypes = {
  workflow: WorkflowEdgeComponent,
}

export default function CuratorWorkflow() {
  const { 
    reactFlowNodes, 
    reactFlowEdges, 
    workflowName, 
    workflowDescription, 
    isLoading,
    isRunning,
    setIsRunning
  } = useWorkflow()
  
  const [nodes, setNodes, onNodesChange] = useNodesState(reactFlowNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(reactFlowEdges)

  // Sync React Flow state with workflow data
  useEffect(() => {
    setNodes(reactFlowNodes)
  }, [reactFlowNodes, setNodes])

  useEffect(() => {
    setEdges(reactFlowEdges)
  }, [reactFlowEdges, setEdges])

  const onConnect = useCallback((connection: Connection) => {
    setEdges((eds) => addEdge(connection, eds))
  }, [setEdges])


  return (
    <div className="min-h-screen bg-black text-slate-100 font-mono">
      {/* Header */}
      <div className="border-b border-slate-100/10 bg-black/90 backdrop-blur-sm sticky top-0 z-10">
        <div className="mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
              <Image width="150" height="31" src="/curator-filled.svg" alt="curator.ai" className="h-8 w-auto text-white" />
               
              <div>
                <p className="text-sm text-gray-400 leading-[14px] mb-[-8px]">Personal <br /> Intelligence Platform</p>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <Badge variant="outline" className="border-yellow-500/50 text-yellow-400">
                v0.1.0
              </Badge>
              <Button
                onClick={() => setIsRunning(!isRunning)}
                className={`${isRunning
                  ? "bg-red-500/20 hover:bg-red-500/30 border-red-500/50 text-red-400"
                  : "bg-yellow-500/20 hover:bg-yellow-500/30 border-yellow-500/50 text-yellow-400"
                  } border font-mono`}
                variant="outline"
              >
                {isRunning ? <Pause className="w-4 h-4 mr-2" /> : <Play className="w-4 h-4 mr-2" />}
                {isRunning ? "PAUSE" : "RUN"}
              </Button>
              <Button variant="outline" className="bg-transparent border-slate-100/30 text-slate-100 hover:bg-slate-100/10">
                <Settings className="w-4 h-4 mr-2" />
                CONFIG
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Workflow Area - Full Screen */}
      <div className="relative h-[calc(100vh-73px)]">
        {/* Workflow Title Overlay */}
        <div className="absolute top-6 left-6 z-30 pointer-events-none">
          <h2 className="text-3xl font-bold mb-2 text-slate-100 drop-shadow-lg">
            {isLoading ? "LOADING..." : workflowName}
          </h2>
          <p className="text-gray-400 drop-shadow-md">
            {isLoading ? "Loading workflow..." : workflowDescription || "Transform scattered information into structured intelligence"}
          </p>
        </div>

        {/* Add Node Button - Overlay */}
        <Button
          className="absolute bottom-8 right-8 w-12 h-12 rounded-full bg-yellow-500/20 hover:bg-yellow-500/30 border-2 border-yellow-500/50 text-yellow-400 z-30 shadow-lg"
          variant="outline"
          style={{
            boxShadow: "0 0 15px rgba(255, 215, 0, 0.2)",
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <Plus className="w-6 h-6" />
        </Button>

        {/* React Flow Canvas */}
        <div className="w-full h-full bg-black/40 relative overflow-hidden">
          <ReactFlow
            nodes={nodes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onConnect={onConnect} 
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            className="bg-transparent"
            fitView
            attributionPosition="bottom-left"
          >
            <Controls position="top-right" />
            <div className="react-flow__background-layer absolute inset-0 pointer-events-none">
              <StarscapeBackground opacity={0.6} />
            </div>
          </ReactFlow>
        </div>
      </div>
    </div>
  )
}
