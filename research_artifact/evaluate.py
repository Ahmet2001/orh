#!/usr/bin/env python3
"""Run the fixed-corpus ORH evaluation and deterministically score outputs."""

from __future__ import annotations

import argparse
import json
import os
import re
import statistics
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path


ROOT = Path(__file__).resolve().parent
PROJECT_ROOT = ROOT.parent
ORH_SOURCE = PROJECT_ROOT / "orh"
DATASET = ROOT / "dataset" / "tasks.json"
ARCHITECTURES = {
    "monolithic": ROOT / "architectures" / "monolithic.orh",
    "composable": ROOT / "architectures" / "composable.orh",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--models", nargs="+", default=["qwen3:1.7b", "hermes3:3b"])
    parser.add_argument("--repetitions", type=int, default=3)
    parser.add_argument("--limit", type=int)
    parser.add_argument("--output-dir", type=Path)
    parser.add_argument("--timeout", type=int, default=180, help="seconds allowed per ORH run")
    return parser.parse_args()


def normalize(value: str) -> str:
    return re.sub(r"[^a-z0-9]+", " ", value.casefold()).strip()


def extract_final(stdout: str) -> str:
    matches = re.findall(r"(?im)^\s*(?:>\s*)?FINAL:\s*(.+?)\s*$", stdout)
    return matches[-1].strip() if matches else ""


def build_cli(output: Path) -> None:
    subprocess.run(
        ["go", "build", "-o", str(output), "./cmd/orh"],
        cwd=ORH_SOURCE,
        check=True,
    )


def main() -> int:
    args = parse_args()
    if args.repetitions < 1:
        raise SystemExit("--repetitions must be positive")

    tasks = json.loads(DATASET.read_text(encoding="utf-8"))
    if args.limit is not None:
        tasks = tasks[: args.limit]

    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    output_dir = (args.output_dir or ROOT / "results" / stamp).resolve()
    provenance_dir = output_dir / "provenance"
    provenance_dir.mkdir(parents=True, exist_ok=True)
    binary = output_dir / "orh"
    build_cli(binary)

    results: list[dict] = []
    total = len(ARCHITECTURES) * len(args.models) * len(tasks) * args.repetitions
    current = 0

    for architecture_name, architecture_path in ARCHITECTURES.items():
        for model in args.models:
            for task in tasks:
                prompt = json.dumps(
                    {"question": task["question"], "evidence": task["evidence"]},
                    ensure_ascii=False,
                    separators=(",", ":"),
                )
                for repetition in range(1, args.repetitions + 1):
                    current += 1
                    safe_model = re.sub(r"[^A-Za-z0-9_.-]", "_", model)
                    run_id = f"{architecture_name}__{safe_model}__{task['id']}__r{repetition}"
                    record_path = provenance_dir / f"{run_id}.json"
                    command = [
                        str(binary),
                        "run",
                        str(architecture_path),
                        "--model",
                        f"ollama:{model}",
                        "--record",
                        str(record_path),
                    ]
                    print(f"[{current}/{total}] {run_id}", flush=True)
                    started = time.perf_counter()
                    environment = os.environ.copy()
                    environment.update(
                        {
                            "ORH_OLLAMA_NUM_PREDICT": "128",
                            "ORH_OLLAMA_TEMPERATURE": "0",
                            "ORH_OLLAMA_SEED": str(repetition),
                        }
                    )
                    try:
                        completed = subprocess.run(
                            command,
                            input=prompt + "\n",
                            text=True,
                            capture_output=True,
                            timeout=args.timeout,
                            env=environment,
                        )
                        exit_code = completed.returncode
                        stdout = completed.stdout
                        stderr = completed.stderr
                    except subprocess.TimeoutExpired as error:
                        exit_code = 124
                        stdout = error.stdout or ""
                        stderr = error.stderr or ""
                        if isinstance(stdout, bytes):
                            stdout = stdout.decode("utf-8", errors="replace")
                        if isinstance(stderr, bytes):
                            stderr = stderr.decode("utf-8", errors="replace")
                        stderr += f"\nrun exceeded {args.timeout}s timeout"
                    latency = time.perf_counter() - started
                    answer = extract_final(stdout)
                    score = int(exit_code == 0 and normalize(answer) == normalize(task["answer"]))
                    results.append(
                        {
                            "runId": run_id,
                            "architecture": architecture_name,
                            "model": model,
                            "taskId": task["id"],
                            "repetition": repetition,
                            "expected": task["answer"],
                            "answer": answer,
                            "score": score,
                            "latencySeconds": round(latency, 6),
                            "exitCode": exit_code,
                            "stdout": stdout.strip(),
                            "stderr": stderr.strip(),
                            "provenance": str(record_path.relative_to(output_dir)),
                        }
                    )

    runs_path = output_dir / "runs.jsonl"
    runs_path.write_text(
        "".join(json.dumps(row, ensure_ascii=False) + "\n" for row in results),
        encoding="utf-8",
    )

    groups: dict[tuple[str, str], list[dict]] = {}
    for row in results:
        groups.setdefault((row["architecture"], row["model"]), []).append(row)

    summary = {
        "schemaVersion": "orh.evaluation/v1",
        "createdAt": datetime.now(timezone.utc).isoformat(),
        "taskCount": len(tasks),
        "repetitions": args.repetitions,
        "groups": [],
    }
    for (architecture, model), rows in sorted(groups.items()):
        latencies = [row["latencySeconds"] for row in rows]
        summary["groups"].append(
            {
                "architecture": architecture,
                "model": model,
                "runs": len(rows),
                "successfulRuns": sum(row["exitCode"] == 0 for row in rows),
                "exactAnswerAccuracy": sum(row["score"] for row in rows) / len(rows),
                "meanLatencySeconds": statistics.mean(latencies),
                "medianLatencySeconds": statistics.median(latencies),
            }
        )

    (output_dir / "summary.json").write_text(
        json.dumps(summary, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    print(json.dumps(summary, indent=2, ensure_ascii=False))
    print(f"Results: {output_dir}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
