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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/plgd-dev/kit/v2/codec/json"
	"github.com/vitu1234/ocf-operator/api/v1alpha1"
	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
)

// OCFDeviceResourceReconciler reconciles a OCFDeviceResource object
type OCFDeviceResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=iot.iot.dev,resources=OCFDeviceResource,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.iot.dev,resources=OCFDeviceResource/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iot.iot.dev,resources=OCFDeviceResource/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OCFDeviceResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *OCFDeviceResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log.Log.Info("Device Resources Controller")

	// TODO(user): your logic here
	// Check if a MyResource object already exists because there has to be one to one resource of this type only
	myDeviceResource := &v1alpha1.OCFDeviceResource{}

	if err := r.Client.Get(ctx, req.NamespacedName, myDeviceResource); err != nil {
		if apierrors.IsNotFound(err) {
			// The resource was deleted
			log.Log.Info("Device Resource deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}

		// An error occurred while trying to retrieve the resource
		log.Log.Error(err, "Device Failed to retrieve resource", "name", req.Name, "namespace", req.Namespace)
		return ctrl.Result{}, err
	}

	if myDeviceResource.ObjectMeta.Generation == 1 {
		// The resource was just created
		log.Log.Info("Device Resource created", "name", req.Name, "namespace", req.Namespace)
	} else {
		// The resource was updated
		log.Log.Info("Device Resource updated", "name", req.Name, "namespace", req.Namespace)
		//get the new object
		for i := 0; i < len(myDeviceResource.Spec.Properties); i++ {
			row_link := myDeviceResource.Spec.Properties[i].Link

			// row_temperature := myDeviceResource.Spec.Properties[i].Temperature
			// row_units := myDeviceResource.Spec.Properties[i].Units
			// row_value := myDeviceResource.Spec.Properties[i].Value
			// log.Log.Info("Link: ", row_link, " Temp.: ", row_link, " Units: ", "row_units", " Value: ", "row_value")
			// fmt.Printf("LInk: %s/n", myDeviceResource.Spec.Properties[i].Units)
			if myDeviceResource.Spec.Properties[i].Value != nil {
				fmt.Printf("Link: %s | DEVICE: %s\n", row_link, myDeviceResource.Spec.DeviceID)
				value := *myDeviceResource.Spec.Properties[i].Value
				if value {
					log.Log.Info("VALUE IS TRUE, DO SOMETHING")
					// key := "value"
					// value := "true"
					jsonString := "{\"value\": true}"
					// jsonString := "{\"" + key + "\": " + value + "}"
					var data interface{}
					err := json.Decode([]byte(jsonString), &data)
					if err != nil {
						println("\nDecoding resource property has failed : " + err.Error())
						break
					}
					dataBytes, err := json.Encode(data)
					if err != nil {
						println("\nEncoding resource property has failed : " + err.Error())
						break
					}
					println("\nProperty data to update : " + string(dataBytes) + " LINK: " + row_link)
					// res, err := ocf_client.GetResource(myDeviceResource.Spec.DeviceID, row_link)
					// if err != nil {
					// 	println("RES FAILED: ", err.Error())
					// }
					// println("RES: ", res)

					err = ocf_client.UpdateResource(myDeviceResource.Spec.DeviceID, row_link, dataBytes)
					if err != nil {
						fmt.Printf("FAILED PUBLISH TO DEVICE: %s\n", err.Error())
					} else {
						log.Log.Info("PUBLISHED TRUE")
					}
				}
			}

		}
	}

	// Perform reconciliation logic here
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCFDeviceResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.OCFDeviceResource{}).
		Complete(r)
}
