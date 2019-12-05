package network

import (
	"context"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	opv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
)

const defaultImagePullPolicy = v1.PullIfNotPresent

func validateImagePullPolicy(conf *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if conf.ImagePullPolicy == "" {
		return []error{}
	}

	if valid := verifyPullPolicyType(conf.ImagePullPolicy); !valid {
		return []error{errors.Errorf("requested imagePullPolicy '%s' is not valid", conf.ImagePullPolicy)}
	}

	return []error{}
}

func fillDefaultsImagePullPolicy(conf, previous *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if conf.ImagePullPolicy == "" {
		if previous != nil && previous.ImagePullPolicy != "" {
			conf.ImagePullPolicy = previous.ImagePullPolicy
		} else {
			conf.ImagePullPolicy = defaultImagePullPolicy
		}
	}

	return []error{}
}

func changeSafeImagePullPolicy(prev, next *opv1alpha1.NetworkAddonsConfigSpec) []error {
	if prev.ImagePullPolicy != "" && prev.ImagePullPolicy != next.ImagePullPolicy {
		return []error{errors.Errorf("cannot modify ImagePullPolicy configuration once components were deployed")}
	}
	return []error{}
}

// Currently not implemented, since there are no obsolete objects under this module
func CleanUpImagePullPolicy(ctx context.Context, client k8sclient.Client, objs []*unstructured.Unstructured) []error {
	return nil
}

// Verify if the value is a valid PullPolicy
func verifyPullPolicyType(imagePullPolicy v1.PullPolicy) bool {
	switch imagePullPolicy {
	case v1.PullAlways:
		return true
	case v1.PullNever:
		return true
	case v1.PullIfNotPresent:
		return true
	default:
		return false
	}
}
