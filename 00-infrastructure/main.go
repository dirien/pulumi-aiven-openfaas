package main

import (
	"encoding/base64"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/yaml"
	"github.com/pulumi/pulumi-linode/sdk/v3/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
	"path/filepath"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create a linode resource (Linode Instance)
		cluster, err := linode.NewLkeCluster(ctx, "linode-lke", &linode.LkeClusterArgs{
			K8sVersion: pulumi.String("1.21"),
			Region:     pulumi.String("eu-central"),
			Label:      pulumi.String("linode-lke"),
			Pools: linode.LkeClusterPoolArray{
				linode.LkeClusterPoolArgs{
					Type:  pulumi.String("g6-standard-4"),
					Count: pulumi.Int(1),
				},
			},
		})
		if err != nil {
			return err
		}

		kubeconfig := cluster.Kubeconfig.ApplyT(func(v string) string {
			decodedStrAsByteSlice, _ := base64.StdEncoding.DecodeString(v)
			return string(decodedStrAsByteSlice)
		}).(pulumi.StringOutput)

		provider, err := kubernetes.NewProvider(ctx, "linode-lke", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		}, nil)
		if err != nil {
			return err
		}
		_, err = yaml.NewConfigGroup(ctx, "pulumi-operator", &yaml.ConfigGroupArgs{
			Files: []string{filepath.Join("yaml", "*.yaml")},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		c := config.New(ctx, "")
		pulumiAccessToken := c.Require("pulumiAccessToken")
		gitAccessToken := c.Require("gitAccessToken")
		apiToken := c.Require("apiToken")
		openfaas := c.Require("openfaas")

		// Create the API token as a Kubernetes Secret.
		accessToken, err := corev1.NewSecret(ctx, "accesstoken", &corev1.SecretArgs{
			StringData: pulumi.StringMap{"accessToken": pulumi.String(pulumiAccessToken)},
		})
		if err != nil {
			return err
		}

		// Create an NGINX deployment in-cluster.
		_, err = apiextensions.NewCustomResource(ctx, "aiven-openfaas", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("pulumi.com/v1alpha1"),
			Kind:       pulumi.String("Stack"),
			Metadata: metav1.ObjectMetaPtr(
				&metav1.ObjectMetaArgs{
					Name: pulumi.String("aiven-openfaas"),
				}).ToObjectMetaPtrOutput(),
			OtherFields: kubernetes.UntypedArgs{
				"spec": map[string]interface{}{
					"accessTokenSecret": accessToken.Metadata.Name(),
					"stack":             "dirien/aiven-openfaas/dev",
					"projectRepo":       "https://github.com/dirien/pulumi-aiven-openfaas",
					"branch":            "refs/remotes/origin/initial",
					"repoDir":           "01-application",
					"gitAuth": map[string]interface{}{
						"accessToken": map[string]interface{}{
							"type": "Literal",
							"literal": map[string]string{
								"value": gitAccessToken,
							},
						},
					},
					"envRefs": map[string]interface{}{
						"AIVEN_TOKEN": map[string]interface{}{
							"type": "Literal",
							"literal": map[string]string{
								"value": apiToken,
							},
						},
						"OPENFAAS_LICENSE": map[string]interface{}{
							"type": "Literal",
							"literal": map[string]string{
								"value": openfaas,
							},
						},
					},
					"destroyOnFinalize": true,
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{accessToken}))

		// Export the DNS name of the instance
		ctx.Export("kubeconfig", pulumi.ToSecret(kubeconfig))
		return nil
	})
}
