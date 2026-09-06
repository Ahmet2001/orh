# ORH Fixed-Corpus Research Workflow Evaluation

This artifact evaluates an ORH research workflow without relying on changing
web-search results. The dataset contains synthetic scientific reports released
with this repository for unrestricted evaluation use. Every answer is stated in
the supplied evidence; models should not need outside knowledge.

The artifact compares:

- `monolithic.orh`: one agent reads evidence and answers;
- `composable.orh`: separate evidence extraction, verification, and reporting
  components communicate through the ORH event graph.

It also runs the same unchanged architecture with multiple Ollama model
bindings and saves one `orh.run/v1` provenance record per execution.

## Run

From this directory:

```bash
python3 evaluate.py --models qwen3:1.7b hermes3:3b --repetitions 3
```

The script builds the local ORH CLI, creates a timestamped directory under
`results/`, and writes:

- `runs.jsonl`: task-level outputs, scores, latency, and record paths;
- `summary.json`: aggregate exact-answer accuracy and latency;
- `provenance/*.json`: ORH execution records.

For a fast smoke test:

```bash
python3 evaluate.py --models qwen3:1.7b --repetitions 1 --limit 2
```

## Scoring

The final answer is parsed from `FINAL: <answer>`. A run receives score 1 only
when the normalized parsed answer exactly matches the gold answer. Missing or
malformed final answers receive score 0. The scorer does not use an LLM.

The dataset is synthetic so that it contains no personal data, copyrighted
paper text, or unstable external retrieval results.
