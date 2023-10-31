# Update an EKS cluster manually

This document builds upon the [Create an EKS cluster manually](docs/create-eks-cluster.md) guide.

## Update node group CloudFormation stack

First, update the CloudFormation stack for the node group:

```shell
aws cloudformation update-stack --stack-name $CLUSTER_NAME-ng-1 --template-body file://etc/aws/eks-nodegroup.yaml --capabilities CAPABILITY_IAM --parameters ParameterKey=ClusterName,UsePreviousValue=true ParameterKey=ClusterControlPlaneSecurityGroup,UsePreviousValue=true ParameterKey=NodeGroupName,UsePreviousValue=true ParameterKey=KeyName,UsePreviousValue=true ParameterKey=VpcId,UsePreviousValue=true ParameterKey=Subnets,UsePreviousValue=true ParameterKey=NodeImageIdSSMParam,ParameterValue=/aws/service/eks/optimized-ami/1.27/amazon-linux-2/recommended/image_id
```

This will cause new nodes to launch with the new Kubernetes version.

Wait for the CloudFormation stack to become ready:

```shell
aws cloudformation wait stack-update-complete --stack-name $CLUSTER_NAME-ng-1
```

## Rotate instances in node group

In order to update each node in the node group, we need to rotate the instances one by one.

Repeat the following steps for each node.

List the available nodes:

```shell
kubectl get nodes
```

Pick the first node and drain it:

```shell
kubectl drain --ignore-daemonsets NODE_NAME
```

> [!NOTE]
> Although it shouldn't be the case in this example, if draining the node fails, you can try the following command instead to force it:
>
> ```shell
> kubectl drain --ignore-daemonsets --delete-emptydir-data --disable-eviction --force NODE_NAME
> ```

Figure out the instance ID of the node:

```shell
kubectl get node -o jsonpath='{.spec.providerID}' NODE_NAME
```

_The provider ID is in the format `aws:///REGION/INSTANCE_ID`._

Next, delete the node from Kubernetes:

```shell
kubectl delete node NODE_NAME
```

> [!NOTE]
> This is not strictly necessary, but some providers may reuse the node name which may cause issues when it tries to rejoin.

Verify that the node is gone:

```shell
kubectl get nodes
```

Grab the autoscaling group name from the node:
```shell
export ASG_NAME=$(aws autoscaling describe-auto-scaling-instances --instance-ids INSTANCE_ID --query "AutoScalingInstances[0].AutoScalingGroupName" --output text)
```

Detach the instance from the autoscaling group:

```shell
aws autoscaling detach-instances --instance-ids INSTANCE_ID --auto-scaling-group-name $ASG_NAME --no-should-decrement-desired-capacity
```

Detaching the instance first leads to the ASG bringing back a new node immediately, without having to wait for the old node to terminate.

Terminate the instance:

```shell
aws ec2 terminate-instances --instance-ids INSTANCE_ID
```

Wait for the instance to terminate:

```shell
aws ec2 wait instance-terminated --instance-ids INSTANCE_ID
```

Wait for the new node to join:

```shell
kubectl get nodes -w
```

Repeat the steps for each node in the node group.
