import type { ScrapeProposal } from '@/lib/schemas';

export default function ExtractionPreviewTable({ proposal }: { proposal: ScrapeProposal }) {
  return (
    <section className="card">
      <h3>Extraction preview</h3>
      <table width="100%" cellPadding={8}>
        <thead>
          <tr>
            <th align="left">Title</th>
            <th align="left">Author</th>
            <th align="left">Published</th>
            <th align="left">URL</th>
          </tr>
        </thead>
        <tbody>
          {proposal.samples.map((sample) => (
            <tr key={sample.url}>
              <td>{sample.title}</td>
              <td>{sample.author}</td>
              <td>{sample.publishedAt}</td>
              <td>
                <a href={sample.url} target="_blank" rel="noreferrer">
                  {sample.url}
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
