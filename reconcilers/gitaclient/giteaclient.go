/*
Copyright 2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package giteaclient

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/go-logr/logr"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type GiteaClient interface {
	Start(ctx context.Context)
	IsInitialized() bool
	Get() *gitea.Client
	GetMyUserInfo() (*gitea.User, *gitea.Response, error)
	DeleteRepo(owner string, repo string) (*gitea.Response, error)
	GetRepo(userName string, repoCRName string) (*gitea.Repository, *gitea.Response, error)
	CreateRepo(createRepoOption gitea.CreateRepoOption) (*gitea.Repository, *gitea.Response, error)
	EditRepo(userName string, repoCRName string, editRepoOption gitea.EditRepoOption) (*gitea.Repository, *gitea.Response, error)
	DeleteAccessToken(value interface{}) (*gitea.Response, error)
	ListAccessTokens(opts gitea.ListAccessTokensOptions) ([]*gitea.AccessToken, *gitea.Response, error)
	CreateAccessToken(opt gitea.CreateAccessTokenOption) (*gitea.AccessToken, *gitea.Response, error)
}

var lock = &sync.Mutex{}

var singleInstance *gc

func GetClient(ctx context.Context, client resource.APIPatchingApplicator) (GiteaClient, error) {
	if ctx == nil {
		return nil, fmt.Errorf("failed creating gitea client, value of ctx cannot be nil")
	}

	if client.Client == nil {
		return nil, fmt.Errorf("failed creating gitea client, value of client.Client cannot be nil")
	}

	// Create singleton instance with check-lock-check pattern
	if singleInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		if singleInstance == nil {
			instance := &gc{
				client: client,
				l:      log.FromContext(ctx),
			}
			instance.Start(ctx) // Initialize the client
			singleInstance = instance
		}
	}

	return singleInstance, nil
}

func GetGiteaSecretUserNamePassword(ctx context.Context, client client.Client) (username, password, giturl string, erro error) {
	log := logf.FromContext(ctx)

	// Get Git server URL from environment variable
	gitURL := os.Getenv("GIT_SERVER_URL")
	if gitURL == "" {
		log.Error(fmt.Errorf("GIT_SERVER_URL environment variable not set"), "Failed to initialize git client")
		return
	}

	// Get Git credentials from secret
	gitSecretName := os.Getenv("GIT_SECRET_NAME")
	if gitSecretName == "" {
		gitSecretName = "git-user-secret" // default secret name
	}

	gitSecretNamespace := os.Getenv("GIT_SECRET_NAMESPACE")
	if gitSecretNamespace == "" {
		if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
			gitSecretNamespace = ns
		} else {
			gitSecretNamespace = "default"
		}
	}

	// Get the secret containing Git credentials
	secret := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      gitSecretName,
		Namespace: gitSecretNamespace,
	}, secret)
	if err != nil {
		log.Error(err, "Failed to get git credentials secret")
		return "", "", "", err
	}

	return string(secret.Data["username"]), string(secret.Data["password"]), gitURL, nil

}

type gc struct {
	client resource.APIPatchingApplicator

	giteaClient *gitea.Client
	l           logr.Logger
}

func (r *gc) Start(ctx context.Context) {
	r.l = log.FromContext(ctx)

	username, password, gitURL, err := GetGiteaSecretUserNamePassword(ctx, r.client)
	if err != nil {
		r.l.Error(err, "Failed to get gitea username and password from secret, make sure it exists")
		return
	}
	// Initialize Gitea client with URL and credentials
	client, err := gitea.NewClient(gitURL,
		gitea.SetBasicAuth(
			username,
			password,
		))
	if err != nil {
		r.l.Error(err, "Failed to create gitea client")
		return
	}

	// Test the connection
	_, _, err = client.GetMyUserInfo()
	if err != nil {
		r.l.Error(err, "Failed to verify gitea connection")
		return
	}

	r.giteaClient = client
	r.l.Info("Successfully initialized gitea client", "url", gitURL)
}

// Add a helper method to wait for initialization
func (r *gc) WaitForInitialization(ctx context.Context, timeout time.Duration) error {
	start := time.Now()
	for {
		if r.IsInitialized() {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for gitea client initialization")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			continue
		}
	}
}

func (r *gc) IsInitialized() bool {
	return r.giteaClient != nil
}

func (r *gc) Get() *gitea.Client {
	return r.giteaClient
}

func (r *gc) GetMyUserInfo() (*gitea.User, *gitea.Response, error) {
	return r.giteaClient.GetMyUserInfo()
}

func (r *gc) DeleteRepo(owner string, repo string) (*gitea.Response, error) {
	return r.giteaClient.DeleteRepo(owner, repo)
}

func (r *gc) GetRepo(userName string, repoCRName string) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.GetRepo(userName, repoCRName)
}

func (r *gc) CreateRepo(createRepoOption gitea.CreateRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.CreateRepo(createRepoOption)
}

func (r *gc) EditRepo(userName string, repoCRName string, editRepoOption gitea.EditRepoOption) (*gitea.Repository, *gitea.Response, error) {
	return r.giteaClient.EditRepo(userName, repoCRName, editRepoOption)
}

func (r *gc) DeleteAccessToken(value interface{}) (*gitea.Response, error) {
	return r.giteaClient.DeleteAccessToken(value)
}

func (r *gc) ListAccessTokens(opts gitea.ListAccessTokensOptions) ([]*gitea.AccessToken, *gitea.Response, error) {
	return r.giteaClient.ListAccessTokens(opts)
}

func (r *gc) CreateAccessToken(opt gitea.CreateAccessTokenOption) (*gitea.AccessToken, *gitea.Response, error) {
	return r.giteaClient.CreateAccessToken(opt)
}
