package main

import (
	"compress/gzip"
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

	"archive/tar"
	"io"
	"os"
	"strings"
	"time"
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

	pvList, err := clientset.CoreV1().PersistentVolumes().List(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	specificStorageClass := "local-path"
	backupTimestamp := time.Now().Format(time.RFC3339)
	os.Mkdir(fmt.Sprintf("/backup/%s", backupTimestamp), os.ModePerm)
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == specificStorageClass {
			fmt.Printf("PersistentVolume: %s, Claim: %s/%s, Path: %s\n", pv.Name, pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, pv.Spec.HostPath.Path)
			if pv.Name == "pvc-01100b40-61f9-4166-a904-c39437696f39" {
				fmt.Printf("Backing up...")
				Tar(fmt.Sprintf("/host%s", pv.Spec.HostPath.Path), fmt.Sprintf("/backup/%s", backupTimestamp))
				fmt.Printf("Finished")
			}
		}

	}
}

func Tar(source, target string) error {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar.gz", filename))
	tarfile, err := os.Create(target)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}
	defer tarfile.Close()

	writer, err := gzip.NewWriterLevel(tarfile, gzip.BestCompression)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}

	tarball := tar.NewWriter(writer)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		fmt.Printf("Source error: %s\n", err)
		return nil
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				fmt.Printf("Error: %s\n", err)
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
			}
			return err
		})
}
