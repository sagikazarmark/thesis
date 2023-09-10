# Create a new EKS cluster manually

Although the project comes with an automated process for launching new clusters,
it roughly mimics the following, manual process.
The two are interchangeable, meaning that a manually launched cluster may be
upgraded and deleted using the automated processes and vice versa.

The contents of this guide are largely based on the [official documentation](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html).

## Preparations

Make sure that the following programs are installed (they are part of the provided Nix development environment):

- curl
- AWS CLI

Required IAM permissions:

- `iam:CreateRole`
- `iam:AttachUserPolicy` or `iam:AttachRolePolicy`

TODO: update the above list

> [!NOTE]
> Some commands require files contained within this repository.
>
> **Make sure to run all commands from the repository root.**

Choose which region do you want to create resources in (the Nix environment comes with the following default):

```shell
export AWS_REGION=eu-west-1
```

Make sure you have a key pair in the selected region (you may set this value in `.env.local`):

```shell
export AWS_KEY_PAIR=my-key
```

Choose a name for your cluster (you may set this value in `.env.local`):

```shell
export CLUSTER_NAME=my-thesis-1
```

## Create an EKS cluster role

Choose a name for your cluster role (the Nix environment comes with the following default):

```shell
export EKS_CLUSTER_ROLE=AmazonEKSClusterRole
```

**The following steps in this section are only necessary the first time you launch a cluster in the same AWS account.**

Create the Amazon EKS cluster IAM role:

```shell
aws iam create-role --role-name $EKS_CLUSTER_ROLE --assume-role-policy-document file://"etc/aws/eks-cluster-role-trust-policy.json"
```

Attach the Amazon EKS managed policy named `AmazonEKSClusterPolicy` to the role:

```shell
aws iam attach-role-policy --policy-arn arn:aws:iam::aws:policy/AmazonEKSClusterPolicy --role-name $EKS_CLUSTER_ROLE
aws iam attach-role-policy --policy-arn arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy --role-name $EKS_CLUSTER_ROLE
```

Save the role ARN for later use:

```shell
export EKS_CLUSTER_ROLE_ARN=$(aws iam get-role --role-name $EKS_CLUSTER_ROLE --query "Role.Arn" --output text)
```

_Read more about the necessary prerequisites for cluster creation [here](https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html#create-cluster-prerequisites)._

## Create a VPC

There are several ways to create a VPC for an EKS cluster.

The easiest is using one of the official CloudFormation templates:

```shell
aws cloudformation create-stack --stack-name $CLUSTER_NAME-vpc --template-body file://etc/aws/eks-vpc-public-subnets.yaml
```

Wait for the CloudFormation stack to become ready:

```shell
aws cloudformation wait stack-create-complete --stack-name $CLUSTER_NAME-vpc
```

Save the following details for later use:

```shell
export EKS_CLUSTER_VPC_ID=$(aws cloudformation describe-stacks --stack-name $CLUSTER_NAME-vpc --query 'Stacks[0].Outputs[?OutputKey==`VpcId`].OutputValue' --output text)
export EKS_CLUSTER_SUBNET_IDS=$(aws cloudformation describe-stacks --stack-name $CLUSTER_NAME-vpc --query 'Stacks[0].Outputs[?OutputKey==`SubnetIds`].OutputValue' --output text)
export EKS_CLUSTER_SECURITY_GROUPS=$(aws cloudformation describe-stacks --stack-name $CLUSTER_NAME-vpc --query 'Stacks[0].Outputs[?OutputKey==`SecurityGroups`].OutputValue' --output text)
```

_Read more about creating a VPC [here](https://docs.aws.amazon.com/eks/latest/userguide/creating-a-vpc.html)._

## Create an EKS cluster

Use the AWS CLI to set up an EKS cluster once all prerequisites are fulfilled:

```shell
aws eks create-cluster --name $CLUSTER_NAME --kubernetes-version 1.27 --role-arn $EKS_CLUSTER_ROLE_ARN --resources-vpc-config subnetIds=$EKS_CLUSTER_SUBNET_IDS,securityGroupIds=$EKS_CLUSTER_SECURITY_GROUPS
```

Wait for the cluster to become active:

```shell
aws eks wait cluster-active --name $CLUSTER_NAME
```

Add the cluster to your kubeconfig:

```shell
aws eks update-kubeconfig --name $CLUSTER_NAME
```

_It may take up to 10 minutes for the cluster to become active. Ideal time for a coffee break. ☕_

## Create a self-managed node group

Although managed node groups offer a number of advantages over self-managed ones,
the process I came up with for cluster creation was invented before managed node groups existed.
Also, the cluster upgrade process in my thesis is a lot more flexible compared to the upgrade process of managed node groups,
hence we create self-managed nodes.

```shell
aws cloudformation create-stack --stack-name $CLUSTER_NAME-ng1 --template-body file://etc/aws/eks-nodegroup.yaml --capabilities CAPABILITY_IAM --parameters ParameterKey=ClusterName,ParameterValue=$CLUSTER_NAME ParameterKey=ClusterControlPlaneSecurityGroup,ParameterValue=$EKS_CLUSTER_SECURITY_GROUPS ParameterKey=NodeGroupName,ParameterValue=$CLUSTER_NAME-ng1 ParameterKey=KeyName,ParameterValue=\"$AWS_KEY_PAIR\" ParameterKey=VpcId,ParameterValue=$EKS_CLUSTER_VPC_ID ParameterKey=Subnets,ParameterValue=\"$EKS_CLUSTER_SUBNET_IDS\"
```

Wait for the CloudFormation stack to become ready:

```shell
aws cloudformation wait stack-create-complete --stack-name $CLUSTER_NAME-ng1
```

Record the node instance role ARN:

```shell
export EKS_NODE_INSTANCE_ROLE_ARN=$(aws cloudformation describe-stacks --stack-name $CLUSTER_NAME-ng1 --query 'Stacks[0].Outputs[?OutputKey==`NodeInstanceRole`].OutputValue' --output text)
```

Apply the `aws-auth` ConfigMap to the cluster:

```shell
sed -e "s|<ARN of instance role (not instance profile)>|$EKS_NODE_INSTANCE_ROLE_ARN|" etc/aws/auth-cm.yaml | kubectl -n kube-system apply -f -
```

Watch the status of your nodes and wait for them to reach the `Ready` status:

```shell
kubectl get nodes --watch
```

_Read more about creating self-managed nodes and other available parameters [here](https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html)._

## Congratulations!

At this point you should have a functioning cluster. 🎉

## Cleanup

To delete cluster and its resources, follow these steps:

Delete the node group CloudFormation stack:

```shell
aws cloudformation delete-stack --stack-name $CLUSTER_NAME-ng1
```

Wait for the CloudFormation stack to be deleted:

```shell
aws cloudformation wait stack-delete-complete --stack-name $CLUSTER_NAME-ng1
```

Delete the EKS cluster:

```shell
aws eks delete-cluster --name $CLUSTER_NAME
```

Wait for the cluster deletion to complete:

```shell
aws eks wait cluster-deleted --name $CLUSTER_NAME
```

Delete the VPC CloudFormation stack:

```shell
aws cloudformation delete-stack --stack-name $CLUSTER_NAME-vpc
```

Wait for the CloudFormation stack to be deleted:

```shell
aws cloudformation wait stack-delete-complete --stack-name $CLUSTER_NAME-vpc
```

> [!NOTE]
> Deleting the VPC may fail if the cluster fails to clean up resources during deletion.
>
> You may have to delete these resources (load balancers, network interfaces, etc)
> before proceeding with VPC deletion.

## References

- https://docs.aws.amazon.com/eks/latest/userguide/create-cluster.html
- https://docs.aws.amazon.com/eks/latest/userguide/what-is-eks.html
- https://docs.aws.amazon.com/eks/latest/userguide/creating-a-vpc.html
- https://docs.aws.amazon.com/eks/latest/userguide/launch-workers.html
- https://docs.aws.amazon.com/eks/latest/userguide/update-stack.html
