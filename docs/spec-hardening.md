# Schema Hardening and Token Optimization

This document outlines the schema hardening methodology implemented by toolcraft to eliminate LLM confusion and reduce token overhead.

---

## 1. The Token Bloat Problem

Standard enterprise OpenAPI specifications are written for human frontend engineers or backend SDK generators. They often contain:
- Internal tracking identifiers (e.g., `X-Correlation-ID`, `X-Trace-Context`).
- Legacy pagination options and database partition hints (e.g., `internal_shard_id`, `use_v1_engine`).
- Undocumented or ambiguous parameter descriptions.
- Complex nested object schemas where only two fields are required.

When presented with 30 endpoints each having 15 parameters, an LLM context window is loaded with over 15,000 tokens of schema metadata before a conversation even begins. This leads to:
1. Slower time to first token.
2. Parameter hallucinations where models populate internal debug flags with arbitrary values.
3. Increased API costs per interaction.

---

## 2. Hardening Strategy

### 2.1 Regex-Based Parameter Pruning

toolcraft matches parameter names against a configurable list of regular expressions. Matching parameters are stripped before code generation or runtime registration.

Default exclusion rules:
```regex
.*_trace_id
.*_correlation_id
^X-.*
internal_.*
legacy_.*
debug_.*
```

Path parameters (parameters in the URL path) are always preserved regardless of regex matches because removing them would produce invalid HTTP requests.

### 2.2 Parameter Capping and Ranking

When an endpoint exposes more parameters than the configured limit (`max_parameters_per_tool`, default: 6), toolcraft ranks parameters by necessity:

1. Required Path Parameters: Rank 1 (essential for routing).
2. Required Query and Body Parameters: Rank 2 (essential for request success).
3. Primary Filters: Parameters named `id`, `name`, `status`, `limit`, `query`, `search`.
4. General Optional Fields: Parameters sorted by description length and utility.
5. Flags and Switches: Parameters with boolean types and default values (culled first).

Only the top N parameters are kept. The rest are dropped, with default values applied where relevant.

### 2.3 Instruction Synthesis and Enrichment

LLMs select tools based on tool name and tool description. Terse descriptions such as "Get items" often lead to incorrect tool selection.

toolcraft enriches tool descriptions using the following template:

```
[Intent Statement]. Use this tool to [Action] [Target Resource].
[Parameter Guidance]: Requires [List of required fields].
[Output Description]: Returns [Expected response format and status].
```

Example transformation:

- Before: `Retrieve invoices`
- After: `Fetch billing invoices for a customer account. Use this tool to inspect payment status and balance records. Requires customer_id. Returns an array of invoice records.`

---

## 3. Token Reduction Measurement

The `toolcraft inspect` command calculates the token reduction ratio:

```bash
toolcraft inspect --spec ./api.yaml
```

Output metrics:
- Total Operations: Number of endpoints extracted.
- Original Parameter Count: Raw parameters across all operations.
- Hardened Parameter Count: Parameters retained after pruning and capping.
- Estimated Token Reduction: Percentage reduction in context window footprint.
