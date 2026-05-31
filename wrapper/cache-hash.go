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
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// serviceRe matches service declarations in proto files.
var serviceRe = regexp.MustCompile(`(?m)^service\s+\w+\s*\{`)

// computePackageHash computes a deterministic hash for a package.
// The hash includes:
// 1. All proto file contents (sorted by name)
// 2. Dependency package hashes (memoized, sorted by name)
// 3. Relevant protoc flags
func (w *Wrapper) computePackageHash(pkg *PackageInfo) string {
	// Check memoization
	if hash, ok := w.hashMemo[pkg.ComputedPackage]; ok {
		return hash
	}

	h := sha256.New()

	// 1. Hash all proto file contents (sorted by name)
	files := make([]*FileInfo, len(pkg.Files))
	copy(files, pkg.Files)
	slices.SortFunc(files, func(a, b *FileInfo) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for _, fi := range files {
		h.Write([]byte(fi.Name))
		if fi.FullPath != "" {
			content, err := os.ReadFile(fi.FullPath)
			if err == nil {
				h.Write(content)
			}
		}
	}

	// 2. Hash dependency packages (sorted by computed package name)
	depPkgNames := pkg.ImportedPackageComputedNames()
	slices.Sort(depPkgNames)

	for _, depPkgName := range depPkgNames {
		if depPkg, ok := w.allPackages[depPkgName]; ok {
			depHash := w.computePackageHash(depPkg)
			h.Write([]byte(depHash))
		}
	}

	// 3. Hash relevant protoc flags
	h.Write([]byte(w.hashProtocFlags()))

	hash := hex.EncodeToString(h.Sum(nil))
	w.hashMemo[pkg.ComputedPackage] = hash
	return hash
}

// hashProtocFlags returns a hash of protoc flags and tool versions that affect output.
func (w *Wrapper) hashProtocFlags() string {
	h := sha256.New()

	// Sort flags for deterministic hashing
	flags := make([]string, len(w.ProtocFlags))
	copy(flags, w.ProtocFlags)
	slices.Sort(flags)

	for _, flag := range flags {
		h.Write([]byte(flag))
	}

	// Hash tool versions string (provided via --tool_versions flag)
	h.Write([]byte(w.ToolVersions))

	return hex.EncodeToString(h.Sum(nil))
}

// hasPlugin checks if a specific protoc plugin is being used.
func (w *Wrapper) hasPlugin(name string) bool {
	for _, flag := range w.ProtocFlags {
		// Check for --plugin= or _out flags
		if strings.Contains(flag, name) {
			return true
		}
	}
	return false
}

// protoHasServices checks if a proto file declares any services.
func (w *Wrapper) protoHasServices(fi *FileInfo) bool {
	if fi.FullPath == "" {
		return false
	}
	content, err := os.ReadFile(fi.FullPath)
	if err != nil {
		return false
	}
	return serviceRe.Match(content)
}

// expectedOutputFiles returns the list of expected output files for a package.
func (w *Wrapper) expectedOutputFiles(pkg *PackageInfo) []string {
	var files []string

	hasGoPlugin := w.hasPlugin("go-lite")
	hasStarpcGo := w.hasPlugin("go-starpc")
	hasEsPlugin := w.hasPlugin("es-lite") || w.hasPlugin("es_out")
	hasStarpcEs := w.hasPlugin("es-starpc")

	for _, fi := range pkg.Files {
		if fi.FullPath == "" {
			continue
		}

		base := strings.TrimSuffix(filepath.Base(fi.Name), ".proto")
		dir := filepath.Dir(fi.FullPath)

		if hasGoPlugin {
			files = append(files, filepath.Join(dir, base+".pb.go"))
		}
		if hasStarpcGo && w.protoHasServices(fi) {
			files = append(files, filepath.Join(dir, base+"_srpc.pb.go"))
		}
		if hasEsPlugin {
			files = append(files, filepath.Join(dir, base+".pb.ts"))
		}
		if hasStarpcEs && w.protoHasServices(fi) {
			files = append(files, filepath.Join(dir, base+"_srpc.pb.ts"))
		}
	}

	return files
}

// allOutputFilesExist checks if all expected output files exist.
func (w *Wrapper) allOutputFilesExist(pkg *PackageInfo, cachedFiles []string) bool {
	// First check cached files
	for _, f := range cachedFiles {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return false
		}
	}

	// Also check currently expected files (in case plugins changed)
	expected := w.expectedOutputFiles(pkg)
	for _, f := range expected {
		if _, err := os.Stat(f); os.IsNotExist(err) {
			return false
		}
	}

	return true
}
