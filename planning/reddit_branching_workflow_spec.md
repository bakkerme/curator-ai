# Localllama Daily Digest – Branch-Capable Workflow Spec

## 1 . Overview  
Adds directed-acyclic-graph (DAG) semantics to Curator workflows so Elements can **branch, route, and re-join** during processing.

## 2 . YAML Schema Additions

| Key | Level | Type | Purpose |
|-----|-------|------|---------|
| `stages` | root | map | explicit node-id → stage definition |
| `next` | stage | string&#124;array | fan-out edges (default linear flow) |
| `router` | stage `type` | enum | evaluates conditions, sends to `routes[].to` |
| `routes` | router | list | ordered `{when, to}` rules (+ optional `else`) |
| `stages` | root | map | explicit node-id → stage definition |
| `next` | stage | string&#124;array | fan-out edges (default linear flow) |
| `router` | stage `type` | enum | evaluates conditions, sends to `routes[].to` |
| `routes` | router | list | ordered `{when, to}` rules (+ optional `else`, supports `extract`) |
| `join` | stage | enum | `all (default)` or `any` – upstream sync policy |
| `batch_join` | stage type | — | barrier that waits until all Elements from the current run arrive from specified upstream nodes, then emits a single aggregated payload |
| `dataID` | stage | string | logical label for the data bucket produced by this stage |
| `extract` | router rule | string | sub-field(s) forwarded to the target stage |
| `sources` | batch_join | id&#124;array | upstream source node(s) the batch barrier monitors |
| `selector_map` | processor | map | bucket-id → selector expression for grouping inputs |

## 4 . Sample DAG Workflow

```yaml
workflow:
  name: "Localllama Daily Digest"

  stages:
    redditSrc:              # ── node ID
      type: source
      dataID: post
      use: reddit_source
      produces: [ObjectBlock]
      config:
        subreddit: "localllama"
        include_comments: true
      next:
        - agg
        - urlRoute
        - imgRoute     # multiple targets = simple fan-out

    urlRoute:
      type: router
      operates_on: [ObjectBlock]
      routes:         # ordered routing table
        - when: "contains_urls == true"
          extract: "external_urls"
          to: urlFetch
        - else: drop  # built-in sink

    imgRoute:
      type: router
      operates_on: [TextBlock]
      routes:
        - when: "has_image_urls == true"
          extract: "image_urls"
          to: imgSum        # imgSum will fetch then summarize images
        - else: drop

    webFetch:
      type: processor
      dataID: web_summaries
      use: web_fetcher
      config:
        readability: true
      produces: [TextBlock]
      next: [agg]

    webSum:
      type: processor
      use: summariser
      operates_on: [TextBlock]
      prompt_template: "./templates/websum.tmpl"

    imgFetch:
      type:processor
      use: url_fetcher
      operates_on: [TextBlock]
      produces: [ImageBlock]
      next: [imgSum]

    imgSum:
      type: processor
      dataID: image_summaries
      use: image_summarizer
      produces: [TextBlock]
      next: [agg]     # converging edge

    agg: # ── join node; receives multiple inbound edges
      type: processor
      use: thread_aggregator
      operates_on: [ObjectBlock, TextBlock]
      produces: [ObjectBlock]
      config:
        template: "reddit_digest_prompt"
      selector_map:
        post_text: 'post.body'
        comment_texts: 'comment'
        web_summaries: web_summaries
        image_summaries: image_summaries
          
    threadSum:
      type: processor
      use: summariser
      operates_on: [ObjectBlock]
      produces: [ObjectBlock] # By signalling an ObjectBlock, we tell the processor to expect JSON to be returned
      prompt_template: "./templates/threadsum.tmpl"
      schema: "./schemas/threadschema.json"
      next: [batchJoin]

    batchJoin: # Wait until everything from reddit source has entered the batch join
      type: batch_join
      sources: redditSrc
      produces: [ObjectBlock]
      max_wait_sec: 60
      next: [emailFmt]

    emailFmt:
      type: formatter
      use: email_formatter
      operates_on: [ObjectBlock]
      produces: [TextBlock]
      template: "./email_templates/email.tmpl"
      next: [email]

    email:
      type: destination
      use: smtp_email
      operates_on: [TextBlock]
      config:
        to: ["digest@example.com"]
        subject: "Localllama Daily Digest"
```
