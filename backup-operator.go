package main

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		// get pods in all the namespaces by omitting namespace
		// Or specify namespace to get pods in particular namespace

		pvList, err := clientset.CoreV1().PersistentVolumes().List(context.Background(), v1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}

		specificStorageClass := "local-path"
		for _, pv := range pvList.Items {
			if pv.Spec.StorageClassName == specificStorageClass {
				fmt.Printf("PersistentVolume: %s, Claim: %s/%s\n", pv.Name, pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
			}
		}

	}
}
