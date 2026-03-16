import { spawn } from 'node:child_process';
import { z } from 'zod';
import { scrapeProposalSchema } from '@/lib/schemas';
import type { OrchestratorEvent, ScrapeDiscoveryInput, ScrapeOrchestrator } from './types';

const cliOutputSchema = z.object({
  proposal: scrapeProposalSchema
});

function extractFirstJsonObject(raw: string): string {
  const start = raw.indexOf('{');
  const end = raw.lastIndexOf('}');
  if (start === -1 || end === -1 || end <= start) {
    throw new Error('No JSON object detected in CLI output');
  }
  return raw.slice(start, end + 1);
}

/**
 * Real runner adapter that invokes a configured CLI command and expects JSON
 * output that can be validated into the proposal schema.
 */
export class CodexCliScrapeOrchestrator implements ScrapeOrchestrator {
  async *runScrapeDiscovery(input: ScrapeDiscoveryInput): AsyncGenerator<OrchestratorEvent> {
    const command = process.env.SCRAPE_CODEX_COMMAND ?? 'codex';
    const args = [
      'exec',
      '--json',
      `Inspect ${input.targetUrl} and return scrape proposal JSON matching schema.`
    ];

    yield { type: 'status', step: 'queued', message: 'Run queued.' };
    yield { type: 'status', step: 'running', message: 'Launching Codex CLI run...' };

    const child = spawn(command, args, {
      stdio: ['ignore', 'pipe', 'pipe'],
      env: {
        PATH: process.env.PATH,
        HOME: process.env.HOME
      }
    });

    let stdout = '';
    let stderr = '';

    child.stdout.on('data', (chunk: Buffer) => {
      stdout += chunk.toString();
    });

    child.stderr.on('data', (chunk: Buffer) => {
      stderr += chunk.toString();
    });

    const exitCode = await new Promise<number>((resolve, reject) => {
      child.on('error', reject);
      child.on('close', (code) => resolve(code ?? 1));
    });

    if (exitCode !== 0) {
      yield {
        type: 'failed',
        step: 'failed',
        message: `Codex CLI exited with code ${exitCode}: ${stderr || 'no stderr output'}`
      };
      return;
    }

    try {
      const jsonText = extractFirstJsonObject(stdout);
      const parsed = cliOutputSchema.parse(JSON.parse(jsonText));
      yield {
        type: 'complete',
        step: 'needs_review',
        message: 'Sample extraction complete.',
        proposal: parsed.proposal
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown parse error';
      yield {
        type: 'failed',
        step: 'failed',
        message: `Codex CLI output validation failed: ${message}`
      };
    }
  }
}
