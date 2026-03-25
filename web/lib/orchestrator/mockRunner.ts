import { buildMockProposal } from '@/lib/mockProposal';
import type { OrchestratorEvent, ScrapeDiscoveryInput, ScrapeOrchestrator } from './types';

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Mock runner used during early UI development. This follows the same event
 * contract as the real CLI runner so the UI/state plumbing can remain stable.
 */
export class MockScrapeOrchestrator implements ScrapeOrchestrator {
  async *runScrapeDiscovery(input: ScrapeDiscoveryInput): AsyncGenerator<OrchestratorEvent> {
    yield { type: 'status', step: 'queued', message: 'Run queued.' };
    await sleep(350);

    yield { type: 'status', step: 'running', message: 'Loading target URL in browser...' };
    yield {
      type: 'output',
      step: 'running',
      stream: 'stderr',
      message: '[mock-agent] browser boot complete\n'
    };
    await sleep(450);

    yield { type: 'status', step: 'running', message: 'Inspecting discovery selector candidates...' };
    yield {
      type: 'output',
      step: 'running',
      stream: 'stdout',
      message: '[mock-agent] found 12 article cards on listing page\n'
    };
    await sleep(450);

    yield { type: 'status', step: 'running', message: 'Inspecting extraction selector candidates...' };
    yield {
      type: 'output',
      step: 'running',
      stream: 'stdout',
      message: '[mock-agent] sampled 3 detail pages for field extraction\n'
    };
    await sleep(450);

    yield {
      type: 'complete',
      step: 'needs_review',
      message: 'Sample extraction complete.',
      proposal: buildMockProposal(input.targetUrl)
    };
  }
}
