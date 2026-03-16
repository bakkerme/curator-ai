import type { ScrapeProposal } from '@/lib/schemas';

export type ScrapeDiscoveryInput = {
  runId: string;
  targetUrl: string;
};

export type OrchestratorEvent =
  | { type: 'status'; step: 'queued' | 'running'; message: string }
  | { type: 'complete'; step: 'needs_review'; message: string; proposal: ScrapeProposal }
  | { type: 'failed'; step: 'failed'; message: string };

export interface ScrapeOrchestrator {
  runScrapeDiscovery(input: ScrapeDiscoveryInput): AsyncGenerator<OrchestratorEvent>;
}
