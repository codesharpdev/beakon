package resolver

import (
	"os"
	"testing"

	"github.com/beakon/beakon/pkg"
)

// --- IsBuiltin ---

func TestIsBuiltin_Go(t *testing.T) {
	for _, sym := range []string{"make", "append", "len", "cap", "panic", "new", "min", "max"} {
		if !IsBuiltin("go", sym) {
			t.Errorf("go: IsBuiltin(%q) = false, want true", sym)
		}
	}
}

func TestIsBuiltin_Python(t *testing.T) {
	for _, sym := range []string{"print", "len", "range", "type", "str", "int"} {
		if !IsBuiltin("python", sym) {
			t.Errorf("python: IsBuiltin(%q) = false, want true", sym)
		}
	}
}

func TestIsBuiltin_TypeScript(t *testing.T) {
	for _, sym := range []string{"console", "JSON", "Math", "Promise", "parseInt"} {
		if !IsBuiltin("typescript", sym) {
			t.Errorf("typescript: IsBuiltin(%q) = false, want true", sym)
		}
	}
}

func TestIsBuiltin_Rust(t *testing.T) {
	for _, sym := range []string{"Box", "Vec", "String", "Option", "Result"} {
		if !IsBuiltin("rust", sym) {
			t.Errorf("rust: IsBuiltin(%q) = false, want true", sym)
		}
	}
}

func TestIsBuiltin_NotBuiltin(t *testing.T) {
	cases := []struct{ lang, sym string }{
		{"go", "json.Marshal"},
		{"go", "zap.New"},
		{"python", "requests.get"},
		{"typescript", "axios.get"},
	}
	for _, c := range cases {
		if IsBuiltin(c.lang, c.sym) {
			t.Errorf("%s: IsBuiltin(%q) = true, want false", c.lang, c.sym)
		}
	}
}

// --- Go import parsing ---

func TestParseGoImports_Simple(t *testing.T) {
	src := []byte(`package main

import "encoding/json"
import "fmt"
`)
	m := parseGoImports(src)
	if imp, ok := m["json"]; !ok {
		t.Error("expected 'json' qualifier")
	} else {
		if imp.Package != "encoding/json" {
			t.Errorf("package = %q, want encoding/json", imp.Package)
		}
		if imp.Stdlib != "yes" {
			t.Errorf("stdlib = %q, want yes", imp.Stdlib)
		}
	}
	if imp, ok := m["fmt"]; !ok {
		t.Error("expected 'fmt' qualifier")
	} else if imp.Stdlib != "yes" {
		t.Errorf("fmt stdlib = %q, want yes", imp.Stdlib)
	}
}

func TestParseGoImports_Block(t *testing.T) {
	src := []byte(`package main

import (
	"encoding/json"
	"github.com/uber-go/zap"
	j "encoding/json"
)
`)
	m := parseGoImports(src)
	if _, ok := m["json"]; !ok {
		t.Error("expected 'json' from block import")
	}
	if imp, ok := m["zap"]; !ok {
		t.Error("expected 'zap' qualifier")
	} else {
		if imp.Stdlib != "no" {
			t.Errorf("zap stdlib = %q, want no", imp.Stdlib)
		}
	}
	if imp, ok := m["j"]; !ok {
		t.Error("expected aliased 'j' qualifier")
	} else if imp.Package != "encoding/json" {
		t.Errorf("j package = %q, want encoding/json", imp.Package)
	}
}

func TestParseGoImports_ThirdParty(t *testing.T) {
	m := parseGoImports([]byte(`package x
import "go.uber.org/zap"
`))
	imp, ok := m["zap"]
	if !ok {
		t.Fatal("expected 'zap'")
	}
	if imp.Stdlib != "no" {
		t.Errorf("zap stdlib = %q, want no", imp.Stdlib)
	}
	if imp.Package != "go.uber.org/zap" {
		t.Errorf("package = %q", imp.Package)
	}
}

// --- TypeScript import parsing ---

func TestParseTSImports_Named(t *testing.T) {
	src := []byte(`import { readFile } from 'node:fs/promises'
import { useState } from 'react'
import axios from 'axios'
import * as path from 'path'
`)
	m := parseTSImports(src)
	if imp, ok := m["readFile"]; !ok {
		t.Error("expected 'readFile'")
	} else if imp.Stdlib != "yes" {
		t.Errorf("readFile stdlib = %q, want yes", imp.Stdlib)
	}

	if imp, ok := m["useState"]; !ok {
		t.Error("expected 'useState'")
	} else if imp.Package != "react" {
		t.Errorf("useState package = %q, want react", imp.Package)
	}

	if _, ok := m["axios"]; !ok {
		t.Error("expected 'axios' default import")
	}

	if imp, ok := m["path"]; !ok {
		t.Error("expected 'path' namespace import")
	} else if imp.Stdlib != "unknown" {
		t.Errorf("path stdlib = %q, want unknown", imp.Stdlib)
	}
}

func TestParseTSImports_Require(t *testing.T) {
	src := []byte(`const express = require('express')
const { Router } = require('express')
`)
	m := parseTSImports(src)
	if _, ok := m["express"]; !ok {
		t.Error("expected 'express' from require")
	}
	if _, ok := m["Router"]; !ok {
		t.Error("expected 'Router' from destructured require")
	}
}

// --- Python import parsing ---

func TestParsePyImports(t *testing.T) {
	src := []byte(`import json
import numpy as np
from pathlib import Path
from os.path import join, exists
`)
	m := parsePyImports(src)
	if imp, ok := m["json"]; !ok {
		t.Error("expected 'json'")
	} else if imp.Stdlib != "yes" {
		t.Errorf("json stdlib = %q, want yes", imp.Stdlib)
	}

	if imp, ok := m["np"]; !ok {
		t.Error("expected 'np' alias")
	} else if imp.Package != "numpy" {
		t.Errorf("np package = %q, want numpy", imp.Package)
	}

	if _, ok := m["Path"]; !ok {
		t.Error("expected direct binding 'Path'")
	}

	if _, ok := m["join"]; !ok {
		t.Error("expected 'join' from os.path")
	}
	if _, ok := m["exists"]; !ok {
		t.Error("expected 'exists' from os.path")
	}
}

// --- Rust import parsing ---

func TestParseRustImports(t *testing.T) {
	src := []byte(`use std::collections::HashMap;
use serde::{Serialize, Deserialize};
use tokio as async_rt;
use std::io::Write;
`)
	m := parseRustImports(src)
	if imp, ok := m["HashMap"]; !ok {
		t.Error("expected 'HashMap'")
	} else if imp.Stdlib != "yes" {
		t.Errorf("HashMap stdlib = %q, want yes", imp.Stdlib)
	}

	if imp, ok := m["Serialize"]; !ok {
		t.Error("expected 'Serialize'")
	} else if imp.Package != "serde" {
		t.Errorf("Serialize package = %q, want serde", imp.Package)
	}

	if _, ok := m["Deserialize"]; !ok {
		t.Error("expected 'Deserialize'")
	}

	if imp, ok := m["async_rt"]; !ok {
		t.Error("expected 'async_rt' alias")
	} else if imp.Package != "tokio" {
		t.Errorf("async_rt package = %q, want tokio", imp.Package)
	}
}

// --- Java import parsing ---

func TestParseJavaImports(t *testing.T) {
	src := []byte(`import java.util.HashMap;
import org.springframework.web.bind.annotation.RestController;
import java.util.*;
`)
	m := parseJavaImports(src)
	if imp, ok := m["HashMap"]; !ok {
		t.Error("expected 'HashMap'")
	} else if imp.Stdlib != "yes" {
		t.Errorf("HashMap stdlib = %q, want yes", imp.Stdlib)
	}

	if imp, ok := m["RestController"]; !ok {
		t.Error("expected 'RestController'")
	} else if imp.Stdlib != "no" {
		t.Errorf("RestController stdlib = %q, want no", imp.Stdlib)
	}

	// Wildcard import should be stored as unresolved sentinel
	if imp, ok := m["__wildcard__java.util"]; !ok {
		t.Error("expected wildcard sentinel for java.util.*")
	} else if imp.Resolution != "unresolved" {
		t.Errorf("wildcard resolution = %q, want unresolved", imp.Resolution)
	}
}

// --- Enrich ---

func TestEnrich_FiltersBuiltins(t *testing.T) {
	calls := []pkg.CallEdge{
		{From: "main", To: "len"},
		{From: "main", To: "append"},
		{From: "main", To: "json.Marshal"},
	}
	src := []byte(`package main
import "encoding/json"
`)
	result := Enrich("/repo", "main.go", "go", src, calls)

	// len and append should be filtered
	for _, e := range result {
		if e.To == "len" || e.To == "append" {
			t.Errorf("builtin %q should be filtered out", e.To)
		}
	}
	// json.Marshal should be present
	found := false
	for _, e := range result {
		if e.To == "json.Marshal" {
			found = true
			if e.Package != "encoding/json" {
				t.Errorf("json.Marshal package = %q, want encoding/json", e.Package)
			}
			if e.Stdlib != "yes" {
				t.Errorf("json.Marshal stdlib = %q, want yes", e.Stdlib)
			}
		}
	}
	if !found {
		t.Error("json.Marshal should be present")
	}
}

func TestEnrich_EnrichesExternalTS(t *testing.T) {
	calls := []pkg.CallEdge{
		{From: "handler", To: "axios.get"},
		{From: "handler", To: "console.log"}, // builtin
	}
	src := []byte(`import axios from 'axios'
`)
	result := Enrich("/repo", "src/handler.ts", "typescript", src, calls)

	var found bool
	for _, e := range result {
		if e.To == "axios.get" {
			found = true
			if e.Package != "axios" {
				t.Errorf("axios.get package = %q, want axios", e.Package)
			}
		}
		if e.To == "console.log" {
			t.Error("console.log is a builtin and should be filtered")
		}
	}
	if !found {
		t.Error("axios.get should be present and enriched")
	}
}

func TestEnrich_InternalCallsPassThrough(t *testing.T) {
	calls := []pkg.CallEdge{
		{From: "AuthService.Login", To: "UserRepo.FindByID"},
	}
	src := []byte(`package auth
import "encoding/json"
`)
	result := Enrich("/repo", "auth/service.go", "go", src, calls)
	if len(result) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(result))
	}
	if result[0].Package != "" {
		t.Errorf("internal call should have no Package, got %q", result[0].Package)
	}
}

func TestParseTSImports_MultiLine(t *testing.T) {
	src := []byte(`import {
  readFile,
  writeFile,
} from 'node:fs/promises'
import {
  useState,
  useEffect,
} from 'react'
`)
	m := parseTSImports(src)
	if _, ok := m["readFile"]; !ok {
		t.Error("expected 'readFile' from multi-line import")
	}
	if _, ok := m["writeFile"]; !ok {
		t.Error("expected 'writeFile' from multi-line import")
	}
	if imp, ok := m["useState"]; !ok {
		t.Error("expected 'useState' from multi-line react import")
	} else if imp.Package != "react" {
		t.Errorf("useState package = %q, want react", imp.Package)
	}
	if _, ok := m["useEffect"]; !ok {
		t.Error("expected 'useEffect' from multi-line react import")
	}
}

func TestDevOnly_TSPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := []byte(`{
  "dependencies": {"react": "^18.0.0"},
  "devDependencies": {"vitest": "^1.0.0", "typescript": "^5.0.0"}
}`)
	if err := os.WriteFile(dir+"/package.json", pkgJSON, 0644); err != nil {
		t.Fatal(err)
	}
	// Write a dummy source file in a subdir
	subdir := dir + "/src"
	os.MkdirAll(subdir, 0755)
	lr := newLockfileReader(dir, "src/index.ts", "typescript")

	// react is in prod deps → dev_only: false
	reactDev := lr.DevOnly("react")
	if reactDev == nil {
		t.Fatal("expected non-nil DevOnly for react")
	}
	if *reactDev {
		t.Error("react should be dev_only: false (it's in dependencies)")
	}

	// vitest is only in devDependencies → dev_only: true
	vitestDev := lr.DevOnly("vitest")
	if vitestDev == nil {
		t.Fatal("expected non-nil DevOnly for vitest")
	}
	if !*vitestDev {
		t.Error("vitest should be dev_only: true (it's in devDependencies)")
	}
}

func TestDevOnly_NoPackageJSON(t *testing.T) {
	lr := newLockfileReader(t.TempDir(), "src/index.ts", "typescript")
	if lr.DevOnly("axios") != nil {
		t.Error("expected nil DevOnly when no package.json found")
	}
}

func TestEnrich_DevOnlyPopulated(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := []byte(`{
  "dependencies": {"axios": "^1.0.0"},
  "devDependencies": {"vitest": "^1.0.0"}
}`)
	os.WriteFile(dir+"/package.json", pkgJSON, 0644)

	calls := []pkg.CallEdge{
		{From: "handler", To: "axios.get"},
		{From: "test", To: "describe"},
	}
	src := []byte(`import axios from 'axios'
import { describe } from 'vitest'
`)
	result := Enrich(dir, "src/handler.ts", "typescript", src, calls)

	for _, e := range result {
		switch e.To {
		case "axios.get":
			if e.DevOnly == nil || *e.DevOnly {
				t.Errorf("axios.get: expected dev_only=false, got %v", e.DevOnly)
			}
		case "describe":
			if e.DevOnly == nil || !*e.DevOnly {
				t.Errorf("describe: expected dev_only=true, got %v", e.DevOnly)
			}
		}
	}
}

func TestEnrich_PythonBuiltins(t *testing.T) {
	calls := []pkg.CallEdge{
		{From: "process_data", To: "len"},
		{From: "process_data", To: "print"},
		{From: "process_data", To: "requests.get"},
	}
	src := []byte(`import requests
`)
	result := Enrich("/repo", "app.py", "python", src, calls)
	for _, e := range result {
		if e.To == "len" || e.To == "print" {
			t.Errorf("python builtin %q should be filtered", e.To)
		}
	}
	found := false
	for _, e := range result {
		if e.To == "requests.get" {
			found = true
			if e.Package != "requests" {
				t.Errorf("requests.get package = %q, want requests", e.Package)
			}
		}
	}
	if !found {
		t.Error("requests.get should be present")
	}
}
