package utils

import (
	"encoding/json"
	"fmt"
        "math/rand"
	"regexp"
	"strings"
        "time"
        
        batchv1 "k8s.io/api/batch/v1"
        corev1 "k8s.io/api/core/v1"
)

func BoolPtr(b bool) *bool    { return &b }
func Int64Ptr(i int64) *int64 { return &i }
func Int32Ptr(i int32) *int32 { return &i }

var wordPairs = []string{
    "basta-fidli",
    "cik-cak",
    "cimpr-campr",
    "cinky-linky",
    "coro-moro",
    "cuchty-cvachty",
    "cury-mury",
    "dumky-zalky",
    "elce-pelce",
    "emo-nono",
    "enyky-benyky",
    "fifty-fifty",
    "fuj-tajbl",
    "hajen-dadej",
    "hala-bala",
    "haya-paya",
    "himl-herdek",
    "hogo-fogo",
    "klarity-klap",
    "klido-pibo",
    "krinda-pana",
    "krindy-pindy",
    "kruci-pisek",
    "kurnik-sopa",
    "kyho-slaka",
    "kyho-vyra",
    "lary-fary",
    "lazo-plazo",
    "matla-patla",
    "morces-hadry",
    "myrnyx-tyrnyx",
    "nota-bene",
    "odsad-pocad",
    "roco-fuco",
    "ruty-suty",
    "safra-porte",
    "sakum-prdum",
    "saky-paky",
    "salto-mortale",
    "sec-mazec",
    "sodoma-gomora",
    "starou-belu",
    "suma-sumarum",
    "spibl-nygl",
    "srummy-srumaida",
    "tando-pede",
    "suby-duby",
    "supito-presto",
    "tagu-figu",
    "techtle-mechtle",
    "tingl-tangl",
    "tip-top",
    "tfun-tajxl",
    "trnky-brnky",
    "tresky-plesky",
    "tudle-nudle",
    "tufu-nunu",
    "tuty-fruty",
    "bam-bam",
    "bim-bam",
    "bum-cink",
    "caky-laky",
    "cary-mary",
    "ciry-miry",
    "dingo-bingo",
    "drapy-hrapy",
    "fiky-miky",
    "frk-brk",
    "gulo-bulo",
    "haldy-baldy",
    "hity-pity",
    "hup-cup",
    "jupi-lupi",
    "kity-mity",
    "kupy-lupy",
    "lity-bity",
    "lup-dup",
    "miky-liky",
    "nany-fany",
    "pity-mity",
    "puf-huf",
    "ram-pam",
    "rara-bara",
    "sity-mity",
    "srum-cum",
    "tiki-miki",
    "trala-lala",
    "zigi-zagi",
}

func GenerateRandomName() string {
        rand.Seed(time.Now().UnixNano())
        randomIndex := rand.Intn(len(wordPairs))
        const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
        b := make([]byte, 6)
        for i := range b {
                b[i] = charset[rand.Intn(len(charset))]
        }
        return wordPairs[randomIndex] + "-" + string(b)
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
                        newVolume := fmt.Sprintf("%s:%s", match[1], match[2])
                        var exists bool
                        for _, v := range volumes {
                                if v == newVolume {
                                        exists = true
                                        break
                                }
                        }
                        if !exists {
                                volumes = append(volumes, newVolume)
                        }
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
        mountPathMap := make(map[string]bool)
	for i, v := range volumes {
		parts := strings.Split(v, ":")
		if len(parts) != 2 {
			err := fmt.Sprintf("Invalid volume format: %s\n", v)
                        panic(err)
		}
		volName := fmt.Sprintf("vol-%d", i)
		mount := parts[1]
                if _, exists := mountPathMap[mount]; exists {
                        err := fmt.Sprintf("Duplicate mount path detected: %s\n", mount)
                        panic(err)
                }
                mountPathMap[mount] = true
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
