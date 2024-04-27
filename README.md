# kubectl-explore

[![.github/workflows/test.yaml](https://github.com/keisku/kubectl-explore/actions/workflows/test.yaml/badge.svg)](https://github.com/keisku/kubectl-explore/actions/workflows/test.yaml)

Fuzzy-find the field to explain from all API resources.

![demo](./demo.gif)

### See also

- [kubectl explore, a better kubectl explain](https://keisku.medium.com/kubectl-explore-a-better-kubectl-explain-46a939fafe3a)
- [kubectl explain にあいまい検索(fuzzy-find)機能を追加したプラグイン kubectl-explore を作った](https://zenn.dev/kskumgk63/articles/d52be6c4a31bbb)

## Motivation

- `kubectl explain` needs knowing in advance the resources name/fields.
- `kubectl explain` needs typing the accurate path to the resource name/field, which is a tedious and typo-prone.

## Usage

```
Fuzzy-find the field to explain from all API resources.

Usage:
  kubectl explore [resource|regex] [flags]

Examples:

# Fuzzy-find the field to explain from all API resources.
kubectl explore

# Fuzzy-find the field to explain from pod.
kubectl explore pod

# Fuzzy-find the field to explain from fields matching the regex.
kubectl explore pod.*node
kubectl explore spec.*containers
kubectl explore lifecycle
kubectl explore sts.*Account

# Fuzzy-find the field to explain from all API resources in the selected cluster.
kubectl explore --context=onecontext


Flags:
      --api-version string             Get different explanations for particular API version (API group/version)
      --as string                      Username to impersonate for the operation. User could be a regular user or a service account in a namespace.
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --as-uid string                  UID to impersonate for the operation.
      --cache-dir string               Default cache directory (default "/root/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --disable-compression            If true, opt-out of response compression for all requests to the server
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

### Nix

The plugin is available in [nixpkgs](https://search.nixos.org/packages?query=kubectl-explore):

```bash
nix-env -iA nixpkgs.kubectl-explore
```

### Download the binary

Download the binary from [GitHub Releases](https://github.com/keisku/kubectl-explore/releases) and drop it in your `$PATH`.

```shell
# Other available architectures are linux_arm64, darwin_amd64, darwin_arm64, windows_amd64.
export ARCH=linux_amd64
export VERSION=v0.8.3
wget -O- "https://github.com/keisku/kubectl-explore/releases/download/${VERSION}/kubectl-explore_${VERSION}_${ARCH}.tar.gz" | sudo tar -xzf - -C /usr/local/bin && sudo chmod +x /usr/local/bin/kubectl-explore
```

From source.

```shell
go install github.com/keisku/kubectl-explore@latest
sudo mv $GOPATH/bin/kubectl-explore /usr/local/bin
```

Validate if `kubectl explore` can be executed.
[The Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#using-a-plugin) explains how to use a plugin.

```bash
kubectl explore --help
```
