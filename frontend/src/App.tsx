import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity, Settings, PlayCircle, BarChart3 } from 'lucide-react'
import { apiClient } from './services/api'

function App() {
  const [activeTab, setActiveTab] = useState<'status' | 'pipeline' | 'analytics'>('status')

  const { data: status, isLoading } = useQuery({
    queryKey: ['status'],
    queryFn: () => apiClient.getStatus(),
    refetchInterval: 30000, // Refresh every 30 seconds
  })

  const { data: pipelineStatus } = useQuery({
    queryKey: ['pipeline-status'],
    queryFn: () => apiClient.getPipelineStatus(),
    refetchInterval: 10000, // Refresh every 10 seconds
  })

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center space-x-4">
              <Activity className="h-8 w-8 text-primary-600" />
              <h1 className="text-xl font-bold text-gray-900">Curator</h1>
              <span className="text-sm text-gray-500">Personal Intelligence Platform</span>
            </div>
            
            <div className="flex items-center space-x-4">
              {isLoading ? (
                <div className="w-3 h-3 bg-gray-400 rounded-full animate-pulse" />
              ) : (
                <div className={`w-3 h-3 rounded-full ${
                  status?.status === 'running' ? 'bg-green-400' : 'bg-red-400'
                }`} />
              )}
              <span className="text-sm text-gray-600">
                {status?.status || 'Unknown'}
              </span>
            </div>
          </div>
        </div>
      </header>

      {/* Navigation */}
      <nav className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex space-x-8">
            {[
              { key: 'status', label: 'Status', icon: Activity },
              { key: 'pipeline', label: 'Pipeline', icon: Settings },
              { key: 'analytics', label: 'Analytics', icon: BarChart3 },
            ].map(({ key, label, icon: Icon }) => (
              <button
                key={key}
                onClick={() => setActiveTab(key as any)}
                className={`flex items-center space-x-2 px-3 py-4 text-sm font-medium border-b-2 ${
                  activeTab === key
                    ? 'border-primary-600 text-primary-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                <Icon className="h-4 w-4" />
                <span>{label}</span>
              </button>
            ))}
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeTab === 'status' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {/* System Status */}
              <div className="card">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">System Status</h3>
                <div className="space-y-3">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Service</span>
                    <span className={`font-medium ${
                      status?.status === 'running' ? 'text-green-600' : 'text-red-600'
                    }`}>
                      {status?.status || 'Unknown'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Version</span>
                    <span className="font-medium">{status?.version || 'N/A'}</span>
                  </div>
                </div>
              </div>

              {/* LLM Configuration */}
              <div className="card">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">LLM Configuration</h3>
                <div className="space-y-3">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Provider</span>
                    <span className="font-medium">{status?.config?.llm_provider || 'N/A'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Endpoint</span>
                    <span className="font-medium text-sm">{status?.config?.llm_endpoint || 'N/A'}</span>
                  </div>
                </div>
              </div>

              {/* Pipeline Status */}
              <div className="card">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Pipeline Status</h3>
                <div className="space-y-3">
                  <div className="flex justify-between">
                    <span className="text-gray-600">Status</span>
                    <span className="font-medium">{pipelineStatus?.status || 'Unknown'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-600">Last Run</span>
                    <span className="font-medium">{pipelineStatus?.last_run || 'Never'}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'pipeline' && (
          <div className="space-y-6">
            <div className="flex justify-between items-center">
              <h2 className="text-2xl font-bold text-gray-900">Pipeline Configuration</h2>
              <button className="btn-primary flex items-center space-x-2">
                <PlayCircle className="h-4 w-4" />
                <span>Run Pipeline</span>
              </button>
            </div>
            
            <div className="card">
              <p className="text-gray-600">Pipeline configuration interface coming soon...</p>
            </div>
          </div>
        )}

        {activeTab === 'analytics' && (
          <div className="space-y-6">
            <h2 className="text-2xl font-bold text-gray-900">Analytics & Performance</h2>
            
            <div className="card">
              <p className="text-gray-600">Analytics dashboard coming soon...</p>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}

export default App