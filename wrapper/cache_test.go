// Copyright 2024 Aperture Robotics, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wrapper

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache()
	if cache.Version != CacheVersion {
		t.Errorf("expected version %d, got %d", CacheVersion, cache.Version)
	}
	if cache.Packages == nil {
		t.Error("expected non-nil Packages map")
	}
}

func TestCacheSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "test-cache.json")

	// Create and save cache
	cache := NewCache()
	cache.ProtocFlagsHash = "abc123"
	cache.Packages["test/pkg;pkg"] = &PackageCache{
		Hash:           "def456",
		GeneratedFiles: []string{"test.pb.go", "test_srpc.pb.go"},
		ProtoFiles:     []string{"test.proto"},
		LastGenerated:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := cache.Save(cachePath); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	// Load cache
	loaded, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}

	if loaded.Version != CacheVersion {
		t.Errorf("expected version %d, got %d", CacheVersion, loaded.Version)
	}
	if loaded.ProtocFlagsHash != "abc123" {
		t.Errorf("expected protocFlagsHash 'abc123', got '%s'", loaded.ProtocFlagsHash)
	}

	pkg, ok := loaded.Packages["test/pkg;pkg"]
	if !ok {
		t.Fatal("expected package 'test/pkg;pkg' in cache")
	}
	if pkg.Hash != "def456" {
		t.Errorf("expected hash 'def456', got '%s'", pkg.Hash)
	}
	if len(pkg.GeneratedFiles) != 2 {
		t.Errorf("expected 2 generated files, got %d", len(pkg.GeneratedFiles))
	}
}

func TestLoadCacheNonexistent(t *testing.T) {
	cache, err := LoadCache("/nonexistent/path/cache.json")
	if err != nil {
		t.Fatalf("unexpected error for nonexistent file: %v", err)
	}
	if cache.Version != CacheVersion {
		t.Errorf("expected version %d, got %d", CacheVersion, cache.Version)
	}
	if len(cache.Packages) != 0 {
		t.Errorf("expected empty packages, got %d", len(cache.Packages))
	}
}

func TestLoadCacheCorrupted(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "corrupted-cache.json")

	// Write corrupted JSON
	if err := os.WriteFile(cachePath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}

	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("unexpected error for corrupted file: %v", err)
	}
	if cache.Version != CacheVersion {
		t.Errorf("expected version %d, got %d", CacheVersion, cache.Version)
	}
}

func TestLoadCacheVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "old-cache.json")

	// Write cache with wrong version
	if err := os.WriteFile(cachePath, []byte(`{"version": 999, "packages": {}}`), 0644); err != nil {
		t.Fatalf("failed to write old cache: %v", err)
	}

	cache, err := LoadCache(cachePath)
	if err != nil {
		t.Fatalf("unexpected error for version mismatch: %v", err)
	}
	if cache.Version != CacheVersion {
		t.Errorf("expected version %d, got %d", CacheVersion, cache.Version)
	}
}

func TestProtoHasServices(t *testing.T) {
	dir := t.TempDir()

	// Proto file with service
	withService := filepath.Join(dir, "with_service.proto")
	if err := os.WriteFile(withService, []byte(`
syntax = "proto3";
package test;

message Request {}
message Response {}

service MyService {
  rpc Call(Request) returns (Response);
}
`), 0644); err != nil {
		t.Fatalf("failed to write proto: %v", err)
	}

	// Proto file without service
	withoutService := filepath.Join(dir, "without_service.proto")
	if err := os.WriteFile(withoutService, []byte(`
syntax = "proto3";
package test;

message Request {}
message Response {}
`), 0644); err != nil {
		t.Fatalf("failed to write proto: %v", err)
	}

	w := &Wrapper{}

	fiWith := &FileInfo{FullPath: withService}
	if !w.protoHasServices(fiWith) {
		t.Error("expected protoHasServices to return true for proto with service")
	}

	fiWithout := &FileInfo{FullPath: withoutService}
	if w.protoHasServices(fiWithout) {
		t.Error("expected protoHasServices to return false for proto without service")
	}
}

func TestHasPlugin(t *testing.T) {
	w := &Wrapper{
		ProtocFlags: []string{
			"--plugin=protoc-gen-go-lite",
			"--go-lite_out=./vendor",
			"--plugin=protoc-gen-go-starpc",
		},
	}

	if !w.hasPlugin("go-lite") {
		t.Error("expected hasPlugin to find go-lite")
	}
	if !w.hasPlugin("go-starpc") {
		t.Error("expected hasPlugin to find go-starpc")
	}
	if w.hasPlugin("nonexistent") {
		t.Error("expected hasPlugin to not find nonexistent plugin")
	}
}

func TestHashProtocFlags(t *testing.T) {
	w1 := &Wrapper{
		ProtocFlags: []string{"--flag1", "--flag2"},
	}
	w2 := &Wrapper{
		ProtocFlags: []string{"--flag2", "--flag1"},
	}
	w3 := &Wrapper{
		ProtocFlags: []string{"--flag1", "--flag3"},
	}

	hash1 := w1.hashProtocFlags()
	hash2 := w2.hashProtocFlags()
	hash3 := w3.hashProtocFlags()

	// Same flags in different order should produce same hash (sorted)
	if hash1 != hash2 {
		t.Errorf("expected same hash for same flags in different order, got %s and %s", hash1, hash2)
	}

	// Different flags should produce different hash
	if hash1 == hash3 {
		t.Error("expected different hash for different flags")
	}
}

func TestComputePackageHash(t *testing.T) {
	dir := t.TempDir()

	// Create a test proto file
	protoPath := filepath.Join(dir, "test.proto")
	if err := os.WriteFile(protoPath, []byte(`
syntax = "proto3";
package test;
message Msg {}
`), 0644); err != nil {
		t.Fatalf("failed to write proto: %v", err)
	}

	w := &Wrapper{
		ProtocFlags: []string{"--go-lite_out=./vendor"},
		allPackages: make(map[string]*PackageInfo),
		hashMemo:    make(map[string]string),
	}

	pkg := &PackageInfo{
		ComputedPackage: "test;test",
		Files: []*FileInfo{
			{Name: "test.proto", FullPath: protoPath},
		},
	}
	w.allPackages["test;test"] = pkg

	hash1 := w.computePackageHash(pkg)
	if hash1 == "" {
		t.Error("expected non-empty hash")
	}

	// Same content should produce same hash
	hash2 := w.computePackageHash(pkg)
	if hash1 != hash2 {
		t.Errorf("expected same hash for same content, got %s and %s", hash1, hash2)
	}

	// Modify proto content
	if err := os.WriteFile(protoPath, []byte(`
syntax = "proto3";
package test;
message Msg { string field = 1; }
`), 0644); err != nil {
		t.Fatalf("failed to write proto: %v", err)
	}

	// Clear memoization
	w.hashMemo = make(map[string]string)

	hash3 := w.computePackageHash(pkg)
	if hash1 == hash3 {
		t.Error("expected different hash for modified content")
	}
}

func TestAllOutputFilesExist(t *testing.T) {
	dir := t.TempDir()

	// Create some output files
	pbGo := filepath.Join(dir, "test.pb.go")
	if err := os.WriteFile(pbGo, []byte("package test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	w := &Wrapper{}
	pkg := &PackageInfo{}

	// All files exist
	if !w.allOutputFilesExist(pkg, []string{pbGo}) {
		t.Error("expected allOutputFilesExist to return true when all files exist")
	}

	// Some files missing
	if w.allOutputFilesExist(pkg, []string{pbGo, filepath.Join(dir, "missing.pb.go")}) {
		t.Error("expected allOutputFilesExist to return false when some files are missing")
	}
}
