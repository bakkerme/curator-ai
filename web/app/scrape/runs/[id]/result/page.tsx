'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import YamlOutputPanel from '@/components/YamlOutputPanel';
import type { ScrapeRun } from '@/lib/schemas';

export default function ResultPage({ params }: { params: { id: string } }) {
  const [run, setRun] = useState<ScrapeRun | null>(null);

  useEffect(() => {
    fetch(`/api/scrape-runs/${params.id}`)
      .then((response) => response.json() as Promise<ScrapeRun>)
      .then(setRun)
      .catch(() => setRun(null));
  }, [params.id]);

  if (!run?.proposal) {
    return (
      <main>
        <p>Loading final YAML...</p>
      </main>
    );
  }

  return (
    <main className="grid">
      <h1>Scrape block ready</h1>
      <p>Copy this snippet into your Curator document.</p>
      <YamlOutputPanel yaml={run.proposal.generatedYaml} />
      <Link href="/scrape/new">Start another run</Link>
    </main>
  );
}
