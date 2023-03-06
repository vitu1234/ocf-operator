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
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OCFDeviceReconciler reconciles a OCFDevice object
type OCFDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdevices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdevices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=iot.iot.dev,resources=ocfdevices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the OCFDevice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *OCFDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	fmt.Println("NORMAL RECONCILER")

	// TODO(user): your logic here //maybe get things which get changed
	// var ocfdevice v1alpha1.OCFDevice
	// r.Client.Get(ctx, req.NamespacedName, &ocfdevice)

	// var ocfdevicelist v1alpha1.OCFDeviceList
	// r.Client.List(ctx, &ocfdevicelist)
	// for i, v := range ocfdevicelist.Items {
	// 	fmt.Printf(v.GetName() + "")
	// 	fmt.Printf(string(i))
	// }

	// device := &v1alpha1.OCFDevice{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name: "device1",
	// 	},
	// 	Spec: v1alpha1.OCFDeviceSpec{
	// 		Id:      "sdhdhgd",
	// 		Name:    "name-here",
	// 		Owned:   false,
	// 		OwnerID: "djkhdkjhd",
	// 	},
	// }

	// create_device := r.Client.Create(ctx, device)
	// fmt.Println(create_device.Error())

	return ctrl.Result{}, nil
}

func (r *OCFDeviceReconciler) PeriodicReconcile() {
	// Handle periodic reconciliation logic here
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("Periodic reconcile at %v", time.Now().Format("15:04:05"))
			// Perform periodic reconciliation logic here

			// Define the OCFDevice object
			ocfDevice := &iotv1alpha1.OCFDevice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ocfdevice-sample",
					Namespace: "default",
				},
				Spec: iotv1alpha1.OCFDeviceSpec{
					Id:   "idsample",
					Name: "ocdfname",
				},
			}

			// Check if the OCFDevice object already exists
			found := &iotv1alpha1.OCFDevice{}
			err := r.Get(context.Background(), types.NamespacedName{Name: "ocfdevice-sample", Namespace: "default"}, found)
			if err != nil && errors.IsNotFound(err) {
				// Create the OCFDevice object if it does not exist
				if err = r.Create(context.Background(), ocfDevice); err != nil {
					fmt.Printf("failed to create OCFDevice: %s\n", err.Error())
					return
				}
			} else if err != nil {
				// Handle any other errors that may occur
				fmt.Printf("failed to get OCFDevice: %s\n", err.Error())
				return
			}

		}
	}

	// return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCFDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&iotv1alpha1.OCFDevice{}).
		// Watches(
		// 	&source.Kind{Type: &v1alpha1.OCFDevice{}},
		// 	handler.EnqueueRequestsFromMapFunc(r.GetAll),
		// ).
		// Register the Reconcile() method for Create and Update events
		Watches(
			&source.Kind{Type: &iotv1alpha1.OCFDevice{}}, &handler.EnqueueRequestForObject{}).
		// Register the PeriodicReconcile() method for periodic reconciliation
		Owns(&iotv1alpha1.OCFDevice{}).
		Complete(r); err != nil {
		return err
	}

	go r.PeriodicReconcile()

	return nil
}

// func (r *OCFDeviceReconciler) GetAll(o client.Client)[] ctrl.Request{

// }
