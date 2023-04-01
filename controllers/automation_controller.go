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
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/vitu1234/ocf-operator/api/v1alpha1"
	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
)

// AutomationReconciler reconciles a Automation object
type AutomationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=iot.iot.dev,resources=automations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.iot.dev,resources=automations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iot.iot.dev,resources=automations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Automation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *AutomationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	fmt.Println("AUTOMATION RECONCILER")

	// TODO(user): your logic here
	// Check if a MyResource object already exists because there has to be one to one resource of this type only
	myAutomation := &v1alpha1.Automation{}

	if err := r.Client.Get(ctx, req.NamespacedName, myAutomation); err != nil {
		if apierrors.IsNotFound(err) {
			// The resource was deleted
			log.Log.Info("Automation deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}

		// An error occurred while trying to retrieve the resource
		log.Log.Error(err, "Automation retrieve retrieving failed", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	trigger := myAutomation.Spec.Trigger
	trigger_device_id := trigger.OnDevice
	trigger_resource := trigger.ForResource

	//get device resource data
	res, err := ocf_client.GetResource(trigger_device_id, trigger_resource)
	if err != nil {
		log.Log.Info("Automation Failed to get device resouce : ", err.Error())
	}

	//get the resource details 1 by 1 and store in the properties struct
	var raw_device_resource_property RawOCFDeviceResourceProperties

	err = json.Unmarshal([]byte(res), &raw_device_resource_property)
	if err != nil {
		log.Log.Info("Automation Failed to convert resource property: ", err.Error())
	}

	myDeviceResource := &iotv1alpha1.OCFDeviceResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trigger_device_id + "-ocf-device-resource",
			Namespace: req.Namespace,
		},
	}

	if err := r.Client.Get(ctx, req.NamespacedName, myDeviceResource); err != nil {
		if apierrors.IsNotFound(err) {
			// The resource was deleted
			log.Log.Info("Automation Resource not found/deleted in namespace", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{Requeue: true}, nil
		}

		// An error occurred while trying to retrieve the resource
		log.Log.Error(err, "Device Failed to retrieve resource", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	switch trigger.InputType {
	case "Numeric":
		log.Log.Info("Numeric type")

		//check if max/minimum is set
		if trigger.Max != nil {
			fmt.Println(*trigger.Max)
		}

		if trigger.Min != nil {
			fmt.Println(*trigger.Min)
		}

		if trigger.Period.Hours != nil {

		}

		if trigger.Period.Minutes != nil {

		}

		if trigger.Period.Seconds != nil {

		}
	case "State":
		log.Log.Info("State type")
	case "Time":
		log.Log.Info("Time type")
	default:
		log.Log.Info("Default case for trigger")
	}

	return ctrl.Result{Requeue: true}, nil
}

// //period reconciler
func (r *AutomationReconciler) TriggerLogic(ctx context.Context, trigger iotv1alpha1.AutomationSpec) {

}

// SetupWithManager sets up the controller with the Manager.
func (r *AutomationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.Automation{}).
		Complete(r)
}
