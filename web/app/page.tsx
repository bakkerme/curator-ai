import Link from 'next/link';

export default function HomePage() {
  return (
    <main className="grid">
      <h1>Curator Web (v0)</h1>
      <p>First vertical slice: generate and review a scrape block from a target URL.</p>
      <div className="card">
        <h2>Start</h2>
        <p>Use the scrape flow to create a reviewable selector proposal and YAML output.</p>
        <Link href="/scrape/new">Go to scrape block generator →</Link>
      </div>
    </main>
  );
}
