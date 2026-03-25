import { describe, expect, it } from 'vitest';
import { scrapeRunInputSchema } from '@/lib/schemas';

describe('scrapeRunInputSchema', () => {
  it('accepts a valid URL payload', () => {
    const parsed = scrapeRunInputSchema.safeParse({
      targetUrl: 'https://example.com/blog',
      contentType: 'blog',
      preferredItemCount: 10
    });

    expect(parsed.success).toBe(true);
  });

  it('rejects invalid URL payload', () => {
    const parsed = scrapeRunInputSchema.safeParse({ targetUrl: 'not-a-url' });
    expect(parsed.success).toBe(false);
  });
});
