# Upgrading HA Kubernetes clusters

This repository contains the source code and related resources for my thesis
titled as **"Upgrading HA (highly available) Kubernetes clusters"**.

The project builds upon earlier work I did for [Banzai Cloud](https://banzaicloud.com).
Specifically, I have applied my knowledge about the topic, which I accumulated during the research I did for that project,
and some of the final processes explained [here](https://banzaicloud.com/blog/kubernetes-nodepool-upgrades/) ([Web Archive link](https://web.archive.org/web/20221227164536/https://banzaicloud.com/blog/kubernetes-nodepool-upgrades/)).

## Tech stack

The project is built using the [Go programming language](https://go.dev/) and relies heavily on [Temporal](https://temporal.io/) for resilient workflow execution.

For brevity, the project only supports Kubernetes clusters running on AWS, but a significant part of the upgrade flow is vendor agnostic,
so adding support for other cloud providers is relatively easy.

## Architecture

TBD

## Setup

For the best developer experience, install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).

Start the Temporal server:

```shell
docker-compose up -d
```

Check if Temporal is accessible:

```shell
tctl namespace list
```

Start the worker (in a separate shell):

```shell
task run --watch
```
