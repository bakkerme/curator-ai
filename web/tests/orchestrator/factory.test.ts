import { afterEach, describe, expect, it } from 'vitest';
import { createScrapeOrchestrator } from '@/lib/orchestrator';
import { CodexCliScrapeOrchestrator } from '@/lib/orchestrator/codexCliRunner';
import { MockScrapeOrchestrator } from '@/lib/orchestrator/mockRunner';

describe('createScrapeOrchestrator', () => {
  afterEach(() => {
    delete process.env.SCRAPE_ORCHESTRATOR_MODE;
  });

  it('returns mock runner by default', () => {
    const runner = createScrapeOrchestrator();
    expect(runner).toBeInstanceOf(MockScrapeOrchestrator);
  });

  it('returns cli runner when configured', () => {
    process.env.SCRAPE_ORCHESTRATOR_MODE = 'cli';
    const runner = createScrapeOrchestrator();
    expect(runner).toBeInstanceOf(CodexCliScrapeOrchestrator);
  });
});
