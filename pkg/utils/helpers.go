package utils

import (
	"encoding/json"
	"fmt"
	"strings"
        "os"
	"regexp"

	"github.com/brianvoe/gofakeit/v7"

        batchv1 "k8s.io/api/batch/v1"
        corev1 "k8s.io/api/core/v1"
)

func BoolPtr(b bool) *bool    { return &b }
func Int64Ptr(i int64) *int64 { return &i }

func GenerateRandomName() string {
	return strings.ToLower(strings.ReplaceAll(fmt.Sprintf("%s-%s", gofakeit.Adjective(), gofakeit.Noun()), " ", "-"))
}

func PrintAsJSON(obj interface{}) {
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal object to JSON: %v", err)
		return
	}
	fmt.Println(string(b))
}

func NormalizeVolumes(args []string, k8sConfig map[string]string) []string {
	var volumes []string

	for i, v := range args {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			continue
		}
		if i == 0 {
			k8sConfig["storageClaimName"] = "'" + parts[0] + "'"
			k8sConfig["storageMountPath"] = "'" + parts[1] + "'"
		} else {
			target := fmt.Sprintf("volumeClaim:'%s', mountPath:'%s'", parts[0], parts[1])

			reDoubleOpen := regexp.MustCompile(`\[\s*\[`)
			reDoubleClose := regexp.MustCompile(`\]\s*\]`)
			reEntrySeparator := regexp.MustCompile(`\]\s*,\s*\[`)

			config := reDoubleOpen.ReplaceAllString(k8sConfig["pod"], "[[")
			config = reDoubleClose.ReplaceAllString(config, "]]" )
			config = reEntrySeparator.ReplaceAllString(config, "],[")

			entries := []string{}
			if strings.TrimSpace(config) != "" {
				trimmed := strings.TrimPrefix(config, "[[")
				trimmed = strings.TrimSuffix(trimmed, "]]")
				entries = strings.Split(trimmed, "],[")
			}

			found := false
			for _, entry := range entries {
				if strings.Contains(entry, fmt.Sprintf("volumeClaim:'%s'", parts[0])) &&
					strings.Contains(entry, fmt.Sprintf("mountPath:'%s'", parts[1])) {
					found = true
					break
				}
			}

			if !found {
				entries = append(entries, target)
			}

			k8sConfig["pod"] = "[[" + strings.Join(entries, "],[") + "]]"
		}
	}

        if Stripped(k8sConfig["storageClaimName"]) != "" && Stripped(k8sConfig["storageMountPath"]) != "" {
 	        volumes = append(volumes, fmt.Sprintf("%s:%s",
		        Stripped(k8sConfig["storageClaimName"]),
		        Stripped(k8sConfig["storageMountPath"])))
        }

	pattern := `(?i)\[\s*volumeClaim\s*:\s*['\"]([^'\"]+)['\"]\s*,\s*mountPath\s*:\s*['\"]([^'\"]+)['\"]\s*\]`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(k8sConfig["pod"], -1)
	for _, match := range matches {
		if len(match) == 3 {
			volumes = append(volumes, fmt.Sprintf("%s:%s", match[1], match[2]))
		}
	}

	return volumes
}

func PrepareFinalConfig(k8sConfig map[string]string, nextflowConfig string) string {
	finalConfig := "k8s {\n"
	for key, value := range k8sConfig {
		finalConfig += fmt.Sprintf("   %s = %s\n", key, value)
	}
	finalConfig += "}\n" + nextflowConfig
	return finalConfig
}

func AttachVolumesToJob(job *batchv1.Job, volumes []string, secretName string) {
	for i, v := range volumes {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid volume format: %s\n", v)
			continue
		}
		volName := fmt.Sprintf("vol-%d", i)
		mount := parts[1]
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: volName,
                        VolumeSource: corev1.VolumeSource{
                           PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
                              ClaimName: parts[0],
                              ReadOnly:  false,
                           },
                        },
		})
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			job.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{Name: volName, MountPath: mount},
		)
	}
	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "nextflow-config",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{SecretName: secretName},
		},
	})
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(
		job.Spec.Template.Spec.Containers[0].VolumeMounts,
		corev1.VolumeMount{Name: "nextflow-config", MountPath: "/etc/nextflow", ReadOnly: true},
	)
}

func Stripped(s string) string {
	return strings.Trim(s, "'\"")
}
