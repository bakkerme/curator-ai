'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import RunTimeline from '@/components/RunTimeline';

type Run = {
  id: string;
  status: string;
  targetUrl: string;
  logs: string[];
};

/**
 * RunPage shows live progression from queued to review-ready and keeps the
 * current run status synchronized with the backend so users can continue the
 * flow without guessing whether the proposal is ready.
 */
export default function RunPage({ params }: { params: { id: string } }) {
  const [run, setRun] = useState<Run | null>(null);
  const [isReadyForReview, setIsReadyForReview] = useState(false);

  const fetchRun = useCallback(async () => {
    const response = await fetch(`/api/scrape-runs/${params.id}`, { cache: 'no-store' });
    if (!response.ok) {
      throw new Error('run not found');
    }
    const nextRun = (await response.json()) as Run;
    setRun(nextRun);
    const nextReady = nextRun.status === 'needs_review' || nextRun.status === 'approved';
    // Preserve a previously unlocked state to avoid regressions from stale responses.
    setIsReadyForReview((current) => current || nextReady);
  }, [params.id]);

  const handleTimelineComplete = useCallback(() => {
    // Fetch the run once timeline reaches completion so the status label and
    // review CTA reflect the latest backend state.
    setIsReadyForReview(true);
    setRun((current) => (current ? { ...current, status: 'needs_review' } : current));
    fetchRun().catch(() => undefined);
  }, [fetchRun]);

  useEffect(() => {
    fetchRun().catch(() => setRun(null));
  }, [fetchRun]);

  if (!run) {
    return (
      <main>
        <p>Loading run...</p>
      </main>
    );
  }

  return (
    <main className="grid">
      <h1>Run {run.id}</h1>
      <p>
        Target URL: <code>{run.targetUrl}</code>
      </p>
      <p>
        Current status: <strong>{run.status}</strong>
      </p>
      <RunTimeline
        runId={run.id}
        initialOutput={run.logs.filter((entry) => entry.startsWith('[stdout]') || entry.startsWith('[stderr]'))}
        onComplete={handleTimelineComplete}
      />
      <div className="card">
        <p>
          {run.status === 'failed'
            ? 'Run failed. Check logs and retry with another URL or runner mode.'
            : isReadyForReview
              ? 'Run is ready. Continue to review the proposed selectors and samples.'
              : 'Run is still generating proposal data. Review will unlock automatically.'}
        </p>
        {run.status === 'failed' ? (
          <span>Go to review → (disabled because run failed)</span>
        ) : isReadyForReview ? (
          <Link href={`/scrape/runs/${run.id}/review`}>Go to review →</Link>
        ) : (
          <span>Go to review → (locked until run completes)</span>
        )}
      </div>
    </main>
  );
}
