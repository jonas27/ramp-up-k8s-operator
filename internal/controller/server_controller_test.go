package controller

import (
	"context"
	"reflect"

	rampupv1alpha1 "github.com/jonas27/ramp-up-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("CronJob controller", func() {

	const (
		name      = "test-cronjob"
		namespace = "default"
	)

	Context("When updating Server Status", func() {
		It("Should create server", func() {
			By("")
			ctx := context.Background()
			selector := make(map[string]string)
			selector["test-label"] = "test"
			server := &rampupv1alpha1.Server{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "ramp-up.joe.ionos.io/v1alpha1",
					Kind:       "Server",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: rampupv1alpha1.ServerSpec{Name: "test", Image: "nginx:1.25", ServicePort: 9090, Selector: selector},
			}
			Expect(k8sClient.Create(ctx, server)).Should(Succeed())

			serverLookupKey := types.NamespacedName{Name: name, Namespace: namespace}
			createdServer := &rampupv1alpha1.Server{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, serverLookupKey, createdServer)
				return err == nil
			}).Should(BeTrue())

			By("By creating a new Pod")
			testPod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "test-container",
							Image: "test-image",
						},
					},
				},
			}
			kind := reflect.TypeOf(v1.Pod{}).Name()
			gvk := rampupv1alpha1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(createdServer, gvk)
			testPod.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testPod)).Should(Succeed())

		})
	})
})
