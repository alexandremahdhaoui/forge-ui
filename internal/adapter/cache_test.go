//go:build unit

// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package adapter

import (
	"sync"
	"testing"
	"time"

	"github.com/alexandremahdhaoui/forge-ui/internal/types"
)

func TestCache_SetAndGetRepoSummary(t *testing.T) {
	t.Parallel()

	c := NewCache()
	data := types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{
			"repo-a": {Name: "repo-a", Branch: "main", IsDirty: true},
		},
		Overviews: map[string]types.RepoOverview{},
		UpdatedAt: time.Now(),
	}
	c.SetWorkspace("ws1", data)

	summary, found := c.GetRepoSummary("ws1", "repo-a")
	if !found {
		t.Fatal("expected repo-a to be found in cache")
	}
	if got, want := summary.Branch, "main"; got != want {
		t.Errorf("Branch = %q, want %q", got, want)
	}
	if got, want := summary.IsDirty, true; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}
}

func TestCache_SetAndGetRepoOverview(t *testing.T) {
	t.Parallel()

	c := NewCache()
	data := types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{},
		Overviews: map[string]types.RepoOverview{
			"repo-b": {Name: "repo-b", Branch: "dev", Ahead: 3},
		},
		UpdatedAt: time.Now(),
	}
	c.SetWorkspace("ws1", data)

	overview, found := c.GetRepoOverview("ws1", "repo-b")
	if !found {
		t.Fatal("expected repo-b to be found in cache")
	}
	if got, want := overview.Branch, "dev"; got != want {
		t.Errorf("Branch = %q, want %q", got, want)
	}
	if got, want := overview.Ahead, 3; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
}

func TestCache_MissReturnsZeroValue(t *testing.T) {
	t.Parallel()

	c := NewCache()

	// Miss: workspace does not exist.
	summary, found := c.GetRepoSummary("nonexistent", "repo")
	if found {
		t.Error("expected found=false for nonexistent workspace")
	}
	if summary.Branch != "" {
		t.Errorf("Branch = %q, want empty string", summary.Branch)
	}
	if summary.IsDirty {
		t.Error("IsDirty = true, want false")
	}

	// Miss: workspace exists but repo does not.
	c.SetWorkspace("ws1", types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{
			"other-repo": {Name: "other-repo", Branch: "main"},
		},
		Overviews: map[string]types.RepoOverview{},
		UpdatedAt: time.Now(),
	})
	summary, found = c.GetRepoSummary("ws1", "missing-repo")
	if found {
		t.Error("expected found=false for missing repo in existing workspace")
	}
	if summary.Branch != "" {
		t.Errorf("Branch = %q, want empty string for missing repo", summary.Branch)
	}
}

func TestCache_AtomicSwap(t *testing.T) {
	t.Parallel()

	c := NewCache()

	// Initial data.
	c.SetWorkspace("ws1", types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{
			"repo-a": {Name: "repo-a", Branch: "old"},
		},
		Overviews: map[string]types.RepoOverview{},
		UpdatedAt: time.Now(),
	})

	// Replace with new data containing updated repo-a and new repo-b.
	c.SetWorkspace("ws1", types.CacheWorkspaceData{
		Summaries: map[string]types.RepoSummary{
			"repo-a": {Name: "repo-a", Branch: "new"},
			"repo-b": {Name: "repo-b", Branch: "feature"},
		},
		Overviews: map[string]types.RepoOverview{},
		UpdatedAt: time.Now(),
	})

	summary, found := c.GetRepoSummary("ws1", "repo-a")
	if !found {
		t.Fatal("expected repo-a to be found after swap")
	}
	if got, want := summary.Branch, "new"; got != want {
		t.Errorf("repo-a Branch = %q, want %q", got, want)
	}

	summary, found = c.GetRepoSummary("ws1", "repo-b")
	if !found {
		t.Fatal("expected repo-b to be found after swap")
	}
	if got, want := summary.Branch, "feature"; got != want {
		t.Errorf("repo-b Branch = %q, want %q", got, want)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := NewCache()
	var wg sync.WaitGroup

	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				repoName := "repo"
				data := types.CacheWorkspaceData{
					Summaries: map[string]types.RepoSummary{
						repoName: {Name: repoName, Branch: "main", Ahead: i},
					},
					Overviews: map[string]types.RepoOverview{
						repoName: {Name: repoName, Branch: "main", Ahead: i},
					},
					UpdatedAt: time.Now(),
				}
				c.SetWorkspace("ws1", data)
				c.GetRepoSummary("ws1", repoName)
				c.GetRepoOverview("ws1", repoName)
			}
		}(g)
	}

	wg.Wait()
}
