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
	logging "log"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jessevdk/go-flags"
	iotv1alpha1 "github.com/vitu1234/ocf-operator/api/v1alpha1"
	OCFClient "github.com/vitu1234/ocf-operator/controllers/ocfclient"
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

	fmt.Println("OCFDevice controller")

	// TODO(user): your logic here //maybe get things which get changed

	return ctrl.Result{}, nil
}

func (r *OCFDeviceReconciler) PeriodicReconcile() {
	// Handle periodic reconciliation logic here
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	//get ocfdeviceboarding resource and get the set values

	//OCF DEVICE DISCOVERY HERE
	var opts iotv1alpha1.Options
	parser := flags.NewParser(&opts, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		logging.Println("Parsing command options has failed : " + err.Error())
	}

	OCFClient.ReadCommandOptions(opts)

	// Create OCF Client
	ocf_client := OCFClient.OCFClient{}
	err = ocf_client.Initialize()
	if err != nil {
		fmt.Println("OCF Client has failed to initialize : " + err.Error())
	}

	for {
		select {
		case <-ticker.C:
			fmt.Printf("Periodic reconcile at %v \n", time.Now().Format("15:04:05"))
			// Perform periodic reconciliation logic here

			// get ocfdeviceboarding resource and get the set values
			onboardingInstances := &iotv1alpha1.OCFDeviceBoardingList{}
			listOpts := []client.ListOption{
				client.InNamespace("default"), // Replace with the namespace you want to list resources from
			}

			// err := r.Client.List(context.Background(), onboardingInstances, listOpts...)
			err := r.Client.List(context.Background(), onboardingInstances, listOpts...)
			if err != nil {
				logging.Printf("Error getting OCFDeviceBoardingList: %s \n", err)
			}
			var device_instance iotv1alpha1.OCFDeviceBoarding
			for _, instance := range onboardingInstances.Items {
				device_instance = instance
			}

			//discover devices only if the Onboarding instance is set
			if len(onboardingInstances.Items) > 0 {
				discoveryTimeout := opts.DiscoveryTimeout
				if discoveryTimeout <= 0 {
					discoveryTimeout = time.Second * time.Duration(device_instance.Spec.DiscoveryTimeout)
				}

				res, err := ocf_client.Discover(discoveryTimeout)
				if err != nil {
					fmt.Printf("Discovering devices has failed : %s\n", err.Error())
				}

				// Define a slice of Device struct to hold the JSON array
				var raw_devices []RawOCFDevice

				// Unmarshal the JSON array to the slice
				err = json.Unmarshal([]byte(res), &raw_devices)
				if err != nil {
					panic(err)
				}

				//check if Onboarding is Manual or Auto
				if device_instance.Spec.OnBoardMode == "Auto" {

					if len(raw_devices) > 0 {

						// Define the OCFDevice object
						ocfDevice := &iotv1alpha1.OCFDevice{
							ObjectMeta: metav1.ObjectMeta{
								Name:      raw_devices[0].ID,
								Namespace: "default",
							},
							Spec: iotv1alpha1.OCFDeviceSpec{
								Id:      raw_devices[0].ID,
								Name:    raw_devices[0].Name,
								Owned:   raw_devices[0].Owned,
								OwnerID: raw_devices[0].OwnerID,
							},
						}

						// Check if the OCFDevice object already exists
						found := &iotv1alpha1.OCFDevice{}
						err = r.Get(context.Background(), types.NamespacedName{Name: raw_devices[0].Name, Namespace: "default"}, found)
						if err != nil && errors.IsNotFound(err) {
							// Create the OCFDevice object if it does not exist
							err = r.Create(context.Background(), ocfDevice)
							if err != nil {
								logging.Printf("failed to create OCFDevice: %s\n, skipping", err.Error())
								// return
							} else {
								//device created
							}
						} else if err != nil {
							// Handle any other errors that may occur
							logging.Printf("failed to get OCFDevice: %s\n, skipping", err.Error())
							// return
						}

						logging.Println("Device Registered ")
						// return
					} else {
						logging.Println("No OCFDevices found on the network")
					}
				} else {
					logging.Println("OCFDeviceBoarding set to Manual")
				}
			} else {
				logging.Println("No need to trigger device discovery, OCFDeviceBoarding not set")
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

// structure of a discovered OCF device
type RawOCFDevice struct {
	Details struct {
		DI   string   `json:"di"`
		RT   []string `json:"rt"`
		IF   []string `json:"if"`
		Name string   `json:"n"`
		DMN  *string  `json:"dmn"`
		DMNO string   `json:"dmno"`
		PIID string   `json:"piid"`
	} `json:"details"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Owned   bool   `json:"owned"`
	OwnerID string `json:"ownerID"`
}
