export default function Home() {
  return (
    <div className="min-h-screen bg-gray-100">
      <div className="container mx-auto px-4 py-8">
        <header className="text-center mb-8">
          <h1 className="text-4xl font-bold text-gray-900 mb-2">
            Curator
          </h1>
          <p className="text-xl text-gray-600">
            Personal Intelligence Platform for Thought Leaders
          </p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">Pipeline Status</h2>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span>Status:</span>
                <span className="text-green-600">Healthy</span>
              </div>
              <div className="flex justify-between">
                <span>Last Run:</span>
                <span className="text-gray-500">N/A</span>
              </div>
              <div className="flex justify-between">
                <span>Next Run:</span>
                <span className="text-gray-500">N/A</span>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">System Health</h2>
            <div className="space-y-2">
              <div className="flex justify-between">
                <span>Service:</span>
                <span className="text-green-600">Running</span>
              </div>
              <div className="flex justify-between">
                <span>Version:</span>
                <span>0.1.0</span>
              </div>
              <div className="flex justify-between">
                <span>LLM Provider:</span>
                <span>OpenAI</span>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-semibold mb-4">Quick Actions</h2>
            <div className="space-y-3">
              <button className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 transition-colors">
                Run Pipeline
              </button>
              <button className="w-full bg-gray-600 text-white py-2 px-4 rounded hover:bg-gray-700 transition-colors">
                View Configuration
              </button>
              <button className="w-full bg-green-600 text-white py-2 px-4 rounded hover:bg-green-700 transition-colors">
                View Reports
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
