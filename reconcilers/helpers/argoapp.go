package helpers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	argo "github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/argo"
	helpers "github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/argo"
	"gopkg.in/yaml.v3"
)

type ArgoAppSkipResourcesIgnoreDifferences struct {
	Group        string   `json:"group,omitempty"`
	Kind         string   `json:"kind,omitempty"`
	Name         string   `json:"name,omitempty"`
	JsonPointers []string `json:"jsonPointers,omitempty"`
}

func CreateAndPushArgoApp(
	ctx context.Context,
	client *gitea.Client,
	username, repoName, folder, sourceRepo, targetClusterRepo string,
	log logr.Logger,
) (string, error) {
	app := argo.ArgoAppSpec{
		APIVersion: "argoproj.io/v1alpha1",
		Kind:       "Application",
	}

	clusterName := repoName

	app.Metadata.Name = fmt.Sprintf("argocd-%s", clusterName)
	app.Metadata.Namespace = "argocd"
	app.Metadata.Finalizers = []string{"resources-finalizer.argocd.argoproj.io"}

	app.Spec.Project = "default"
	app.Spec.Source.RepoURL = sourceRepo
	app.Spec.Source.TargetRevision = "HEAD"
	app.Spec.Source.Path = "."
	app.Spec.Source.Directory.Recurse = true

	app.Spec.Destination.Server = "https://kubernetes.default.svc"
	app.Spec.Destination.Namespace = "default"

	// app.Spec.SyncPolicy.Automated.Prune = true
	app.Spec.SyncPolicy.Automated.SelfHeal = true
	app.Spec.SyncPolicy.Automated.AllowEmpty = true
	app.Spec.SyncPolicy.SyncOptions = []string{"CreateNamespace=true"}

	// Correct placement of IgnoreDifferences
	app.Spec.IgnoreDifferences = []helpers.IgnoreDifference{
		{Group: "fn.kpt.dev", Kind: "ApplyReplacements"},
		{Group: "fn.kpt.dev", Kind: "StarlarkRun"},
		{Group: "infra.nephio.org", Kind: "WorkloadCluster"}, // <-- Add this
	}

	//add more ignore differences if provided
	// Add more ignore differences if provided
	// if len(ignoreDifferences) > 0 {
	// 	for _, diff := range ignoreDifferences {
	// 		app.Spec.IgnoreDifferences = append(app.Spec.IgnoreDifferences, helpers.IgnoreDifference{
	// 			Group:        diff.Group,
	// 			Kind:         diff.Kind,
	// 			Name:         diff.Name,
	// 			JSONPointers: diff.JsonPointers,
	// 		})
	// 	}
	// }
	yamlData, err := yaml.Marshal(app)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Argo application YAML: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s/argo-app-%s.yaml", folder, timestamp)
	message := fmt.Sprintf("argocd-app-%s-", clusterName)

	encodedContent := base64.StdEncoding.EncodeToString(yamlData)

	fileOpts := gitea.CreateFileOptions{
		Content: encodedContent,
		FileOptions: gitea.FileOptions{
			Message:    message,
			BranchName: "main",
		},
	}

	_, _, err = client.CreateFile(username, targetClusterRepo, filename, fileOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create file in Gitea: %w", err)
	}

	log.Info("Successfully pushed Argo application", "repo", targetClusterRepo, "file", filename)
	return filename, nil
}
