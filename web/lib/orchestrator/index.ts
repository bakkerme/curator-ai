import { CodexCliScrapeOrchestrator } from './codexCliRunner';
import { MockScrapeOrchestrator } from './mockRunner';
import type { ScrapeOrchestrator } from './types';

/**
 * Chooses runner implementation. Default remains mock for local stability.
 * Set SCRAPE_ORCHESTRATOR_MODE=cli to run Codex CLI.
 */
export function createScrapeOrchestrator(): ScrapeOrchestrator {
  if (process.env.SCRAPE_ORCHESTRATOR_MODE === 'cli') {
    return new CodexCliScrapeOrchestrator();
  }

  return new MockScrapeOrchestrator();
}
