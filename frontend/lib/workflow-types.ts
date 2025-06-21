export type BlockType = 'TextBlock' | 'ObjectBlock' | 'ImageBlock' | 'BinaryBlock' | 'AudioBlock' | 'DocumentBlock'

export type StageType = 'source' | 'processor' | 'router' | 'formatter' | 'destination' | 'batch_join'

export interface WorkflowStage {
  id: string
  type: StageType
  dataID?: string
  use?: string
  produces?: BlockType[]
  operates_on?: BlockType[]
  config?: Record<string, any>
  next?: string[]
  routes?: Array<{
    when?: string
    extract?: string
    to?: string
    else?: string
  }>
  selector_map?: Record<string, string>
  prompt_template?: string
  schema?: string
  sources?: string
  max_wait_sec?: number
  template?: string
}

export interface WorkflowConfig {
  name: string
  stages: Record<string, WorkflowStage>
}

export interface WorkflowNode {
  id: string
  type: StageType
  title: string
  subtitle: string
  status: 'active' | 'processing' | 'idle'
  position: { x: number; y: number }
  blockType: BlockType
  config?: Record<string, any>
  connections: string[]
}

export interface WorkflowConnection {
  from: string
  to: string
}