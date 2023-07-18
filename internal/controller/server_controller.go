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

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	rampupv1alpha1 "github.com/jonas27/ramp-up-k8s-operator/api/v1alpha1"
)

// ServerReconciler reconciles a Server object.
type ServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    logr.Logger
}

//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=servers/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)
	r.log.Info("start reconcile for", "name", req.Name)

	var server rampupv1alpha1.Server
	if err := r.Get(ctx, req.NamespacedName, &server); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("unable to fetch Server: %w", err))
	}
	if err := r.reconcileService(ctx, server); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, r.reconcilePod(ctx, server)
}

// CreateOrUpdate infos here https://github.com/pivotal/blog/blob/master/content/post/gp4k-kubebuilder-lessons.md
func (r *ServerReconciler) reconcileService(ctx context.Context, server rampupv1alpha1.Server) error {
	var service corev1.Service
	service.Name = server.Name
	service.Namespace = server.Namespace
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error {
		modifyService(server, &service)
		return ctrl.SetControllerReference(&server, &service, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("could not create or update service: %w", err)
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("reconcile service successfully", "operation", op)
	}
	return nil
}

func modifyService(server rampupv1alpha1.Server, service *corev1.Service) {
	service.Labels = server.Spec.Selector
	service.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{{
			Port:       server.Spec.ServicePort,
			TargetPort: intstr.FromInt(int(server.Spec.ContainerPort)),
		}},
		Selector: server.Spec.Selector,
	}
}

// CreateOrUpdate infos here https://github.com/pivotal/blog/blob/master/content/post/gp4k-kubebuilder-lessons.md
func (r *ServerReconciler) reconcilePod(ctx context.Context, server rampupv1alpha1.Server) error {
	var pod corev1.Pod
	pod.Name = server.Name
	pod.Namespace = server.Namespace
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, &pod, func() error {
		modifyPod(server, &pod)
		return ctrl.SetControllerReference(&server, &pod, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("could not create or update pod: %w", err)
	}
	if op != controllerutil.OperationResultNone {
		r.log.Info("reconcile pod successfully", "operation", op)
	}
	return nil
}

func modifyPod(server rampupv1alpha1.Server, pod *corev1.Pod) {
	pod.Labels = server.Spec.Selector
	podSpec := &pod.Spec
	if len(podSpec.Containers) == 0 {
		podSpec.Containers = make([]corev1.Container, 1)
		podSpec.Containers[0] = corev1.Container{
			Name:  server.Name,
			Ports: []corev1.ContainerPort{{ContainerPort: server.Spec.ContainerPort}},
		}
	}
	podSpec.Containers[0].Image = server.Spec.Image
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rampupv1alpha1.Server{}).
		Owns(&corev1.Service{}).Owns(&corev1.Pod{}).
		Complete(r)
}
