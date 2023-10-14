# Upgrading HA Kubernetes clusters

This repository contains the source code and related resources for my thesis
titled as **"Upgrading HA (highly available) Kubernetes clusters"**.

The project builds upon earlier work I did for [Banzai Cloud](https://banzaicloud.com).
Specifically, I have applied my knowledge about the topic, which I accumulated during the research I did for that project,
and some of the final processes explained [here](https://banzaicloud.com/blog/kubernetes-nodepool-upgrades/) ([Web Archive link](https://web.archive.org/web/20221227164536/https://banzaicloud.com/blog/kubernetes-nodepool-upgrades/)).

## Tech stack

The project is built using the [Go programming language](https://go.dev/) and relies heavily on [Temporal](https://temporal.io/) for resilient workflow execution.

For brevity, the project only supports Kubernetes clusters running on AWS (more specifically, [EKS](https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html) clusters),
but a significant part of the upgrade flow is vendor agnostic, so adding support for other cloud providers is relatively easy.

## Architecture

TBD

## Setup

**For an optimal developer experience, it is recommended to install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).**

<details><summary><i>Installing Nix and direnv</i></summary><br>

**Note: These are instructions that _SHOULD_ work in most cases. Consult the links above for the official instructions for your OS.**

Install Nix:

```sh
sh <(curl -L https://nixos.org/nix/install) --daemon
```

Consult the [installation instructions](https://direnv.net/docs/installation.html) to install direnv using your package manager.

On MacOS:

```sh
brew install direnv
```

Install from binary builds:

```sh
curl -sfL https://direnv.net/install.sh | bash
```

The last step is to configure your shell to use direnv. For example for bash, add the following lines at the end of your `~/.bashrc`:

    eval "\$(direnv hook bash)"

**Then restart the shell.**

For other shells, see [https://direnv.net/docs/hook.html](https://direnv.net/docs/hook.html).

**MacOS specific instructions**

Nix may stop working after a MacOS upgrade. If it does, follow [these instructions](https://github.com/NixOS/nix/issues/3616#issuecomment-662858874).

<hr>
</details>

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

## Usage

The primary purpose of the project is to demonstrate Kubernetes cluster upgrades on EKS.

That however requires a running EKS cluster.

Creating one manually is possible by following [this](./docs/create-eks-cluster.md) document.

It's much easier to use the provided Temporal workflow:

```console
$ tctl wf start --tq thesis --wt "CreateCluster" --if examples/cluster.json
```

> [!NOTE]
> Make sure the global account prerequisites described [here](./docs/create-eks-cluster.md) are met.

Similarly, you can also delete a cluster using the following command:

```console
$ tctl wf start --tq thesis --wt "DeleteCluster" --if examples/cluster.json
```
