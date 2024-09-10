# kubectl-explore

[![.github/workflows/test.yaml](https://github.com/keisku/kubectl-explore/actions/workflows/test.yaml/badge.svg)](https://github.com/keisku/kubectl-explore/actions/workflows/test.yaml)

![demo](./demo.gif)

## What’s lacking in the original `kubectl-explain`?

- `kubectl explain` requires knowing in advance the resource name/fields.
- `kubectl explain` requires typing the accurate path to the resource name/field, which is a tedious and typo-prone.

## Example Usage

```bash
# Fuzzy-find a resource, then a field to explain
kubectl explore

# Fuzzy-find from all fields of a specific resource.
kubectl explore pod
kubectl explore sts

# Fuzzy-find from fields that a given regex matches.
kubectl explore sts.*Account
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

```bash
# Other available architectures are linux_arm64, darwin_amd64, darwin_arm64, windows_amd64.
export ARCH=linux_amd64
# Check the latest version, https://github.com/keisku/kubectl-explore/releases/latest
export VERSION=<LATEST_VERSION>
wget -O- "https://github.com/keisku/kubectl-explore/releases/download/${VERSION}/kubectl-explore_${VERSION}_${ARCH}.tar.gz" | sudo tar -xzf - -C /usr/local/bin && sudo chmod +x /usr/local/bin/kubectl-explore
```

From source.

```bash
go install github.com/keisku/kubectl-explore@latest
sudo mv $GOPATH/bin/kubectl-explore /usr/local/bin
```

Validate if `kubectl explore` can be executed.
[The Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#using-a-plugin) explains how to use a plugin.

```bash
kubectl explore --help
```
