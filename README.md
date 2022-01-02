# kubectl-explore

[![Go Reference](https://pkg.go.dev/badge/github.com/kei6u/kubectl-explore.svg)](https://pkg.go.dev/github.com/kei6u/kubectl-explore)
[![.github/workflows/test.yaml](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml/badge.svg)](https://github.com/kei6u/kubectl-explore/actions/workflows/go_test.yaml)

This command is a better `kubectl explain` with the fuzzy-finder.

## What

`kubectl-explore` finds fields associated with each supported API resource to explain.

## Motivation

`kubectl explain` is already helpful, but typing the accurate path to the filed to explain is a tedious and typo-prone.
