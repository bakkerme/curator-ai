# Scrape Source Guide (Web UI Agent Layer)

When generating scrape selectors for a target URL:

1. Identify discovery selectors:
   - container/list selector
   - item link selector
   - pagination selector (if present)
2. Identify extraction selectors for each post page:
   - title
   - canonical URL
   - author
   - published date
   - summary
   - content/body
3. Prefer stable selectors:
   - semantic tags and attributes
   - stable classes/ids
   - avoid nth-child, nth-of-type selectors
4. Validate selectors across multiple items/pages.
5. Return JSON matching the UI proposal schema with confidence values.
