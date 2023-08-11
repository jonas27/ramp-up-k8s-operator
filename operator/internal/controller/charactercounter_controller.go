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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	rampupv1alpha1 "github.com/jonas27/ramp-up-k8s-operator/operator/api/v1alpha1"
)

// CharacterCounterReconciler reconciles a CharacterCounter object
type CharacterCounterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type CharacterCounterStep struct{}

//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=charactercounters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=charactercounters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ramp-up.joe.ionos.io,resources=charactercounters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CharacterCounter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *CharacterCounterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("start reconcile for", "name", req.Name)

	var cc rampupv1alpha1.CharacterCounter
	if err := r.Get(ctx, req.NamespacedName, &cc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("unable to fetch ChracterCounter CR: %w", err))
	}

	if cc.Spec.Namespace == "" {
		cc.Spec.Namespace = "default"
	}

	log.Info("server reconcileService")
	if err := r.reconcileService(ctx, log, cc, cc.Spec.Server); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("server reconcileDeployment")
	if err := r.reconcileDeployement(ctx, log, cc, cc.Spec.Server, false); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("frontend reconcileService")
	if err := r.reconcileService(ctx, log, cc, cc.Spec.Frontend); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("frontend reconcileDeployment")
	return ctrl.Result{}, r.reconcileDeployement(ctx, log, cc, cc.Spec.Frontend, true)
}

func (r *CharacterCounterReconciler) reconcileService(ctx context.Context, log logr.Logger, cc rampupv1alpha1.CharacterCounter, component rampupv1alpha1.CharacterCounterComponent) error {
	var service corev1.Service
	service.Name = component.Name
	service.Namespace = cc.Namespace
	service.Labels = cc.Spec.Labels
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, &service, func() error {
		modifyService(cc, component, &service)
		return ctrl.SetControllerReference(&cc, &service, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("could not create or update service: %w", err)
	}
	if op != controllerutil.OperationResultNone {
		log.Info("reconcile service successfully", "operation", op)
	}
	return nil
}

func modifyService(cc rampupv1alpha1.CharacterCounter, component rampupv1alpha1.CharacterCounterComponent, service *corev1.Service) {
	service.Labels = cc.Spec.Labels
	service.Spec = corev1.ServiceSpec{
		Ports:    component.ServicePorts,
		Selector: component.Selector,
	}
}

func (r *CharacterCounterReconciler) reconcileDeployement(ctx context.Context, log logr.Logger, cc rampupv1alpha1.CharacterCounter, component rampupv1alpha1.CharacterCounterComponent, frontend bool) error {
	args := []string{}
	if frontend {
		args = append(args, "-grpc-addr", fmt.Sprintf("%s:80", cc.Spec.Server.Name))
	}
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      component.Name,
			Namespace: cc.Spec.Namespace,
			Labels:    cc.Spec.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: component.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: component.Selector,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: component.Selector,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Args:            args,
							Name:            component.Name,
							Image:           component.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports:           component.Ports,
						},
					},
				},
			},
		},
	}
	op, err := ctrl.CreateOrUpdate(ctx, r.Client, &deployment, func() error {
		modifyDeployment(cc, component, &deployment, frontend)
		return ctrl.SetControllerReference(&cc, &deployment, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("could not create or update pod: %w", err)
	}
	if op != controllerutil.OperationResultNone {
		log.Info("reconcile pod successfully", "operation", op)
	}
	return nil
}

func modifyDeployment(cc rampupv1alpha1.CharacterCounter, component rampupv1alpha1.CharacterCounterComponent, deployment *appsv1.Deployment, frontend bool) {
	args := []string{}
	if frontend {
		args = append(args, "-grpc-addr", cc.Spec.Server.Name)
	}

	deployment.Labels = cc.Spec.Labels
	deploySepc := &deployment.Spec
	if len(deploySepc.Template.Spec.Containers) == 0 {
		deploySepc.Template.Spec.Containers = []corev1.Container{
			{
				Args:            args,
				Name:            component.Name,
				Image:           component.Image,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Ports:           component.Ports,
			},
		}
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *CharacterCounterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rampupv1alpha1.CharacterCounter{}).
		Owns(&corev1.Service{}).Owns(&appsv1.Deployment{}).
		Complete(r)
}
