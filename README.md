# Nextflow Kubernetes Launcher

`nextflow-go` is a Go-based alternative to the `nextflow kuberun` command, designed to address several limitations of the original `kuberun` driver. These limitations include:

- Incompatibility with `nextflow.config` files that contain functions (e.g., `def check_resource(obj)`).
- Problems handling multiline strings (e.g., `beforeText = """`).
- Inability to pass custom command-line arguments not compiled into the `nextflow` driver.

As a statically compiled Go binary, `nextflow-go` requires only the LibC library from Ubuntu 22.04 or later. A precompiled `x86_64` binary is available in the `bin` folder.

## Key Features

- **Full CLI Argument Support**: Arbitrary command-line arguments are passed directly to `nextflow run` without modification.
- **Minimal Config Changes**: The tool only modifies the `k8s` section of `nextflow.config`, ensuring full compatibility with custom functions, multiline strings, and future additions.
- **No Shared Storage Required**: You can run `nextflow-go` from a local machine without shared storage access. The only requirement is a working Kubernetes configuration (`kubeconfig`) -- refer to the [Cerit-SC Kubernetes documentation](https://docs.cerit.io/en/docs/kubernetes/kubectl).

## Usage

```bash
nextflow-go run [arguments]
```

## Supported Options

The following options are currently supported (mirroring the original `kuberun` functionality). All options are optional:

- `-v pvc:dir`  
  Mounts a Persistent Volume Claim (`pvc`) into the specified `dir` in both the launcher and worker pods. Can be specified multiple times.

- `-head-image`, `-pod-image`  
  Specifies the container image for the driver pod. Defaults to `cerit.io/nextflow/nextflow:24.10.5`.

- `-head-cpus`  
  Sets the CPU limit for the driver pod. Default is `1` (requests are set to half).

- `-head-memory`  
  Sets the memory limit for the driver pod. Default is `8Gi`.

- `-head-prescript`  
  This option is currently ignored.

- `-C`  
  Specifies the main configuration file. Defaults to `nextflow.config` in the current directory.

- `-c`  
  Specifies a custom configuration file.

- `-params-file`  
  Provides an additional parameters file.

- `-name`  
  Sets a custom name for the run. If not provided, a random name will be generated.

## Default Behavior

If the `nextflow.config` does not define the `computeResourceType`, the launcher defaults to using the `Job` compute type.

