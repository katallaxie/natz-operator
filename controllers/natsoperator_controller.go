package controllers

import (
	"context"
	"fmt"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	natsv1alpha1 "github.com/katallaxie/natz-operator/api/v1alpha1"
	"github.com/katallaxie/natz-operator/pkg/status"
	"github.com/katallaxie/pkg/conv"
	"github.com/katallaxie/pkg/slices"
	"github.com/katallaxie/pkg/utilx"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	corev1 "k8s.io/api/core/v1"
)

const (
	EventRecorderLabel = "natz-controller"
)

type EventReason string

const (
	EventReasonOperatorCreateFailed          EventReason = "OperatorCreateFailed"
	EventReasonOperatorUpdateFailed          EventReason = "OperatorUpdateFailed"
	EventReasonOperatorDeleteFailed          EventReason = "OperatorDeleteFailed"
	EventReasonOperatorSecretCreateSucceeded EventReason = "OperatorSecretCreateSucceeded"
	EventReasonOperatorSecretCreateFailed    EventReason = "OperatorSecretCreateFailed"
	EventReasonOperatorSynchronized          EventReason = "OperatorSynchronized"
	EventReasonOperatorFailed                EventReason = "OperatorFailed"
	EventReasonOperatorSynchronizeFailed     EventReason = "OperatorSynchronizeFailed"
)

// NatsOperatorReconciler ...
type NatsOperatorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// NewNatsOperatorReconciler ...
func NewNatsOperatorReconciler(mgr ctrl.Manager) *NatsOperatorReconciler {
	return &NatsOperatorReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(EventRecorderLabel),
	}
}

//+kubebuilder:rbac:groups=natz.katallaxie.dev,resources=natsoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=natz.katallaxie.dev,resources=natsoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=natz.katallaxie.dev,resources=natsoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile ...
// nolint:gocyclo
func (r *NatsOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	operator := &natsv1alpha1.NatsOperator{}
	if err := r.Get(ctx, req.NamespacedName, operator); err != nil {
		// Request object not found, could have been deleted after reconcile request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !operator.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, operator)
	}

	if operator.Spec.Paused {
		return r.reconcilePaused(ctx, operator)
	}

	// get latest version of the account
	if err := r.Get(ctx, req.NamespacedName, operator); err != nil {
		return reconcile.Result{}, err
	}

	err := r.reconcileResources(ctx, operator)
	if err != nil {
		return r.ManageError(ctx, operator, err)
	}

	return r.ManageSuccess(ctx, operator)
}

func (r *NatsOperatorReconciler) reconcilePaused(ctx context.Context, sk *natsv1alpha1.NatsOperator) (ctrl.Result, error) {
	if sk.Status.ControlPaused {
		return ctrl.Result{}, nil
	}

	if sk.Spec.Paused {
		sk.Status.ControlPaused = true
		return ctrl.Result{}, r.Status().Update(ctx, sk)
	}

	return ctrl.Result{}, nil
}

func (r *NatsOperatorReconciler) reconcileResources(ctx context.Context, operator *natsv1alpha1.NatsOperator) error {
	return r.reconcileOperator(ctx, operator)
}

func (r *NatsOperatorReconciler) reconcileOperator(ctx context.Context, obj *natsv1alpha1.NatsOperator) error {
	pk := &corev1.Secret{}
	pkName := client.ObjectKey{
		Namespace: obj.Namespace,
		Name:      obj.Spec.PrivateKey.Name,
	}

	if err := r.Get(ctx, pkName, pk); err != nil {
		return err
	}

	seed, ok := pk.Data[natsv1alpha1.SecretSeedDataKey]
	if !ok {
		return fmt.Errorf("public key not found")
	}

	sk, err := nkeys.FromSeed(seed)
	if err != nil {
		return err
	}

	public, err := sk.PublicKey()
	if err != nil {
		return err
	}

	token := jwt.NewOperatorClaims(public)
	jwt, err := token.Encode(sk)
	if err != nil {
		return err
	}

	obj.Status.JWT = jwt
	obj.Status.PublicKey = public

	if !controllerutil.ContainsFinalizer(obj, natsv1alpha1.FinalizerName) {
		controllerutil.AddFinalizer(obj, natsv1alpha1.FinalizerName)
	}

	return nil
}

func (r *NatsOperatorReconciler) reconcileDelete(ctx context.Context, operator *natsv1alpha1.NatsOperator) (ctrl.Result, error) {
	// Remove our finalizer from the list.
	controllerutil.RemoveFinalizer(operator, natsv1alpha1.FinalizerName)

	if !operator.DeletionTimestamp.IsZero() {
		// Remove our finalizer from the list.
		controllerutil.RemoveFinalizer(operator, natsv1alpha1.FinalizerName)

		// Stop reconciliation as the object is being deleted.
		return ctrl.Result{}, r.Update(ctx, operator)
	}

	return ctrl.Result{Requeue: true}, nil
}

// IsCreating ...
func (r *NatsOperatorReconciler) IsCreating(obj *natsv1alpha1.NatsOperator) bool {
	return utilx.Or(obj.Status.Conditions == nil, slices.Size(0, obj.Status.Conditions))
}

// IsSynchronized ...
func (r *NatsOperatorReconciler) IsSynchronized(obj *natsv1alpha1.NatsOperator) bool {
	return obj.Status.Phase == natsv1alpha1.OperatorPhaseSynchronized
}

// IsPaused ...
func (r *NatsOperatorReconciler) IsPaused(obj *natsv1alpha1.NatsOperator) bool {
	return obj.Status.ControlPaused
}

// ManageError ...
func (r *NatsOperatorReconciler) ManageError(ctx context.Context, obj *natsv1alpha1.NatsOperator, err error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Error(err, "reconciling operator", "operator", obj.Name)

	if errors.IsNotFound(err) {
		return ctrl.Result{Requeue: true}, nil
	}

	status.SetNatzOperatorCondition(obj, status.NewOperatorFailedCondition(obj, err))
	obj.Status.Phase = natsv1alpha1.OperatorPhaseFailed

	if err := r.Client.Status().Update(ctx, obj); err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, err
	}

	r.Recorder.Event(obj, corev1.EventTypeWarning, conv.String(EventReasonOperatorSynchronizeFailed), "operator synchronization failed")

	var retryInterval time.Duration

	return reconcile.Result{
		RequeueAfter: time.Duration(math.Min(float64(retryInterval.Nanoseconds()*2), float64(time.Hour.Nanoseconds()*6))),
		Requeue:      true,
	}, nil
}

// ManageSuccess ...
func (r *NatsOperatorReconciler) ManageSuccess(ctx context.Context, obj *natsv1alpha1.NatsOperator) (ctrl.Result, error) {
	obj.Status.Phase = natsv1alpha1.OperatorPhaseSynchronized
	obj.Status.LastUpdate = metav1.Now()
	status.SetNatzOperatorCondition(obj, status.NewOperatorSychronizedCondition(obj))

	err := r.Status().Update(ctx, obj)
	if err != nil {
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second}, err
	}

	r.Recorder.Event(obj, corev1.EventTypeNormal, conv.String(EventReasonOperatorSynchronized), "operator synchronized")

	return ctrl.Result{}, nil
}

// IsControlPaused ...
func (r *NatsOperatorReconciler) IsControlPaused(obj *natsv1alpha1.NatsOperator) bool {
	return obj.Status.ControlPaused
}

// SetupWithManager sets up the controller with the Manager.
func (r *NatsOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&natsv1alpha1.NatsOperator{}).
		Owns(&natsv1alpha1.NatsAccount{}).
		Owns(&corev1.Secret{}).
		WithEventFilter(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{})).
		Complete(r)
}
