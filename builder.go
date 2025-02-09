package main

import (
	"context"
	"fmt"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type AbstractChannelComponentConfig struct {
	Id         string                 `yaml:"id"`
	Kind       string                 `yaml:"kind"`
	Properties map[string]interface{} `yaml:"properties"`
}

func (c *AbstractChannelComponentConfig) validate() error {
	if c.Id == "" {
		return fmt.Errorf("id is required")
	}

	if c.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	return nil
}

type AbstractChannelComponentBuilder func(id string, properties map[string]interface{}) (AbstractChannelComponent, error)

var senderBuilderMap map[string]AbstractChannelComponentBuilder = make(map[string]AbstractChannelComponentBuilder)
var receiverBuilderMap map[string]AbstractChannelComponentBuilder = make(map[string]AbstractChannelComponentBuilder)

func init() {
	receiverBuilderMap["dummy"] = func(id string, properties map[string]interface{}) (AbstractChannelComponent, error) {
		return &Receiver{&dummyReceiverImpl{id: id}}, nil
	}
	receiverBuilderMap["HTTP"] = func(id string, properties map[string]interface{}) (AbstractChannelComponent, error) {
		listenAddr, ok := properties["listenAddress"]
		if !ok {
			return nil, fmt.Errorf("listenAddress is required")
		}

		listenAddrStr, ok := listenAddr.(string)
		if !ok {
			return nil, fmt.Errorf("listenAddress should be string")
		}

		return &Receiver{&HTTPReceiverImpl{id: id, listenAddr: listenAddrStr}}, nil
	}

	senderBuilderMap["dummy"] = func(id string, properties map[string]interface{}) (AbstractChannelComponent, error) {
		return &Sender{&dummySenderImpl{id: id}}, nil
	}

	senderBuilderMap["webPush"] = func(id string, properties map[string]interface{}) (AbstractChannelComponent, error) {
		listenAddr, ok := properties["listenAddress"]
		if !ok {
			return nil, fmt.Errorf("listenAddress is required")
		}

		listenAddrStr, ok := listenAddr.(string)
		if !ok {
			return nil, fmt.Errorf("listenAddress should be string")
		}

		defaultSubscriberStr := ""
		defaultSubscriber, ok := properties["defaultSubscriber"]
		if ok {
			defaultSubscriberStr, ok = defaultSubscriber.(string)
			if !ok {
				return nil, fmt.Errorf("defaultSubscriber should be string")
			}
		}

		repositoryType, ok := properties["repositoryType"]
		if !ok {
			return nil, fmt.Errorf("repositoryType is required")
		}

		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-northeast-1"))
		if err != nil {
			return nil, fmt.Errorf("Load AWS config failed. err: %s", err)
		}
		dynamodbClient := dynamodb.NewFromConfig(cfg)

		var subscriptionRepository SubscriptionRepository

		switch repositoryType {
		case "DynamoDB":
			subscriptionRepository = NewDynamoDBSubscriptionRepository(dynamodbClient)
		case "InMemory":
			subscriptionRepository = NewInMemorySubscriptionRepository()
		default:
			return nil, fmt.Errorf("repositoryType is invalid. repositoryType: %s", repositoryType)
		}

		output, err := dynamodbClient.GetItem(context.Background(), &dynamodb.GetItemInput{
			TableName: aws.String("notifier-config"),
			Key: map[string]types.AttributeValue{
				"Key": &types.AttributeValueMemberS{
					Value: "vapid",
				},
			},
		})

		if err != nil {
			return nil, fmt.Errorf("GetItem for vapid failed. err: %s", err)
		}

		var vapidPrivateKey, vapidPublicKey string

		if output.Item == nil {
			vapidPrivateKey, vapidPublicKey, _ = webpush.GenerateVAPIDKeys()

			_, err := dynamodbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
				TableName: aws.String("notifier-config"),
				Item: map[string]types.AttributeValue{
					"Key": &types.AttributeValueMemberS{
						Value: "vapid",
					},
					"Value": &types.AttributeValueMemberS{
						Value: fmt.Sprintf("%s %s", vapidPrivateKey, vapidPublicKey),
					},
				},
			})
			if err != nil {
				return nil, fmt.Errorf("PutItem for vapid failed. err: %s", err)
			}

		} else {
			vapid := struct {
				Key   string `dynamodbav:"Key"`
				Value string `dynamodbav:"Value"`
			}{}

			err = attributevalue.UnmarshalMap(output.Item, &vapid)
			if err != nil {
				return nil, fmt.Errorf("Unmarshal for vapid failed. err: %s", err)
			}

			splitted := strings.Split(vapid.Value, " ")
			vapidPrivateKey = splitted[0]
			vapidPublicKey = splitted[1]
		}

		return &Sender{&webPushSenderImpl{
			id:                     id,
			listenAddress:          listenAddrStr,
			defaultSubscriber:      defaultSubscriberStr,
			subscriptionRepository: subscriptionRepository,
			vapidPrivateKey:        vapidPrivateKey,
			vapidPublicKey:         vapidPublicKey,
		}}, nil
	}
}

func Build(receiverConfigs []AbstractChannelComponentConfig, senderConfigs []AbstractChannelComponentConfig) (receivers []*AutonomousChannelComponent, senders []*AutonomousChannelComponent, err error) {
	receivers = make([]*AutonomousChannelComponent, 0)

	for _, config := range receiverConfigs {
		builder, ok := receiverBuilderMap[config.Kind]
		if !ok {
			err = fmt.Errorf("receiver kind: %s for %s is not found", config.Kind, config.Id)
			return nil, nil, err
		}

		component, err := builder(config.Id, config.Properties)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build receiver id: %s, kind: %s, error: %s", config.Id, config.Kind, err.Error())
		}

		receivers = append(receivers, NewAutonomousChannelComponent(component))
	}

	senders = make([]*AutonomousChannelComponent, 0)

	for _, config := range senderConfigs {
		builder, ok := senderBuilderMap[config.Kind]
		if !ok {
			err = fmt.Errorf("sender kind: %s for %s is not found", config.Kind, config.Id)
			return nil, nil, err
		}

		component, err := builder(config.Id, config.Properties)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build sender id: %s, kind: %s, error: %s", config.Id, config.Kind, err.Error())
		}

		senders = append(senders, NewAutonomousChannelComponent(component))
	}
	return
}
