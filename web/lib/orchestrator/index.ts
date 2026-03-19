import { CodexSdkScrapeOrchestrator } from './codexSdkRunner';
import { MockScrapeOrchestrator } from './mockRunner';
import type { ScrapeOrchestrator } from './types';

/**
 * Chooses runner implementation. Default remains mock for local stability.
 * Set SCRAPE_ORCHESTRATOR_MODE=sdk to run Codex from the web server.
 */
export function createScrapeOrchestrator(): ScrapeOrchestrator {
  if (process.env.SCRAPE_ORCHESTRATOR_MODE === 'sdk') {
    return new CodexSdkScrapeOrchestrator();
  }

  return new MockScrapeOrchestrator();
}
