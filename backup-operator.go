package main

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	filepath "path/filepath"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"os"
	"strings"
	"time"
)

const DEFAULT_BACKUP_RETENTION = 10
const DO_BACKUP_ANNOTATION = "volume-backup-operator/do-backup"
const RETENTION_ANNOTATION = "volume-backup-operator/backup-retention"

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

	pvList, err := clientset.CoreV1().PersistentVolumes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	storageClassesToBackup := "local-path"
	backupTimestampRaw := time.Now().Format(time.RFC3339)
	backupTimestamp := strings.Replace(backupTimestampRaw, ":", "-", -1)
	err = os.Mkdir(fmt.Sprintf("/backup/%s", backupTimestamp), os.ModePerm)

	if err != nil {
		fmt.Printf("Could not create backup dir: %s", err)
		return
	}
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == storageClassesToBackup {
			fmt.Printf("PersistentVolume: %s, Claim: %s/%s, Path: %s\n", pv.Name, pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, pv.Spec.HostPath.Path)

			fmt.Printf("Backing up...\n")
			err = compress(fmt.Sprintf("/host%s", pv.Spec.HostPath.Path), fmt.Sprintf("/backup/%s/%s.tar.gz", backupTimestamp, filepath.Base(pv.Spec.HostPath.Path)))
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return
			}
			fmt.Printf("Finished\n\n")
		}
	}

}
