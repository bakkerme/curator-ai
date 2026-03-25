import { describe, expect, it } from 'vitest';
import { MockScrapeOrchestrator } from '@/lib/orchestrator/mockRunner';

describe('MockScrapeOrchestrator', () => {
  it('emits status events and completes with a proposal', async () => {
    const runner = new MockScrapeOrchestrator();
    const events = [];

    for await (const event of runner.runScrapeDiscovery({
      runId: 'run-1',
      targetUrl: 'https://example.com/blog'
    })) {
      events.push(event);
    }

    expect(events.length).toBe(8);
    expect(events.some((event) => event.type === 'output')).toBe(true);
    expect(events.at(-1)?.type).toBe('complete');
    if (events.at(-1)?.type === 'complete') {
      expect(events.at(-1)?.proposal.generatedYaml).toContain('sources:');
    }
  });
});
