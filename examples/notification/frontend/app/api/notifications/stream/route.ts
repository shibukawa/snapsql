import { NextRequest } from 'next/server';

const BACKEND_API_URL = process.env.BACKEND_API_URL || 'http://localhost:8080';
const POLLING_INTERVAL = 5000; // 5 seconds
const KEEP_ALIVE_INTERVAL = 30000; // 30 seconds

export async function GET(request: NextRequest) {
  const userId = process.env.NEXT_PUBLIC_USER_ID || 'EMP001';
  
  // Create a readable stream for SSE
  const encoder = new TextEncoder();
  let lastUpdatedAt: string | null = null;
  let pollingIntervalId: NodeJS.Timeout | null = null;
  let keepAliveIntervalId: NodeJS.Timeout | null = null;

  const stream = new ReadableStream({
    async start(controller) {
      console.log(`SSE connection established for user ${userId}`);

      // Send initial connection message
      controller.enqueue(encoder.encode(': connected\n\n'));

      // Function to poll Go backend for new notifications
      const pollBackend = async () => {
        try {
          let url = `${BACKEND_API_URL}/api/users/${userId}/notifications`;
          if (lastUpdatedAt) {
            url += `?since=${encodeURIComponent(lastUpdatedAt)}`;
            console.log(`[POLLING] Requesting: ${url}`);
            console.log(`[POLLING] lastUpdatedAt: ${lastUpdatedAt}`);
          } else {
            console.log(`[POLLING] Initial request (no since parameter): ${url}`);
          }

          const response = await fetch(url);
          
          if (!response.ok) {
            console.error('[POLLING] Failed to poll backend:', response.statusText);
            return;
          }

          const notifications = await response.json();
          
          console.log(`[POLLING] Received ${notifications.length} notification(s) from backend`);
          
          if (Array.isArray(notifications) && notifications.length > 0) {
            console.log(`[POLLING] Notifications:`, notifications.map((n: any) => ({
              id: n.id,
              title: n.title,
              created_at: n.created_at,
              updated_at: n.updated_at,
              read_at: n.read_at
            })));
            
            // Send each notification as an SSE event
            for (const notification of notifications) {
              const event = {
                type: 'notification',
                payload: notification
              };
              
              console.log(`[POLLING] Sending notification ${notification.id} to client via SSE`);
              controller.enqueue(
                encoder.encode(`data: ${JSON.stringify(event)}\n\n`)
              );
            }

            // Update lastUpdatedAt to the latest from notifications
            const latestUpdatedAt = notifications.reduce((latest: string | null, n: any) => {
              const updatedAt = n.updated_at || n.created_at;
              if (!latest || new Date(updatedAt) > new Date(latest)) {
                return updatedAt;
              }
              return latest;
            }, lastUpdatedAt);
            
            console.log(`[POLLING] Previous lastUpdatedAt: ${lastUpdatedAt}`);
            console.log(`[POLLING] New lastUpdatedAt: ${latestUpdatedAt}`);
            lastUpdatedAt = latestUpdatedAt;
          } else {
            console.log(`[POLLING] No new notifications`);
          }
        } catch (error) {
          console.error('[POLLING] Error polling backend:', error);
        }
      };

      // Start polling
      await pollBackend(); // Initial poll
      pollingIntervalId = setInterval(pollBackend, POLLING_INTERVAL);

      // Send keep-alive comments
      keepAliveIntervalId = setInterval(() => {
        try {
          controller.enqueue(encoder.encode(': keep-alive\n\n'));
        } catch (error) {
          console.error('Error sending keep-alive:', error);
        }
      }, KEEP_ALIVE_INTERVAL);
    },

    cancel() {
      console.log(`SSE connection closed for user ${userId}`);
      
      // Clean up intervals
      if (pollingIntervalId) {
        clearInterval(pollingIntervalId);
      }
      if (keepAliveIntervalId) {
        clearInterval(keepAliveIntervalId);
      }
    }
  });

  // Return SSE response
  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
    },
  });
}
