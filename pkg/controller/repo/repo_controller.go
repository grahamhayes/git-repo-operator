package repo

import (
	"os"
	"context"
	"github.com/google/go-github/v31/github"
	"golang.org/x/oauth2"
	repov1alpha1 "github.com/grahamhayes/git-repo-operator/pkg/apis/repo/v1alpha1"
	// corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"github.com/go-logr/logr"
)

var log = logf.Log.WithName("controller_repo")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Repo Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRepo{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("repo-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Repo
	err = c.Watch(&source.Kind{Type: &repov1alpha1.Repo{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRepo implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRepo{}

// ReconcileRepo reconciles a Repo object
type ReconcileRepo struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

const repoFinalizer = "finalizer.repo.gra.ham.ie"

// Reconcile reads that state of the cluster for a Repo object and makes changes based on the state read
// and what is in the Repo.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRepo) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Repo")

	var token = os.Getenv("GH_ACCESS_TOKEN")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	gh_client := github.NewClient(tc)



	// Fetch the Repo instance
	instance := &repov1alpha1.Repo{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("Repo CRD Deleted")
			// _, err := gh_client.Repositories.Delete(ctx, instance.Spec.Organisation, instance.Spec.Repository)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.GetDeletionTimestamp() != nil {
		if err := r.finalizeRepo(reqLogger, instance, gh_client, ctx); err != nil {
			return reconcile.Result{}, err
		}
		instance.SetFinalizers(remove(instance.GetFinalizers(), repoFinalizer))
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

		// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), repoFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	repo, resp, err := gh_client.Repositories.Get(ctx, instance.Spec.Organisation, instance.Spec.Repository)
	if err == nil {
		reqLogger.Info("Skip reconcile: Repo already exists", "URL", repo.SSHURL)
		return reconcile.Result{}, nil
	}

	if resp.StatusCode == 404 {
		repo := &github.Repository{
			Name:    github.String(instance.Spec.Repository),
			Private: github.Bool(true),
		}

		repo, _, err := gh_client.Repositories.Create(ctx, instance.Spec.Organisation, repo)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Created new Repo: Repo created", "URL", repo.SSHURL)
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), repoFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, err
}

func (r *ReconcileRepo) finalizeRepo(reqLogger logr.Logger, m *repov1alpha1.Repo, gh_client *github.Client, ctx context.Context) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	resp, err := gh_client.Repositories.Delete(ctx, m.Spec.Organisation, m.Spec.Repository)
	if resp.StatusCode == 404 {
		err = nil
	}
	reqLogger.Info("Successfully finalized repo")
	return err
}

func (r *ReconcileRepo) addFinalizer(reqLogger logr.Logger, m *repov1alpha1.Repo) error {
	reqLogger.Info("Adding Finalizer for the Repo")
	m.SetFinalizers(append(m.GetFinalizers(), repoFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Repo with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}