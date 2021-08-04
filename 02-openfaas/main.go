package main

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		conf := config.New(ctx, "")

		infra, err := pulumi.NewStackReference(ctx, "dirien/00-infrastructure/dev", nil)
		if err != nil {
			return err
		}
		aiven, err := pulumi.NewStackReference(ctx, "dirien/01-aiven/dev", nil)
		if err != nil {
			return err
		}
		caCert := aiven.GetStringOutput(pulumi.String("caCert"))
		serviceUri := aiven.GetStringOutput(pulumi.String("serviceUri"))
		topicName := aiven.GetStringOutput(pulumi.String("topicName"))
		accessCert := aiven.GetStringOutput(pulumi.String("accessCert"))
		accessKey := aiven.GetStringOutput(pulumi.String("accessKey"))

		kubeconfig := infra.GetStringOutput(pulumi.String("kubeconfig"))

		provider, err := kubernetes.NewProvider(ctx, "linode-lke", &kubernetes.ProviderArgs{
			Kubeconfig: kubeconfig,
		}, nil)
		if err != nil {
			return err
		}

		namespace, err := v1.NewNamespace(ctx, "openfaas", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("openfaas"),
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		_, err = v1.NewNamespace(ctx, "openfaas-fn", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("openfaas-fn"),
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		_, err = helm.NewChart(ctx, "openfaas", helm.ChartArgs{
			Chart:     pulumi.String("openfaas"),
			Repo:      pulumi.String("openfaas"),
			Version:   pulumi.String("8.0.2"),
			Namespace: pulumi.String("openfaas"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://openfaas.github.io/faas-netes/"),
			},
			Values: pulumi.Map{
				"functionNamespace": pulumi.String("openfaas-fn"),
				"generateBasicAuth": pulumi.Bool(true),
				"openfaasPRO":       pulumi.Bool(true),
				"serviceType":       pulumi.String("LoadBalancer"),
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		kafkaBrokerCa, err := v1.NewSecret(ctx, "kafka-broker-ca", &v1.SecretArgs{
			Type: pulumi.String("generic"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("kafka-broker-ca"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"broker-ca": caCert,
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		kafkaBrokerCert, err := v1.NewSecret(ctx, "kafka-broker-cert", &v1.SecretArgs{
			Type: pulumi.String("generic"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("kafka-broker-cert"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"broker-cert": accessCert,
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		kafkaBrokerKey, err := v1.NewSecret(ctx, "kafka-broker-key", &v1.SecretArgs{
			Type: pulumi.String("generic"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("kafka-broker-key"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"broker-key": accessKey,
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		openfaas := conf.Require("openfaas")
		_, err = v1.NewSecret(ctx, "openfaas-license", &v1.SecretArgs{
			Type: pulumi.String("generic"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("openfaas-license"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"license": pulumi.String(openfaas),
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		_, err = helm.NewChart(ctx, "kafka-connector", helm.ChartArgs{
			Chart:     pulumi.String("kafka-connector"),
			Repo:      pulumi.String("openfaas"),
			Version:   pulumi.String("0.6.3"),
			Namespace: pulumi.String("openfaas"),
			FetchArgs: helm.FetchArgs{

				Repo: pulumi.String("https://openfaas.github.io/faas-netes/"),
			},

			Values: pulumi.Map{
				"brokerHost": serviceUri,
				"tls":        pulumi.Bool(true),
				"saslAuth":   pulumi.Bool(false),
				"caSecret":   kafkaBrokerCa.Metadata.Name(),
				"certSecret": kafkaBrokerCert.Metadata.Name(),
				"keySecret":  kafkaBrokerKey.Metadata.Name(),
				"topics":     topicName,
			},
		}, pulumi.Providers(provider))
		if err != nil {
			return err
		}

		return nil
	})
}
