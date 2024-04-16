# eBPF Container Auditor

Container-Level Auditing in Container - Orchestrators with eBPF

We propose an eBPF-based solution that enhances transparency with respect to operations performed within containers. Overall, this study suggests that the use of eBPF for container-level auditing can provide valuable insights into container behavior and improve the security of containerized applications.

We address the challenges associated with auditing container behavior and highlight the advantages of leveraging eBPF to monitor container activities at the kernel level.


## Description
In this implementation, we utilized an eBPF program to capture the commands executed within the Bash shell interpreter. This was achieved by instrumenting the readline function, which is used by Bash to read user-provided commands. Our focus on Bash stems from its status as the default shell in most contemporary Linux distributions.

The eBPF program we employed monitors commands executed by any user on the system. While this may pose challenges in systems with multiple users, it becomes advantageous within the context of Linux containers. Containers have a limited perception of the system, and the program running inside a container perceives the container itself as the entire system.

When the eBPF program is running within a container, it captures all commands executed in any Bash process and communicates this information back to the service on the host machine, using a Unix Domain Socket. The service then creates the corresponding Events resources in the Kubernetes API server, allowing cluster administrators to gain a comprehensive view of the actions executed within the containers.

## Architecture

The implementation consists of the following components:

1. **eBPF Program**: Responsible for capturing the commands executed in the Bash shell interpreter.
2. **runc Wrapper**: Detects the PID of the Bash process within each container and sends this information to the service.
3. **Worker Node Service**: Receives the Bash process PIDs, runs the eBPF program in the correct namespace using `nsenter`, and sends the captured information to the Kubernetes API server.

## Overview

Developed by Fábio Junior Bertinatto, Daniel Arioza, Jéferson Campos Nobre, and Lisandro Zambenedetti Granville from the Instituto de Informática - Universidade Federal do Rio Grande do Sul.

## Features

- **Real-time Auditing**: Captures every command executed within the Bash shell of a container.
- **Enhanced Security**: Monitors and logs activities for security analysis and compliance.
- **Low Overhead**: Optimized to minimize CPU and memory usage, ensuring minimal impact on the system.
- **Easy Integration**: Seamlessly integrates with Kubernetes using a custom runc wrapper and RuntimeClass for Kubernetes deployments.

## Getting Started

### Prerequisites

- Kubernetes cluster (Version 1.26 or higher recommended)
- Containerd configured with our custom runc wrapper

### Installation

1. **Configure Containerd**: Update the `/etc/containerd/config.toml` file to use the custom wrapper:

    ```toml
    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.wrapper]
    runtime_type = "io.containerd.runc.v1"
    pod_annotations = ["*"]
    container_annotations = ["*"]

    [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.wrapper.options]
    BinaryName="/usr/bin/wrapper"
    ```

2. **Create RuntimeClass in Kubernetes**:

    Use the following YAML to create a `RuntimeClass` resource:

    ```yaml
    apiVersion: node.k8s.io/v1
    kind: RuntimeClass
    metadata:
      name: my-wrapper-name
    handler: wrapper
    ```

3. **Deploy your containerized application**:

    Specify the `runtimeClassName: wrapper` in your Deployment to enable monitoring:

    ```yaml
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: nginx-deployment
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          runtimeClassName: wrapper
          containers:
          - name: nginx
            image: nginx:latest
            volumeMounts:
            - mountPath: /ebpf
              name: ebpf-program-mount-point
          volumes:
          - name: ebpf-program-mount-point
            hostPath:
              path: /path-on-the-host
    ```

## Usage

Once deployed, the eBPF Container Auditor will automatically start monitoring any Bash shells initiated within the container. All commands executed will be logged and can be viewed via Kubernetes Events.

## Contributing

Contributions are welcome! Feel free to fork the repository and submit pull requests.

## License

This project is licensed under the Apache License, Version 2.0 - see the [LICENSE.md](LICENSE.md) file for details.

## Acknowledgements

This project was supported by The São Paulo Research Foundation (FAPESP), grant number 2020/05152-7, under the PROFISSA project.

