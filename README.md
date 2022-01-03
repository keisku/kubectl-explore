# kubectl-explore

[![.github/workflows/test.yaml](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml/badge.svg)](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml)

This is a plugin for `kubectl` to fuzzy-find and explain the field supported API resource.

## What

`kubectl-explore` finds and explains the field associated with each supported API resource.

## Motivation

`kubectl explain` is already helpful, but typing the accurate path to the filed is a tedious and typo-prone.

## Install

Download the binary from [GitHub Releases](https://github.com/kei6u/kubectl-explore/releases) and drop it in your `$PATH`.

### Linux

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/0.1.0/kubectl-explore_linux_x86_64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

### OSX

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/0.1.0/kubectl-explore_darwin_x86_64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

### Source

```shell
go install github.com/kei6u/kubectl-explore
sudo mv $GOPATH/bin/kubectl-explore /usr/local/bin
```

### Validation

Validate if `kubectl explore` can be executed.
[The Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#using-a-plugin) explains how to use a plugin.

```shell
kubectl explore --help
```
