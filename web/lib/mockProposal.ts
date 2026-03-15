import type { ScrapeProposal } from './schemas';

export function buildMockProposal(targetUrl: string): ScrapeProposal {
  return {
    discovery: {
      listPageSelector: { selector: 'main article', confidence: 0.85 },
      itemLinkSelector: { selector: 'h2 a', confidence: 0.88 }
    },
    extraction: {
      title: { selector: 'h1', confidence: 0.9 },
      url: { selector: 'link[rel="canonical"]', confidence: 0.95 },
      author: { selector: '.byline a', confidence: 0.71 },
      publishedAt: { selector: 'time[datetime]', confidence: 0.8 },
      summary: { selector: 'meta[name="description"]', confidence: 0.75 },
      content: { selector: 'article', confidence: 0.84 }
    },
    samples: Array.from({ length: 5 }).map((_, idx) => ({
      title: `Sample post ${idx + 1}`,
      url: `${targetUrl.replace(/\/$/, '')}/sample-post-${idx + 1}`,
      author: idx % 2 === 0 ? 'Editorial Team' : 'Guest Author',
      publishedAt: `2026-01-${String(10 + idx).padStart(2, '0')}`,
      summary: 'A short extracted summary from metadata or leading paragraph.',
      contentSnippet: 'This is the extracted content snippet to let users verify selector quality.'
    })),
    overallConfidence: 0.84,
    generatedYaml: `sources:\n  - scrape:\n      urls:\n        - ${targetUrl}\n      discovery:\n        list_selector: "main article"\n        item_link_selector: "h2 a"\n      extraction:\n        title: "h1"\n        url: "link[rel=\\"canonical\\"]"\n        author: ".byline a"\n        published_at: "time[datetime]"\n        summary: "meta[name=\\"description\\"]"\n        content: "article"`
  };
}
