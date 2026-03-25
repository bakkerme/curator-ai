'use client';

import { useState } from 'react';

export default function YamlOutputPanel({ yaml }: { yaml: string }) {
  const [copied, setCopied] = useState(false);

  async function onCopy() {
    await navigator.clipboard.writeText(yaml);
    setCopied(true);
  }

  return (
    <section className="card grid">
      <h3>Generated YAML</h3>
      <pre style={{ overflowX: 'auto' }}>{yaml}</pre>
      <button type="button" onClick={onCopy}>
        {copied ? 'Copied!' : 'Copy YAML'}
      </button>
    </section>
  );
}
