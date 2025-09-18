package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	gitplumbing "github.com/go-git/go-git/v5/plumbing"
	cudureconfigv1 "github.com/vitu1234/oai-cu-du-reconfiguration/v1/api/v1"
	"gopkg.in/yaml.v3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

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
