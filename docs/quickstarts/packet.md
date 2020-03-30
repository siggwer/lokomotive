# Lokomotive Packet quickstart guide

## Contents

* [Introduction](#introduction)
* [Requirements](#requirements)
* [Step 1: Install lokoctl](#step-1-install-lokoctl)
* [Step 2: Set up a working directory](#step-2-set-up-a-working-directory)
* [Step 3: Set up credentials from environment variables](#step-3-set-up-credentials-from-environment-variables)
* [Step 4: Define cluster configuration](#step-4-define-cluster-configuration)
* [Step 5: Create Lokomotive cluster](#step-5-create-lokomotive-cluster)
* [Verification](#verification)
* [Cleanup](#cleanup)
* [Troubleshooting](#troubleshooting)
* [Conclusion](#conclusion)
* [Next steps](#next-steps)

## Introduction

This quickstart guide walks through the steps needed to create a Lokomotive cluster on Packet with
Flatcar Container Linux using Route53 as the DNS provider.

By the end of this guide, you'll have a production-ready Kubernetes cluster running on Packet.

## Requirements

* Basic understanding of Kubernetes concepts.
* Packet account, Project ID and auth token (sometimes also referred to as [User Level API
  key](https://www.packet.com/developers/docs/API/getting-started/)).
* AWS account and IAM credentials (optional for Route53 DNS configuration).
* AWS Route53 DNS Zone (registered Domain Name or delegated subdomain).
* Terraform v0.12.x and [terraform-provider-ct](https://github.com/poseidon/terraform-provider-ct)
  installed locally.
* Local BGP enabled. More information on how to enable Local BGP for the Packet Project is found in
  the [Packet support document](https://support.packet.com/kb/articles/bgp).
* An SSH key pair for management access.
* `kubectl` installed locally to access the Kubernetes cluster.

## Steps

### Step 1: Install lokoctl

lokoctl is a command-line interface for Lokomotive.

To install `lokoctl`, follow the instructions in the [lokoctl installation](../installer/lokoctl.md)
guide.

### Step 2: Set up a working directory

It's better to start fresh in a new working directory, as the state of the cluster is stored in this
directory.

This also makes the cleanup task easier.

```console
mkdir -p lokomotive-infra/mycluster
cd lokomotive-infra/mycluster
```

### Step 3: Set up credentials from environment variables

#### Packet

* Log in to your Packet account and obtain the Project ID from the `Project Settings` tab.
* Obtain an API key from the `User Settings` menu.
* Set the environment variable `PACKET_AUTH_TOKEN` with the API key.

```console
export PACKET_AUTH_TOKEN=<PACKET_API_KEY>
```
#### AWS

Lokomotive requires AWS credentials for configuring Route53 DNS. To manually configure DNS entries
refer the DNS configuration settings for Packet(Add link).

```console
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### Step 4: Define cluster configuration

To create a Lokomotive cluster, we need to define a configuration.

A [production-ready configuration](../../examples/packet-production) is already provided for ease of
use. Copy the example configuration to the working directory and modify accordingly.

The provided configuration installs the Lokomotive cluster and the following components:

* [metrics-server](../configuration-reference/components/metrics-server.md)
* [openebs-operator](../configuration-reference/components/openebs-operator.md)
* [contour](../configuration-reference/components/contour.md)
* [metallb](../configuration-reference/components/metallb.md)
* [cert-manager](../configuration-reference/components/cert-manager.md)
* [flatcar-linux-update-operator](../configuration-reference/components/flatcar-linux-upate-operator.md)
* [openebs-storage-class](../configuration-reference/components/openebs-storage-class.md)
* [prometheus-operator](../configuration-reference/components/prometheus-operator.md)

You can configure the components as per your requirements.

Create a variables file named `lokocfg.vars` in working directory to set values for variables
defined in the configuration file.

```console
#lokocfg.vars

packet_project_id = "PACKET_PROJECT_ID"
ssh_public_keys = ["public-ssh-key-1", "public-ssh-key-2", ...]

state_s3_bucket = "name-of-the-s3-bucket-to-store-the-cluster-state"
lock_dynamodb_table = "name-of-the-dynamodb-table-for-state-locking"

dns_zone = "dns-zone-name"
route53_zone_id = "zone-id-of-the-dns-zone"

management_cidrs = "public-ip-address-cidr-to-access-the-cluster"
node_private_cidr = "private-subnet-assigned-by-packet-to-the-project"

cert_manager_email = "email-address-used-for-cert-manager-component"
grafana_admin_password = "password-for-grafana"
```

**NOTE**: You can separate component configurations from cluster configuration in separate
configuration files if doing so fits your needs.

Example:
```console
$ ls lokomotive-infra/mycluster
cluster.lokocfg  metallb.lokocfg  cert-manager.lokocfg lokocfg.vars
```

For advanced cluster configurations and more information refer to the [Packet configuration
guide](../configuration-reference/platforms/packet.md).

### Step 5: Create Lokomotive cluster

Run the following command to create the cluster:

```console
lokoctl cluster apply
```

Once the command finishes, your Lokomotive cluster details are stored in the path you've specified
under `asset_dir`.

## Verification

A successful installation results in the output:

```console
module.packet-mycluster.null_resource.bootkube-start: Still creating... [4m10s elapsed]
module.packet-mycluster.null_resource.bootkube-start: Still creating... [4m20s elapsed]
module.packet-mycluster.null_resource.bootkube-start: Creation complete after 4m25s [id=1122239320434737682]

Apply complete! Resources: 74 added, 0 changed, 0 destroyed.

Your configurations are stored in /home/imran/lokoctl-assets/mycluster

Now checking health and readiness of the cluster nodes ...

Node                                          Ready    Reason          Message

mycluster-controller-0                        True     KubeletReady    kubelet is posting ready status
mycluster-controller-1                        True     KubeletReady    kubelet is posting ready status
mycluster-controller-2                        True     KubeletReady    kubelet is posting ready status
mycluster-worker-pool-1-worker-0              True     KubeletReady    kubelet is posting ready status
mycluster-worker-pool-1-worker-1              True     KubeletReady    kubelet is posting ready status
mycluster-worker-pool-1-worker-2              True     KubeletReady    kubelet is posting ready status

Success - cluster is healthy and nodes are ready!
```

Use the generated `kubeconfig` file to access the Kubernetes cluster and list nodes.

```console
export KUBECONFIG=./lokomotive-assets/cluster-assets/auth/kubeconfig
kubectl get nodes
```

## Using the cluster

At this point you have access to the Kubernetes cluster and can use it!
If you don't have Kubernetes experience you can check out the [Kubernetes
Basics official
documentation](https://kubernetes.io/docs/tutorials/kubernetes-basics/deploy-app/deploy-intro/)
to learn about its usage.

**Note**: Lokomotive sets up a pretty restrictive Pod Security Policy that
disallows running containers as root by default, check the [Pod Security Policy
documentation](../concepts/securing-lokomotive-cluster.md#cluster-wide-pod-security-policy)
for more details.

## Cleanup

To destroy the Lokomotive cluster, execute the following command:

```console
lokoctl cluster destroy --confirm
```

You can safely delete the working directory created for this quickstart guide if you no longer
require it.

## Troubleshooting

### Stuck at copy controller secrets

If there is an execution error or no progress beyond the output provided below:

```console
...
module.packet-mycluster.null_resource.copy-controller-secrets: Still creating... (8m30s elapsed)
module.packet-mycluster.null_resource.copy-controller-secrets: Still creating... (8m40s elapsed)
...
```

The error probably happens because the `ssh_pubkeys` provided in the configuration is missing in the
`ssh-agent`.

To rectify the error, you need to:

1. Follow the steps [to add the SSH key to the
   ssh-agent](https://help.github.com/en/github/authenticating-to-github/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent#adding-your-ssh-key-to-the-ssh-agent).
2. Retry [Step 5](#step-5-create-lokomotive-cluster).

### Packet provisioning failed

For failed machine provisioning on Packet end, retry [Step 5](#step-5-create-lokomotive-cluster).

### Insufficient availability of nodes types on Packet

In the event of failed Packet provisioning due to machines of type `controller_type` or
`workers_type` not available.  You can check the Packet API [capacity
endpoint](https://www.packet.com/developers/api/capacity/) to get the current capacity and decide on
changing the facility or the machine type.

### Permission issues

  * If the failure is due to insufficient permissions on Packet, check the permission on the Packet
    console.
  * This generally happens if user is using `Project Level API Key` and not `User Level API Key`.

### Failed installation of components that require disk storage

For components that require disk storage such as [Openebs storage
class](../configuration-reference/components/openebs-storage-class.md), [Prometheus
Operator](../configuration-reference/components/prometheus-operator.md) machine types with spare disks
should be used.

## Conclusion

After walking through this guide, you've learned how to set up a Lokomotive cluster on Packet.

## Next steps

You can now start deploying your workloads on the cluster.

For more information on installing supported Lokomotive components, you can visit the [component
configuration references](../configuration-reference/components).
