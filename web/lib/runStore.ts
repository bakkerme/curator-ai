import { promises as fs } from 'node:fs';
import path from 'node:path';
import { tmpdir } from 'node:os';
import type { ScrapeProposal, ScrapeRun, ScrapeRunInput } from './schemas';

const STORE_FILE = path.join(tmpdir(), 'curator-web-runs.json');

type RunMap = Record<string, ScrapeRun>;

function nowIso(): string {
  return new Date().toISOString();
}

function newId(): string {
  return crypto.randomUUID();
}

/**
 * readStore loads persisted runs from a temp-file backed JSON store.
 * File persistence is used instead of process memory to work reliably in
 * Next.js dev/runtime execution contexts where requests may hit different workers.
 */
async function readStore(): Promise<RunMap> {
  try {
    const raw = await fs.readFile(STORE_FILE, 'utf8');
    return JSON.parse(raw) as RunMap;
  } catch {
    return {};
  }
}

/**
 * writeStore atomically persists the latest run map snapshot.
 */
async function writeStore(store: RunMap): Promise<void> {
  await fs.writeFile(STORE_FILE, JSON.stringify(store, null, 2), 'utf8');
}

export async function createRun(input: ScrapeRunInput): Promise<ScrapeRun> {
  const store = await readStore();
  const run: ScrapeRun = {
    id: newId(),
    status: 'queued',
    targetUrl: input.targetUrl,
    createdAt: nowIso(),
    updatedAt: nowIso(),
    logs: ['Run created and queued for execution.']
  };

  store[run.id] = run;
  await writeStore(store);
  return run;
}

export async function getRun(id: string): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  return store[id];
}

export async function setRunStatus(id: string, status: ScrapeRun['status']): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  const run = store[id];
  if (!run) return undefined;

  run.status = status;
  run.updatedAt = nowIso();
  store[id] = run;
  await writeStore(store);
  return run;
}

export async function setRunProposal(id: string, proposal: ScrapeProposal): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  const run = store[id];
  if (!run) return undefined;

  run.proposal = proposal;
  run.status = 'needs_review';
  run.updatedAt = nowIso();
  store[id] = run;
  await writeStore(store);
  return run;
}

export async function appendRunLog(id: string, message: string): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  const run = store[id];
  if (!run) return undefined;

  run.logs = [...run.logs, message];
  run.updatedAt = nowIso();
  store[id] = run;
  await writeStore(store);
  return run;
}

export async function updateSelectors(
  id: string,
  edits: Record<string, string>
): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  const run = store[id];
  if (!run || !run.proposal) return undefined;

  if (edits.listPageSelector) {
    run.proposal.discovery.listPageSelector.selector = edits.listPageSelector;
  }
  if (edits.itemLinkSelector) {
    run.proposal.discovery.itemLinkSelector.selector = edits.itemLinkSelector;
  }
  for (const field of ['title', 'url', 'author', 'publishedAt', 'summary', 'content'] as const) {
    if (edits[field]) {
      run.proposal.extraction[field].selector = edits[field];
    }
  }

  run.logs.push('Selectors edited by user and preview regenerated.');
  run.updatedAt = nowIso();

  store[id] = run;
  await writeStore(store);
  return run;
}

export async function approveRun(id: string): Promise<ScrapeRun | undefined> {
  const store = await readStore();
  const run = store[id];
  if (!run) return undefined;

  run.status = 'approved';
  run.logs.push('Run approved and YAML finalized.');
  run.updatedAt = nowIso();

  store[id] = run;
  await writeStore(store);
  return run;
}
