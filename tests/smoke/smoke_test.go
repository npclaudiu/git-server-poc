package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const defaultServerURL = "http://localhost:8080"

func TestGitSmoke(t *testing.T) {
	serverURL := os.Getenv("GIT_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultServerURL
	}

	// Wait for server to be healthy
	waitForServer(t, serverURL)

	repoName := fmt.Sprintf("smoke-test-%d", rand.New(rand.NewSource(time.Now().UnixNano())).Int())
	t.Logf("Running smoke test with repo: %s", repoName)

	// 1. Create Repository via API
	createRepo(t, serverURL, repoName)
	defer deleteRepo(t, serverURL, repoName)

	// 2. Prepare local workspace
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	repoURL := fmt.Sprintf("%s/repositories/%s.git", serverURL, repoName)

	// 3. Clone empty repository
	t.Log("Cloning empty repository...")
	runGit(t, tmpDir, "clone", repoURL, "repo-1")

	repo1Dir := filepath.Join(tmpDir, "repo-1")

	// 4. Create a commit
	t.Log("Creating initial commit...")
	runGit(t, repo1Dir, "checkout", "-b", "main")
	testFile := "hello.txt"
	content := []byte("Hello, Git Server!")
	if err := os.WriteFile(filepath.Join(repo1Dir, testFile), content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	runGit(t, repo1Dir, "add", ".")
	runGit(t, repo1Dir, "commit", "-m", "Initial commit")

	// 5. Push to server
	t.Log("Pushing to server...")
	runGit(t, repo1Dir, "push", "origin", "main")

	// 6. Clone again to verify
	t.Log("Cloning repository to second location...")
	runGit(t, tmpDir, "clone", repoURL, "repo-2")

	repo2Dir := filepath.Join(tmpDir, "repo-2")

	// 7. Verify content
	t.Log("Verifying content...")
	readContent, err := os.ReadFile(filepath.Join(repo2Dir, testFile))
	if err != nil {
		t.Fatalf("failed to read file from cloned repo: %v", err)
	}

	if !bytes.Equal(readContent, content) {
		t.Errorf("content mismatch: expected %q, got %q", content, readContent)
	}

	// 8. Verify git log
	t.Log("Verifying git log...")
	out := runGitOutput(t, repo2Dir, "log", "-1", "--pretty=%B")
	if string(bytes.TrimSpace(out)) != "Initial commit" {
		t.Errorf("log mismatch: expected 'Initial commit', got %q", out)
	}

	// 9. Modify file in repo-1 and push
	t.Log("Modifying file in repo-1...")
	newContent := []byte("Hello, Git Server! (v2)")
	if err := os.WriteFile(filepath.Join(repo1Dir, testFile), newContent, 0644); err != nil {
		t.Fatalf("failed to write updated file: %v", err)
	}
	runGit(t, repo1Dir, "add", ".")
	runGit(t, repo1Dir, "commit", "-m", "Second commit")
	t.Log("Pushing changes...")
	runGit(t, repo1Dir, "push", "origin", "main")

	// 10. Fetch in repo-2
	t.Log("Fetching in repo-2...")
	runGit(t, repo2Dir, "fetch", "origin")

	// 11. Verify fetch
	t.Log("Verifying fetch...")
	out = runGitOutput(t, repo2Dir, "log", "origin/main", "-1", "--pretty=%B")
	if string(bytes.TrimSpace(out)) != "Second commit" {
		t.Errorf("fetched log mismatch: expected 'Second commit', got %q", out)
	}

	// 12. Pull in repo-2
	t.Log("Pulling in repo-2...")
	runGit(t, repo2Dir, "pull", "origin", "main")

	// 13. Verify content in repo-2
	t.Log("Verifying content in repo-2...")
	readContent, err = os.ReadFile(filepath.Join(repo2Dir, testFile))
	if err != nil {
		t.Fatalf("failed to read file from repo-2: %v", err)
	}
	if !bytes.Equal(readContent, newContent) {
		t.Errorf("content mismatch after pull: expected %q, got %q", newContent, readContent)
	}

	t.Log("Smoke test passed!")

	// Restore working directory (though t.TempDir handles cleanup of the temp directory itself)
	if err := os.Chdir(originalWd); err != nil {
		t.Fatalf("failed to restore working directory: %v", err)
	}
}

func waitForServer(t *testing.T, url string) {
	healthURL := fmt.Sprintf("%s/health", url)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for server at %s", url)
		case <-ticker.C:
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				return
			}
		}
	}
}

func createRepo(t *testing.T, url, name string) {
	apiURL := fmt.Sprintf("%s/repositories", url)
	body, _ := json.Marshal(map[string]string{"name": name})

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("failed to create repo: status %d", resp.StatusCode)
	}
}

func deleteRepo(t *testing.T, url, name string) {
	apiURL := fmt.Sprintf("%s/repositories/%s", url, name)
	req, err := http.NewRequest(http.MethodDelete, apiURL, nil)
	if err != nil {
		t.Fatalf("failed to create delete request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("failed to delete repo %s: %v", name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Logf("failed to delete repo %s: status %d", name, resp.StatusCode)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Prevent git from asking for credentials or using local config that validates things too strictly
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_AUTHOR_NAME=Smoke Test",
		"GIT_AUTHOR_EMAIL=smoke@test.local",
		"GIT_COMMITTER_NAME=Smoke Test",
		"GIT_COMMITTER_EMAIL=smoke@test.local",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) []byte {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
	return out
}
