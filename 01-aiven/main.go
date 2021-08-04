package main

import (
	"github.com/pulumi/pulumi-aiven/sdk/v4/go/aiven"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const projectName = "kafka-test"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		kafka, err := aiven.NewKafka(ctx, "kafka", &aiven.KafkaArgs{
			Project:     pulumi.String(projectName),
			CloudName:   pulumi.String("azure-westeurope"),
			Plan:        pulumi.String("startup-2"),
			ServiceName: pulumi.String("openfaas-kafka"),
			KafkaUserConfig: &aiven.KafkaKafkaUserConfigArgs{
				KafkaRest:      pulumi.String("true"),
				KafkaConnect:   pulumi.String("false"),
				SchemaRegistry: pulumi.String("false"),
				KafkaVersion:   pulumi.String("2.8"),
				PublicAccess: &aiven.KafkaKafkaUserConfigPublicAccessArgs{
					KafkaRest: pulumi.String("true"),
				},
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

		project, err := aiven.LookupProject(ctx, &aiven.LookupProjectArgs{
			Project: projectName,
		}, nil)
		if err != nil {
			return err
		}

		ctx.Export("caCert", pulumi.ToSecret(project.CaCert))
		ctx.Export("serviceUri", pulumi.ToSecret(kafka.ServiceUri))
		ctx.Export("topicName", pulumi.ToSecret(openfaasTopic.TopicName))
		ctx.Export("accessCert", pulumi.ToSecret(user.AccessCert))
		ctx.Export("accessKey", pulumi.ToSecret(user.AccessKey))
		ctx.Export("restUri", pulumi.ToSecret(kafka.Kafka.RestUri()))
		return nil
	})
}
