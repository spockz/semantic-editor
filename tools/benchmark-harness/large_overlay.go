package main

import "maps"

// LargeContextOverlayFiles returns shared and task-specific sibling code for large-context trials.
func LargeContextOverlayFiles(taskID string) map[string]string {
	files := map[string]string{
		"api/router.go": `package api

type Route struct {
	Path   string
	Method string
}

func DefaultRoutes() []Route {
	return []Route{
		{Path: "/healthz", Method: "GET"},
		{Path: "/metrics", Method: "GET"},
	}
}
`,
		"api/router_test.go": `package api

import "testing"

func TestDefaultRoutes(t *testing.T) {
	routes := DefaultRoutes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}
}
`,
		"auth/auth.go": `package auth

type Session struct {
	Token string
}

func ValidateToken(token string) bool {
	return token != ""
}
`,
		"config/config.go": `package config

type Config struct {
	Port int
	Env  string
}

func LoadConfig() Config {
	return Config{Port: 8080, Env: "dev"}
}
`,
		"db/db.go": `package db

type Connection struct {
	DSN string
}

func Connect(dsn string) (*Connection, error) {
	return &Connection{DSN: dsn}, nil
}
`,
		"cache/cache.go": `package cache

type MemoryCache struct {
	data map[string]string
}

func New() *MemoryCache {
	return &MemoryCache{data: make(map[string]string)}
}
`,
		"metrics/metrics.go": `package metrics

type Counter struct {
	Name  string
	Value int64
}

func NewCounter(name string) *Counter {
	return &Counter{Name: name}
}
`,
		"models/models.go": `package models

type User struct {
	ID    int64
	Email string
}

type Product struct {
	SKU   string
	Price float64
}
`,
		"logging/logging.go": `package logging

type Logger struct {
	Prefix string
}

func NewLogger(p string) *Logger {
	return &Logger{Prefix: p}
}
`,
		"utils/utils.go": `package utils

import "strings"

func Sanitize(s string) string {
	return strings.TrimSpace(s)
}
`,
		"health/health.go": `package health

type Status struct {
	Alive bool
}

func Check() Status {
	return Status{Alive: true}
}
`,
	}

	if taskID == "task-11-mixed-sink-api-migration" {
		maps.Copy(files, mixedSinkMigrationOverlayFiles())
	}

	return files
}

// mixedSinkMigrationOverlayFiles adds real API consumers for the task-11 large variant.
func mixedSinkMigrationOverlayFiles() map[string]string {
	return map[string]string{
		"audit/fanout.go": `// FanoutSink delivers an event to each configured audit destination.
package audit

type FanoutSink struct {
	First  Sink
	Second Sink
}

func (f FanoutSink) Write(event Event) error {
	if err := f.First.Write(event); err != nil {
		return err
	}
	return f.Second.Write(event)
}
`,
		"audit/transaction.go": `// BufferedSink exposes transactional delivery without owning the persistence protocol.
package audit

type BufferedSink struct {
	Sink
	queue []Event
}

func (b *BufferedSink) Begin() error {
	b.queue = b.queue[:0]
	return nil
}

func (b *BufferedSink) Commit() error {
	for _, event := range b.queue {
		if err := b.Sink.Write(event); err != nil {
			return err
		}
	}
	return nil
}
`,
		"journal/file.go": `// Package journal contains a separate audit implementation used by batch jobs.
package journal

import "example.com/auditapp/audit"

type JournalSink struct {
	Entries []audit.Event
}

func (j *JournalSink) Write(event audit.Event) error {
	j.Entries = append(j.Entries, event)
	return nil
}

var _ audit.Sink = (*JournalSink)(nil)
`,
		"service/replay.go": `// Package service contains replay helpers for previously captured audit events.
package service

import "example.com/auditapp/audit"

func Replay(sink audit.Sink, events []audit.Event) error {
	writeEvent := audit.Sink.Write
	for _, event := range events {
		if err := writeEvent(sink, event); err != nil {
			return err
		}
	}
	return nil
}
`,
		"cmd/replay/main.go": `// Command replay demonstrates the default audit delivery path.
package main

import (
	"example.com/auditapp/audit"
	"example.com/auditapp/service"
)

func main() {
	sink := service.NewDispatcher(audit.NewMemorySink())
	_ = sink.Record(audit.Event{ID: "replay", Kind: "replayed"})
}
`,
	}
}
