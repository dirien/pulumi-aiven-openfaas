# Running OpenFaas Pro on Linode K8s (feat. Aiven and Pulumi) - Automation- API

Alex Ellis did a great job, when he wrote a tutorial
about [Event-driven OpenFaaS with Managed Kafka from Aiven](https://www.openfaas.com/blog/openfaas-kafka-aiven/).

So this got me hooked. My personal challenge was: **How can I fully automate the deployment of the whole stack.**

To spice up the challenge, I decided to use only Pulumi for this in Go.

## Preparation

Set the API keys for Linode and Aiven and th OpenFaas Pro license via the

```bash
export LINODE_TOKEN=xx
export AIVEN_TOKEN=zzz
export LICENSE=yyy
```

Otherwise, you can't replay the deployment.

## How to

Firs I cut the whole stack into three different Pulumi `Stacks`:

- 00-infrastructure
- 01-aiven
- 02-openfaas

Creating this kind of independent IaC modules, I would consider as good practice. So at the end you can work independent
on a specific stack, without running having this all-in-one single deployment files.

The only dependency we have between the different stacks are the:

[Export](https://www.pulumi.com/docs/intro/concepts/stack/#outputs) of properties, a different stack would need for the
services to provision.

```go
ctx.Export("caCert", pulumi.ToSecret(project.CaCert))
```

On the other side to consume the outputs in a different stack, you just need to get
the [stack reference](https://www.pulumi.com/docs/intro/concepts/stack/#stackreferences).

```go
aiven, err := pulumi.NewStackReference(ctx, "dirien/01-aiven/dev", nil)
if err != ...
caCert := aiven.GetStringOutput(pulumi.String("caCert"))
```

To start via the automation api simply:

```bash
cd pulumi-automation-api
go run . 
```

to destroy the stacks: 

```bash
go run . destroy
```