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
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
func (r *OCFDeviceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logging.Println("OCFDevice Controller")
	// ocf_client.DisownDevice("7e2b109e-7452-4e9b-4102-b25b754a4d0c")
	// TODO(user): your logic here //maybe get things which get changed
	// Fetch the OCFDevice resource that was modified
	instance := &iotv1alpha1.OCFDevice{}
	_ = r.Get(context.TODO(), request.NamespacedName, instance)

	// Define the desired OCFDevice object
	ocfDevice := &iotv1alpha1.OCFDevice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: iotv1alpha1.OCFDeviceSpec{
			Id:                instance.Spec.Id,
			Name:              instance.Spec.Name,
			OwnerID:           instance.Spec.OwnerID,
			PreferedResources: instance.Spec.PreferedResources,
		}, Status: iotv1alpha1.OCFDeviceStatus{
			Status:  iotv1alpha1.OCFDeviceCreating,
			Message: "Device created, onboarding",
		},
	}

	//register finalizers
	// name of our custom finalizer
	myFinalizerName := "iot.iot.v1alpha1device.dev/finalizer"
	// examine DeletionTimestamp to determine if object is under deletion
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(instance, myFinalizerName) {
			controllerutil.AddFinalizer(instance, myFinalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(instance, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if !r.deleteExternalResources(instance) {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				logging.Println("FAILED TO DELETE EXTRA RESOURCES")
				return ctrl.Result{}, nil
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(instance, myFinalizerName)
			if err := r.Update(ctx, instance); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if err := controllerutil.SetControllerReference(instance, ocfDevice, r.Scheme); err != nil {
		logging.Printf("Error:  %s\n", err.Error())
		return ctrl.Result{}, err
	}

	var old_ocfdevice iotv1alpha1.OCFDevice
	err := r.Get(context.Background(), types.NamespacedName{Name: ocfDevice.Name, Namespace: ocfDevice.Namespace}, &old_ocfdevice)
	if err != nil {
		return ctrl.Result{}, err
	}

	// TODO(user): Change this for the object type created by your controller
	// Check if the Deployment already exists
	found := &iotv1alpha1.OCFDevice{}
	err = r.Get(context.Background(), types.NamespacedName{Name: ocfDevice.Name, Namespace: ocfDevice.Namespace}, found)
	if err == nil && !found.Status.Owned {

		logging.Printf("Creating OCFDevice %s/%s\n", ocfDevice.Namespace, ocfDevice.Name)

		logging.Println("Discover OCFDevice on the network")

		found.Status.Status = iotv1alpha1.OCFDeviceDiscovery
		found.Status.Message = "Discover OCFDevice on the network"
		r.Client.Status().Update(context.TODO(), found)

		discoveryTimeout := opts.DiscoveryTimeout
		if discoveryTimeout <= 0 {
			discoveryTimeout = time.Second * time.Duration(10)
		}

		res, err := ocf_client.Discover(discoveryTimeout)
		if err != nil {
			logging.Printf("Discovering devices has failed : %s\n", err.Error())
			found.Status.Status = iotv1alpha1.OCFDeviceNotFound
			found.Status.Message = "Discovering devices has failed, Id not found"
			err = r.Client.Status().Update(context.TODO(), found)
			if err != nil {
				//An error occured updating the resource so requeue
				return ctrl.Result{}, err
			}

		}

		// Define a slice of Device struct to hold the JSON array
		var raw_devices []RawOCFDevice

		// Unmarshal the JSON array to the slice
		err = json.Unmarshal([]byte(res), &raw_devices)
		if err != nil {
			found.Status.Status = iotv1alpha1.OCFDeviceError
			r.Client.Status().Update(context.TODO(), found)
			logging.Printf("%s\n", err.Error())
		}

		//create the device
		//check number of OCFDevices on the network
		if len(raw_devices) > 0 {
			for _, raw_device := range raw_devices {

				if raw_device.ID == instance.Spec.Id && instance.Spec.Id != "" {
					//device created | own/onboard it and register its resources in K8s
					ocf_client.OwnDevice(raw_device.ID)
					found.Spec.Id = raw_device.ID
					found.Spec.Name = raw_device.Details.Name
					found.Status.Owned = raw_device.Owned
					found.Spec.OwnerID = raw_device.OwnerID
					r.Client.Update(context.TODO(), found)
					links_resources, err := ocf_client.GetResources(raw_device.ID)
					if err != nil {
						found.Status.Status = iotv1alpha1.OCFDeviceError
						logging.Printf("Failed to get device resources, DeviceID: %s | Error: %s \n", raw_device.ID, err.Error())
						r.Client.Status().Update(context.TODO(), found)

					}
					//get the resources 1 by 1 and register
					var raw_device_resources []RawOCFDeviceResource
					err = json.Unmarshal([]byte(links_resources), &raw_device_resources)
					if err != nil {
						found.Status.Status = iotv1alpha1.OCFDeviceError
						logging.Printf("Processing device resources to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
						r.Client.Status().Update(context.TODO(), found)

					}
					// Define the Properties array
					properties := []iotv1alpha1.OCFDeviceResourceProperties{}

					//get the device resources and store for this device
					for _, d := range raw_device_resources {
						resource_properties, err := ocf_client.GetResource(raw_device.ID, d.Href)
						if err != nil {
							found.Status.Status = iotv1alpha1.OCFDeviceError
							found.Status.Message = "Failed to get device resources details: " + raw_device.ID
							logging.Printf("Failed to get device resources details, DeviceID: %s | Error: %s \n", raw_device.ID, err.Error())
							r.Client.Status().Update(context.TODO(), found)
						}

						//get the resource details 1 by 1 and store in the properties struct
						var raw_device_resource_properties RawOCFDeviceResourceProperties

						err = json.Unmarshal([]byte(resource_properties), &raw_device_resource_properties)
						if err != nil {
							logging.Printf("Processing device resource properties to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
						}

						//append all the properties
						newProperty := iotv1alpha1.OCFDeviceResourceProperties{}

						newProperty.Name = d.Href

						if raw_device_resource_properties.Value != nil {

							if strconv.FormatBool(*raw_device_resource_properties.Value) == "true" {
								newProperty.Value = raw_device_resource_properties.Value
							} else if strconv.FormatBool(*raw_device_resource_properties.Value) == "false" {
								newProperty.Value = raw_device_resource_properties.Value
							} else {
								newProperty.Value = raw_device_resource_properties.Value
							}

						}

						if raw_device_resource_properties.Units != "" {
							newProperty.Units = raw_device_resource_properties.Units
						}

						if raw_device_resource_properties.Units != "" {
							newProperty.Temperature = strconv.FormatFloat(raw_device_resource_properties.Temperature, 'f', 2, 64)
						}
						properties = append(properties, newProperty)

					}

					//create the device resource
					// Define the OCFDevice object
					ocfDeviceResource := &iotv1alpha1.OCFDeviceResource{
						ObjectMeta: metav1.ObjectMeta{
							Name:      raw_device.ID + "-ocf-device-resource",
							Namespace: found.Namespace,
						},
						Spec: iotv1alpha1.OCFDeviceResourceSpec{
							DeviceID:   raw_device.ID,
							Properties: properties,
						},
					}
					err = r.Create(context.Background(), ocfDeviceResource)
					if err != nil {
						logging.Printf("failed to create OCFDeviceResource: %s\n, skipping", err.Error())
						// return
					}

					found.Status.Status = iotv1alpha1.OCFDeviceRunning
					found.Status.Owned = true
					found.Status.Message = "Device registered"
					r.Client.Status().Update(context.TODO(), found)

				} else {

					//if device ID is not set on Spec.ID, search for the device using prefferedresourcestypes
					if len(instance.Spec.PreferedResources) > 0 {

						preferredResources := instance.Spec.PreferedResources
						//get device resources
						links_resources, err := ocf_client.GetResources(raw_device.ID)

						for i := 0; i < len(preferredResources); i++ {
							//search for a device with the resource until it find one matches
							if err != nil {
								found.Status.Status = iotv1alpha1.OCFDeviceError
								found.Status.Message = "Failed searching for device resources: " + err.Error()
								err = r.Client.Status().Update(context.TODO(), found)
								if err != nil {
									//An error occured updating the resource so requeue
									return ctrl.Result{}, err
								}
							}
							count := strings.Count(links_resources, preferredResources[i].Name)
							if count > 0 {
								//onboard this device
								ocf_client.OwnDevice(raw_device.ID)
								found.Spec.Id = raw_device.ID
								found.Spec.Name = raw_device.Details.Name
								found.Status.Owned = raw_device.Owned
								found.Spec.OwnerID = raw_device.OwnerID
								found.Status.Owned = true
								r.Client.Update(context.TODO(), found)

								//get the resources 1 by 1 and register
								var raw_device_resources []RawOCFDeviceResource
								err = json.Unmarshal([]byte(links_resources), &raw_device_resources)
								if err != nil {
									found.Status.Status = iotv1alpha1.OCFDeviceError
									logging.Printf("Processing device resources to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
									r.Client.Status().Update(context.TODO(), found)

								}
								// Define the Properties array
								properties := []iotv1alpha1.OCFDeviceResourceProperties{}

								//get the device resources and store for this device
								for _, d := range raw_device_resources {
									resource_properties, err := ocf_client.GetResource(raw_device.ID, d.Href)
									if err != nil {
										found.Status.Status = iotv1alpha1.OCFDeviceError
										found.Status.Message = "Failed to get device resources details: " + raw_device.ID
										logging.Printf("Failed to get device resources details, DeviceID: %s | Error: %s \n", raw_device.ID, err.Error())
										r.Client.Status().Update(context.TODO(), found)

									}

									//get the resource details 1 by 1 and store in the properties struct
									var raw_device_resource_properties RawOCFDeviceResourceProperties

									err = json.Unmarshal([]byte(resource_properties), &raw_device_resource_properties)
									if err != nil {
										logging.Printf("Processing device resource properties to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
									}

									//append all the properties
									newProperty := iotv1alpha1.OCFDeviceResourceProperties{}

									newProperty.Name = d.Href

									if raw_device_resource_properties.Value != nil {

										if strconv.FormatBool(*raw_device_resource_properties.Value) == "true" {
											newProperty.Value = raw_device_resource_properties.Value
										} else if strconv.FormatBool(*raw_device_resource_properties.Value) == "false" {
											newProperty.Value = raw_device_resource_properties.Value
										} else {
											newProperty.Value = raw_device_resource_properties.Value
										}

									}

									if raw_device_resource_properties.Units != "" {
										newProperty.Units = raw_device_resource_properties.Units
									}

									if raw_device_resource_properties.Units != "" {
										newProperty.Temperature = strconv.FormatFloat(raw_device_resource_properties.Temperature, 'f', 2, 64)
									}
									properties = append(properties, newProperty)

								}

								//create the device resource
								// Define the OCFDevice object
								ocfDeviceResource := &iotv1alpha1.OCFDeviceResource{
									ObjectMeta: metav1.ObjectMeta{
										Name:      raw_device.ID + "-ocf-device-resource",
										Namespace: found.Namespace,
									},
									Spec: iotv1alpha1.OCFDeviceResourceSpec{
										DeviceID:   raw_device.ID,
										Properties: properties,
									},
								}
								err = r.Create(context.Background(), ocfDeviceResource)
								if err != nil {
									logging.Printf("failed to create OCFDeviceResource: %s\n, skipping", err.Error())
									// return
								}

								found.Status.Status = iotv1alpha1.OCFDeviceRunning
								found.Status.Message = "Device registered"
								found.Status.Owned = true
								r.Client.Status().Update(context.TODO(), found)
							}
						}

					} else {

						found.Status.Status = iotv1alpha1.OCFDeviceError
						found.Status.Message = "No OCFDevices found on the network matching the provided Spec.Id or Spec.preferredResources"
						r.Client.Status().Update(context.TODO(), found)

					}
				}

			}

		} else {
			found.Status.Status = iotv1alpha1.OCFDeviceError
			found.Status.Message = "No OCFDevices un-onboarded found on the network"
			logging.Println("No OCFDevices un-onboarded found on the network")
			err = r.Client.Status().Update(context.TODO(), found)
			if err != nil {
				//An error occured updating the resource so requeue
				return ctrl.Result{}, err
			}
		}

	} else if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			logging.Printf("OCFDevice Object not found, return.  Created objects are automatically garbage collected. %s\n", err.Error())
			//delete related device resources
			// The object has been deleted, you can get the object's specification

			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logging.Printf("Error reading the object - requeue the request. %s\n", err.Error())
		return ctrl.Result{}, err
	}

	// perfom update action
	if found.ObjectMeta.Generation > 1 {
		if found.Status.Owned {
			logging.Println("Device is Owned is true ")
			if found.Spec.Id != "" {
				//onboard device or reconcile
				return ctrl.Result{}, nil
			}
		}

		if !found.Status.Owned {
			logging.Println("Device is Owned is false ")
			found.Spec.Id = ""
			found.Spec.Name = ""
			found.Spec.OwnerID = ""
			found.Status.Status = iotv1alpha1.OCFDeviceCompleted
			//dete device resources
		}
	}

	return ctrl.Result{}, nil
}

func (r *OCFDeviceReconciler) deleteExternalResources(device *iotv1alpha1.OCFDevice) bool {
	//
	// delete any external resources associated with the device
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.
	logging.Printf("DELETING FOR: %s\n", device.Spec.Id)
	deviceId := device.Spec.Id

	resource := &iotv1alpha1.OCFDeviceResource{}
	err := r.Get(context.Background(), types.NamespacedName{Name: deviceId + "-ocf-device-resource", Namespace: device.Namespace}, resource)
	if err != nil {
		logging.Printf("Error getting resource %s\n", err.Error())
		return false
	}

	// we don't care if the job was already deleted
	if err := r.Delete(context.Background(), resource, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
		logging.Printf("unable to delete active resource %s\n", err.Error())
		return false
	}

	//disown device
	ocf_client.DisownDevice(deviceId)

	return true
}

func (r *OCFDeviceReconciler) PeriodicReconcile() {
	// Handle periodic reconciliation logic here
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	//get ocfdeviceboarding resource and get the set values

	for {
		select {
		case <-ticker.C:
			// fmt.Printf("Periodic reconcile at %v \n", time.Now().Format("15:04:05"))
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

					//check number of OCFDevices on the network
					if len(raw_devices) > 0 {

						for _, raw_device := range raw_devices {

							// Define the OCFDevice object
							ocfDevice := &iotv1alpha1.OCFDevice{
								ObjectMeta: metav1.ObjectMeta{
									Name:      raw_device.ID + "-ocf-device",
									Namespace: "default",
								},
								Spec: iotv1alpha1.OCFDeviceSpec{
									Id:      raw_device.ID,
									Name:    raw_device.Details.Name,
									OwnerID: raw_device.OwnerID,
								},
								Status: iotv1alpha1.OCFDeviceStatus{
									Status: iotv1alpha1.OCFDeviceCreating,
									Owned:  raw_device.Owned,
								},
							}

							// Check if the OCFDevice object already exists
							found := &iotv1alpha1.OCFDevice{}
							err = r.Get(context.Background(), types.NamespacedName{Name: raw_device.ID + "-ocf-device", Namespace: "default"}, found)
							if err != nil && errors.IsNotFound(err) {
								// Create the OCFDevice object if it does not exist
								err = r.Create(context.Background(), ocfDevice)
								if err != nil {
									logging.Printf("failed to create OCFDevice: %s\n, skipping", err.Error())
									// return
								} else {

									found.Status.Status = iotv1alpha1.OCFDeviceDiscovery
									r.Client.Status().Update(context.TODO(), found)

									//device created | own/onboard it and register its resources in K8s
									ocf_client.OwnDevice(raw_device.ID)
									links_resources, err := ocf_client.GetResources(raw_device.ID)
									if err != nil {
										found.Status.Status = iotv1alpha1.OCFDeviceError
										r.Client.Status().Update(context.TODO(), found)

										logging.Printf("Failed to get device resources, DeviceID: %s | Error: %s \n", raw_device.ID, err.Error())
									}

									//get the resources 1 by 1 and register
									var raw_device_resources []RawOCFDeviceResource

									err = json.Unmarshal([]byte(links_resources), &raw_device_resources)
									if err != nil {
										logging.Printf("Processing device resources to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
									}
									// Define the Properties array
									properties := []iotv1alpha1.OCFDeviceResourceProperties{}

									//get the device resources and store for this device
									for _, d := range raw_device_resources {

										resource_properties, err := ocf_client.GetResource(raw_device.ID, d.Href)
										if err != nil {
											logging.Printf("Failed to get device resources details, DeviceID: %s | Error: %s \n", raw_device.ID, err.Error())
										}

										//get the resource details 1 by 1 and store in the properties struct
										var raw_device_resource_properties RawOCFDeviceResourceProperties

										err = json.Unmarshal([]byte(resource_properties), &raw_device_resource_properties)
										if err != nil {
											logging.Printf("Processing device resource properties to json failed, DeviceID: %s | Error: %s \n ", raw_device.ID, err.Error())
										}

										//append all the properties
										newProperty := iotv1alpha1.OCFDeviceResourceProperties{}

										newProperty.Name = d.Href

										if raw_device_resource_properties.Value != nil {

											if strconv.FormatBool(*raw_device_resource_properties.Value) == "true" {
												newProperty.Value = raw_device_resource_properties.Value
											} else if strconv.FormatBool(*raw_device_resource_properties.Value) == "false" {
												newProperty.Value = raw_device_resource_properties.Value
											} else {
												newProperty.Value = raw_device_resource_properties.Value
											}

										}

										if raw_device_resource_properties.Units != "" {
											newProperty.Units = raw_device_resource_properties.Units
										}

										if raw_device_resource_properties.Units != "" {
											newProperty.Temperature = strconv.FormatFloat(raw_device_resource_properties.Temperature, 'f', 2, 64)
										}
										properties = append(properties, newProperty)

										// fmt.Println("PROPERTIES HERE")
										// for _, prop := range properties {
										// 	fmt.Printf("Name: %s, Value: %s, Units: %s\n", prop.Name, prop.Value, prop.Units)
										// }

									}

									found.Status.Status = iotv1alpha1.OCFDeviceRunning
									r.Client.Status().Update(context.TODO(), found)

									//create the device resource
									// Define the OCFDevice object
									ocfDeviceResource := &iotv1alpha1.OCFDeviceResource{
										ObjectMeta: metav1.ObjectMeta{
											Name:      raw_device.ID + "-ocf-device-resource",
											Namespace: "default",
										},
										Spec: iotv1alpha1.OCFDeviceResourceSpec{
											DeviceID:   raw_device.ID,
											Properties: properties,
										},
									}
									err = r.Create(context.Background(), ocfDeviceResource)
									if err != nil {
										logging.Printf("failed to create OCFDeviceResource: %s\n, skipping", err.Error())
										// return
									}

								}
							} else if err != nil {

								// Handle any other errors that may occur
								logging.Printf("failed to get OCFDevice: %s\n, skipping", err.Error())
								// return
							}

							// logging.Println("Device Registered ")
							// return
						}
					} else {
						logging.Println("No OCFDevices found on the network")
					}
				} else {
					// logging.Println("OCFDeviceBoarding set to Manual")
				}
			} else {
				logging.Println("No need to trigger device discovery, OCFDeviceBoarding not set")
			}
		}
	}

	// return ctrl.Result{}, nil
}

func (r *OCFDeviceReconciler) OCFDeviceOnboard() {}

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
			&source.Kind{Type: &iotv1alpha1.OCFDevice{}},
			&handler.EnqueueRequestForObject{}).

		// Register the PeriodicReconcile() method for periodic reconciliation
		Owns(&iotv1alpha1.OCFDevice{}).
		Complete(r); err != nil {
		return err
	}

	go r.PeriodicReconcile()

	return nil
}

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

// structure of raw device resources
type RawOCFDeviceResource struct {
	Href string `json:"href"`
}

// structure of raw device resources
type RawOCFDeviceResourceProperties struct {
	If          []string `json:"if"`
	Rt          []string `json:"rt"`
	Name        string   `json:"name"`
	Value       *bool    `json:"value"`
	Units       string   `json:"units"`
	Temperature float64  `json:"temperature"`
}
