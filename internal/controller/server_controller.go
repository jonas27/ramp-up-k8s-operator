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

package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	rampupv1alpha1 "github.com/jonas27/ramp-up-k8s-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServerReconciler reconciles a Server object.
type ServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    logr.Logger
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = rampupv1alpha1.GroupVersion.String()
)

//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Server object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)

	r.log.Info("start reconcile for", "name", req.Name)

	var server rampupv1alpha1.Server
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("unable to fetch Server: %w", err))
	}

	desiredService := v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
			Labels:    server.Spec.Selector,
		},
		Spec: v1.ServiceSpec{
			Ports:    []v1.ServicePort{{Port: server.Spec.ServicePort, TargetPort: intstr.FromInt(int(server.Spec.ContainerPort))}},
			Selector: server.Spec.Selector,
		}}
	if err := controllerutil.SetControllerReference(&server, &desiredService, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to set control of servive: %w", err)
	}

	foundService := v1.Service{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(&desiredService), &foundService); err != nil {
		// create the desired pod if the pod doesn't exist already
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, &desiredService); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	// pod was found
	// validate the labels of the found pod
	foundLabels := foundService.Labels
	updateService := false
	// if the foundLabels match the expectedLabels, no need to do anything: just exit peacefully
	if !reflect.DeepEqual(server.Spec.Selector, foundLabels) {
		updateService = true
		// else, update the foundPod with expectedLabels
		foundService.Labels = server.Spec.Selector
	}
	foundSpec := foundService.Spec
	// if the foundLabels match the expectedLabels, no need to do anything: just exit peacefully
	if !reflect.DeepEqual(desiredService.Spec, foundSpec) {
		updateService = true
		// else, update the foundPod with expectedLabels
		foundService.Spec = desiredService.Spec
	}

	if updateService {
		if err := r.Update(ctx, &foundService); err != nil {
			return ctrl.Result{}, err
		}
	}

	desiredPod := v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      server.Name,
			Namespace: server.Namespace,
			Labels:    server.Spec.Selector,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{{
				Name:  server.Name,
				Ports: []v1.ContainerPort{{ContainerPort: server.Spec.ContainerPort}},
				Image: server.Spec.Image,
			}}}}
	if err := controllerutil.SetControllerReference(&server, &desiredPod, r.Scheme); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to set control of pod: %w", err)
	}

	foundPod := v1.Pod{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(&desiredPod), &foundPod); err != nil {
		// create the desired pod if the pod doesn't exist already
		if errors.IsNotFound(err) {
			return ctrl.Result{}, r.Create(ctx, &desiredPod)
		}
		return ctrl.Result{}, err
	}
	// pod was found
	// validate the labels of the found pod
	foundLabels = foundPod.Labels
	// if the foundLabels match the expectedLabels, no need to do anything: just exit peacefully
	if reflect.DeepEqual(server.Spec.Selector, foundLabels) {
		return ctrl.Result{}, nil
	}

	// else, update the foundPod with expectedLabels
	foundPod.Labels = server.Spec.Selector
	return ctrl.Result{}, r.Update(ctx, &foundPod)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rampupv1alpha1.Server{}).
		Owns(&v1.Pod{}).Owns(&v1.Service{}).
		Complete(r)
}
