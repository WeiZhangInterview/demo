/*
Copyright 2024.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	demov1 "nameclass/api/v1"
	"nameclass/util"
)

// NameSapceReconciler reconciles a NameSapce object
type NameSapceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	labelClass     = "namespaceclass.akuity.io/name"
	cheatNameSpace = "default"
)

var DefaultNameSpace = []any{"default", "kube-system", "kube-node-lease", "demo", "kube-public", "demo-system"}

//+kubebuilder:rbac:groups=namesapceclass.demo.com,resources=namesapces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=namesapceclass.demo.com,resources=namesapces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=namesapceclass.demo.com,resources=namesapces/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NameSapce object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *NameSapceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// return for default namespace
	if util.Contains(DefaultNameSpace, req.Name) {
		return ctrl.Result{}, nil
	}

	logger := log.FromContext(ctx)
	var namespace corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &namespace); err != nil {
		if errors.IsNotFound(err) {
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch namespace")
		return ctrl.Result{}, err
	}
	if namespace.Status.Phase == corev1.NamespaceTerminating {
		logger.Info(fmt.Sprintf("byte byte  %s", req.Name))
		return ctrl.Result{}, nil
	}

	if value, ok := namespace.Labels[labelClass]; ok {
		var nameClass demov1.NameClass
		if err := r.Get(ctx, types.NamespacedName{Name: value, Namespace: cheatNameSpace}, &nameClass); err != nil {
			logger.Error(err, "unable to fetch crd")
		}
		nameClassCopy := nameClass.DeepCopy()
		logger.Info(fmt.Sprintf("find from defaut namespace  %s", req.Name))
		logger.Info(fmt.Sprintf("image is   %s", nameClassCopy.Spec.Image))

		_, err := r.MatchResource(ctx, req, nameClassCopy)
		if err != nil {
			logger.Error(err, "update crd failed")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NameSapceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

func (r *NameSapceReconciler) MatchResource(ctx context.Context, req ctrl.Request, class *demov1.NameClass) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("start Reconcile Match Resource  value and name is %s, %s", req.Namespace, req.Name))
	if class == nil {
		return reconcile.Result{}, nil
	}

	/*
		create/update deploy
	*/
	oldDeploy := &appsv1.Deployment{}
	newDeploy := NewDeploy(class, req, r.Scheme)
	if err := r.Get(ctx, req.NamespacedName, oldDeploy); err != nil && errors.IsNotFound(err) {
		logger.Info("creating deploy")
		// 1. create Deploy
		if err := r.Create(ctx, newDeploy); err != nil {
			logger.Error(err, "create deploy failed")
			return reconcile.Result{}, err
		}
		logger.Info("create deploy done")
	} else {
		if !reflect.DeepEqual(oldDeploy.Spec, newDeploy.Spec) {
			logger.Info("updating deploy")
			oldDeploy.Spec = newDeploy.Spec
			if err := r.Update(ctx, oldDeploy); err != nil {
				logger.Error(err, "update old deploy failed")
				return reconcile.Result{}, err
			}
			logger.Info("update deploy done")
		}
	}
	/*
		create/update Service
	*/
	oldService := &corev1.Service{}
	newService := NewService(class, req, r.Scheme)
	if err := r.Get(ctx, req.NamespacedName, oldService); err != nil && errors.IsNotFound(err) {
		logger.Info("Creating service")
		if err := r.Create(ctx, newService); err != nil {
			logger.Error(err, "create service failed")
			return reconcile.Result{}, err
		}
		logger.Info("create service done")
		return reconcile.Result{}, nil
	} else {
		if !reflect.DeepEqual(oldService.Spec, newService.Spec) {
			logger.Info("updating service")
			clstip := oldService.Spec.ClusterIP
			oldService.Spec = newService.Spec
			oldService.Spec.ClusterIP = clstip
			if err := r.Update(ctx, oldService); err != nil {
				logger.Error(err, "update service failed")
				return reconcile.Result{}, err
			}
			logger.Info("update service done")
			return reconcile.Result{}, nil
		}
	}
	/*
		TODO: support other resource
	*/

	return ctrl.Result{}, nil
}

// create a new deploy object
func NewDeploy(owner *demov1.NameClass, req ctrl.Request, scheme *runtime.Scheme) *appsv1.Deployment {
	labels := map[string]string{"app": owner.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      owner.Name,
			Namespace: req.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: owner.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            owner.Name,
							Image:           owner.Spec.Image,
							Ports:           []corev1.ContainerPort{{ContainerPort: owner.Spec.Port}},
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             owner.Spec.Envs,
						},
					},
				},
			},
			Selector: selector,
		},
	}
	// add ControllerReference for deployment
	if err := controllerutil.SetControllerReference(owner, deploy, scheme); err != nil {
		fmt.Printf("setControllerReference for Deployment %s/%s failed", owner.Namespace, owner.Name)
	}
	return deploy
}

// create a new service object
func NewService(owner *demov1.NameClass, req ctrl.Request, scheme *runtime.Scheme) *corev1.Service {
	srv := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      owner.Name,
			Namespace: req.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: owner.Spec.Port, NodePort: owner.Spec.Nodeport}},
			Selector: map[string]string{
				"app": owner.Name,
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}
	// add ControllerReference for service
	if err := controllerutil.SetControllerReference(owner, srv, scheme); err != nil {
		fmt.Printf("setcontrollerReference for Service %s/%s failed", owner.Namespace, owner.Name)
	}
	return srv
}
