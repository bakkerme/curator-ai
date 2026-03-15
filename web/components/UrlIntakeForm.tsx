'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';

export default function UrlIntakeForm() {
  const router = useRouter();
  const [targetUrl, setTargetUrl] = useState('');
  const [contentType, setContentType] = useState('blog');
  const [preferredItemCount, setPreferredItemCount] = useState(10);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setPending(true);

    const response = await fetch('/api/scrape-runs', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ targetUrl, contentType, preferredItemCount })
    });

    if (!response.ok) {
      setError('Unable to start run. Please check the URL and try again.');
      setPending(false);
      return;
    }

    const run = (await response.json()) as { id: string };
    router.push(`/scrape/runs/${run.id}`);
  }

  return (
    <form className="grid card" onSubmit={onSubmit}>
      <h2>Create scrape run</h2>
      <label>
        Target URL
        <input
          type="url"
          placeholder="https://example.com/blog"
          required
          value={targetUrl}
          onChange={(event) => setTargetUrl(event.target.value)}
        />
      </label>

      <label>
        Content type
        <select value={contentType} onChange={(event) => setContentType(event.target.value)}>
          <option value="blog">Blog</option>
          <option value="news">News</option>
          <option value="docs">Docs</option>
        </select>
      </label>

      <label>
        Preferred item count
        <input
          type="number"
          min={1}
          max={50}
          value={preferredItemCount}
          onChange={(event) => setPreferredItemCount(Number(event.target.value))}
        />
      </label>

      {error && <p style={{ color: '#b40000' }}>{error}</p>}

      <button disabled={pending} type="submit">
        {pending ? 'Starting run...' : 'Start run'}
      </button>
    </form>
  );
}
