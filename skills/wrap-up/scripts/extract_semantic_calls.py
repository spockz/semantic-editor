#!/usr/bin/env python3
"""Read local Codex evidence for wrap-up without losing results to history summaries."""

import argparse
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import sys


THREAD = re.compile(r"[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\Z")
SCHEMA = 1


def thread_id(value):
    value = value.removeprefix("codex://threads/")
    if not THREAD.fullmatch(value):
        raise ValueError("expected a thread UUID or codex://threads/<UUID>")
    return value


def locate(home, identity):
    candidates = []
    for directory in ("sessions", "archived_sessions"):
        base = home / directory
        if base.exists():
            candidates.extend(base.rglob(f"*{identity}.jsonl"))
    if len(candidates) != 1:
        raise ValueError(f"thread {identity}: expected one transcript, found {len(candidates)}")
    return candidates[0].resolve()


def read_snapshot(path, frozen=None):
    with path.open("rb") as stream:
        data = stream.read(frozen["bytes"] if frozen else -1)
    digest = hashlib.sha256(data).hexdigest()
    if frozen and (len(data) != frozen["bytes"] or digest != frozen["sha256"]):
        raise ValueError(f"transcript snapshot changed: {path}; restart without cursor")
    records = []
    for number, line in enumerate(data.splitlines(keepends=True), 1):
        if not line.endswith(b"\n"):
            break
        try:
            record = json.loads(line)
        except (ValueError, UnicodeError) as error:
            raise ValueError(f"{path}:{number}: malformed JSON: {error}") from error
        if not isinstance(record, dict) or not isinstance(record.get("payload"), dict):
            raise ValueError(f"{path}:{number}: expected an event object with payload")
        records.append((number, record))
    return records, {"bytes": len(data), "sha256": digest}, bool(data and not data.endswith(b"\n"))


def semantic_name(name):
    if not isinstance(name, str):
        return None
    for prefix in ("mcp__semedit__", "mcp__semedit.", "semedit/", "semedit-"):
        name = name.removeprefix(prefix)
    return name if re.fullmatch(r"semantic_[a-zA-Z0-9_]+", name) else None


def batch_results(value):
    """Decode only recorded MCP envelopes, never code snippets or suggested calls."""
    if isinstance(value, dict):
        if isinstance(value.get("results"), list):
            return value["results"]
        for key in ("structuredContent", "result"):
            found = batch_results(value.get(key))
            if found is not None:
                return found
        for block in value.get("content", []):
            if not isinstance(block, dict) or block.get("type") != "text":
                continue
            text = block.get("text", "")
            if "Critical result (JSON): " in text:
                text = text.split("Critical result (JSON): ", 1)[1]
            try:
                decoded, _ = json.JSONDecoder().raw_decode(text)
            except (ValueError, TypeError):
                continue
            found = batch_results(decoded)
            if found is not None:
                return found
    return None


def children(arguments, result):
    edits = arguments.get("edits") if isinstance(arguments, dict) else None
    if not isinstance(edits, list):
        return [], "batch arguments do not contain an edits array"
    outcomes = batch_results(result)
    rows = []
    for index, edit in enumerate(edits):
        if not isinstance(edit, dict):
            raise ValueError(f"batch child {index}: expected object")
        if "tool" in edit:
            operation = edit["tool"]
            params = edit.get("params", {key: value for key, value in edit.items() if key != "tool"})
        elif len(edit) == 1:
            operation, params = next(iter(edit.items()))
        else:
            raise ValueError(f"batch child {index}: unrecognized request shape")
        outcome = outcomes[index] if outcomes is not None and index < len(outcomes) else None
        rows.append({"index": index, "tool": operation, "arguments": params,
                     "outcome_available": outcomes is not None and index < len(outcomes),
                     "outcome": outcome, "timing": "unavailable: only parent timing is recorded"})
    issue = None
    if outcomes is None:
        issue = "child execution unknown: no decodable recorded child results"
    elif len(outcomes) != len(edits):
        issue = f"requested {len(edits)} children, recorded {len(outcomes)} outcomes; unmatched execution unknown"
    return rows, issue


def extract(records, identity, path):
    calls, candidates, child_ids, limitations = [], [], set(), []
    seen = {}
    metadata = [row["payload"] for _, row in records if row.get("type") == "session_meta"]
    if not metadata or metadata[0].get("id") != identity:
        raise ValueError(f"transcript identity does not match {identity}: {path}")
    for line, event in records:
        payload = event["payload"]
        if event.get("type") != "event_msg" or payload.get("type") != "item_completed":
            continue
        if payload.get("thread_id", identity) != identity:
            continue
        item = payload.get("item")
        if not isinstance(item, dict):
            raise ValueError(f"{path}:{line}: completed item is missing")
        if item.get("type") == "SubAgentActivity" and item.get("agent_thread_id"):
            child_ids.add(thread_id(item["agent_thread_id"]))
        if item.get("type") == "CollabAgentToolCall" and item.get("tool") == "spawn_agent":
            child_ids.update(thread_id(value) for value in item.get("receiver_thread_ids", []))
        reference = {"thread_id": identity, "turn_id": payload.get("turn_id"),
                     "line": line, "timestamp": event.get("timestamp"), "id": item.get("id")}
        if item.get("type") == "CommandExecution":
            command = item.get("command", [])
            if re.search(r"semedit|semantic_", json.dumps(command)):
                candidates.append({**reference, "kind": "command_review_candidate", "recorded_item": item,
                                   "reason": "manual review required; mentions do not prove semantic execution"})
        if item.get("type") != "McpToolCall" or not semantic_name(item.get("tool")):
            continue
        key = item.get("id")
        if not key:
            raise ValueError(f"{path}:{line}: semantic call has no stable ID")
        if key in seen:
            if seen[key] != item:
                raise ValueError(f"{path}:{line}: conflicting duplicate call {key}")
            continue
        seen[key] = item
        call = {**reference, "kind": "semantic_call", "tool": semantic_name(item["tool"]),
                "started_at_ms": payload.get("started_at_ms"),
                "completed_at_ms": payload.get("completed_at_ms"),
                "result_available": "result" in item, "recorded_item": item}
        if not call["result_available"]:
            limitations.append(f"{key}: result absent from recorded item")
        if call["tool"] == "semantic_batch":
            call["children"], call["batch_limitation"] = children(item.get("arguments"), item.get("result"))
            if call["batch_limitation"]:
                limitations.append(f"{key}: {call['batch_limitation']}")
        calls.append(call)
    direct_ids = {event["payload"].get("call_id") for _, event in records
                  if event.get("type") == "response_item"
                  and event["payload"].get("type") in ("function_call", "custom_tool_call")
                  and semantic_name(event["payload"].get("name"))}
    responses = {}
    for line, event in records:
        payload = event["payload"]
        if event.get("type") == "response_item" and payload.get("type") in ("function_call_output", "custom_tool_call_output"):
            key = payload.get("call_id")
            if key not in direct_ids or key is None:
                continue
            if key in responses and responses[key][1]["payload"] != payload:
                raise ValueError(f"{path}:{line}: conflicting duplicate response {key}")
            responses[key] = (line, event)
    direct_seen = {}
    for line, event in records:
        payload = event["payload"]
        if event.get("type") != "response_item":
            continue
        if payload.get("type") in ("function_call", "custom_tool_call") and semantic_name(payload.get("name")):
            key = payload.get("call_id")
            if key in seen or payload.get("id") in seen:
                continue
            if seen:
                limitations.append(f"line {line}: direct semantic response item requires manual reconciliation")
                continue
            if not key:
                raise ValueError(f"{path}:{line}: missing direct call ID")
            if key in direct_seen:
                if direct_seen[key] != payload:
                    raise ValueError(f"{path}:{line}: conflicting duplicate direct call {key}")
                continue
            direct_seen[key] = payload
            response = responses.get(key)
            arguments = payload.get("arguments", payload.get("input"))
            if isinstance(arguments, str):
                try:
                    arguments = json.loads(arguments)
                except ValueError:
                    limitations.append(f"{key}: arguments are not JSON; original request retained")
            result = response[1]["payload"].get("output") if response else None
            item = {"id": key, "type": "DirectSemanticCall", "tool": semantic_name(payload["name"]),
                    "arguments": arguments, "status": "unknown", "request": payload}
            call = {"thread_id": identity, "turn_id": None, "line": line,
                    "timestamp": event.get("timestamp"), "id": key, "kind": "semantic_call",
                    "tool": item["tool"], "result_available": response is not None, "recorded_item": item}
            if response:
                item["result"] = result
                item["response"] = response[1]["payload"]
                call["response_line"] = response[0]
                call["response_timestamp"] = response[1].get("timestamp")
            else:
                limitations.append(f"{key}: response absent from transcript snapshot")
            if call["tool"] == "semantic_batch":
                decoded = result
                if isinstance(result, str):
                    try:
                        decoded = json.loads(result)
                    except ValueError:
                        decoded = None
                call["children"], call["batch_limitation"] = children(arguments, decoded)
                if call["batch_limitation"]:
                    limitations.append(f"{key}: {call['batch_limitation']}")
            calls.append(call)
    return calls, candidates, child_ids, limitations


def decode_cursor(token):
    try:
        value = json.loads(base64.urlsafe_b64decode(token + "=" * (-len(token) % 4)))
        if value["schema_version"] != SCHEMA or type(value["offset"]) is not int or value["offset"] < 0:
            raise ValueError("unsupported cursor version or offset")
        if not isinstance(value["snapshots"], dict) or not value["snapshots"]:
            raise ValueError("cursor has no snapshots")
        for identity, snapshot in value["snapshots"].items():
            thread_id(identity)
            if type(snapshot["bytes"]) is not int or snapshot["bytes"] < 0:
                raise ValueError("invalid snapshot size")
        return value
    except (ValueError, KeyError, TypeError) as error:
        raise ValueError(f"invalid cursor: {error}") from error


def page(home, identity, limit=20, cursor=None, include_children=False):
    identity = thread_id(identity)
    home = Path(home).expanduser().resolve()
    state = decode_cursor(cursor) if cursor else {
        "schema_version": SCHEMA, "thread_id": identity, "home": str(home),
        "include_children": include_children, "offset": 0, "snapshots": {}, "missing_children": []}
    if (state["thread_id"], state["home"], state["include_children"]) != (identity, str(home), include_children):
        raise ValueError("cursor scope mismatch; repeat the original thread, home, and child option")
    if cursor and identity not in state["snapshots"]:
        raise ValueError("cursor is missing the root transcript snapshot")
    if not 1 <= limit <= 100:
        raise ValueError("limit must be between 1 and 100")
    calls, candidates, sources, limitations = [], [], [], []
    pending = list(state["snapshots"]) if cursor else [identity]
    visited = set()
    missing = list(state["missing_children"])
    while pending:
        current = pending.pop(0)
        if current in visited:
            continue
        visited.add(current)
        try:
            path = locate(home, current)
        except ValueError as error:
            if current == identity or cursor:
                raise
            missing.append({"thread_id": current, "reason": str(error)})
            state["missing_children"] = missing
            continue
        records, snapshot, partial = read_snapshot(path, state["snapshots"].get(current))
        state["snapshots"][current] = snapshot
        found, review, child_ids, issues = extract(records, current, path)
        calls.extend(found)
        candidates.extend(review)
        limitations.extend(f"{current}: {issue}" for issue in issues)
        if partial:
            limitations.append(f"{current}: incomplete final JSONL record omitted")
        sources.append({"thread_id": current, "path": str(path), **snapshot,
                        "records": len(records), "compactions": sum(row.get("type") == "compacted" for _, row in records),
                        "child_thread_ids": sorted(child_ids)})
        if not include_children and child_ids:
            limitations.append(f"{current}: child transcripts excluded; rerun with --include-children")
        elif include_children and not cursor:
            pending.extend(sorted(child_ids - visited))
    calls.sort(key=lambda call: (call["timestamp"] or "", call["thread_id"], call["line"]))
    candidates.sort(key=lambda call: (call["timestamp"] or "", call["thread_id"], call["line"]))
    items = sorted(calls + candidates, key=lambda call: (call["timestamp"] or "", call["thread_id"], call["line"]))
    offset = state["offset"]
    if offset > len(items):
        raise ValueError("cursor offset exceeds snapshot inventory")
    next_offset = min(offset + limit, len(items))
    next_cursor = None
    if next_offset < len(items):
        state["offset"] = next_offset
        next_cursor = base64.urlsafe_b64encode(json.dumps(state, sort_keys=True).encode()).decode().rstrip("=")
    return {"schema_version": SCHEMA, "thread_id": identity, "sources": sources,
            "coverage": {"scope": "retained completed MCP events, or direct semantic response pairs in legacy logs, at frozen byte boundaries",
                         "limitations": limitations, "missing_children": missing,
                         "command_candidates_require_review": len(candidates),
                         "direct_semantic_calls": len(calls),
                         "results_recorded": sum(call["result_available"] for call in calls),
                         "results_without_recorded_value": sum(not call["result_available"] for call in calls),
                         "batch_parents": sum(call["tool"] == "semantic_batch" for call in calls),
                         "batch_requested_children": sum(len(call.get("children", [])) for call in calls),
                         "batch_recorded_child_outcomes": sum(child["outcome_available"] for call in calls for child in call.get("children", [])),
                         "historical_completeness": "unknown: deleted, unrecorded, or truncated evidence cannot be recovered"},
            "page": {"order": "oldest_first", "offset": offset, "limit": limit,
                     "total_items": len(items), "has_more": next_cursor is not None, "next_cursor": next_cursor},
            "items": items[offset:next_offset]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--thread-id", required=True)
    parser.add_argument("--codex-home", default=os.environ.get("CODEX_HOME", "~/.codex"))
    parser.add_argument("--include-children", action="store_true")
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--cursor")
    args = parser.parse_args()
    try:
        result = page(args.codex_home, args.thread_id, args.limit, args.cursor, args.include_children)
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(json.dumps({"error": str(error)}), file=sys.stderr)
        return 1
    json.dump(result, sys.stdout, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
