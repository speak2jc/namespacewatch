package watcher

import (
	"context"
	samplev1alpha1 "github.com/speak2jc/namespacewatch/pkg/apis/sample/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_watcher")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Watcher Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	_ = rbac.AddToScheme(mgr.GetScheme())
	return &ReconcileWatcher{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("watcher-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Watcher
	err = c.Watch(&source.Kind{Type: &samplev1alpha1.Watcher{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Watcher
	err = c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	// TODO Use ownerRef instead where netpol owned by namespace - then changes in netpol or namespace will come here
	//err = c.Watch(&source.Kind{Type: &v1.NetworkPolicy{}}, &handler.EnqueueRequestForObject{})
	//if err != nil {
	//	return err
	//}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Watcher
	err = c.Watch(&source.Kind{Type: &v1.NetworkPolicy{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &corev1.Namespace{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileWatcher implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileWatcher{}

// ReconcileWatcher reconciles a Watcher object
type ReconcileWatcher struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Watcher object and makes changes based on the state read
// and what is in the Watcher.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWatcher) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	//reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reqLogger.Info("Reconciling Watcher")

	// Fetch the Watcher instance
	instance := &corev1.Namespace{}
	//if strings.Contains(request.NamespacedName.Name, "netpol") {
	//	instance = &v1.NetworkPolicy{}
	//}
	//instance := &samplev1alpha1.Watcher{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue

			namespacedName := types.NamespacedName{
				Namespace: "",
				Name:      request.Name,
			}
			err := r.client.Get(context.TODO(), namespacedName, instance)
			if err != nil {
				if errors.IsNotFound(err) {
					return reconcile.Result{}, nil
				}
				// Error reading the object - requeue the request.
				return reconcile.Result{}, err
			}
		}

	}

	namespacePhase := instance.Status.Phase
	namespace := instance.ObjectMeta.Name
	log.Info("Processing namespace " + namespace + " in phase " + string(namespacePhase))

	annotations := instance.Annotations
	for annotation, _ := range annotations {
		log.Info("Annotations on namespace " + namespace + ": " + annotation)
	}
	labels := instance.Labels
	//global := false
	if labels != nil {

		for label, _ := range labels {
			log.Info("Labels on namespace " + namespace + ": " + label)
		}
		if val, ok := labels["network"]; ok {
			if val == "global" {
				//global = true
			}
		}
	}

	networkPolicies := &v1.NetworkPolicyList{}
	listOptions := &client.ListOptions{Namespace: namespace}

	err = r.client.List(context.TODO(), listOptions, networkPolicies)
	if err != nil {

	}

	if namespace == "default" || namespace == "kube-admin" || namespace == "kube-public" || namespace == "kube-system" {
		return reconcile.Result{}, nil
	}

	// TODO - even easier
	// When a namespace is created or updated this controller will fire
	// If the namespace does not have three netpols in it, we will create any missing ones here

	// Security
	// If the three exist, I guess I could compare (cache uid) - but for now I will assume existence is enough
	// When I create the netpols I could stamp them with ownerRef of the namespace - in case someone changes them manually, it would also swing by here
	// It would also be good to "watch" for new netpols and reject any that are not in the designated list - would need to inspect content
	// to ensure it had not changed

	// All pods in namespace
	podSelector := metav1.LabelSelector{}

	if r.findNetworkPolicy("deny-ingress-by-default", networkPolicies, namespace) == false {
		r.createNetworkPolicy("deny-ingress-by-default", namespace, podSelector, nil)
	}

	if r.findNetworkPolicy("deny-ingress-by-default", networkPolicies, namespace) == false {
		allowFromSameNamespace := ingressRuleAllowFromSameNamespace()
		r.createNetworkPolicy("allow-from-same-namespace", namespace, podSelector, &allowFromSameNamespace)
	}

	if r.findNetworkPolicy("allow-from-global-namespaces", networkPolicies, namespace) == false {
		allowFromGlobalNamespaces := ingressRuleAllowFromGlobalNamespaces()
		r.createNetworkPolicy("allow-from-global-namespaces", namespace, podSelector, &allowFromGlobalNamespaces)
	}

	subjectImagePuller := rbac.Subject{
		Kind: "SystemGroup",
		Name: "system:serviceaccounts:" + namespace,
	}
	err = r.createRolebinding("system:image-pullers", namespace, "system:image-puller", &subjectImagePuller)
	if err != nil {
		log.Error(err, "Cannot create RB")
	}

	subjectImageBuilder := rbac.Subject{
		Kind: "ServiceAccount",
		Name: "builder",
	}
	//usernames := "system:serviceaccounts:" + namespace +":builder"
	err = r.createRolebinding("system:image-builders", namespace, "system:image-builder", &subjectImageBuilder)
	if err != nil {
		log.Error(err, "Cannot create RB")
	}

	subjectDeployer := rbac.Subject{
		Kind: "ServiceAccount",
		Name: "deployer",
	}
	//usernames := "system:serviceaccounts:" + namespace +":builder"
	err = r.createRolebinding("system:deployers", namespace, "system:deployer", &subjectDeployer)
	if err != nil {
		log.Error(err, "Cannot create RB")
	}
	//subject4 := rbac.Subject{
	//	Kind:      "User",
	//	Name:      "U",
	//}
	////usernames := "system:serviceaccounts:" + namespace +":builder"
	//err = r.createRolebinding("admin", namespace, "admin", &subject4)
	if err != nil {
		log.Error(err, "Cannot create RB")
	}

	/*
		foundGlobal := false
		if networkPolicies != nil {
			log.Info("Found network policies in namespace " + namespace, "size", len(networkPolicies.Items))
			if global == true {
				for _, np := range networkPolicies.Items {
					log.Info("NP: " + np.Name)
					if np.Name == "netpol-global" {
						log.Info("Found global in namespace " +namespace )
						foundGlobal = true
					} else {
						log.Info("Deleting non-global in namespace " +namespace )
						r.client.Delete(context.TODO(), &v1.NetworkPolicy{
							ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: np.Name},
						})
					}
				}
			}

		}
	*/

	/*
		if global == true && foundGlobal == false {
			// Delete all netpols other than netpol-global

			policyTypes := []v1.PolicyType{"Ingress"}
			ingressRuleAllowFromAllNamespaces := v1.NetworkPolicyIngressRule {
				From: []v1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{},
					},
				},
			}
			ingressRules := []v1.NetworkPolicyIngressRule{ingressRuleAllowFromAllNamespaces}
			// Create a new netpol
			r.client.Create(context.TODO(), &v1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: "netpol-global",},
				Spec:       v1.NetworkPolicySpec{
					PodSelector: metav1.LabelSelector{},
					PolicyTypes: policyTypes,
					Ingress: ingressRules,
				},
			})
		}
	*/

	// Define a new Pod object
	//pod := newPodForCR(instance)

	// Set Watcher instance as the owner and controller
	//if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
	//	return reconcile.Result{}, err
	//}
	//
	//// Check if this Pod already exists
	//found := &corev1.Pod{}
	//err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	//if err != nil && errors.IsNotFound(err) {
	//	reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
	//	err = r.client.Create(context.TODO(), pod)
	//	if err != nil {
	//		return reconcile.Result{}, err
	//	}
	//
	//	// Pod created successfully - don't requeue
	//	return reconcile.Result{}, nil
	//} else if err != nil {
	//	return reconcile.Result{}, err
	//}

	// Pod already exists - don't requeue
	//	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

func (r *ReconcileWatcher) findNetworkPolicy(policyName string, networkPolicies *v1.NetworkPolicyList, namespace string) bool {
	if networkPolicies != nil {
		log.Info("Found network policies in namespace "+namespace, "size", len(networkPolicies.Items))
		for _, np := range networkPolicies.Items {
			if np.Name == policyName {
				return true
			}
		}
	}
	return false
}

func ingressRuleAllowFromGlobalNamespaces() v1.NetworkPolicyIngressRule {
	matchLabels := make(map[string]string)
	matchLabels["namespace-network"] = "global"
	return v1.NetworkPolicyIngressRule{
		From: []v1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{MatchLabels: matchLabels},
			},
		},
	}
}

func ingressRuleAllowFromSameNamespace() v1.NetworkPolicyIngressRule {
	matchLabels := make(map[string]string)
	matchLabels["namespace-network"] = "global"
	return v1.NetworkPolicyIngressRule{
		From: []v1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{},
			},
		},
	}
}

func (r *ReconcileWatcher) createNetworkPolicy(name string, namespace string, podSelector metav1.LabelSelector, ingressRule *v1.NetworkPolicyIngressRule) error {

	ingressRules := []v1.NetworkPolicyIngressRule{}
	if ingressRule != nil {
		ingressRules = append(ingressRules, *ingressRule)
	}

	return r.client.Create(context.TODO(), &v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name},
		Spec: v1.NetworkPolicySpec{
			PodSelector: podSelector,
			Ingress:     ingressRules,
		},
	})
}

func (r *ReconcileWatcher) createRolebinding(name string, namespace string, roleRefName string, subject *rbac.Subject) error {

	subjects := []rbac.Subject{}
	if subject != nil {
		subjects = append(subjects, *subject)
	}

	return r.client.Create(context.TODO(), &rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name},
		RoleRef: rbac.RoleRef{
			Name: roleRefName,
		},
		Subjects: subjects,
	})
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *samplev1alpha1.Watcher) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
