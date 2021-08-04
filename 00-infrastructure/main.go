package main

import (
	"encoding/base64"
	"github.com/pulumi/pulumi-linode/sdk/v3/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
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

		ctx.Export("kubeconfig", pulumi.ToSecret(kubeconfig))
		return nil
	})
}
