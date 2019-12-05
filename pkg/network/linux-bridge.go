package network

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/render"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	"github.com/kubevirt/cluster-network-addons-operator/pkg/network/cni"
)

func changeSafeLinuxBridge(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.LinuxBridge != nil && !reflect.DeepEqual(prev.LinuxBridge, next.LinuxBridge) {
		return []error{errors.Errorf("cannot modify Linux Bridge configuration once it is deployed")}
	}
	return nil
}

// In older versions of the operator, we used daemon sets of type 'extensions/v1beta1', later we
// changed that to 'apps/v1'. Because of this change, we are not able to seamlessly upgrade using
// only Update methods. Following this we find and delete the old daemonSet if configured
func CleanUpLinuxBridge(ctx context.Context, client k8sclient.Client, objs []*unstructured.Unstructured) []error {

	// Get existing
	existing := &unstructured.Unstructured{}
	gvk := schema.GroupVersionKind{Group: "extensions", Version: "v1beta1", Kind: "DaemonSet"}
	existing.SetGroupVersionKind(gvk)

	err := client.Get(ctx, types.NamespacedName{Name: "bridge-marker", Namespace: "linux-bridge"}, existing)
	//log.Printf("ram sam sam 3 after get: \nexisting = %v\nerr = %v! ! !", existing, err)
	if err == nil || !(apierrors.IsNotFound(err) || strings.Contains(err.Error(), "no matches for kind")) {
		log.Printf("ram sam sam 5.0 Found Existing, Deleteing the object")
		//Delete the object
		err = client.Delete(ctx, existing)
		if err != nil {
			log.Printf("ram sam sam 5.1 failed Cleaning up Linux-Bridge Object.")
		} else {
			log.Printf("ram sam sam 5.2 Delete Success, err = %v! ! !", err)
		}
		return nil
	}
	return nil
}

// renderLinuxBridge generates the manifests of Linux Bridge
func renderLinuxBridge(conf *opv1alpha1.NetworkAddonsConfigSpec, manifestDir string, clusterInfo *ClusterInfo) ([]*unstructured.Unstructured, error) {
	if conf.LinuxBridge == nil {
		return nil, nil
	}

	// render the manifests on disk
	data := render.MakeRenderData()
	data.Data["Namespace"] = os.Getenv("OPERAND_NAMESPACE")
	data.Data["LinuxBridgeMarkerImage"] = os.Getenv("LINUX_BRIDGE_MARKER_IMAGE")
	data.Data["LinuxBridgeImage"] = os.Getenv("LINUX_BRIDGE_IMAGE")
	data.Data["ImagePullPolicy"] = conf.ImagePullPolicy
	if clusterInfo.OpenShift4 {
		data.Data["CNIBinDir"] = cni.BinDirOpenShift4
	} else {
		data.Data["CNIBinDir"] = cni.BinDir
	}
	data.Data["EnableSCC"] = clusterInfo.SCCAvailable

	objs, err := render.RenderDir(filepath.Join(manifestDir, "linux-bridge"), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render linux-bridge manifests")
	}

	return objs, nil
}
