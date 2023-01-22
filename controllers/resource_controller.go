/*
Copyright 2023 Ian Rudie.

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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	demov1alpha1 "github.com/ilrudie/observed-generation-demo/api/v1alpha1"
)

// ResourceReconciler reconciles a Resource object
type ResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.demo.demo,resources=resources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.demo.demo,resources=resources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.demo.demo,resources=resources/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Resource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	resource := demov1alpha1.Resource{}
	err := r.Get(ctx, req.NamespacedName, &resource)
	if err != nil {
		// hopefully a delete, just give up I guess
		return ctrl.Result{}, nil
	}

	resource.Status.ReconciledBody = resource.Spec.Body
	resource.Status.ObservedGeneration = resource.ObjectMeta.Generation
	resource.Status.ReconcileTimestamp = time.Now().UTC().Format(time.UnixDate)

	r.Client.Status().Update(ctx, &resource)

	return ctrl.Result{}, nil
}

func checkObservedGeneration(o client.Object) bool {
	// safe type assert
	resource, ok := o.(*demov1alpha1.Resource)
	if !ok {
		//something strange, just reconcile I guess
		return true
	}

	if resource.ObjectMeta.Generation == resource.Status.ObservedGeneration {
		return false
	}
	return true
}

func makePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return checkObservedGeneration(e.ObjectNew)
		},

		// on startup nothing is already known so it will look like a create to your operator
		CreateFunc: func(e event.CreateEvent) bool {
			return checkObservedGeneration(e.Object)
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1alpha1.Resource{}).
		WithEventFilter(makePredicate()).
		Complete(r)
}
