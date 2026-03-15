import { getRun, progressRunToReview } from '@/lib/runStore';

const encoder = new TextEncoder();

function sseData(payload: unknown): Uint8Array {
  return encoder.encode(`data: ${JSON.stringify(payload)}\n\n`);
}

/**
 * SSE endpoint that emits a short deterministic event sequence for the mock run.
 * It guards against stream cancellation to avoid enqueueing into a closed stream.
 */
export async function GET(_: Request, { params }: { params: { id: string } }) {
  const run = await getRun(params.id);

  if (!run) {
    return new Response('run not found', { status: 404 });
  }

  let isClosed = false;
  let timer: ReturnType<typeof setInterval> | undefined;

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      const events = [
        { step: 'queued', message: 'Run queued.' },
        { step: 'running', message: 'Loading target URL in browser...' },
        { step: 'running', message: 'Inspecting discovery selector candidates...' },
        { step: 'running', message: 'Inspecting extraction selector candidates...' },
        { step: 'needs_review', message: 'Sample extraction complete.' }
      ];

      let index = 0;
      timer = setInterval(() => {
        if (isClosed) {
          clearInterval(timer);
          return;
        }

        if (index >= events.length) {
          void progressRunToReview(params.id);
          isClosed = true;
          controller.close();
          clearInterval(timer);
          return;
        }

        controller.enqueue(sseData(events[index]));
        index += 1;
      }, 450);
    },
    cancel() {
      // The controller may be cancelled by the browser during navigations.
      // We stop the timer to prevent writes into a closed stream.
      isClosed = true;
      if (timer) {
        clearInterval(timer);
      }
    }
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive'
    }
  });
}
