import { getRun, progressRunToReview } from '@/lib/runStore';

const encoder = new TextEncoder();

function sseData(payload: unknown): Uint8Array {
  return encoder.encode(`data: ${JSON.stringify(payload)}\n\n`);
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * SSE endpoint that emits a short deterministic event sequence for the mock run.
 * It updates run status before emitting the terminal needs_review event so the
 * client unlock state and backend status stay consistent.
 */
export async function GET(_: Request, { params }: { params: { id: string } }) {
  const run = await getRun(params.id);

  if (!run) {
    return new Response('run not found', { status: 404 });
  }

  let isClosed = false;

  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      const events = [
        { step: 'queued', message: 'Run queued.' },
        { step: 'running', message: 'Loading target URL in browser...' },
        { step: 'running', message: 'Inspecting discovery selector candidates...' },
        { step: 'running', message: 'Inspecting extraction selector candidates...' }
      ];

      for (const event of events) {
        if (isClosed) {
          return;
        }
        controller.enqueue(sseData(event));
        await sleep(450);
      }

      if (isClosed) {
        return;
      }

      // Persist status before sending final event to prevent unlock/status races.
      await progressRunToReview(params.id);

      if (isClosed) {
        return;
      }

      controller.enqueue(sseData({ step: 'needs_review', message: 'Sample extraction complete.' }));
      controller.close();
    },
    cancel() {
      // The controller may be cancelled by the browser during navigations.
      isClosed = true;
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
