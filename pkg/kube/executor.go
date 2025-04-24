package kube

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

        "nextflow-go/pkg/args"
        "nextflow-go/pkg/config"
        "nextflow-go/pkg/utils"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
)

func Execute(dryRun bool) {
	args := args.ParseArgs()
	k8sConfig, restConfigStr, _ := config.ReadNextflowConfig(args.ConfigName)

	volumes := utils.NormalizeVolumes(args.Volumes, k8sConfig)
	finalConfig := utils.PrepareFinalConfig(k8sConfig, restConfigStr)
	config, err := getKubeConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil && !dryRun {
		panic(err)
	}

	namespace := corev1.NamespaceDefault
        if nsBytes, err := os.ReadFile("/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
                if ns := strings.TrimSpace(string(nsBytes)); ns != "" {
                        namespace = ns
                }
        }
	if ns, ok := k8sConfig["namespace"]; ok {
		namespace = strings.Trim(ns, "'\"")
	}

	launchDir, _ := os.Getwd()
	if dir, ok := k8sConfig["launchDir"]; ok {
		launchDir = strings.Trim(dir, "'\"")
	}

	initScript := fmt.Sprintf("mkdir -p '%s'; cd '%s'; cp /etc/nextflow/nextflow.config .", launchDir, launchDir)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "nf-config-"},
		Type:       corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"init.sh":         []byte(initScript),
			"nextflow.config": []byte(finalConfig),
		},
	}

        secretName := "nf-config-"
        var createdSecret *corev1.Secret
        ctx := context.Background()

        if ! dryRun {
  	        createdSecret, err := clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	        if err != nil {
		        panic(err)
	        }
	        secretName = createdSecret.Name
        } else {
                utils.PrintAsJSON(secret)
        }

	mainCmd := fmt.Sprintf("source /etc/nextflow/init.sh; nextflow run %s", strings.Join(args.Nextflow, " "))
	command := []string{"/bin/bash", "-c", mainCmd}

	resources := prepareResources(args.HeadCPUs, args.HeadMemory)
	envVars := prepareEnvVars()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: args.JobName,
			Labels: map[string]string{
				"app":     "nextflow",
				"runName": args.JobName,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"job-name": args.JobName}},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						FSGroupChangePolicy: func() *corev1.PodFSGroupChangePolicy { p := corev1.PodFSGroupChangePolicy("OnRootMismatch"); return &p }(),
						RunAsNonRoot:         utils.BoolPtr(true),
						SeccompProfile:       &corev1.SeccompProfile{Type: "RuntimeDefault"},
					},
					Containers: []corev1.Container{{
						Name:            args.JobName,
						Image:           args.HeadImage,
						Command:         command,
						Resources:       resources,
						Env:             envVars,
						SecurityContext: &corev1.SecurityContext{RunAsUser: utils.Int64Ptr(1000), AllowPrivilegeEscalation: utils.BoolPtr(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
					}}},
			},
		},
	}

	utils.AttachVolumesToJob(job, volumes, secretName)

        if ! dryRun {
 	        createdJob, err := clientset.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	        if err != nil {
		        panic(err)
	        }
                ownerRef := metav1.OwnerReference{
                        APIVersion:         "batch/v1",
                        Kind:               "Job",
                        Name:               createdJob.Name,
                        UID:                createdJob.UID,
                        Controller:         utils.BoolPtr(true),
                        BlockOwnerDeletion: utils.BoolPtr(true),
                }
                secretPatch := &corev1.Secret{
                        ObjectMeta: metav1.ObjectMeta{
                                Name:            createdSecret.Name,
                                OwnerReferences: []metav1.OwnerReference{ownerRef},
                        },
                }
                _, _ = clientset.CoreV1().Secrets(namespace).Update(ctx, secretPatch, metav1.UpdateOptions{})
        } else {
                utils.PrintAsJSON(job)
        }

	fmt.Printf("Kubernetes Job '%s' created successfully.\n", args.JobName)
}

func getKubeConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	return clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
}

func prepareResources(cpus, memory string) corev1.ResourceRequirements {
	cpusFloat, _ := strconv.ParseFloat(cpus, 64)
	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpus),
			corev1.ResourceMemory: resource.MustParse(memory),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%.1f", cpusFloat/2)),
			corev1.ResourceMemory: resource.MustParse(memory),
		},
	}
}

func prepareEnvVars() []corev1.EnvVar {
	env := os.Environ()
	envVars := []corev1.EnvVar{
		{Name: "NXF_EXECUTOR", Value: "k8s"},
		{Name: "NXF_ANSI_LOG", Value: "false"},
	}
	for _, e := range env {
		if strings.HasPrefix(e, "NXF_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envVars = append(envVars, corev1.EnvVar{Name: parts[0], Value: parts[1]})
			}
		}
	}
	return envVars
}

