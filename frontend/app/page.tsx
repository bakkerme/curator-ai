"use client"

import type React from "react"

import { useState, useRef, useCallback, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { StarscapeBackground } from "@/components/starscape-background"
import { WorkflowConnector } from "@/components/workflow-connector"
import { useWorkflow } from "@/lib/use-workflow"
import { getStageIcon, getStageTypeColor } from "@/lib/workflow-icons"
import {
  Play,
  Pause,
  Settings,
  Plus,
  ZoomIn,
  ZoomOut,
  RotateCcw,
} from "lucide-react"


export default function CuratorWorkflow() {
  const { nodes: workflowNodes, connections, workflowName, isLoading } = useWorkflow()
  const [isRunning, setIsRunning] = useState(false)
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [zoom, setZoom] = useState(1)
  const [pan, setPan] = useState({ x: 0, y: 0 })
  const [isPanning, setIsPanning] = useState(false)
  const [lastPanPoint, setLastPanPoint] = useState({ x: 0, y: 0 })
  const canvasRef = useRef<HTMLDivElement>(null)

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

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? -0.1 : 0.1
    setZoom((prev) => Math.max(0.2, Math.min(3, prev + delta)))
  }, [])

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button === 0) {
      // Left mouse button
      setIsPanning(true)
      setLastPanPoint({ x: e.clientX, y: e.clientY })
    }
  }, [])

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (isPanning) {
        const deltaX = e.clientX - lastPanPoint.x
        const deltaY = e.clientY - lastPanPoint.y
        setPan((prev) => ({
          x: prev.x + deltaX,
          y: prev.y + deltaY,
        }))
        setLastPanPoint({ x: e.clientX, y: e.clientY })
      }
    },
    [isPanning, lastPanPoint],
  )

  const handleMouseUp = useCallback(() => {
    setIsPanning(false)
  }, [])

  const zoomIn = useCallback(() => {
    setZoom((prev) => Math.min(3, prev + 0.2))
  }, [])

  const zoomOut = useCallback(() => {
    setZoom((prev) => Math.max(0.2, prev - 0.2))
  }, [])

  const resetView = useCallback(() => {
    setZoom(1)
    setPan({ x: 0, y: 0 })
  }, [])

  useEffect(() => {
    const handleGlobalMouseUp = () => setIsPanning(false)
    document.addEventListener("mouseup", handleGlobalMouseUp)
    return () => document.removeEventListener("mouseup", handleGlobalMouseUp)
  }, [])

  return (
    <div className="min-h-screen bg-black text-white font-mono">
      {/* Header */}
      <div className="border-b border-white/10 bg-black/90 backdrop-blur-sm sticky top-0 z-10">
        <div className="mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
              <div className="flex items-baseline space-x-4">
              <img src="/curator-logo.svg" alt="Curator.ai" className="h-8 w-auto" />
              <div>
                <p className="text-sm text-gray-400 leading-[14px] ">Personal <br /> Intelligence Platform</p>
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
              <Button variant="outline" className="bg-transparent border-white/30 text-white hover:bg-white/10">
                <Settings className="w-4 h-4 mr-2" />
                CONFIG
              </Button>
            </div>
          </div>
        </div>
      </div>

      {/* Main Workflow Area - Full Screen */}
      <div className="relative h-[calc(100vh-73px)]">
        {" "}
        {/* 73px is header height */}
        {/* Workflow Title Overlay */}
        <div className="absolute top-6 left-6 z-30 pointer-events-none">
          <h2 className="text-3xl font-bold mb-2 text-white drop-shadow-lg">
            {isLoading ? "LOADING..." : workflowName.toUpperCase()}
          </h2>
          <p className="text-gray-400 drop-shadow-md">Transform scattered information into structured intelligence</p>
        </div>
        {/* Workflow Canvas */}
        <Card className="bg-black/40 border-white/10 w-full h-full relative overflow-hidden rounded-none border-0">
          {/* Zoom Controls */}
          <div className="absolute top-4 right-4 z-30 flex flex-col space-y-2">
            <Button
              onClick={zoomIn}
              size="sm"
              className="w-10 h-10 bg-black/60 hover:bg-black/80 border-white/20 text-white"
              variant="outline"
            >
              <ZoomIn className="w-4 h-4" />
            </Button>
            <Button
              onClick={zoomOut}
              size="sm"
              className="w-10 h-10 bg-black/60 hover:bg-black/80 border-white/20 text-white"
              variant="outline"
            >
              <ZoomOut className="w-4 h-4" />
            </Button>
            <Button
              onClick={resetView}
              size="sm"
              className="w-10 h-10 bg-black/60 hover:bg-black/80 border-white/20 text-white"
              variant="outline"
            >
              <RotateCcw className="w-4 h-4" />
            </Button>
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

          {/* Zoom Level Indicator */}
          <div className="absolute top-4 left-4 z-30">
            <Badge variant="outline" className="border-white/20 text-white bg-black/60">
              {Math.round(zoom * 100)}%
            </Badge>
          </div>

          {/* Infinite Starscape Background */}
          <StarscapeBackground
            transform={`translate(${pan.x}px, ${pan.y}px) scale(${zoom})`}
            transformOrigin="0 0"
            transition={isPanning ? "none" : "transform 0.1s ease-out"}
          />

          {/* Canvas Container */}
          <div
            ref={canvasRef}
            className="w-full h-full cursor-grab active:cursor-grabbing"
            onWheel={handleWheel}
            onMouseDown={handleMouseDown}
            onMouseMove={handleMouseMove}
            onMouseUp={handleMouseUp}
            style={{ cursor: isPanning ? "grabbing" : "grab" }}
          >
            {/* Transformable Content */}
            <div
              className="relative w-full h-full"
              style={{
                transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
                transformOrigin: "0 0",
                transition: isPanning ? "none" : "transform 0.1s ease-out",
              }}
            >

              {/* Connection Lines */}
              <svg className="absolute inset-0 w-full h-full pointer-events-none z-10">
                {connections.map((connection, index) => (
                  <WorkflowConnector
                    key={index}
                    connection={connection}
                    nodes={workflowNodes}
                    isRunning={isRunning}
                  />
                ))}
              </svg>

              {/* Workflow Nodes */}
              {workflowNodes.map((node) => {
                const NodeIcon = getStageIcon(node.type)
                return (
                  <div
                    key={node.id}
                    className={`absolute cursor-pointer transition-all duration-300 z-20 ${selectedNode === node.id ? "scale-105" : "hover:scale-102"
                      }`}
                    style={{
                      left: node.position.x,
                      top: node.position.y,
                      transform: selectedNode === node.id ? "scale(1.05)" : undefined,
                    }}
                    onClick={(e) => {
                      e.stopPropagation()
                      setSelectedNode(selectedNode === node.id ? null : node.id)
                    }}
                  >
                    <Card
                      className={`w-40 bg-black/80 backdrop-blur-sm border-2 ${getStageTypeColor(node.type)} hover:border-yellow-500/50 transition-all shadow-lg workflow-card`}
                      style={{
                        boxShadow:
                          selectedNode === node.id ? "0 0 20px rgba(255, 215, 0, 0.3)" : "0 4px 20px rgba(0, 0, 0, 0.5)",
                      }}
                    >
                      <div className="p-3">
                        <div className="flex items-center justify-between mb-2">
                          <NodeIcon className="w-5 h-5 text-yellow-400" />
                          <div className="flex flex-col items-end space-y-1">
                            <Badge className={`text-xs font-mono ${getStatusColor(node.status)}`}>
                              {node.status.toUpperCase()}
                            </Badge>
                            <Badge variant="outline" className="text-xs text-gray-400 border-gray-600">
                              {node.blockType}
                            </Badge>
                          </div>
                        </div>
                        <h3 className="font-bold text-xs mb-1">{node.title}</h3>
                        <p className="text-xs text-gray-400">{node.subtitle}</p>
                      </div>
                    </Card>
                  </div>
                )
              })}
            </div>
          </div>
        </Card>
      </div>
    </div>
  )
}
