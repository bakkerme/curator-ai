import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
})

export interface SystemStatus {
  status: string
  version: string
  config: {
    llm_provider: string
    llm_endpoint: string
  }
}

export interface PipelineStatus {
  status: string
  last_run: string | null
  next_run: string | null
}

export const apiClient = {
  async getHealth() {
    const response = await api.get('/health')
    return response.data
  },

  async getStatus(): Promise<SystemStatus> {
    const response = await api.get('/status')
    return response.data
  },

  async getPipelineStatus(): Promise<PipelineStatus> {
    const response = await api.get('/pipeline/status')
    return response.data
  },

  async getPipelineConfig() {
    const response = await api.get('/pipeline/config')
    return response.data
  },

  async updatePipelineConfig(config: any) {
    const response = await api.post('/pipeline/config', config)
    return response.data
  },

  async runPipeline() {
    const response = await api.post('/pipeline/run')
    return response.data
  },
}