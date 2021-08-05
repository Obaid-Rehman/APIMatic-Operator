/*
Copyright 2021 APIMatic.io.

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

	"reflect"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apicodegenv1beta1 "github.com/apimatic/apimatic-operator/api/v1beta1"
)

// APIMaticReconciler reconciles a APIMatic object
type APIMaticReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apicodegen.apimatic.io,resources=apimatics,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apicodegen.apimatic.io,resources=apimatics/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apicodegen.apimatic.io,resources=apimatics/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the APIMatic object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *APIMaticReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// your logic here
	// Fetch the APIMatic instance
	apimatic := &apicodegenv1beta1.APIMatic{}
	err := r.Get(ctx, req.NamespacedName, apimatic)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("APIMatic resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		log.Error(err, "Failed to get APIMatic")
		return ctrl.Result{}, err
	}

	// validates APIMatic instance, setting default values
	err = r.validateAPIMatic(apimatic, &ctx, &log)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if service already exists, if not create a new one
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: apimatic.Name, Namespace: apimatic.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		// Define a new Service
		dep := r.serviceForAPIMatic(apimatic)
		log.Info("Creating a new service", "Service.Namespace", dep.Namespace, "Service.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", dep.Namespace, "Service.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Service created successfully- return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	// Check if statefulset already exists, if not create a new one
	foundStatefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: apimatic.Name, Namespace: apimatic.Namespace}, foundStatefulSet)
	if err != nil && errors.IsNotFound(err) {
		// Define a new StatefulSet
		dep := r.statefulSetForAPIMatic(apimatic)
		log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", dep.Namespace, "StatefulSet.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new StatefulSet", "StatefulSet.Namespace", dep.Namespace, "StatefulSet.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// StatefulSet created successfuly- return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		return ctrl.Result{}, err
	}
	/*
		// Check for any changes in the apimatic spec
		hasChanged := r.checkAndUpdateStatefulSet(apimatic, foundStatefulSet)

		if hasChanged {
			err = r.Update(ctx, foundStatefulSet)
			if err != nil {
				log.Error(err, "Failed to update stateful set", "StatefulSet.Namespace", foundStatefulSet.Namespace, "StatefulSet.Name", foundStatefulSet.Name)
				// Ask to requeue after 1 minute in order to give enough time for the
			 // pods be created on the cluster side and the operand be able
			 // to do the next update step accurately.
				return ctrl.Result{RequeueAfter: time.Minute},nil
			}
		}
	*/

	var foundServiceStatus = foundService.Status
	var foundStatefulSetStatus = foundStatefulSet.Status

	if !reflect.DeepEqual(foundServiceStatus, apimatic.Status.ServiceStatus) || !reflect.DeepEqual(foundStatefulSetStatus, apimatic.Status.StatefulSetStatus) {
		apimatic.Status.ServiceStatus = foundServiceStatus
		apimatic.Status.StatefulSetStatus = foundStatefulSetStatus

		err = r.Status().Update(ctx, apimatic)
		if err != nil {
			log.Error(err, "Failed to update APIMatic status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func labelsForAPIMatic(name string) map[string]string {
	return map[string]string{"app": "apimatic", "apimatic_cr": name}
}


func (r *APIMaticReconciler) validateAPIMatic(a *apicodegenv1beta1.APIMatic, ctx *context.Context, logr *logr.Logger) error {

	log := *logr
	var needsUpdating bool = false

		// Add default replica size of 1 if replicas field not set
		if a.Spec.Replicas == nil {
			a.Spec.Replicas = new(int32)
			*a.Spec.Replicas = 1
			needsUpdating = true
		}
		
		// Add default license volume mounth path /usr/local/apimatic if not provided
		if a.Spec.PodVolumeSpec.APIMaticLicensePath == nil {
			a.Spec.PodVolumeSpec.APIMaticLicensePath = new(string)
			*a.Spec.PodVolumeSpec.APIMaticLicensePath = "/usr/local/apimatic"
			needsUpdating = true
		}

		// Add default container name of apimatic if container name not provided
		if a.Spec.PodSpec.Name == nil {
			a.Spec.PodSpec.Name = new(string)
			*a.Spec.PodSpec.Name = "apimatic"
			needsUpdating = true
		}
  
		if needsUpdating {
			log.Info("Updating APIMatic instance with default values", "APIMatic.Namespace", a.Namespace, "APIMatic.Name", a.Name)
	  err := r.Update(*ctx, a)
		 if err != nil {
			log.Error(err, "Failed to update APIMatic instance", "APIMatic.Namespace", a.Namespace, "APIMatic.Name", a.Name)
		} else {
			log.Info("Successfully updated APIMatic", "APIMatic.Namespace", a.Namespace, "APIMatic.Name", a.Name)
		}
		 return err
		}
		return nil
	}


func (r *APIMaticReconciler) serviceForAPIMatic(a *apicodegenv1beta1.APIMatic) *corev1.Service {
	ls := labelsForAPIMatic(a.Name)

	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: ls,
			Ports: []corev1.ServicePort{{
				Port: a.Spec.ServiceSpec.APIMaticServicePort.Port,
			}},
		},
	}

	if a.Spec.ServiceSpec.Type != nil {
		dep.Spec.Type = *a.Spec.ServiceSpec.Type
	} else {
		dep.Spec.Type = corev1.ServiceTypeClusterIP
	}

	if dep.Spec.Type == corev1.ServiceTypeLoadBalancer || dep.Spec.Type == corev1.ServiceTypeNodePort {
		if a.Spec.ServiceSpec.APIMaticServicePort.NodePort != nil {
			dep.Spec.Ports[0].NodePort = *a.Spec.ServiceSpec.APIMaticServicePort.NodePort
		}
	}

	if dep.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if a.Spec.ServiceSpec.LoadBalancerIP != nil {
			dep.Spec.LoadBalancerIP = *a.Spec.ServiceSpec.LoadBalancerIP
		}
	}

	if a.Spec.ServiceSpec.SessionAffinity != nil {
		dep.Spec.SessionAffinity = *a.Spec.ServiceSpec.SessionAffinity
	} else {
		dep.Spec.SessionAffinity = corev1.ServiceAffinityNone
	}

	if a.Spec.ServiceSpec.ExternalTrafficPolicy != nil {
		dep.Spec.ExternalTrafficPolicy = *a.Spec.ServiceSpec.ExternalTrafficPolicy
	}
	
	if dep.Spec.Type == corev1.ServiceTypeLoadBalancer && dep.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
		if a.Spec.ServiceSpec.HealthCheckNodePort != nil {
			dep.Spec.HealthCheckNodePort = *a.Spec.ServiceSpec.HealthCheckNodePort
		}
	}

	if a.Spec.ServiceSpec.PublishNotReadyAddresses != nil {
		dep.Spec.PublishNotReadyAddresses = *a.Spec.ServiceSpec.PublishNotReadyAddresses
	} else {
		dep.Spec.PublishNotReadyAddresses = false
	}

	if a.Spec.ServiceSpec.SessionAffinityConfig != nil {
		dep.Spec.SessionAffinityConfig = a.Spec.ServiceSpec.SessionAffinityConfig
	}

	if a.Spec.ServiceSpec.TopologyKeys != nil {
		dep.Spec.TopologyKeys = []string{}
		dep.Spec.TopologyKeys = append(dep.Spec.TopologyKeys, a.Spec.ServiceSpec.TopologyKeys...)
	}

	if a.Spec.ServiceSpec.IPFamilyPolicy != nil {
		dep.Spec.IPFamilyPolicy = a.Spec.ServiceSpec.IPFamilyPolicy
	} 

	if (dep.Spec.Type == corev1.ServiceTypeClusterIP || dep.Spec.Type == corev1.ServiceTypeLoadBalancer || dep.Spec.Type == corev1.ServiceTypeNodePort) && (*dep.Spec.IPFamilyPolicy != corev1.IPFamilyPolicySingleStack) {
		if a.Spec.ServiceSpec.IPFamilies != nil {
			dep.Spec.IPFamilies = []corev1.IPFamily{}
			dep.Spec.IPFamilies = append(dep.Spec.IPFamilies, a.Spec.ServiceSpec.IPFamilies...)
		}
	}

	if dep.Spec.Type == corev1.ServiceTypeLoadBalancer {
		if a.Spec.ServiceSpec.AllocateLoadBalancerNodePorts != nil {
			dep.Spec.AllocateLoadBalancerNodePorts = a.Spec.ServiceSpec.AllocateLoadBalancerNodePorts
		}
	}

	// Set APIMatic instance as owner and controller
	ctrl.SetControllerReference(a, dep, r.Scheme)
	return dep
}

func (r *APIMaticReconciler) statefulSetForAPIMatic(a *apicodegenv1beta1.APIMatic) *appsv1.StatefulSet {
	ls := labelsForAPIMatic(a.Name)
	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.Name,
			Namespace: a.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: a.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			ServiceName: a.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           a.Spec.PodSpec.Image,
						Name:            *a.Spec.PodSpec.Name,
						Env: []corev1.EnvVar{{
							Name: "LICENSEPATH",
							Value: *a.Spec.PodVolumeSpec.APIMaticLicensePath,
						}},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
						}},
						VolumeMounts: []corev1.VolumeMount{{
							ReadOnly: true,
							MountPath: *a.Spec.PodVolumeSpec.APIMaticLicensePath,
							Name: a.Spec.PodVolumeSpec.APIMaticLicenseVolumeName,
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: a.Spec.PodVolumeSpec.APIMaticLicenseVolumeName,
						VolumeSource: a.Spec.PodVolumeSpec.APIMaticLicenseVolumeSource,
					}},
				},
			},
		},
	}

	if a.Spec.PodSpec.ImagePullPolicy != nil {
		dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = *a.Spec.PodSpec.ImagePullPolicy
	} else {
		dep.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullIfNotPresent
	}

	if a.Spec.VolumeClaimTemplates != nil {
		dep.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{}
		dep.Spec.VolumeClaimTemplates = append(dep.Spec.VolumeClaimTemplates, a.Spec.VolumeClaimTemplates...)
		}			
		
	if a.Spec.PodSpec.Resources != nil {
		dep.Spec.Template.Spec.Containers[0].Resources = *a.Spec.PodSpec.Resources
	}

	if a.Spec.PodSpec.SideCars != nil {
		dep.Spec.Template.Spec.Containers = append(dep.Spec.Template.Spec.Containers, a.Spec.PodSpec.SideCars...)
	}

	if a.Spec.PodSpec.InitContainers != nil {
		dep.Spec.Template.Spec.InitContainers = []corev1.Container{}
		dep.Spec.Template.Spec.InitContainers = append(dep.Spec.Template.Spec.InitContainers, a.Spec.PodSpec.InitContainers...)
	}

	if a.Spec.PodVolumeSpec.AdditionalVolumes != nil {
		dep.Spec.Template.Spec.Volumes = append(dep.Spec.Template.Spec.Volumes, a.Spec.PodVolumeSpec.AdditionalVolumes...)
	}
	// Set APIMatic instance as owner and controller
	ctrl.SetControllerReference(a, dep, r.Scheme)
	return dep
}

/*
func (r *APIMaticReconciler) checkAndUpdateStatefulSet(a *apicodegenv1beta1.APIMatic, s *appsv1.StatefulSet) bool {
	hasChanged := false
	// Check for any changes in the apimatic spec
	size := a.Spec.Replicas;
 if *s.Spec.Replicas != size {
		s.Spec.Replicas = &size
		hasChanged = true
	}

	if s.Spec.Template.Spec.Containers[0].Image != a.Spec.PodSpec.Image {
		s.Spec.Template.Spec.Containers[0].Image = a.Spec.PodSpec.Image
		hasChanged = true
	}

	if s.Spec.Template.Spec.Containers[0].ImagePullPolicy != a.Spec.PodSpec.ImagePullPolicy {
		s.Spec.Template.Spec.Containers[0].ImagePullPolicy = a.Spec.PodSpec.ImagePullPolicy
		hasChanged = true
	}

	// Check if no of sidecar containers in APIMatic pod has changed. This is one less than the total containers listed in the StatefulSet PodSpec field


	return hasChanged
}
*/
// SetupWithManager sets up the controller with the Manager.
func (r *APIMaticReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apicodegenv1beta1.APIMatic{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
