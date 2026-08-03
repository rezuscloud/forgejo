#!/usr/bin/env python3
"""
Generate a ranked, token-budgeted map.txt rollup from a SCIP index.

Reads a .scip file (via `scip print --json`), extracts the symbol/dependency
graph, clusters by package, ranks by relationship count (approximate PageRank),
and emits a text file the wiki agent reads to derive C4 architecture diagrams.

Usage: generate-map.py <index.scip> [--max-tokens 8000]
"""
import json, subprocess, sys, argparse, collections, re
from pathlib import Path

def main():
    ap = argparse.ArgumentParser(description="Generate map.txt from SCIP index")
    ap.add_argument("scip_file", help="Path to .scip index file")
    ap.add_argument("--max-tokens", type=int, default=8000, help="Token budget for output")
    args = ap.parse_args()

    # Run scip print --json to get the structured index
    result = subprocess.run(
        ["scip", "print", "--json", args.scip_file],
        capture_output=True, text=True
    )
    if result.returncode != 0:
        print(f"Error: scip print failed: {result.stderr}", file=sys.stderr)
        sys.exit(1)

    data = json.loads(result.stdout)
    docs = data.get("documents", [])

    # ---- Phase 1: Extract packages and their symbols ----
    packages = {}  # pkg_name -> {symbols, files, deps}

    KIND_MAP = {
        0: "unknown", 1: "assoc", 2: "package", 3: "type", 4: "enum",
        5: "struct", 6: "interface", 7: "trait", 8: "method",
        9: "function", 10: "macro", 11: "parameter", 12: "local",
        13: "self", 14: "field", 15: "var", 16: "const",
        17: "constructor", 18: "enum_member", 19: "label",
        20: "type_parameter", 21: "namespace", 22: "type_alias",
        23: "module",
        49: "class_like", 61: "local_var",
    }

    # Map of symbol -> {package, kind, display_name, relationships: set}
    all_symbols = {}

    for doc in docs:
        rel_path = doc.get("relative_path", "?")
        # Derive package from the symbol strings
        for sym in doc.get("symbols", []):
            symbol_str = sym.get("symbol", "")
            display_name = sym.get("display_name", "?")
            kind = KIND_MAP.get(sym.get("kind", 0), f"kind{sym.get('kind',0)}")

            # Parse SCIP symbol to extract package
            # Format: scip-go gomod <module> <hash> `<pkg>`/<sym>
            pkg_match = re.search(r'`([^`]+)`/', symbol_str)
            pkg = pkg_match.group(1) if pkg_match else "unknown"

            # Extract relationship info
            rels = set()
            for rel in sym.get("relationships", []):
                rel_sym = rel.get("symbol", "")
                rel_pkg_match = re.search(r'`([^`]+)`/', rel_sym)
                rel_pkg = rel_pkg_match.group(1) if rel_pkg_match else "external"
                rels.add(rel_pkg)

            all_symbols[symbol_str] = {
                "package": pkg,
                "kind": kind,
                "display": display_name,
                "file": rel_path,
                "deps": rels,
            }

    # ---- Phase 2: Cluster by package, count cross-package deps ----
    pkg_symbols = collections.defaultdict(list)
    pkg_dep_counts = collections.defaultdict(collections.Counter)

    for sym_str, info in all_symbols.items():
        pkg = info["package"]
        pkg_symbols[pkg].append((info["display"], info["kind"], info["file"]))
        for dep_pkg in info["deps"]:
            if dep_pkg != pkg:
                pkg_dep_counts[pkg][dep_pkg] += 1

    # ---- Phase 3: Rank packages by total relationship count ----
    pkg_ranking = sorted(
        pkg_symbols.keys(),
        key=lambda p: sum(pkg_dep_counts[p].values()),
        reverse=True
    )

    # ---- Phase 4: Emit the map.txt (token-budgeted) ----
    lines = []
    lines.append(f"# Code Graph Map — derived from {Path(args.scip_file).name}")
    lines.append(f"# Tool: scip-go (Sourcegraph SCIP indexer for Go)")
    lines.append(f"# Generated: deterministic from source code")
    lines.append(f"# Total symbols: {len(all_symbols)} | Packages: {len(pkg_symbols)}")
    lines.append("")

    # Package dependency graph (the architecture backbone)
    lines.append("## Package Dependency Graph")
    lines.append("")
    for pkg in pkg_ranking:
        deps = pkg_dep_counts[pkg]
        if deps:
            dep_str = ", ".join(f"{d}({c})" for d, c in deps.most_common(10))
            lines.append(f"  {pkg} -> {dep_str}")
        else:
            lines.append(f"  {pkg} (leaf)")
    lines.append("")

    # Symbol inventory per package (sorted by ranking, token-budgeted)
    lines.append("## Package Inventory (ranked by relationship count)")
    lines.append("")

    token_budget = args.max_tokens
    current_tokens = len("\n".join(lines)) // 4  # rough token estimate

    for pkg in pkg_ranking:
        syms = pkg_symbols[pkg]
        exported = [(d, k, f) for d, k, f in syms if k in ("struct", "interface", "class_like", "function", "constructor", "type")]
        internal = [(d, k, f) for d, k, f in syms if k not in ("struct", "interface", "class_like", "function", "constructor", "type")]

        pkg_text = f"### {pkg}\n"
        pkg_text += f"  files: {len(set(f for _, _, f in syms))} | symbols: {len(syms)} | exported: {len(exported)} | deps: {sum(pkg_dep_counts[pkg].values())}\n"
        pkg_text += f"  imports from: {', '.join(d for d, c in pkg_dep_counts[pkg].most_common(5))}\n"

        # List exported symbols (the C4-relevant ones)
        if exported:
            pkg_text += "  exported:\n"
            for disp, kind, file in sorted(exported, key=lambda x: x[0])[:30]:
                pkg_text += f"    [{kind}] {disp} ({file})\n"

        pkg_tokens = len(pkg_text) // 4
        if current_tokens + pkg_tokens > token_budget:
            pkg_text = f"### {pkg} (truncated — token budget reached)\n"
            current_tokens += len(pkg_text) // 4
            lines.append(pkg_text)
            break

        lines.append(pkg_text)
        current_tokens += pkg_tokens

    print("\n".join(lines))

if __name__ == "__main__":
    main()
