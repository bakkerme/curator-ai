'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import ExtractionPreviewTable from '@/components/ExtractionPreviewTable';
import SelectorEditor from '@/components/SelectorEditor';
import type { ScrapeRun } from '@/lib/schemas';

/**
 * ReviewPage lets users edit selector values and regenerate preview output.
 * It also gracefully handles runs that are still processing by polling until
 * proposal data is available.
 */
export default function ReviewPage({ params }: { params: { id: string } }) {
  const router = useRouter();
  const [run, setRun] = useState<ScrapeRun | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchRun = useCallback(async (): Promise<ScrapeRun | null> => {
    try {
      const response = await fetch(`/api/scrape-runs/${params.id}`, { cache: 'no-store' });
      if (!response.ok) {
        throw new Error('run not found');
      }

      const nextRun = (await response.json()) as ScrapeRun;
      setRun(nextRun);
      setIsLoading(false);
      return nextRun;
    } catch {
      setRun(null);
      setIsLoading(false);
      return null;
    }
  }, [params.id]);

  useEffect(() => {
    let isCancelled = false;
    let intervalId: ReturnType<typeof setInterval> | undefined;

    const start = async () => {
      const nextRun = await fetchRun();
      if (isCancelled) {
        return;
      }

      // Only poll while proposal is unavailable; stop once review data is ready.
      if (!nextRun?.proposal) {
        intervalId = setInterval(async () => {
          const polledRun = await fetchRun();
          if (polledRun?.proposal && intervalId) {
            clearInterval(intervalId);
          }
        }, 1200);
      }
    };

    void start();

    return () => {
      isCancelled = true;
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [fetchRun]);

  const selectorValues = useMemo(() => {
    if (!run?.proposal) return {};
    return {
      listPageSelector: run.proposal.discovery.listPageSelector.selector,
      itemLinkSelector: run.proposal.discovery.itemLinkSelector.selector,
      title: run.proposal.extraction.title.selector,
      url: run.proposal.extraction.url.selector,
      author: run.proposal.extraction.author.selector,
      publishedAt: run.proposal.extraction.publishedAt.selector,
      summary: run.proposal.extraction.summary.selector,
      content: run.proposal.extraction.content.selector
    };
  }, [run]);

  async function onPreview(nextValues: Record<string, string>) {
    const response = await fetch(`/api/scrape-runs/${params.id}/preview`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(nextValues)
    });

    if (response.ok) {
      setRun((await response.json()) as ScrapeRun);
    }
  }

  async function onApprove() {
    const response = await fetch(`/api/scrape-runs/${params.id}/approve`, {
      method: 'POST'
    });
    if (response.ok) {
      router.push(`/scrape/runs/${params.id}/result`);
    }
  }

  if (isLoading) {
    return (
      <main>
        <p>Loading review data...</p>
      </main>
    );
  }

  if (!run?.proposal) {
    return (
      <main className="grid">
        <h1>Review scrape proposal</h1>
        <p>Proposal not ready yet. Return to run timeline and wait for generation.</p>
        <Link href={`/scrape/runs/${params.id}`}>Back to run</Link>
      </main>
    );
  }

  return (
    <main className="grid">
      <h1>Review scrape proposal</h1>
      <p>
        Overall confidence: <strong>{Math.round(run.proposal.overallConfidence * 100)}%</strong>
      </p>
      <SelectorEditor initialValues={selectorValues} onPreview={onPreview} />
      <ExtractionPreviewTable proposal={run.proposal} />
      <button onClick={onApprove} type="button">
        Approve and generate YAML
      </button>
    </main>
  );
}
