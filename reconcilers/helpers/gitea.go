package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	gitplumbing "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	cudureconfigv1 "github.com/vitu1234/oai-cu-du-reconfiguration/v1/api/v1"
	"gopkg.in/yaml.v3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type IPInfo struct {
	Address string
	Gateway string
}

// CheckRepoForMatchingManifests clones a repo and searches YAML files for a name/namespace match.
func CheckRepoForMatchingManifests(ctx context.Context, repoURL string, branch string, nf *cudureconfigv1.NFReconfig) (cloneDirectory string, matchingFiles []string, err error) {

	log := logf.FromContext(ctx)

	log.Info("url " + repoURL)
	// clone into a temp dir
	tmpDir, err := os.MkdirTemp("", "nf-manifests-*")
	if err != nil {
		return "", nil, err
	}
	//remove all the files
	// defer os.RemoveAll(tmpDir)

	_, err = git.PlainCloneContext(ctx, tmpDir, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: gitplumbing.ReferenceName("refs/heads/" + branch),
		Depth:         1,
		SingleBranch:  true,
	})
	if err != nil {
		return "", nil, fmt.Errorf("git clone failed: %w", err)
	}

	var matches []string

	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		// Only .yaml or .yml
		if ext := strings.ToLower(filepath.Ext(path)); ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Files may contain multiple YAML docs separated by ---
		dec := yaml.NewDecoder(bytes.NewReader(data))
		for {
			var obj struct {
				Metadata struct {
					Name      string `yaml:"name"`
					Namespace string `yaml:"namespace"`
				} `yaml:"metadata"`
			}
			if err := dec.Decode(&obj); err != nil {
				if err.Error() == "EOF" {
					break
				}
				return err
			}

			// if obj.Metadata.Name == nf.Name && obj.Metadata.Namespace == nf.Namespace {
			// 	matches = append(matches, path)
			// 	// no break: a file can have multiple docs
			// }
			if obj.Metadata.Name == "cucp-regional" && obj.Metadata.Namespace == "oai-ran-cucp" {
				matches = append(matches, path)
				// no break: a file can have multiple docs
			}
		}
		return nil
	}

	if err := filepath.WalkDir(tmpDir, walkFn); err != nil {
		return "", nil, err
	}

	return tmpDir, matches, nil
}

func UpdateInterfaceIPs(path string, newIPs map[string]IPInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return nil
	}
	ifaces, ok := spec["interfaces"].([]interface{})
	if !ok {
		return nil
	}

	for _, iface := range ifaces {
		m := iface.(map[string]interface{})
		name := m["name"].(string)
		if info, ok := newIPs[name]; ok {
			if ipv4, ok := m["ipv4"].(map[string]interface{}); ok {
				if info.Address != "" {
					ipv4["address"] = info.Address
				}
				if info.Gateway != "" {
					ipv4["gateway"] = info.Gateway
				}
			}
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

// CommitAndPush commits all staged changes in cloneDir and pushes to the remote branch.
// userName and password are Gitea credentials (from your secret).
func CommitAndPush(ctx context.Context, cloneDir, branch, repoURL, userName, password, commitMsg string) error {
	// userName, password, _, err := giteaclient.GetGiteaSecretUserNamePassword(ctx)

	// Open the repository
	r, err := git.PlainOpen(cloneDir)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	// Stage all changes
	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Create commit
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  userName,
			Email: fmt.Sprintf("%s@example.com", userName),
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Push commit
	if err := r.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)),
		},
		Auth: &http.BasicAuth{
			Username: userName,
			Password: password,
		},
	}); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}
