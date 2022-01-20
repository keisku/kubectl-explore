# kubectl-explore

[![.github/workflows/test.yaml](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml/badge.svg)](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml)

![demo](./demo.gif)

## What

This is a `kubectl` plugin to fuzzy-find the field explanation from supported API resources.

## Motivation

- `kubectl explain` needs knowing in advance the resources name/fields.
- `kubectl explain` needs typing the accurate path to the resource name/field, which is a tedious and typo-prone.

## Usage

```
Fields are identified via a simple JSONPath identifier:
        <type>.<fieldName>[.<fieldName>]

Usage:
  kubectl explore RESOURCE [options] [flags]

Examples:

# Find the field explanation from supported API resources.
kubectl explore

# Find the field explanation from "pod"
kubectl explore pod

# Find the field explanation from "node.spec"
kubectl explore pod.spec.containers

# Find the field explanation from supported API resources in the selected cluster.
kubectl explore --context=onecontext

Flags:
      --api-version string             Get different explanations for particular API version (API group/version)
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "/Users/keisukeumegaki/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
  -h, --help                           help for kubectl
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
      --match-server-version           Require server version to match client version
  -n, --namespace string               If present, the namespace scope for this CLI request
      --password string                Password for basic authentication to the API server
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
      --username string                Username for basic authentication to the API server
```

## Installation

### Krew

Use [krew](https://krew.sigs.k8s.io/) plugin manager to install.
See [the guide](https://krew.sigs.k8s.io/docs/user-guide/setup/install/) to install [krew](https://krew.sigs.k8s.io/).

```bash
kubectl krew install explore
kubectl explore --help
```

### Download the binary

Download the binary from [GitHub Releases](https://github.com/kei6u/kubectl-explore/releases) and drop it in your `$PATH`.

#### Linux

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/v0.4.4/kubectl-explore_v0.4.4_linux_amd64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

#### Darwin(amd64)

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/v0.4.4/kubectl-explore_v0.4.4_darwin_amd64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

#### Darwin(arm64)

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/v0.4.4/kubectl-explore_v0.4.4_darwin_arm64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

#### Source

```shell
go install github.com/kei6u/kubectl-explore@latest
sudo mv $GOPATH/bin/kubectl-explore /usr/local/bin
```

#### Validation

Validate if `kubectl explore` can be executed.
[The Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#using-a-plugin) explains how to use a plugin.

```bash
kubectl explore --help
```
