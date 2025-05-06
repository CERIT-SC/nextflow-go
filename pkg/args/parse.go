package args

import (
	"os"
        "nextflow-go/pkg/utils"
        "path/filepath"
	"strings"
)

type Args struct {
	JobName     string
	Nextflow    []string
	Volumes     []string
	HeadImage   string
	HeadCPUs    string
	HeadMemory  string
        ConfigName  string
        ParamsFile  string
        Ttl         int32
}

func ParseArgs() Args {
	args := os.Args[1:]
	jobName := utils.GenerateRandomName()
	hasName := false
        ttl := int32(3600)

	for i, arg := range args {
                if arg == "-help" || arg == "-h" {
                        hasName = true // hack
                        ttl = 10
                        break
                }
		if arg == "-name" && i+1 < len(args) {
			jobName = args[i+1]
			hasName = true
			break
		}
	}

	nextflowArgs := []string{}
	volumesArgs := []string{}
        paramsFile := ""
        configName := "nextflow.config"
        headCPUs := "1"
        headMemory := "8Gi"
        headImage := "cerit.io/nextflow/nextflow:24.10.5"
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			if !hasName {
				nextflowArgs = append(nextflowArgs, "-name", jobName)
				hasName = true
			}
			switch arg {
			case "-v":
				volumesArgs = append(volumesArgs, args[i+1])
				skipNext = true
			case "-head-image", "-pod-image":
				headImage = args[i+1]
				skipNext = true
			case "-head-cpus":
				headCPUs = args[i+1]
				skipNext = true
			case "-head-memory":
				headMemory = args[i+1]
				skipNext = true
			case "-name", "-head-prescript":
				skipNext = true
                        case "-C":
                                configName = args[i+1]
                                skipNext = true
                        case "-params-file":
                                paramsFile = args[i+1]
                                skipNext = true
                                filename := filepath.Base(paramsFile)
                                nextflowArgs = append(nextflowArgs, "-params-file", "/etc/nextflow/"+filename)
			default:
				nextflowArgs = append(nextflowArgs, arg)
			}
		} else if arg != "run" && arg != "kuberun" {
			nextflowArgs = append(nextflowArgs, arg)
		}
	}

	return Args{
		JobName:    jobName,
		Nextflow:   nextflowArgs,
		Volumes:    volumesArgs,
		HeadImage:  headImage,
		HeadCPUs:   headCPUs,
		HeadMemory: headMemory,
                ConfigName: configName,
                ParamsFile: paramsFile,
                Ttl:        ttl,
	}
}

