import { afterEach, describe, expect, it } from 'vitest';
import { createScrapeOrchestrator } from '@/lib/orchestrator';
import { CodexSdkScrapeOrchestrator } from '@/lib/orchestrator/codexSdkRunner';
import { MockScrapeOrchestrator } from '@/lib/orchestrator/mockRunner';

describe('createScrapeOrchestrator', () => {
  afterEach(() => {
    delete process.env.SCRAPE_ORCHESTRATOR_MODE;
  });

  it('returns mock runner by default', () => {
    const runner = createScrapeOrchestrator();
    expect(runner).toBeInstanceOf(MockScrapeOrchestrator);
  });

  it('returns sdk runner when configured', () => {
    process.env.SCRAPE_ORCHESTRATOR_MODE = 'sdk';
    const runner = createScrapeOrchestrator();
    expect(runner).toBeInstanceOf(CodexSdkScrapeOrchestrator);
  });
});
