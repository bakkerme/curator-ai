import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ScrapeProposal } from '@/lib/schemas';

const startThreadMock = vi.fn();
const codexConstructorMock = vi.fn(() => ({
  startThread: startThreadMock
}));

vi.mock('@openai/codex-sdk', () => ({
  Codex: codexConstructorMock
}));

function createMockProposal(): ScrapeProposal {
  return {
    discovery: {
      listPageSelector: { selector: '.post-card', confidence: 0.9 },
      itemLinkSelector: { selector: '.post-card a', confidence: 0.91 }
    },
    extraction: {
      title: { selector: 'h1', confidence: 0.94 },
      url: { selector: 'link[rel="canonical"]', confidence: 0.88 },
      author: { selector: '.author', confidence: 0.73 },
      publishedAt: { selector: 'time', confidence: 0.82 },
      summary: { selector: '.summary', confidence: 0.77 },
      content: { selector: 'article', confidence: 0.79 }
    },
    samples: [
      {
        title: 'Example title',
        url: 'https://example.com/post-1',
        author: 'Example author',
        publishedAt: '2026-03-18',
        summary: 'Example summary',
        contentSnippet: 'Example snippet'
      }
    ],
    generatedYaml: 'sources:\n  - kind: scrape',
    overallConfidence: 0.87
  };
}

describe('CodexSdkScrapeOrchestrator', () => {
  beforeEach(() => {
    startThreadMock.mockReset();
    codexConstructorMock.mockClear();
    delete process.env.SCRAPE_CODEX_MODEL;
    delete process.env.SCRAPE_CODEX_WEB_SEARCH;
    delete process.env.SCRAPE_CODEX_WORKING_DIRECTORY;
  });

  it('streams structured SDK events and completes with a proposal', async () => {
    const proposal = createMockProposal();
    const runStreamedMock = vi.fn(async () => ({
      events: (async function* () {
        yield { type: 'thread.started', thread_id: 'thread-123' };
        yield { type: 'turn.started' };
        yield {
          type: 'item.started',
          item: {
            id: 'cmd-1',
            type: 'command_execution',
            command: 'playwright open https://example.com/blog',
            aggregated_output: '',
            status: 'in_progress'
          }
        };
        yield {
          type: 'item.updated',
          item: {
            id: 'cmd-1',
            type: 'command_execution',
            command: 'playwright open https://example.com/blog',
            aggregated_output: 'loaded listing page\n',
            status: 'in_progress'
          }
        };
        yield {
          type: 'item.completed',
          item: {
            id: 'msg-1',
            type: 'agent_message',
            text: JSON.stringify({ proposal })
          }
        };
        yield {
          type: 'turn.completed',
          usage: { input_tokens: 10, cached_input_tokens: 0, output_tokens: 20 }
        };
      })()
    }));
    startThreadMock.mockReturnValue({ runStreamed: runStreamedMock });

    const { CodexSdkScrapeOrchestrator } = await import('@/lib/orchestrator/codexSdkRunner');
    const runner = new CodexSdkScrapeOrchestrator();
    const events = [];

    for await (const event of runner.runScrapeDiscovery({
      runId: 'run-1',
      targetUrl: 'https://example.com/blog'
    })) {
      events.push(event);
    }

    expect(codexConstructorMock).toHaveBeenCalled();
    expect(startThreadMock).toHaveBeenCalled();
    expect(runStreamedMock).toHaveBeenCalled();
    expect(events[0]).toMatchObject({ type: 'status', step: 'queued' });
    expect(events[1]).toMatchObject({ type: 'status', step: 'running' });
    expect(events.some((event) => event.type === 'output' && event.message.includes('thread started'))).toBe(true);
    expect(events.some((event) => event.type === 'output' && event.message.includes('loaded listing page'))).toBe(true);
    expect(events.at(-1)).toMatchObject({ type: 'complete', step: 'needs_review' });
    expect(events.at(-1)).toHaveProperty('proposal.generatedYaml', proposal.generatedYaml);
  });
});
