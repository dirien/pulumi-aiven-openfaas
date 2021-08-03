package main

import (
	"github.com/pulumi/pulumi-aiven/sdk/v4/go/aiven"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const projectName = "kafka-test\n"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		kafka, err := aiven.NewKafka(ctx, "kafka", &aiven.KafkaArgs{
			Project:     pulumi.String(projectName),
			CloudName:   pulumi.String("azure-westeurope"),
			Plan:        pulumi.String("startup-2"),
			ServiceName: pulumi.String("openfaas-kafka"),
			KafkaUserConfig: &aiven.KafkaKafkaUserConfigArgs{
				KafkaRest:    pulumi.String("true"),
				KafkaVersion: pulumi.String("2.7"),
			},
		})
		if err != nil {
			return err
		}
		openfaasTopic, err := aiven.NewKafkaTopic(ctx, "openfaas", &aiven.KafkaTopicArgs{
			Project:     pulumi.String(projectName),
			ServiceName: kafka.ServiceName,
			Partitions:  pulumi.Int(4),
			Replication: pulumi.Int(2),
			TopicName:   pulumi.String("openfaas-pro"),
		})

		if err != nil {
			return err
		}

		user, err := aiven.NewServiceUser(ctx, "openfaas-reader", &aiven.ServiceUserArgs{
			Project:     pulumi.String(projectName),
			ServiceName: kafka.ServiceName,
			Username:    pulumi.String("openfaas-reader"),
		})
		if err != nil {
			return err
		}

		_, err = aiven.NewKafkaAcl(ctx, "openfaas-acl", &aiven.KafkaAclArgs{
			Project:     pulumi.String(projectName),
			ServiceName: kafka.ServiceName,
			Username:    user.Username,
			Topic:       openfaasTopic.TopicName,
			Permission:  pulumi.String("read"),
		})
		if err != nil {
			return err
		}

		ctx.Export("name", kafka.ServiceName)
		ctx.Export("AccessKey", pulumi.ToSecret(user.AccessKey))
		ctx.Export("AccessCert", pulumi.ToSecret(user.AccessCert))

		project, err := aiven.LookupProject(ctx, &aiven.LookupProjectArgs{
			Project: projectName,
		}, nil)
		if err != nil {
			return err
		}

		namespace, err := v1.NewNamespace(ctx, "openfaas", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("openfaas"),
			},
		})
		if err != nil {
			return err
		}

		_, err = v1.NewNamespace(ctx, "openfaas-fn", &v1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.StringPtr("openfaas-fn"),
			},
		})
		if err != nil {
			return err
		}

		_, err = helm.NewChart(ctx, "openfaas", helm.ChartArgs{
			Chart:     pulumi.String("openfaas/openfaas"),
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
		})
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
				"broker-ca": pulumi.String(project.CaCert),
			},
		})
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
				"broker-cert": user.AccessCert,
			},
		})
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
				"broker-key": user.AccessKey,
			},
		})
		if err != nil {
			return err
		}

		_, err = v1.NewSecret(ctx, "openfaas-license", &v1.SecretArgs{
			Type: pulumi.String("generic"),
			Metadata: &metav1.ObjectMetaArgs{
				Name:      pulumi.StringPtr("openfaas-license"),
				Namespace: namespace.Metadata.Name(),
			},
			StringData: pulumi.StringMap{
				"license": pulumi.String("xxx"),
			},
		})
		if err != nil {
			return err
		}

		_, err = helm.NewChart(ctx, "kafka-connector", helm.ChartArgs{
			Chart:     pulumi.String("openfaas/kafka-connector"),
			Version:   pulumi.String("0.6.3"),
			Namespace: pulumi.String("openfaas"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://openfaas.github.io/faas-netes/"),
			},

			Values: pulumi.Map{
				"brokerHost": kafka.ServiceHost,
				"tls":        pulumi.Bool(true),
				"saslAuth":   pulumi.Bool(false),
				"caSecret":   kafkaBrokerCa.Metadata.Name(),
				"certSecret": kafkaBrokerCert.Metadata.Name(),
				"keySecret":  kafkaBrokerKey.Metadata.Name(),
				"topics":     openfaasTopic.TopicName,
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
}
