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
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CacheVersion is the current cache format version.
// Increment this when the cache format changes incompatibly.
const CacheVersion = 1

// Cache represents the cached state of proto generation.
type Cache struct {
	Version         int                      `json:"version"`
	ProtocFlagsHash string                   `json:"protocFlagsHash"`
	Packages        map[string]*PackageCache `json:"packages"`
}

// PackageCache represents the cached state of a single package.
type PackageCache struct {
	Hash           string    `json:"hash"`
	GeneratedFiles []string  `json:"generatedFiles"`
	ProtoFiles     []string  `json:"protoFiles"`
	LastGenerated  time.Time `json:"lastGenerated"`
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{
		Version:  CacheVersion,
		Packages: make(map[string]*PackageCache),
	}
}

// LoadCache loads a cache from the given path.
// Returns a new empty cache if the file doesn't exist or is corrupted.
func LoadCache(path string) (*Cache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCache(), nil
		}
		return nil, fmt.Errorf("reading cache file: %w", err)
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Corrupted cache, return a new one
		fmt.Fprintf(os.Stderr, "Warning: cache file corrupted, regenerating all packages: %v\n", err)
		return NewCache(), nil
	}

	// Check version compatibility
	if cache.Version != CacheVersion {
		fmt.Fprintf(os.Stderr, "Warning: cache version mismatch (got %d, expected %d), regenerating all packages\n", cache.Version, CacheVersion)
		return NewCache(), nil
	}

	if cache.Packages == nil {
		cache.Packages = make(map[string]*PackageCache)
	}

	return &cache, nil
}

// Save writes the cache to the given path.
func (c *Cache) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	return nil
}
