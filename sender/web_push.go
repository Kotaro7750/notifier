package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/cors"
)

func WebPushSenderBuilder(id string, properties map[string]interface{}) (abstraction.AbstractChannelComponent, error) {
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

	var subscriptionRepository SubscriptionRepository
	vapidPrivateKey, vapidPublicKey, _ := webpush.GenerateVAPIDKeys()

	switch repositoryType {
	case "DynamoDB":
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("ap-northeast-1"))
		if err != nil {
			return nil, fmt.Errorf("Load AWS config failed. err: %s", err)
		}
		dynamodbClient := dynamodb.NewFromConfig(cfg)

		subscriptionRepository = NewDynamoDBSubscriptionRepository(dynamodbClient)

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

		if output.Item == nil {
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

	case "InMemory":
		subscriptionRepository = NewInMemorySubscriptionRepository()
	default:
		return nil, fmt.Errorf("repositoryType is invalid. repositoryType: %s", repositoryType)
	}

	return NewSender(&webPushSenderImpl{
		id:                     id,
		logger:                 nil,
		listenAddress:          listenAddrStr,
		defaultSubscriber:      defaultSubscriberStr,
		subscriptionRepository: subscriptionRepository,
		vapidPrivateKey:        vapidPrivateKey,
		vapidPublicKey:         vapidPublicKey,
	}), nil
}

type SubscriptionRepository interface {
	LoadAll() ([]webpush.Subscription, error)
	Store(subscription webpush.Subscription) error
	Delete(subscription webpush.Subscription) error
}

type InMemorySubscriptionRepository struct {
	subscriptionMap sync.Map
}

func NewInMemorySubscriptionRepository() *InMemorySubscriptionRepository {
	return &InMemorySubscriptionRepository{
		subscriptionMap: sync.Map{},
	}
}

func (imsr *InMemorySubscriptionRepository) LoadAll() ([]webpush.Subscription, error) {
	subscriptions := make([]webpush.Subscription, 0)

	imsr.subscriptionMap.Range(func(key, value interface{}) bool {
		subscription := value.(webpush.Subscription)

		subscriptions = append(subscriptions, subscription)

		return true
	})

	return subscriptions, nil

}

func (imsr *InMemorySubscriptionRepository) Store(subscription webpush.Subscription) error {
	imsr.subscriptionMap.Store(subscription.Endpoint, subscription)

	return nil
}

func (imsr *InMemorySubscriptionRepository) Delete(subscription webpush.Subscription) error {

	_, ok := imsr.subscriptionMap.Load(subscription.Endpoint)
	if ok {
		imsr.subscriptionMap.Delete(subscription.Endpoint)
	}

	return nil
}

type DynamoDBSubscriptionRepository struct {
	dynamodbClient *dynamodb.Client
}

func NewDynamoDBSubscriptionRepository(dynamodbClient *dynamodb.Client) *DynamoDBSubscriptionRepository {
	return &DynamoDBSubscriptionRepository{
		dynamodbClient: dynamodbClient,
	}
}

func (ddbr *DynamoDBSubscriptionRepository) LoadAll() ([]webpush.Subscription, error) {
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String("notifier-subscriptions"),
	}

	subscriptions := make([]webpush.Subscription, 0)

	for {
		output, err := ddbr.dynamodbClient.Scan(context.Background(), scanInput)

		for _, item := range output.Items {
			subscription := webpush.Subscription{}

			err = attributevalue.UnmarshalMap(item, &subscription)
			if err != nil {
				return nil, fmt.Errorf("Unmarshaling subscription from DynamoDB AttributeValue failed. err: %s", err.Error())
			}

			subscriptions = append(subscriptions, subscription)
		}

		if output.LastEvaluatedKey == nil {
			break
		}

		scanInput.ExclusiveStartKey = output.LastEvaluatedKey
	}
	return subscriptions, nil
}

func (ddbr *DynamoDBSubscriptionRepository) Store(subscription webpush.Subscription) error {
	av, err := attributevalue.MarshalMap(subscription)
	if err != nil {
		return fmt.Errorf("Marshaling subscription to DynamoDB AttributeValue failed. err: %s", err.Error())
	}

	_, err = ddbr.dynamodbClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String("notifier-subscriptions"),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("PutItem to DynamoDB failed. err: %s", err.Error())
	}

	return nil
}

func (ddbr *DynamoDBSubscriptionRepository) Delete(subscription webpush.Subscription) error {
	_, err := ddbr.dynamodbClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String("notifier-subscriptions"),
		Key: map[string]types.AttributeValue{
			"Endpoint": &types.AttributeValueMemberS{
				Value: subscription.Endpoint,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("DeleteItem to DynamoDB failed. err %s", err.Error())
	}

	return nil
}

type webPushSenderImpl struct {
	id                     string
	logger                 *slog.Logger
	listenAddress          string
	defaultSubscriber      string
	vapidPrivateKey        string
	vapidPublicKey         string
	subscriptionRepository SubscriptionRepository
}

func (wpsi *webPushSenderImpl) GetId() string {
	return fmt.Sprintf("%s", wpsi.id)
}

func (wpsi *webPushSenderImpl) GetLogger() *slog.Logger {
	return wpsi.logger
}

func (wpsi *webPushSenderImpl) SetLogger(logger *slog.Logger) {
	wpsi.logger = logger
}

func (wpsi *webPushSenderImpl) Start(inputCh <-chan notification.Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	serveMux := http.NewServeMux()

	serveMux.HandleFunc("GET /publickey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(wpsi.vapidPublicKey))
	})

	serveMux.HandleFunc("POST /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		var subscription webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			wpsi.GetLogger().Error("Decoding posted subscription to JSON failed", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = wpsi.subscriptionRepository.Store(subscription)
		if err != nil {
			wpsi.GetLogger().Error("Store subscription to repository failed", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		wpsi.GetLogger().Info("Receive subscription")
		w.WriteHeader(http.StatusOK)
	})

	serveMux.HandleFunc("DELETE /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		var subscription webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			wpsi.GetLogger().Error("Decoding passed subscription to JSON failed", "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = wpsi.subscriptionRepository.Delete(subscription)
		if err != nil {
			wpsi.GetLogger().Error("Delete subscription from repository failed", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		wpsi.GetLogger().Info("Delete subscription")

		w.WriteHeader(http.StatusNoContent)
		return
	})

	serveMux.HandleFunc("GET /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		subscriptions, err := wpsi.subscriptionRepository.LoadAll()
		if err != nil {
			wpsi.GetLogger().Error("LoadAll subscription from repository failed", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		endpoints := make([]string, 0)

		for _, subscription := range subscriptions {
			endpoints = append(endpoints, subscription.Endpoint)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(endpoints)
	})

	s := &http.Server{
		Addr: wpsi.listenAddress,
		// TODO セキュリティ的によくないので環境変数経由で指定できるように設定する
		Handler: cors.AllowAll().Handler(serveMux),
	}

	shutdownFunc := func() {
		s.Shutdown(context.Background())
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		err := s.ListenAndServe()
		errCh <- err
	}()

	go func() {
		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					wpsi.GetLogger().Info("inputCh closed")
				} else {
					subscriptions, err := wpsi.subscriptionRepository.LoadAll()

					if err != nil {
						wpsi.GetLogger().Error("LoadAll subscription from repository failed", "err", err)
						errCh <- err
						return
					}

					data, err := json.Marshal(n)
					if err != nil {
						wpsi.GetLogger().Error("Marshal notification failed", "err", err)
						errCh <- err
						return
					}

					for _, subscription := range subscriptions {
						res, err := webpush.SendNotification(data, &subscription, &webpush.Options{
							Subscriber:      wpsi.defaultSubscriber,
							VAPIDPublicKey:  wpsi.vapidPublicKey,
							VAPIDPrivateKey: wpsi.vapidPrivateKey,
						})

						wpsi.GetLogger().Info("Notify send to WebPush Endpoint from webPushSender", "response", res.Status)

						if err != nil {
							wpsi.GetLogger().Error("SendNotification failed", "err", err)
							errCh <- err
							break
						}
					}
				}
			case <-done:
				return
			}
		}
	}()

	go func() {
		defer close(retCh)
		select {
		case err := <-errCh:
			shutdownFunc()
			retCh <- err
			return
		case <-done:
			shutdownFunc()
			return
		}
	}()

	return retCh
}
