import { createScrapeOrchestrator } from '@/lib/orchestrator';
import { appendRunLog, getRun, setRunProposal, setRunStatus } from '@/lib/runStore';

const encoder = new TextEncoder();

function sseData(payload: unknown): Uint8Array {
  return encoder.encode(`data: ${JSON.stringify(payload)}\n\n`);
}

/**
 * SSE endpoint that streams live orchestrator events.
 * Uses the configured runner (mock or CLI) so the UI can be tested against the
 * same event contract that future real agent execution will use.
 */
export async function GET(_: Request, { params }: { params: { id: string } }) {
  const run = await getRun(params.id);

  if (!run) {
    return new Response('run not found', { status: 404 });
  }

  const orchestrator = createScrapeOrchestrator();
  let isClosed = false;

  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      try {
        for await (const event of orchestrator.runScrapeDiscovery({
          runId: run.id,
          targetUrl: run.targetUrl
        })) {
          if (isClosed) {
            return;
          }

          controller.enqueue(sseData({ step: event.step, message: event.message }));

          if (event.type === 'status') {
            if (event.step === 'running') {
              await setRunStatus(run.id, 'running');
            }
            await appendRunLog(run.id, event.message);
          }

          if (event.type === 'complete') {
            await setRunProposal(run.id, event.proposal);
            await appendRunLog(run.id, event.message);
            controller.close();
            return;
          }

          if (event.type === 'failed') {
            await setRunStatus(run.id, 'failed');
            await appendRunLog(run.id, event.message);
            controller.close();
            return;
          }
        }

        controller.close();
      } catch (error) {
        await setRunStatus(run.id, 'failed');
        const message = error instanceof Error ? error.message : 'Unknown orchestrator error';
        await appendRunLog(run.id, message);
        if (!isClosed) {
          controller.enqueue(sseData({ step: 'failed', message }));
          controller.close();
        }
      }
    },
    cancel() {
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
