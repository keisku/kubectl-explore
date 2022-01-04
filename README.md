# kubectl-explore

[![.github/workflows/test.yaml](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml/badge.svg)](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml)

This is a plugin for `kubectl` to fuzzy-find and explain the field supported API resource.

![demo](./demo.gif)

## What

`kubectl-explore` finds and explains the field associated with each supported API resource.

## Motivation

`kubectl explain` is already helpful, but typing the accurate path to the filed is a tedious and typo-prone.

## Usage

```
This command finds fields associated with each supported API resource to explain.

Fields are identified via a simple JSONPath identifier:
        <type>.<fieldName>[.<fieldName>]

Usage:
  kubectl-explore RESOURCE [options] [flags]

Examples:

# Explore pod fields.
kubectl-explore pod

# Explore "pod.spec.containers" fields.
kubectl-explore pod.spec.containers

# Explore the resource selected by interactive fuzzy-finder.
kubectl-explore


Flags:
      --api-version string   Get different explanations for particular API version (API group/version)
  -h, --help                 help for kubectl-explore
```

## Install

Download the binary from [GitHub Releases](https://github.com/kei6u/kubectl-explore/releases) and drop it in your `$PATH`.

### Linux

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/0.2.0/kubectl-explore_linux_x86_64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

### OSX

```shell
curl -L -o kubectl-explore.tar.gz https://github.com/kei6u/kubectl-explore/releases/download/0.2.0/kubectl-explore_darwin_x86_64.tar.gz
tar -xvf kubectl-explore.tar.gz
sudo mv kubectl-explore /usr/local/bin
```

### Source

```shell
go install github.com/kei6u/kubectl-explore@latest
sudo mv $GOPATH/bin/kubectl-explore /usr/local/bin
```

### Validation

Validate if `kubectl explore` can be executed.
[The Kubernetes documentation](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/#using-a-plugin) explains how to use a plugin.

```bash
kubectl explore --help
```
