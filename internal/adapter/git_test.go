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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoInfo_CleanRepo(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.IsDirty, false; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}
	if got, want := len(result.StatusFiles), 0; got != want {
		t.Errorf("len(StatusFiles) = %d, want %d", got, want)
	}
	if got, want := result.Branch, tr.branch; got != want {
		t.Errorf("Branch = %q, want %q", got, want)
	}
	if result.LastCommitTime.IsZero() {
		t.Error("LastCommitTime is zero, want non-zero")
	}
	if got := len(result.RecentLogs); got < 1 {
		t.Errorf("len(RecentLogs) = %d, want >= 1", got)
	}
}

func TestRepoInfo_DirtyRepo_UntrackedFile(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)

	// Write an untracked file.
	if err := os.WriteFile(filepath.Join(tr.dir, "foo.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.IsDirty, true; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}
	if got := len(result.StatusFiles); got < 1 {
		t.Errorf("len(StatusFiles) = %d, want >= 1", got)
	}

	found := false
	for _, entry := range result.StatusFiles {
		if entry.Code == "??" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a StatusFiles entry with Code %q, got %v", "??", result.StatusFiles)
	}
}

func TestRepoInfo_DirtyRepo_StagedFile(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)

	// Write and stage a new file.
	if err := os.WriteFile(filepath.Join(tr.dir, "staged.txt"), []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	gitRunGitCmd(t, tr.dir, "add", "staged.txt")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.IsDirty, true; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}

	found := false
	for _, entry := range result.StatusFiles {
		// "A " after TrimRight becomes "A"
		if entry.Code == "A" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a StatusFiles entry with Code %q, got %v", "A", result.StatusFiles)
	}
}

func TestRepoInfo_DirtyRepo_UnstagedModification(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)

	// Modify the README.md created by the initial commit, without staging.
	if err := os.WriteFile(filepath.Join(tr.dir, "README.md"), []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.IsDirty, true; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}

	found := false
	for _, entry := range result.StatusFiles {
		if strings.Contains(entry.Code, "M") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a StatusFiles entry with Code containing %q, got %v", "M", result.StatusFiles)
	}
}

func TestRepoInfo_DetachedHead(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)

	gitRunGitCmd(t, tr.dir, "checkout", "--detach", "HEAD")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.Branch, ""; got != want {
		t.Errorf("Branch = %q, want %q (detached HEAD)", got, want)
	}
	if got, want := result.IsDirty, false; got != want {
		t.Errorf("IsDirty = %v, want %v", got, want)
	}
}

func TestRepoInfo_LogEntries(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)

	// Add 3 more commits (total 4 including init).
	tr.makeCommit(t, "a.txt", "a", "commit a")
	tr.makeCommit(t, "b.txt", "b", "commit b")
	tr.makeCommit(t, "c.txt", "c", "commit c")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(result.RecentLogs), 4; got != want {
		t.Errorf("len(RecentLogs) = %d, want %d", got, want)
	}
	for i, entry := range result.RecentLogs {
		if entry.Hash == "" {
			t.Errorf("RecentLogs[%d].Hash is empty", i)
		}
		if entry.Message == "" {
			t.Errorf("RecentLogs[%d].Message is empty", i)
		}
	}
}

func TestRepoInfo_Synced(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	tr.addRemote(t)

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.HasUpstream, true; got != want {
		t.Errorf("HasUpstream = %v, want %v", got, want)
	}
	if got, want := result.Ahead, 0; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
	if got, want := result.Behind, 0; got != want {
		t.Errorf("Behind = %d, want %d", got, want)
	}
}

func TestRepoInfo_AheadOfUpstream(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	tr.addRemote(t)

	// Make 2 local commits without pushing.
	tr.makeCommit(t, "local1.txt", "local1", "local commit 1")
	tr.makeCommit(t, "local2.txt", "local2", "local commit 2")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.HasUpstream, true; got != want {
		t.Errorf("HasUpstream = %v, want %v", got, want)
	}
	if got, want := result.Ahead, 2; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
	if got, want := result.Behind, 0; got != want {
		t.Errorf("Behind = %d, want %d", got, want)
	}
}

func TestRepoInfo_BehindUpstream(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	tr.addRemote(t)

	// Simulate 3 remote commits by another developer.
	tr.makeRemoteCommit(t, "remote1.txt", "r1", "remote commit 1")
	tr.makeRemoteCommit(t, "remote2.txt", "r2", "remote commit 2")
	tr.makeRemoteCommit(t, "remote3.txt", "r3", "remote commit 3")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.HasUpstream, true; got != want {
		t.Errorf("HasUpstream = %v, want %v", got, want)
	}
	if got, want := result.Ahead, 0; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
	if got, want := result.Behind, 3; got != want {
		t.Errorf("Behind = %d, want %d", got, want)
	}
}

func TestRepoInfo_AheadAndBehind(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	tr.addRemote(t)

	// Make 1 local commit (not pushed).
	tr.makeCommit(t, "local.txt", "local", "local commit")

	// Simulate 2 remote commits.
	tr.makeRemoteCommit(t, "remote1.txt", "r1", "remote commit 1")
	tr.makeRemoteCommit(t, "remote2.txt", "r2", "remote commit 2")

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.HasUpstream, true; got != want {
		t.Errorf("HasUpstream = %v, want %v", got, want)
	}
	if got, want := result.Ahead, 1; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
	if got, want := result.Behind, 2; got != want {
		t.Errorf("Behind = %d, want %d", got, want)
	}
}

func TestRepoInfo_NoRemote_GracefulFallback(t *testing.T) {
	t.Parallel()

	tr := newTestRepo(t)
	// Do NOT call addRemote -- no remote configured.

	gi := NewGitInfo()
	result, err := gi.RepoInfo(tr.dir)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := result.HasUpstream, false; got != want {
		t.Errorf("HasUpstream = %v, want %v", got, want)
	}
	if got, want := result.Ahead, 0; got != want {
		t.Errorf("Ahead = %d, want %d", got, want)
	}
	if got, want := result.Behind, 0; got != want {
		t.Errorf("Behind = %d, want %d", got, want)
	}
}
