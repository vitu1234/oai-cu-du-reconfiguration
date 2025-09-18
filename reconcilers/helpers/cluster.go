package helpers

import (
	"context"

	capictrl "github.com/vitu1234/oai-cu-du-reconfiguration/v1/reconcilers/capi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetWorkloadClusterClient(ctx context.Context, mgmtClient client.Client, clusterName string) (workloadClusterClient client.Client, erro error) {
	log := logf.FromContext(ctx)
	clusterList := &capiv1beta1.ClusterList{}
	if err := mgmtClient.List(ctx, clusterList); err != nil {
		log.Error(err, "error listing clusters")
		return nil, err
	}

	var workloadCluster capiv1beta1.Cluster

	for _, workload_cluster := range clusterList.Items {
		if workload_cluster.Name == clusterName {
			workloadCluster = workload_cluster
		}
	}

	capiCluster, err := capictrl.GetCapiClusterFromName(ctx, workloadCluster.Name, workloadCluster.Namespace, mgmtClient)
	if err != nil {
		log.Error(err, "Failed to get CAPI cluster")
		return
	}

	//get workload cluster client
	clusterClient, _, ready, err := capiCluster.GetClusterClient(ctx)
	if err != nil {
		log.Error(err, "Failed to get workload cluster client", "cluster", capiCluster.GetClusterName())
		return nil, err
	}
	if !ready {
		log.Info("Cluster is not ready", "cluster", capiCluster.GetClusterName())
		return nil, err
	}

	return clusterClient, nil

}

// Delete resource on workload cluster
// DeleteNFDeployment deletes the given NFDeployment resource and all its Pods
// DeleteNFDeployment deletes the NFDeployment CR and matching pods
func DeleteNFDeployment(ctx context.Context, k8sClient client.Client,
	name, namespace string) error {

	// 1. Delete the NFDeployment custom resource
	nf := &unstructured.Unstructured{}
	nf.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "workload.nephio.org",
		Version: "v1alpha1",
		Kind:    "NFDeployment",
	})
	nf.SetName(name)
	nf.SetNamespace(namespace)

	if err := k8sClient.Delete(ctx, nf); err != nil {
		return err
	}

	// 2.delete all pods in the namespace

	if err := k8sClient.DeleteAllOf(
		ctx,
		&corev1.Pod{},
		client.InNamespace(namespace),
	); err != nil {
		return err
	}

	return nil
}
