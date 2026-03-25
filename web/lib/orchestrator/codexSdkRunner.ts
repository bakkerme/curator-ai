import path from 'node:path';
import { Codex } from '@openai/codex-sdk';
import { scrapeProposalSchema } from '@/lib/schemas';
import type { OrchestratorEvent, ScrapeDiscoveryInput, ScrapeOrchestrator } from './types';

const fullPrompt = `# Scrape Source Guide (Web UI Agent Layer)
When generating scrape selectors for a target URL:

1. Identify discovery selectors:
   - container/list selector
   - item link selector
   - pagination selector (if present)
2. Identify extraction selectors for each post page:
   - title
   - canonical URL
   - author
   - published date
   - summary
   - content/body
3. Prefer stable selectors:
   - semantic tags and attributes
   - stable classes/ids
   - avoid nth-child, nth-of-type selectors
4. Validate selectors across multiple items/pages.
5. Return JSON matching the UI proposal schema with confidence values.`;

const promptTemplate =
  'Using the playwright skill, create a list of selectors to extract blog post data from the provided URL. ' + fullPrompt +  ' The site is: %s';

const proposalOutputSchema = {
  type: 'object',
  additionalProperties: false,
  required: ['proposal'],
  properties: {
    proposal: {
      type: 'object',
      additionalProperties: false,
      required: ['discovery', 'extraction', 'samples', 'generatedYaml', 'overallConfidence'],
      properties: {
        discovery: {
          type: 'object',
          additionalProperties: false,
          required: ['listPageSelector', 'itemLinkSelector', 'paginationSelector'],
          properties: {
            listPageSelector: selectorFieldSchema(),
            itemLinkSelector: selectorFieldSchema(),
            paginationSelector: {
              anyOf: [selectorFieldSchema(), { type: 'null' }]
            }
          }
        },
        extraction: {
          type: 'object',
          additionalProperties: false,
          required: ['title', 'url', 'author', 'publishedAt', 'summary', 'content'],
          properties: {
            title: selectorFieldSchema(),
            url: selectorFieldSchema(),
            author: selectorFieldSchema(),
            publishedAt: selectorFieldSchema(),
            summary: selectorFieldSchema(),
            content: selectorFieldSchema()
          }
        },
        samples: {
          type: 'array',
          items: {
            type: 'object',
            additionalProperties: false,
            required: ['title', 'url', 'author', 'publishedAt', 'summary', 'contentSnippet'],
            properties: {
              title: { type: 'string' },
              url: { type: 'string' },
              author: { type: 'string' },
              publishedAt: { type: 'string' },
              summary: { type: 'string' },
              contentSnippet: { type: 'string' }
            }
          }
        },
        generatedYaml: { type: 'string' },
        overallConfidence: { type: 'number' }
      }
    }
  }
} as const;

function selectorFieldSchema() {
  return {
    type: 'object',
    additionalProperties: false,
    required: ['selector', 'confidence'],
    properties: {
      selector: { type: 'string' },
      confidence: { type: 'number' }
    }
  } as const;
}

function buildPrompt(targetUrl: string): string {
  return promptTemplate.replace('%s', targetUrl);
}

function resolveWorkingDirectory(): string {
  return process.env.SCRAPE_CODEX_WORKING_DIRECTORY ?? path.resolve(process.cwd(), '..');
}

function createCodexClient(): Codex {
  return new Codex({
    // apiKey: process.env.OPENAI_API_KEY,
    // baseUrl: process.env.OPENAI_BASE_URL
  });
}

function formatCommandStatus(status: 'in_progress' | 'completed' | 'failed', command: string, exitCode?: number): string {
  if (status === 'completed') {
    return `[command] completed (${exitCode ?? 0}): ${command}\n`;
  }
  if (status === 'failed') {
    return `[command] failed (${exitCode ?? 'unknown'}): ${command}\n`;
  }
  return `[command] running: ${command}\n`;
}

/**
 * SDK-backed runner that keeps the scrape-agent path fully inside the web
 * application while still using Codex's streamed event model.
 */
export class CodexSdkScrapeOrchestrator implements ScrapeOrchestrator {
  async *runScrapeDiscovery(input: ScrapeDiscoveryInput): AsyncGenerator<OrchestratorEvent> {
    const codex = createCodexClient();
    const thread = codex.startThread({
      model: process.env.SCRAPE_CODEX_MODEL,
      sandboxMode: 'read-only',
      // workingDirectory: resolveWorkingDirectory(),
      workingDirectory: '/tmp',
      approvalPolicy: 'on-request',
      skipGitRepoCheck: true,
      // webSearchMode: process.env.SCRAPE_CODEX_WEB_SEARCH === 'live' ? 'live' : 'disabled'
      webSearchMode: 'disabled'
    });
    const prompt = buildPrompt(input.targetUrl);
    const commandOutputOffsets = new Map<string, number>();
    let proposalText = '';

    yield { type: 'status', step: 'queued', message: 'Run queued.' };
    yield { type: 'status', step: 'running', message: 'Starting Codex SDK run...' };

    try {
      const { events } = await thread.runStreamed(prompt, {
        outputSchema: proposalOutputSchema
      });

      for await (const event of events) {
        if (event.type === 'thread.started') {
          yield {
            type: 'output',
            step: 'running',
            stream: 'stderr',
            message: `[codex] thread started: ${event.thread_id}\n`
          };
          continue;
        }

        if (event.type === 'turn.started') {
          yield { type: 'status', step: 'running', message: 'Codex is inspecting the target site...' };
          continue;
        }

        if (event.type === 'item.started' || event.type === 'item.updated' || event.type === 'item.completed') {
          const item = event.item;

          if (item.type === 'command_execution') {
            const previousOffset = commandOutputOffsets.get(item.id) ?? 0;
            const nextOutput = item.aggregated_output.slice(previousOffset);
            commandOutputOffsets.set(item.id, item.aggregated_output.length);

            if (event.type === 'item.started') {
              yield {
                type: 'output',
                step: 'running',
                stream: 'stderr',
                message: formatCommandStatus('in_progress', item.command)
              };
            }
            if (nextOutput) {
              yield {
                type: 'output',
                step: 'running',
                stream: 'stdout',
                message: nextOutput
              };
            }
            if (event.type === 'item.completed') {
              yield {
                type: 'output',
                step: 'running',
                stream: item.status === 'failed' ? 'stderr' : 'stdout',
                message: formatCommandStatus(item.status, item.command, item.exit_code)
              };
            }
            continue;
          }

          if (item.type === 'reasoning' && event.type === 'item.completed' && item.text.trim()) {
            yield {
              type: 'output',
              step: 'running',
              stream: 'stderr',
              message: `[reasoning] ${item.text.trim()}\n`
            };
            continue;
          }

          if (item.type === 'todo_list') {
            yield {
              type: 'output',
              step: 'running',
              stream: 'stderr',
              message: `[todo] ${item.items.map((todo) => `${todo.completed ? '[x]' : '[ ]'} ${todo.text}`).join(' | ')}\n`
            };
            continue;
          }

          if (item.type === 'web_search' && event.type === 'item.started') {
            yield {
              type: 'output',
              step: 'running',
              stream: 'stderr',
              message: `[web-search] ${item.query}\n`
            };
            continue;
          }

          if (item.type === 'mcp_tool_call') {
            const suffix = item.status === 'failed' && item.error ? `: ${item.error.message}` : '';
            yield {
              type: 'output',
              step: 'running',
              stream: item.status === 'failed' ? 'stderr' : 'stdout',
              message: `[mcp] ${item.server}/${item.tool} ${item.status}${suffix}\n`
            };
            continue;
          }

          if (item.type === 'file_change' && event.type === 'item.completed') {
            yield {
              type: 'output',
              step: 'running',
              stream: 'stdout',
              message: `[files] ${item.status}: ${item.changes.map((change) => `${change.kind} ${change.path}`).join(', ')}\n`
            };
            continue;
          }

          if (item.type === 'error') {
            yield {
              type: 'output',
              step: 'running',
              stream: 'stderr',
              message: `[agent-error] ${item.message}\n`
            };
            continue;
          }

          if (item.type === 'agent_message' && event.type === 'item.completed') {
            proposalText = item.text;
            yield {
              type: 'output',
              step: 'running',
              stream: 'stdout',
              message: '[agent] structured response received\n'
            };
            continue;
          }
        }

        if (event.type === 'turn.failed') {
          yield {
            type: 'failed',
            step: 'failed',
            message: `Codex SDK run failed: ${event.error.message}`
          };
          return;
        }

        if (event.type === 'error') {
          yield {
            type: 'failed',
            step: 'failed',
            message: `Codex SDK stream error: ${event.message}`
          };
          return;
        }

        if (event.type === 'turn.completed') {
          if (!proposalText) {
            yield {
              type: 'failed',
              step: 'failed',
              message: 'Codex SDK completed without returning a structured proposal.'
            };
            return;
          }

          try {
            const rawProposal = JSON.parse(proposalText).proposal as Record<string, unknown>;
            if (
              rawProposal &&
              typeof rawProposal === 'object' &&
              rawProposal.discovery &&
              typeof rawProposal.discovery === 'object' &&
              'paginationSelector' in rawProposal.discovery &&
              rawProposal.discovery.paginationSelector === null
            ) {
              delete rawProposal.discovery.paginationSelector;
            }

            const parsed = scrapeProposalSchema.parse(rawProposal);
            yield {
              type: 'complete',
              step: 'needs_review',
              message: 'Sample extraction complete.',
              proposal: parsed
            };
            return;
          } catch (error) {
            const message = error instanceof Error ? error.message : 'Unknown parse error';
            yield {
              type: 'failed',
              step: 'failed',
              message: `Codex SDK output validation failed: ${message}`
            };
            return;
          }
        }
      }

      yield {
        type: 'failed',
        step: 'failed',
        message: 'Codex SDK stream ended before producing a completed turn.'
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown SDK error';
      yield {
        type: 'failed',
        step: 'failed',
        message: `Codex SDK runner failed: ${message}`
      };
    }
  }
}
