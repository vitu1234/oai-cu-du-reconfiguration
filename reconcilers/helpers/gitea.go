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

	argov1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	gitplumbing "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	cudureconfigv1 "github.com/vitu1234/oai-cu-du-reconfiguration/v1/api/v1"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type IPInfo struct {
	Address string
	Gateway string
}

// CheckRepoForMatchingManifests clones a repo and searches YAML files for a name/namespace match.
func CheckRepoForMatchingManifests(ctx context.Context, repoURL string, branch string, clusterInfo *cudureconfigv1.ClusterInfo, typeResource string) (cloneDirectory string, matchingFiles []string, err error) {

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

			if typeResource == "NFDeployment" {
				if obj.Metadata.Name == clusterInfo.NFDeployment.Name && obj.Metadata.Namespace == clusterInfo.NFDeployment.Namespace {
					matches = append(matches, path)
					// no break: a file can have multiple docs
				}
			} else {
				if obj.Metadata.Name == clusterInfo.ConfigRef.Name && obj.Metadata.Namespace == clusterInfo.ConfigRef.Namespace {
					matches = append(matches, path)
					// no break: a file can have multiple docs
				}
			}

			// if obj.Metadata.Name == "cucp-regional" && obj.Metadata.Namespace == "oai-ran-cucp" {
			// 	matches = append(matches, path)
			// 	// no break: a file can have multiple docs
			// }
		}
		return nil
	}

	if err := filepath.WalkDir(tmpDir, walkFn); err != nil {
		return "", nil, err
	}

	return tmpDir, matches, nil
}

func UpdateInterfaceIPsNFDeployment(path string, newIPs map[string]IPInfo) error {
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

// UpdateInterfaceIPsConfigRefs updates the IP addresses and gateways
// in a manifest of kind Config that contains nested NFDeployment spec.
func UpdateInterfaceIPsConfigRefs(path string, newIPs map[string]IPInfo) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	// Drill down to spec.config.spec.interfaces
	spec, ok := doc["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	config, ok := spec["config"].(map[string]interface{})
	if !ok {
		return nil
	}

	nfSpec, ok := config["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	ifaces, ok := nfSpec["interfaces"].([]interface{})
	if !ok {
		return nil
	}

	for _, iface := range ifaces {
		ifaceMap, ok := iface.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := ifaceMap["name"].(string)
		if !ok {
			continue
		}
		if info, ok := newIPs[name]; ok {
			if ipv4, ok := ifaceMap["ipv4"].(map[string]interface{}); ok {
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

// argocd sync
func TriggerArgoCDSyncWithKubeClient(k8sClient client.Client, appName, namespace string) error {
	ctx := context.TODO()

	// Get the current Application object
	var app argov1alpha1.Application
	err := k8sClient.Get(ctx, types.NamespacedName{
		Name:      appName,
		Namespace: namespace,
	}, &app)
	if err != nil {
		return fmt.Errorf("failed to get Argo CD application: %w", err)
	}

	// Deep copy to modify
	// updated := app.DeepCopy()
	// now := time.Now().UTC()

	// Update Operation field to trigger sync
	app.Operation = &argov1alpha1.Operation{
		Sync: &argov1alpha1.SyncOperation{
			Revision: "HEAD",
			SyncStrategy: &argov1alpha1.SyncStrategy{
				Apply: &argov1alpha1.SyncStrategyApply{
					Force: true, // This enables kubectl apply --force
				},
			},
		},
		InitiatedBy: argov1alpha1.OperationInitiator{
			Username: "gitea-client",
		},
		// StartedAt: &now,
	}

	//doing operations for this argocd application
	// fmt.Printf("Triggering sync for application %v", app)
	// fmt.Printf("Triggering sync for application %s in namespace %s\n", appName, namespace)

	if err := k8sClient.Update(ctx, &app); err != nil {
		return fmt.Errorf("failed to update application with sync operation: %w", err)
	}

	return nil
}
