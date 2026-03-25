import { z } from 'zod';

export const runStatusSchema = z.enum(['queued', 'running', 'needs_review', 'approved', 'failed']);

export const scrapeRunInputSchema = z.object({
  targetUrl: z.string().url(),
  contentType: z.enum(['blog', 'news', 'docs']).optional(),
  preferredItemCount: z.number().int().min(1).max(50).optional(),
  includePattern: z.string().optional(),
  excludePattern: z.string().optional()
});

const selectorFieldSchema = z.object({
  selector: z.string(),
  confidence: z.number().min(0).max(1)
});

export const scrapeProposalSchema = z.object({
  discovery: z.object({
    listPageSelector: selectorFieldSchema,
    itemLinkSelector: selectorFieldSchema,
    paginationSelector: selectorFieldSchema.optional()
  }),
  extraction: z.object({
    title: selectorFieldSchema,
    url: selectorFieldSchema,
    author: selectorFieldSchema,
    publishedAt: selectorFieldSchema,
    summary: selectorFieldSchema,
    content: selectorFieldSchema
  }),
  samples: z.array(
    z.object({
      title: z.string(),
      url: z.string().url(),
      author: z.string(),
      publishedAt: z.string(),
      summary: z.string(),
      contentSnippet: z.string()
    })
  ),
  generatedYaml: z.string(),
  overallConfidence: z.number().min(0).max(1)
});

export const scrapeRunSchema = z.object({
  id: z.string(),
  status: runStatusSchema,
  targetUrl: z.string().url(),
  createdAt: z.string(),
  updatedAt: z.string(),
  logs: z.array(z.string()),
  proposal: scrapeProposalSchema.optional()
});

export type ScrapeRunInput = z.infer<typeof scrapeRunInputSchema>;
export type ScrapeProposal = z.infer<typeof scrapeProposalSchema>;
export type ScrapeRun = z.infer<typeof scrapeRunSchema>;
