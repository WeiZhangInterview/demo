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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"nameclass/util"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	demov1 "nameclass/api/v1"
)

// NameClassReconciler reconciles a NameClass object
type NameClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.demo.com,resources=nameclasses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.demo.com,resources=nameclasses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.demo.com,resources=nameclasses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NameClass object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *NameClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("---start Reconcile--- %s, %s", req.Namespace, req.Name))
	mgr, _ := manager.New(ctrl.GetConfigOrDie(), manager.Options{})
	generatedClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())

	namespaces := generatedClient.CoreV1().Namespaces()
	namespaceList, _ := namespaces.List(ctx, metav1.ListOptions{})
	customNameSpace := make([]string, 0)
	for _, ns := range namespaceList.Items {
		if util.Contains(DefaultNameSpace, ns.Name) {
			continue
		}
		customNameSpace = append(customNameSpace, ns.Name)
	}
	err := r.UpdateSpec(ctx, customNameSpace)
	if err != nil {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NameClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.NameClass{}).
		Complete(r)
}

func (r *NameClassReconciler) UpdateSpec(ctx context.Context, namespaceNames []string) error {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("---start Reconcile UpdateSpec ---"))

	for _, name := range namespaceNames {
		var namespace corev1.Namespace
		if err := r.Get(ctx, types.NamespacedName{Name: name}, &namespace); err != nil {
			if errors.IsNotFound(err) {
				// we'll ignore not-found errors, since they can't be fixed by an immediate
				// requeue (we'll need to wait for a new notification), and we can get them
				// on deleted requests.
				return nil
			}
			logger.Error(err, "unable to fetch namespace")
			return err
		}
		namespaceCopy := namespace.DeepCopy()
		err := r.Update(ctx, namespaceCopy)
		if err != nil {
			logger.Error(err, "UpdateSpec failed")
			return err
		}
	}
	return nil
}
