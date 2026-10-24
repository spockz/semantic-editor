// WHY: This file streams Codex's local JSONL chat records into the analysis schema without loading the private transcript corpus into memory.

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const maxCodexTranscriptLine = 128 << 20

var errAnalysisPageFull = errors.New("analysis page is full")

type analysisPageCursor struct {
	BenchmarkIndex int    `json:"benchmark_index"`
	TranscriptFile string `json:"transcript_file,omitempty"`
	Ordinal        int64  `json:"ordinal,omitempty"`
}

func encodeAnalysisPageCursor(cursor analysisPageCursor) (string, error) {
	data, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func decodeAnalysisPageCursor(token string) (*analysisPageCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("invalid URL-safe base64: %w", err)
	}
	var cursor analysisPageCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid cursor JSON: %w", err)
	}
	if cursor.BenchmarkIndex < 0 || cursor.Ordinal < 0 {
		return nil, errors.New("cursor positions cannot be negative")
	}
	return &cursor, nil
}

type analysisEventEmitter func(analysisEvent, analysisPageCursor) error

type analysisPageWriter struct {
	encoder    *json.Encoder
	limit      int
	returned   int
	nextCursor string
	hasMore    bool
	since      *time.Time
}

func (writer *analysisPageWriter) emit(event analysisEvent, position analysisPageCursor) error {
	if writer.since != nil {
		occurredAt, err := time.Parse(time.RFC3339Nano, event.OccurredAt)
		if event.OccurredAt == "" {
			return nil
		}
		if err != nil {
			return fmt.Errorf("parse event timestamp %q: %w", event.OccurredAt, err)
		}
		if occurredAt.Before(*writer.since) {
			return nil
		}
	}
	if writer.limit > 0 && writer.returned >= writer.limit {
		token, err := encodeAnalysisPageCursor(position)
		if err != nil {
			return err
		}
		writer.nextCursor, writer.hasMore = token, true
		return errAnalysisPageFull
	}
	if err := writer.encoder.Encode(event); err != nil {
		return fmt.Errorf("encode analysis event: %w", err)
	}
	writer.returned++
	return nil
}

func (writer *analysisPageWriter) writePageInfo() error {
	if writer.limit == 0 {
		return nil
	}
	return writer.encoder.Encode(analysisEvent{
		SchemaVersion: analysisExtractSchemaVersion,
		Type:          "page_info",
		Source:        "analysis_export",
		Pagination:    &analysisPagination{Limit: writer.limit, Returned: writer.returned, HasMore: writer.hasMore, NextCursor: writer.nextCursor},
	})
}

type codexTranscriptLine struct {
	Type      string          `json:"type"`
	Ordinal   *int64          `json:"ordinal"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type codexTranscriptPayload struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	TurnID    string          `json:"turn_id"`
	Role      string          `json:"role"`
	Author    string          `json:"author"`
	Phase     string          `json:"phase"`
	Status    string          `json:"status"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace"`
	CallID    string          `json:"call_id"`
	Recipient string          `json:"recipient"`
	Arguments json.RawMessage `json:"arguments"`
	Input     json.RawMessage `json:"input"`
	Output    json.RawMessage `json:"output"`
	Content   json.RawMessage `json:"content"`
	Usage     json.RawMessage `json:"usage"`
	TurnUsage json.RawMessage `json:"turn_token_usage"`
}

type analysisCodexUsage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	CacheWriteInputTokens int `json:"cache_write_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

type codexTranscriptContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func listCodexTranscriptFiles(home string) ([]string, error) {
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return nil, fmt.Errorf("resolve Codex home: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return nil, fmt.Errorf("inspect Codex home: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("codex home %q is not a directory", home)
	}
	var files []string
	foundRoot := false
	for _, name := range []string{"sessions", "archived_sessions"} {
		root := filepath.Join(canonical, name)
		rootInfo, err := os.Lstat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("inspect Codex transcript directory %q: %w", name, err)
		}
		if !rootInfo.IsDir() {
			continue
		}
		foundRoot = true
		rootFiles, err := codexTranscriptFilesUnder(root)
		if err != nil {
			return nil, fmt.Errorf("walk Codex transcript directory %q: %w", name, err)
		}
		files = append(files, rootFiles...)
	}
	if !foundRoot {
		return nil, fmt.Errorf("codex home %q contains neither sessions/ nor archived_sessions/", home)
	}
	slices.Sort(files)
	return files, nil
}

func codexTranscriptFilesUnder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() || (!strings.HasSuffix(entry.Name(), ".jsonl") && !strings.HasSuffix(entry.Name(), ".jsonl.zst")) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

func prepareCodexTranscriptInputs(home, zstdBin string) ([]string, error) {
	if home == "" {
		return nil, nil
	}
	files, err := listCodexTranscriptFiles(home)
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		if !strings.HasSuffix(path, ".jsonl.zst") {
			continue
		}
		if _, err := exec.LookPath(zstdBin); err != nil {
			return nil, fmt.Errorf("compressed transcripts require zstd on PATH or --zstd-bin: %w", err)
		}
		break
	}
	return files, nil
}

func writeAnalysisOutput(output string, events []analysisEvent, codexHome string, codexFiles []string, zstdBin string, pageLimit int, cursor *analysisPageCursor, since *time.Time) error {
	write := func(writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetEscapeHTML(false)
		page := &analysisPageWriter{encoder: encoder, limit: pageLimit, since: since}
		return writeAnalysisPage(page, events, codexHome, codexFiles, zstdBin, cursor)
	}
	if output == "" {
		return write(os.Stdout)
	}
	return writeAnalysisAtomically(output, write)
}

func writeAnalysisPage(page *analysisPageWriter, events []analysisEvent, codexHome string, codexFiles []string, zstdBin string, cursor *analysisPageCursor) error {
	startIndex := 0
	if cursor != nil {
		startIndex = cursor.BenchmarkIndex
	}
	if startIndex > len(events) {
		return fmt.Errorf("page cursor benchmark index %d exceeds %d available events", startIndex, len(events))
	}
	err := emitBenchmarkAnalysisEvents(page, events, startIndex)
	if err != nil && !errors.Is(err, errAnalysisPageFull) {
		return err
	}
	if err == nil && codexHome != "" {
		err = streamCodexTranscripts(context.Background(), codexHome, codexFiles, zstdBin, len(events), cursor, page.emit)
		if err != nil && !errors.Is(err, errAnalysisPageFull) {
			return err
		}
	}
	if err := page.writePageInfo(); err != nil {
		return fmt.Errorf("encode page information: %w", err)
	}
	return nil
}

func emitBenchmarkAnalysisEvents(page *analysisPageWriter, events []analysisEvent, start int) error {
	for i := start; i < len(events); i++ {
		if err := page.emit(events[i], analysisPageCursor{BenchmarkIndex: i}); err != nil {
			return err
		}
	}
	return nil
}

func streamCodexTranscripts(ctx context.Context, home string, files []string, zstdBin string, benchmarkIndex int, cursor *analysisPageCursor, emit analysisEventEmitter) error {
	canonical, err := filepath.EvalSymlinks(home)
	if err != nil {
		return fmt.Errorf("resolve Codex home: %w", err)
	}
	resumePath, resumeOrdinal, err := resolveTranscriptResume(canonical, files, benchmarkIndex, cursor)
	if err != nil {
		return err
	}
	for _, path := range files {
		relative, err := filepath.Rel(canonical, path)
		if err != nil {
			return fmt.Errorf("make transcript path relative: %w", err)
		}
		relative = filepath.ToSlash(relative)
		if resumePath != "" && relative < resumePath {
			continue
		}
		startOrdinal := int64(0)
		if relative == resumePath {
			startOrdinal = resumeOrdinal
		}
		if err := streamCodexTranscriptFile(ctx, path, relative, zstdBin, benchmarkIndex, startOrdinal, emit); err != nil {
			return err
		}
	}
	return nil
}

func resolveTranscriptResume(canonical string, files []string, benchmarkIndex int, cursor *analysisPageCursor) (string, int64, error) {
	if cursor == nil || cursor.TranscriptFile == "" {
		return "", 0, nil
	}
	if cursor.BenchmarkIndex != benchmarkIndex {
		return "", 0, fmt.Errorf("cursor belongs to %d benchmark events, current inputs have %d", cursor.BenchmarkIndex, benchmarkIndex)
	}
	resumePath := filepath.ToSlash(cursor.TranscriptFile)
	for _, path := range files {
		relative, err := filepath.Rel(canonical, path)
		if err != nil {
			return "", 0, fmt.Errorf("make transcript path relative: %w", err)
		}
		if filepath.ToSlash(relative) == resumePath {
			return resumePath, cursor.Ordinal, nil
		}
	}
	return "", 0, fmt.Errorf("cursor transcript %q is no longer present; restart pagination", resumePath)
}

func streamCodexTranscriptFile(ctx context.Context, path, relative, zstdBin string, benchmarkIndex int, startOrdinal int64, emit analysisEventEmitter) error {
	if strings.HasSuffix(path, ".jsonl.zst") {
		if err := streamCompressedCodexTranscript(ctx, path, relative, zstdBin, benchmarkIndex, startOrdinal, emit); err != nil {
			return fmt.Errorf("codex transcript %q: %w", relative, err)
		}
		return nil
	}
	file, err := os.Open(path) // #nosec G304 -- paths are regular transcript files discovered below an explicitly selected Codex home.
	if err != nil {
		return fmt.Errorf("open transcript %q: %w", relative, err)
	}
	err = streamCodexTranscript(file, relative, benchmarkIndex, startOrdinal, emit)
	closeErr := file.Close()
	if err != nil {
		return fmt.Errorf("codex transcript %q: %w", relative, err)
	}
	if closeErr != nil {
		return fmt.Errorf("close transcript %q: %w", relative, closeErr)
	}
	return nil
}

func streamCompressedCodexTranscript(ctx context.Context, path, relative, zstdBin string, benchmarkIndex int, startOrdinal int64, emit analysisEventEmitter) error {
	command := exec.CommandContext(ctx, zstdBin, "-q", "-dc", "--", path) // #nosec G204 -- the executable is an explicit local CLI setting; flags and transcript path are separate arguments.
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open zstd output: %w", err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return fmt.Errorf("start zstd: %w", err)
	}
	streamErr := streamCodexTranscript(stdout, relative, benchmarkIndex, startOrdinal, emit)
	if streamErr != nil {
		_ = command.Process.Kill()
	}
	waitErr := command.Wait()
	if streamErr != nil {
		return streamErr
	}
	if waitErr != nil {
		return fmt.Errorf("decompress transcript: %w: %s", waitErr, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func streamCodexTranscript(reader io.Reader, relative string, benchmarkIndex int, startOrdinal int64, emit analysisEventEmitter) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxCodexTranscriptLine)
	chatID := transcriptIDFromPath(relative)
	var lineIndex int64
	for scanner.Scan() {
		currentIndex := lineIndex
		lineIndex++
		var line codexTranscriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			return fmt.Errorf("decode JSONL record: %w", err)
		}
		var payload codexTranscriptPayload
		if len(line.Payload) != 0 {
			if err := json.Unmarshal(line.Payload, &payload); err != nil {
				return fmt.Errorf("decode %s payload at ordinal %v: %w", line.Type, line.Ordinal, err)
			}
		}
		if line.Type == "session_meta" && payload.ID != "" {
			chatID = payload.ID
		}
		position := currentIndex
		if line.Ordinal != nil {
			position = *line.Ordinal
		}
		if position < startOrdinal {
			continue
		}
		record, include := codexAnalysisRecord(line, payload, chatID, relative)
		if !include {
			continue
		}
		event := analysisEvent{
			SchemaVersion: analysisExtractSchemaVersion,
			Type:          "codex_chat",
			Source:        "codex_home:" + relative,
			OccurredAt:    line.Timestamp,
			RunID:         chatID,
			TaskID:        chatID,
			CodexChat:     &record,
		}
		if err := emit(event, analysisPageCursor{BenchmarkIndex: benchmarkIndex, TranscriptFile: relative, Ordinal: position}); err != nil {
			return fmt.Errorf("emit record at ordinal %v: %w", line.Ordinal, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read JSONL: %w", err)
	}
	return nil
}

func codexAnalysisRecord(line codexTranscriptLine, payload codexTranscriptPayload, chatID, relative string) (analysisCodexChatRecord, bool) {
	record := analysisCodexChatRecord{
		ChatID: chatID, File: relative, Ordinal: line.Ordinal, Timestamp: line.Timestamp,
		TurnID: payload.TurnID, ItemID: payload.ID, Status: payload.Status,
	}
	switch line.Type {
	case "response_item":
		switch payload.Type {
		case "message":
			if payload.Role != "user" && payload.Role != "assistant" {
				return analysisCodexChatRecord{}, false
			}
			if payload.Role == "assistant" && payload.Phase != "" && payload.Phase != "final_answer" && payload.Phase != "commentary" {
				return analysisCodexChatRecord{}, false
			}
			record.Kind, record.Role, record.Phase = "message", payload.Role, payload.Phase
			record.Text, record.ContentTypes = codexTranscriptText(payload.Content)
		case "function_call", "custom_tool_call":
			record.Kind, record.ToolName, record.Namespace, record.CallID = "tool_call", payload.Name, payload.Namespace, payload.CallID
			record.Arguments, record.Input = payload.Arguments, payload.Input
		case "function_call_output", "custom_tool_call_output":
			record.Kind, record.CallID, record.Output = "tool_result", payload.CallID, payload.Output
		default:
			return analysisCodexChatRecord{}, false
		}
	case "token_usage_record":
		rawUsage := payload.TurnUsage
		if len(rawUsage) == 0 || string(rawUsage) == "null" {
			rawUsage = payload.Usage
		}
		var usage analysisCodexUsage
		if len(rawUsage) == 0 || string(rawUsage) == "null" || json.Unmarshal(rawUsage, &usage) != nil {
			return analysisCodexChatRecord{}, false
		}
		record.Kind, record.Usage = "token_usage", &usage
		return record, true
	default:
		return analysisCodexChatRecord{}, false
	}
	return record, true
}

func codexTranscriptText(raw json.RawMessage) (string, []string) {
	var parts []codexTranscriptContent
	if err := json.Unmarshal(raw, &parts); err != nil {
		var text string
		if json.Unmarshal(raw, &text) == nil {
			return text, []string{"text"}
		}
		return "", nil
	}
	var text strings.Builder
	var types []string
	for _, part := range parts {
		if part.Type != "" {
			types = append(types, part.Type)
		}
		if part.Text != "" {
			text.WriteString(part.Text)
		}
	}
	return text.String(), types
}

func transcriptIDFromPath(relative string) string {
	base := filepath.Base(relative)
	base = strings.TrimSuffix(base, ".jsonl.zst")
	base = strings.TrimSuffix(base, ".jsonl")
	return base
}

func writeAnalysisAtomically(path string, write func(io.Writer) error) error {
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing output: %w", err)
	}
	if info != nil && !info.Mode().IsRegular() {
		return fmt.Errorf("output %q is not a regular file", path)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryPath)
	}()
	buffered := bufio.NewWriter(temporary)
	if err := write(buffered); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := buffered.Flush(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("flush temporary output: %w", err)
	}
	if info != nil {
		if err := temporary.Chmod(info.Mode().Perm()); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("preserve output permissions: %w", err)
		}
		modified := time.Now()
		if minimum := info.ModTime().Add(time.Millisecond); modified.Before(minimum) {
			modified = minimum
		}
		if err := os.Chtimes(temporaryPath, modified, modified); err != nil {
			_ = temporary.Close()
			return fmt.Errorf("advance output modification time: %w", err)
		}
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish analysis output: %w", err)
	}
	return nil
}
