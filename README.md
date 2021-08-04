# Running OpenFaas Pro on Linode K8s (feat. Aiven and Pulumi)

Alex Ellis did a great job, when he wrote a tutorial
about [Event-driven OpenFaaS with Managed Kafka from Aiven](https://www.openfaas.com/blog/openfaas-kafka-aiven/).

So this got me hooked. My personal challenge was: **How can I fully automate the deployment of the whole stack.**

To spice up the challenge, I decided to use only Pulumi for this in Go.

## Preparation

Set the API keys for Linode and Aiven and th OpenFaas Pro license via the

```bash
cd 00-infrastructure
pulumi config set linode:token xxx --secret

cd 01-aiven
pulumi config set aiven:apiToken xxx --secret

02-openfaas
pulumi config set openfaas xxx --secret
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

Now you can `pulumi up` every folder and your whole stack gets deployed.

So running a deployment of a larger app with different layers (infra, managed services and app) is becoming more and
more accessible and enables us to work more in a DevOps fashion inside our team.

## Alternative solutions

An alternative solution would be, to use Pulumi for the provisioning of the infrastructure and manged services and
bootstrapping a GitOps engine (like Flux2 or ArgoCD).

Another solution could be to just provision your kubernetes infrastructure and use [Crossplane](https://crossplane.io/)
to provision the "real" infrastructure.
