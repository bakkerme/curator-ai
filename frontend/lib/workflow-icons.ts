import {
  Database,
  Filter,
  Zap,
  Brain,
  Target,
  Sparkles,
  Mail,
  FileText,
  Image,
  Binary,
  Volume2,
  File
} from "lucide-react"
import type { StageType, BlockType } from './workflow-types'

export const getStageIcon = (type: StageType) => {
  switch (type) {
    case 'source':
      return Database
    case 'router':
      return Filter
    case 'processor':
      return Brain
    case 'formatter':
      return Sparkles
    case 'destination':
      return Mail
    case 'batch_join':
      return Target
    default:
      return Zap
  }
}

export const getBlockTypeIcon = (blockType: BlockType) => {
  switch (blockType) {
    case 'TextBlock':
      return FileText
    case 'ObjectBlock':
      return Target
    case 'ImageBlock':
      return Image
    case 'BinaryBlock':
      return Binary
    case 'AudioBlock':
      return Volume2
    case 'DocumentBlock':
      return File
    default:
      return FileText
  }
}

export const getStageTypeColor = (type: StageType) => {
  switch (type) {
    case 'source':
      return "border-yellow-500/70"
    case 'destination':
      return "border-yellow-500/70"
    case 'processor':
      return "border-blue-500/50"
    case 'router':
      return "border-purple-500/50"
    case 'formatter':
      return "border-green-500/50"
    case 'batch_join':
      return "border-orange-500/50"
    default:
      return "border-white/30"
  }
}