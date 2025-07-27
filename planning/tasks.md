# Tasks
## Spikes

### Define Core Data Structures (Blocks)
#### Goal
 To define the primary Go structs that will represent data as it moves through the curation flow. This will form the foundation of the system's internal data handling.

#### Context References
- **Block Design Specification** (`block_design.md` lines 3-42): Complete struct definitions for PostBlock, CommentBlock, WebBlock, and ImageBlock
- **Technical Design Block Flow** (`td.md` lines 225-286): Block flow through processing stages and expected field usage
- **Example Flow Structure** (`example_flow.yml` lines 13-47): Real configuration showing data requirements

#### Tasks
- Create a new file internal/core/blocks.go.
- Define the PostBlock struct, including fields for source data (ID, URL, Title, Content), metadata, and placeholders for results from later processing steps (Quality, Summary).
- Define the supporting block structs (CommentBlock, WebBlock, ImageBlock) that can be attached to a PostBlock.
- Define the structs for processor outputs, such as QualityResult and SummaryResult.
- Define the internal representation of the entire parsed flow, including the trigger configuration and an ordered list of processor instances.
- Ensure all structs have appropriate JSON/YAML tags for serialization and are well-documented with comments.

#### Success Criteria
A Go file exists with the core data structures defined, reviewed, and agreed upon. These structs should be sufficient to represent a post from the Reddit source and carry it through all the processing steps defined in the MVP.

### Specify the Curator Document Schema
#### Goal
To create a concrete and validated specification for the curator.yaml document. This will serve as the contract for what the parser needs to handle.

#### Context References
- **Complete Example Flow** (`example_flow.yml` lines 1-64): Full working example with all processor types
- **Technical Design Curator Documents** (`td.md` lines 200-217): YAML structure overview and processor categories
- **Technical Design Architecture** (`td.md` lines 105-150): Parser responsibilities and Runner interfaces

#### Tasks

- Create a new file planning/curator_document_spec.md.
- Define the top-level structure of the YAML file (e.g., version, name, trigger, source, processors, outputs).
- For each processor in the MVP scope (Cron, Reddit, Quality Rule, LLM Quality, LLM Summary, LLM Run Summary, Email), define its specific configuration keys and value types.
- Create a complete example-flow.yaml that uses every feature defined in the spec.
- Define the corresponding Go structs in a new file (internal/config/schema.go) that gopkg.in/yaml.v3 can unmarshal the document into.
- Success Criteria: A specification document and a corresponding set of Go structs exist. The structs can successfully parse the example-flow.yaml file without errors.

### Detail the Flow Runner Execution Logic
#### Goal
To clarify the step-by-step operational logic of the Curator Flow Runner, including data flow, concurrency, and synchronization.

#### Context References
- **Technical Design Block Flow Example** (`td.md` lines 250-302): Complete execution flow from trigger to output
- **Curation Flow Runner Design** (`td.md` lines 140-165): Runner responsibilities and implementation patterns
- **Technical Design Overview** (`td.md` lines 105-125): Engine architecture and state management

#### Tasks
- Create a new file planning/runner_logic.md.
- Write pseudocode or create a sequence diagram for the runner's main loop, starting from a trigger event.
- Explicitly define how a batch of PostBlocks moves between processors. Confirm if it's one-by-one or as a full batch at each stage.
- Design the synchronization mechanism for the Run Summary processor. How does the runner wait for all posts to be processed before executing the run summary? (e.g., using channels or a sync.WaitGroup).
- Define the common Go interface that all processors must implement. This should include methods for configuration and execution (e.g., Configure(map[string]any) error, Process([]*PostBlock) ([]*PostBlock, error)).
- Success Criteria: A document exists that clearly explains the runner's execution model, and a Go interface for processors is defined. This should be clear enough for a developer to begin implementing the runner orchestrator.