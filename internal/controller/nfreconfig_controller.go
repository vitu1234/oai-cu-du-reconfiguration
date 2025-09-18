/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"

	"code.gitea.io/sdk/gitea"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nephio-project/nephio/controllers/pkg/resource"
	cudureconfigv1 "github.com/vitu1234/oai-cu-du-reconfiguration/v1/api/v1"
	giteaclient "github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/gitaclient"
	giteaclient2 "github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/gitaclient"
	"github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/helpers"
)

// NFReconfigReconciler reconciles a NFReconfig object
type NFReconfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cu-du-reconfig.cu-du-reconfig.dcnlab.ssu.ac.kr,resources=nfreconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cu-du-reconfig.cu-du-reconfig.dcnlab.ssu.ac.kr,resources=nfreconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cu-du-reconfig.cu-du-reconfig.dcnlab.ssu.ac.kr,resources=nfreconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NFReconfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *NFReconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling NFReconfig")

	// Fetch the Checkpoint instance
	var nfReconfig cudureconfigv1.NFReconfig
	if err := r.Get(ctx, req.NamespacedName, &nfReconfig); err != nil {
		if errors.IsNotFound(err) {
			log.Info("nfReconfig resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get nfReconfig")
		return ctrl.Result{}, err
	}

	// initialize gitea client
	giteaclient, _, err := r.InitializeGiteaClient(ctx)
	if err != nil {
		log.Error(err, "gitea initialization failed")
		return ctrl.Result{}, nil
	}

	// find the target cluster
	var clusterInfoTarget cudureconfigv1.ClusterInfo
	for _, clusterInfo := range nfReconfig.Spec.ClusterInfo {

		//check if has NFDeployment key
		if clusterInfo.NFDeployment == (cudureconfigv1.NFDeployment{}) {
			continue
		}

		clusterInfoTarget = clusterInfo
	}

	err = r.HandleTargetClusterPkgNF(ctx, clusterInfoTarget, giteaclient, &nfReconfig)
	if err != nil {
		log.Error(err, err.Error())
	}

	// //get all repositories
	// repos, resp, err := giteaclient.Get().ListMyRepos(gitea.ListReposOptions{})
	// if err != nil {
	// 	log.Error(err, "Failed to list Gitea repositories", "response", resp)
	// }
	// if len(repos) == 0 {
	// 	log.Info("No repositories found for user", "username", user.UserName)
	// }

	// for _, repo := range repos {

	// 	if repo.Name != "regional" {
	// 		// log.Info("repo is not regional, skipping: " + repo.Name)
	// 		continue
	// 	}
	// 	gitURL := os.Getenv("GIT_SERVER_URL")

	// 	log.Info("repo is  regional, url: " + repo.CloneURL + " <------> ")
	// 	_, matches, err := helpers.CheckRepoForMatchingManifests(ctx, gitURL+"/nephio/"+repo.Name+".git", repo.DefaultBranch, &nfReconfig)
	// 	if err != nil {
	// 		log.Error(err, "error scanning repo", "repo", repo.Name)
	// 		continue
	// 	}
	// 	if len(matches) > 0 {
	// 		log.Info("Found matching manifests",
	// 			"repo", repo.Name,
	// 			"files", matches)
	// 	}
	// }

	return ctrl.Result{}, nil
}

func (r *NFReconfigReconciler) HandleTargetClusterPkgNF(ctx context.Context, clusterInfo cudureconfigv1.ClusterInfo, giteaclient giteaclient.GiteaClient, nfReconfig *cudureconfigv1.NFReconfig) error {
	log := logf.FromContext(ctx)

	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		return err
	}

	//get all repositories
	repos, resp, err := giteaclient.Get().ListMyRepos(gitea.ListReposOptions{})
	if err != nil {
		log.Error(err, "Failed to list Gitea repositories", "response", resp)
	}
	if len(repos) == 0 {
		log.Info("No repositories found for user", "username", user.UserName)
	}

	tmpDir, matches, err := helpers.CheckRepoForMatchingManifests(ctx, clusterInfo.Repo, "main", &clusterInfo, "NFDeployment")
	if err != nil {
		log.Error(err, "error scanning repo", "repo", clusterInfo.Name)
		return err
	}
	if len(matches) > 0 {
		log.Info("Found matching manifests",
			"repo", clusterInfo.Name,
			"tmpDir", tmpDir,
			"files", matches)

		// build new IP map from nfReconfig.Spec.Interfaces
		newIPs := map[string]helpers.IPInfo{}

		for _, iface := range nfReconfig.Spec.Interfaces {
			if iface.IPv4.Address != "" || iface.IPv4.Gateway != "" {
				newIPs[iface.Name] = helpers.IPInfo{
					Address: iface.IPv4.Address,
					Gateway: iface.IPv4.Gateway,
				}
			}
		}

		for _, f := range matches {
			// fullPath := filepath.Join(tmpDir, f)
			if err := helpers.UpdateInterfaceIPsNFDeployment(f, newIPs); err != nil {
				log.Error(err, "failed to update IPs in manifest", "file", f)
				return err
			}
			log.Info("Updated IPs in manifest", "file", f)
		}

		// commit & push changes back to Gitea
		log.Info("will commit and push changes back to git here")
		commitMsg := fmt.Sprintf("Update interface IPs for %s/%s", clusterInfo.NFDeployment.Namespace, clusterInfo.NFDeployment.Name)

		username, password, _, err := giteaclient2.GetGiteaSecretUserNamePassword(ctx, r.Client)
		if err != nil {
			log.Error(err, "failed to get gitea")
			return err
		}

		if err := helpers.CommitAndPush(ctx, tmpDir, "main", clusterInfo.Repo, username, password, commitMsg); err != nil {
			log.Error(err, "failed to commit & push changes")
			return err
		}

		log.Info("Changes committed and pushed", "repo", clusterInfo.Repo)

		//handle dependent cluster pkg nfs
		err = r.HandleDependentClusterPkgNF(ctx, giteaclient, nfReconfig)
		if err != nil {
			log.Error(err, "failed to commit and push dependent nfs configs")
			return err
		}
	}

	return err

}

func (r *NFReconfigReconciler) HandleDependentClusterPkgNF(ctx context.Context, giteaclient giteaclient.GiteaClient, nfReconfig *cudureconfigv1.NFReconfig) error {
	log := logf.FromContext(ctx)

	user, _, err := giteaclient.GetMyUserInfo()
	if err != nil {
		return err
	}

	//get all repositories
	repos, resp, err := giteaclient.Get().ListMyRepos(gitea.ListReposOptions{})
	if err != nil {
		log.Error(err, "Failed to list Gitea repositories", "response", resp)
	}
	if len(repos) == 0 {
		log.Info("No repositories found for user", "username", user.UserName)
	}

	for _, clusterInfo := range nfReconfig.Spec.ClusterInfo {

		//check if has NFDeployment key
		if clusterInfo.ConfigRef == (cudureconfigv1.ConfigRef{}) {
			continue
		}

		tmpDir, matches, err := helpers.CheckRepoForMatchingManifests(ctx, clusterInfo.Repo, "main", &clusterInfo, "ConfigRef")
		if err != nil {
			log.Error(err, "error scanning repo", "repo", clusterInfo.Name)
			return err
		}
		if len(matches) > 0 {
			log.Info("Found matching manifests",
				"repo", clusterInfo.Name,
				"tmpDir", tmpDir,
				"files", matches)

			// build new IP map from nfReconfig.Spec.Interfaces
			newIPs := map[string]helpers.IPInfo{}

			for _, iface := range nfReconfig.Spec.Interfaces {
				if iface.IPv4.Address != "" || iface.IPv4.Gateway != "" {
					newIPs[iface.Name] = helpers.IPInfo{
						Address: iface.IPv4.Address,
						Gateway: iface.IPv4.Gateway,
					}
				}
			}

			for _, f := range matches {
				// fullPath := filepath.Join(tmpDir, f)
				if err := helpers.UpdateInterfaceIPsConfigRefs(f, newIPs); err != nil {
					log.Error(err, "failed to update IPs in manifest", "file", f)
					return err
				}
				log.Info("Updated IPs in manifest", "file", f)
			}

			// commit & push changes back to Gitea
			log.Info("will commit and push changes back to git here")
			commitMsg := fmt.Sprintf("Update interface IPs for %s/%s", nfReconfig.Spec.ClusterInfo[0].NFDeployment.Namespace, nfReconfig.Spec.ClusterInfo[0].NFDeployment.Name)

			username, password, _, err := giteaclient2.GetGiteaSecretUserNamePassword(ctx, r.Client)
			if err != nil {
				log.Error(err, "failed to get gitea")
				return err
			}

			if err := helpers.CommitAndPush(ctx, tmpDir, "main", clusterInfo.Repo, username, password, commitMsg); err != nil {
				log.Error(err, "failed to commit & push changes")
				return err
			}

			log.Info("Changes committed and pushed", "repo", clusterInfo.Repo)
		}
	}

	return err
}

func (r *NFReconfigReconciler) InitializeGiteaClient(ctx context.Context) (giteaclient.GiteaClient, *gitea.User, error) {
	log := logf.FromContext(ctx)

	apiClient := resource.NewAPIPatchingApplicator(r.Client)
	giteaClient, err := giteaclient.GetClient(ctx, apiClient)
	if err != nil {
		log.Error(err, "Failed to initialize Gitea client")
		return nil, nil, err
	}
	if !giteaClient.IsInitialized() {
		log.Info("Gitea client not yet initialized, retrying later")
		return nil, nil, err
	}

	user, resp, err := giteaClient.GetMyUserInfo()
	if err != nil {
		log.Error(err, "Failed to get Gitea user info", "response", resp)
		return nil, nil, err
	}
	log.Info("Authenticated with Gitea", "username", user.UserName)

	return giteaClient, user, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *NFReconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cudureconfigv1.NFReconfig{}).
		Named("nfreconfig").
		Complete(r)
}
