package main

// LargeContextOverlayFiles returns synthetic sibling packages to scale up context size
// dynamically at extraction time, keeping the golden txtar files as the single source of truth.
func LargeContextOverlayFiles() map[string]string {
	return map[string]string{
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
}
