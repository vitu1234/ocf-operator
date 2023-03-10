/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/vitu1234/ocf-operator/api/v1alpha1"
	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// OCFDeviceBoardingReconciler reconciles a OCFDeviceBoarding object
type OCFDeviceBoardingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdeviceboardings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdeviceboardings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdeviceboardings/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OCFDeviceBoarding object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *OCFDeviceBoardingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	fmt.Println("Onboarding Controller")

	// TODO(user): your logic here
	// Check if a MyResource object already exists because there has to be one to one resource of this type only
	myDeviceBoardingResource := &v1alpha1.OCFDeviceBoarding{}

	if err := r.Client.Get(ctx, req.NamespacedName, myDeviceBoardingResource); err != nil {
		if apierrors.IsNotFound(err) {
			// The resource was deleted
			fmt.Println("Resource deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}

		// An error occurred while trying to retrieve the resource
		fmt.Println(err, "Failed to retrieve resource", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if myDeviceBoardingResource.ObjectMeta.Generation == 1 {
		// The resource was just created
		fmt.Println("Resource created", "name", req.Name, "namespace", req.Namespace)
	} else {
		// The resource was updated
		fmt.Println("Resource updated", "name", req.Name, "namespace", req.Namespace)
	}

	// Perform reconciliation logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCFDeviceBoardingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.OCFDeviceBoarding{}).
		Complete(r)
}
