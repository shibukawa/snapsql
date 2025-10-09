import JobTrigger from '@/components/JobTrigger';

export default function Home() {
  return (
    <div className="min-h-screen bg-gray-50 p-4 sm:p-6 md:p-8 lg:p-12">
      <main className="max-w-4xl mx-auto">
        {/* Page Header */}
        <div className="mb-6 md:mb-8">
          <h1 className="text-2xl sm:text-3xl lg:text-4xl font-bold mb-2 text-gray-900">
            Notification System Demo
          </h1>
          <p className="text-sm sm:text-base text-gray-600">
            Execute asynchronous jobs and observe the notification system in action.
          </p>
        </div>

        {/* Job Execution Card */}
        <div className="bg-white rounded-lg sm:rounded-xl shadow-sm sm:shadow-md p-4 sm:p-6 mb-6 md:mb-8">
          <h2 className="text-lg sm:text-xl font-semibold mb-3 sm:mb-4 text-gray-900">
            Execute Asynchronous Jobs
          </h2>
          <p className="text-xs sm:text-sm text-gray-600 mb-4 sm:mb-6">
            Click a button to execute a job in the background. You will receive a notification when it completes.
          </p>

          <div className="space-y-4 sm:space-y-5">
            <div className="border-b border-gray-100 pb-4 sm:pb-5 last:border-b-0 last:pb-0">
              <h3 className="text-sm font-medium text-gray-700 mb-2">Success Job</h3>
              <JobTrigger 
                type="success" 
                label="Run Job (Success)" 
              />
            </div>

            <div className="border-b border-gray-100 pb-4 sm:pb-5 last:border-b-0 last:pb-0">
              <h3 className="text-sm font-medium text-gray-700 mb-2">Error Job</h3>
              <JobTrigger 
                type="error" 
                label="Run Job (Error)" 
              />
            </div>

            <div>
              <h3 className="text-sm font-medium text-gray-700 mb-2">Fix Job</h3>
              <JobTrigger 
                type="fix" 
                label="Fix Last Error" 
              />
              <p className="text-xs text-gray-500 mt-2 italic">
                * Available after executing an error job
              </p>
            </div>
          </div>
        </div>

        {/* Instructions Card */}
        <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-lg sm:rounded-xl p-4 sm:p-6 border border-blue-100">
          <h2 className="text-base sm:text-lg font-semibold mb-3 sm:mb-4 text-gray-900 flex items-center gap-2">
            <svg className="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            How to Use
          </h2>
          <ol className="list-decimal list-inside space-y-2 text-xs sm:text-sm text-gray-700 leading-relaxed">
            <li className="pl-1">Click one of the buttons above to execute a job</li>
            <li className="pl-1">After 3 seconds, the job will complete and a notification will be created</li>
            <li className="pl-1">Check the notification icon in the top right to see new notifications</li>
            <li className="pl-1">Click on a notification to view its details</li>
          </ol>
        </div>
      </main>
    </div>
  );
}
