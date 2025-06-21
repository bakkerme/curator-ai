import { useState, useEffect } from 'react'
import { WorkflowParser } from './workflow-parser'
import type { WorkflowNode, WorkflowConnection } from './workflow-types'

export function useWorkflow() {
  const [nodes, setNodes] = useState<WorkflowNode[]>([])
  const [connections, setConnections] = useState<WorkflowConnection[]>([])
  const [workflowName, setWorkflowName] = useState<string>('')
  const [workflowDescription, setWorkflowDescription] = useState<string>('')
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const loadWorkflow = async () => {
      try {
        const response = await fetch('/config/workflow.yaml')
        if (!response.ok) {
          throw new Error(`Failed to load workflow: ${response.statusText}`)
        }
        const yamlContent = await response.text()
        
        const parser = new WorkflowParser(yamlContent)
        setNodes(parser.getNodes())
        setConnections(parser.getConnections())
        setWorkflowName(parser.getWorkflowName())
        setWorkflowDescription(parser.getWorkflowDescription())
      } catch (error) {
        console.error('Failed to parse workflow:', error)
      } finally {
        setIsLoading(false)
      }
    }

    loadWorkflow()
  }, [])

  return {
    nodes,
    connections,
    workflowName,
    workflowDescription,
    isLoading
  }
}