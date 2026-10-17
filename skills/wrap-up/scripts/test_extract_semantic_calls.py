"""Exercise the wrap-up CLI against recorded evidence and changing transcript snapshots."""

import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("extract_semantic_calls.py")
ROOT = Path(__file__).resolve().parents[3]
ROOT_ID = "00000000-0000-0000-0000-000000000001"
CHILD_ID = "00000000-0000-0000-0000-000000000002"


class TranscriptCLI(unittest.TestCase):
    def setUp(self):
        scratch = ROOT / ".scratch"
        scratch.mkdir(exist_ok=True)
        self.directory = tempfile.TemporaryDirectory(dir=scratch)
        self.addCleanup(self.directory.cleanup)
        self.home = Path(self.directory.name)
        (self.home / "sessions").mkdir()

    def write(self, events, identity=ROOT_ID):
        path = self.home / "sessions" / f"rollout-{identity}.jsonl"
        rows = [{"type": "session_meta", "payload": {"id": identity}}, *events]
        path.write_text("".join(json.dumps(row) + "\n" for row in rows))
        return path

    def event(self, item, identity=ROOT_ID):
        return {"timestamp": "2026-10-17T09:00:00Z", "type": "event_msg", "payload": {
            "type": "item_completed", "thread_id": identity, "turn_id": "turn-1",
            "started_at_ms": 10, "completed_at_ms": 35, "item": item}}

    def call(self, identity="call-1", **extra):
        return self.event({"type": "McpToolCall", "id": identity, "server": "semedit",
                           "tool": "semantic_lookup", "arguments": {"symbol": "Foo"},
                           "status": "completed", "duration": {"secs": 0, "nanos": 1000},
                           "result": {"content": [{"type": "text", "text": "found"}],
                                      "structuredContent": {"result": {"file": "foo.go"}}}, **extra})

    def run_cli(self, *arguments, success=True):
        process = subprocess.run([sys.executable, str(SCRIPT), "--thread-id", ROOT_ID,
                                  "--codex-home", str(self.home), *arguments],
                                 text=True, capture_output=True, check=False)
        self.assertEqual(process.returncode, 0 if success else 1, process.stderr)
        if success:
            return json.loads(process.stdout)
        self.assertEqual(process.stdout, "")
        return json.loads(process.stderr)["error"]

    def test_exact_results_dedup_compaction_and_catalog_exclusion(self):
        call = self.call()
        rows = [call, call, {"type": "compacted", "payload": {}}, self.call("after"),
                {"type": "response_item", "payload": {"type": "custom_tool_call", "name": "exec",
                 "input": 'text(ALL_TOOLS.filter(x => x.name === "semantic_lookup"))'}},
                self.event({"type": "AgentMessage", "id": "suggestion", "text": "semantic_rename"})]
        self.write(rows)
        result = self.run_cli()
        self.assertEqual(result["coverage"]["direct_semantic_calls"], 2)
        self.assertEqual(result["sources"][0]["compactions"], 1)
        self.assertEqual(result["items"][0]["recorded_item"], call["payload"]["item"])
        self.assertEqual(result["items"][0]["started_at_ms"], 10)

    def test_pagination_survives_append_and_rejects_rewrite(self):
        path = self.write([self.call("first"), self.call("second"), self.call("third")])
        first = self.run_cli("--limit", "1")
        cursor = first["page"]["next_cursor"]
        with path.open("a") as stream:
            stream.write(json.dumps(self.call("later")) + "\n")
        second = self.run_cli("--limit", "1", "--cursor", cursor)
        third = self.run_cli("--limit", "1", "--cursor", second["page"]["next_cursor"])
        self.assertEqual([row["items"][0]["id"] for row in (first, second, third)], ["first", "second", "third"])
        self.assertFalse(third["page"]["has_more"])
        self.assertEqual(third["coverage"]["direct_semantic_calls"], 3)
        self.assertEqual(self.run_cli()["coverage"]["direct_semantic_calls"], 4)
        path.write_text(path.read_text().replace("Foo", "Bar"))
        self.assertIn("snapshot changed", self.run_cli("--cursor", cursor, success=False))

    def test_failed_batch_partial_results_and_request_forms(self):
        outcomes = [{"status": "ok", "result": {"edit_receipt": {"status": "applied_working"}}},
                    {"status": "error", "error": "rejected"}]
        result = {"isError": True, "content": [{"type": "text", "text": json.dumps({
            "structuredContent": {"result": {"status": "error", "results": outcomes}}})}]}
        edits = [{"tool": "semantic_rename", "params": {"symbol": "A", "to": "B"}},
                 {"tool": "semantic_replace_construct", "file": "foo.go"},
                 {"semantic_remove_construct": {"file": "foo.go"}}]
        self.write([self.call(tool="semantic_batch", status="failed", arguments={"edits": edits}, result=result)])
        row = self.run_cli()["items"][0]
        self.assertEqual(row["recorded_item"]["result"], result)
        self.assertEqual(row["children"][0]["outcome"], outcomes[0])
        self.assertEqual(row["children"][1]["outcome"], outcomes[1])
        self.assertFalse(row["children"][2]["outcome_available"])
        self.assertIn("execution unknown", row["batch_limitation"])
        self.assertEqual(row["children"][1]["arguments"], {"file": "foo.go"})
        self.assertEqual(row["children"][2]["tool"], "semantic_remove_construct")

    def test_missing_null_empty_and_truncated_results_remain_distinct(self):
        missing = self.call("missing")
        del missing["payload"]["item"]["result"]
        truncated = {"content": [{"type": "text", "text": '{"structuredContent": [truncated'}]}
        self.write([missing, self.call("null", result=None), self.call("empty", result={}),
                    self.call("truncated", tool="semantic_batch", arguments={"edits": [{"tool": "semantic_rename"}]}, result=truncated)])
        result = self.run_cli()
        self.assertEqual(result["coverage"]["results_recorded"], 3)
        self.assertFalse(result["items"][0]["result_available"])
        self.assertIsNone(result["items"][1]["recorded_item"]["result"])
        self.assertEqual(result["items"][2]["recorded_item"]["result"], {})
        self.assertIn("unknown", result["items"][3]["batch_limitation"])

    def test_children_cycle_missing_and_command_candidates(self):
        child = self.event({"type": "SubAgentActivity", "id": "spawn", "agent_thread_id": CHILD_ID})
        command = self.event({"type": "CommandExecution", "id": "cmd", "command": ["semedit", "lookup", "Foo"],
                              "aggregated_output": "found", "exit_code": 0})
        self.write([self.call(), child, command])
        self.assertTrue(self.run_cli()["coverage"]["limitations"])
        self.assertEqual(len(self.run_cli("--include-children")["coverage"]["missing_children"]), 1)
        back = self.event({"type": "SubAgentActivity", "agent_thread_id": ROOT_ID}, CHILD_ID)
        self.write([self.event(self.call("child")["payload"]["item"], CHILD_ID), back], CHILD_ID)
        result = self.run_cli("--include-children", "--limit", "1")
        self.assertEqual(result["coverage"]["direct_semantic_calls"], 2)
        self.assertEqual(result["coverage"]["command_candidates_require_review"], 1)
        self.assertEqual(len(result["sources"]), 2)
        next_page = self.run_cli("--include-children", "--cursor", result["page"]["next_cursor"])
        self.assertEqual(next_page["coverage"], result["coverage"])

    def test_invalid_inputs_fail_explicitly(self):
        path = self.write([self.call(), self.call("second")])
        with self.subTest("malformed complete record"):
            with path.open("a") as stream:
                stream.write("invalid\n")
            self.assertIn("malformed JSON", self.run_cli(success=False))
        self.write([self.call(), self.call("second")])
        with self.subTest("invalid cursor"):
            self.assertIn("invalid cursor", self.run_cli("--cursor", "nonsense", success=False))
        with self.subTest("invalid limit"):
            self.assertIn("limit", self.run_cli("--limit", "0", success=False))
        with self.subTest("scope mismatch"):
            cursor = self.run_cli("--limit", "1")["page"]["next_cursor"]
            self.assertIn("scope mismatch", self.run_cli("--cursor", cursor, "--include-children", success=False))
        with self.subTest("conflicting duplicate"):
            self.write([self.call(), self.call(arguments={"symbol": "Different"})])
            self.assertIn("conflicting duplicate", self.run_cli(success=False))

    def test_incomplete_tail_and_unsupported_direct_call_are_reported(self):
        path = self.write([self.call(), {"type": "response_item", "payload": {
            "type": "function_call", "name": "mcp__semedit__semantic_rename", "call_id": "direct"}}])
        with path.open("a") as stream:
            stream.write('{"unfinished":')
        result = self.run_cli()
        self.assertEqual(len(result["coverage"]["limitations"]), 2)
        self.assertEqual(result["coverage"]["direct_semantic_calls"], 1)

    def test_legacy_direct_call_pairing_preserves_output_and_pending_attempt(self):
        output = json.dumps({"isError": True, "message": "rejected"})
        self.write([{"type": "response_item", "payload": {
            "type": "function_call", "name": "mcp__semedit__semantic_rename", "call_id": "legacy",
            "arguments": json.dumps({"symbol": "Foo", "to": "Bar"})}},
            {"type": "response_item", "payload": {"type": "function_call_output", "call_id": "legacy", "output": output}},
            {"type": "response_item", "payload": {"type": "custom_tool_call", "name": "semantic_lookup",
                                                   "call_id": "pending", "input": '{"symbol":"Foo"}'}}])
        result = self.run_cli()
        self.assertEqual(result["coverage"]["direct_semantic_calls"], 2)
        self.assertEqual(result["coverage"]["results_recorded"], 1)
        self.assertEqual(result["items"][0]["recorded_item"]["result"], output)
        self.assertEqual(result["items"][0]["recorded_item"]["arguments"], {"symbol": "Foo", "to": "Bar"})
        self.assertFalse(result["items"][1]["result_available"])


if __name__ == "__main__":
    unittest.main()
