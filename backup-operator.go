package main

import (
	"compress/gzip"
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	filepath "path/filepath"
	"syscall"

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
	backupTimestampRaw := time.Now().Format(time.RFC3339)
	backupTimestamp := strings.Replace(backupTimestampRaw, ":", "-", -1)
	os.Mkdir(fmt.Sprintf("/backup/%s", backupTimestamp), os.ModePerm)
	for _, pv := range pvList.Items {
		if pv.Spec.StorageClassName == specificStorageClass {
			fmt.Printf("PersistentVolume: %s, Claim: %s/%s, Path: %s\n", pv.Name, pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name, pv.Spec.HostPath.Path)
			fmt.Printf("Backing up...")
			//Tar(fmt.Sprintf("/host%s", pv.Spec.HostPath.Path), fmt.Sprintf("/backup/%s", backupTimestamp))
			//err := CopyDirectory(fmt.Sprintf("/host%s", pv.Spec.HostPath.Path), fmt.Sprintf("/backup/%s/%s", backupTimestamp, pv.Name))
			err := compress(fmt.Sprintf("/host%s", pv.Spec.HostPath.Path), fmt.Sprintf("/backup/%s/%s.tar.gz", backupTimestamp, filepath.Base(pv.Spec.HostPath.Path)))
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				return
			}
			fmt.Printf("Finished")

		}

	}
}

func CopyDirectory(scrDir, dest string) error {
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := CopyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := CopySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err := Copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func Copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	defer in.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
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
