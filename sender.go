package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/cors"
)

type SenderImpl interface {
	GetId() string
	Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error
}

type Sender struct {
	impl SenderImpl
}

func (s *Sender) Start(inputCh chan Notification, done <-chan struct{}) <-chan error {
	return s.impl.Start(inputCh, done)
}

func (s *Sender) GetId() string {
	return s.impl.GetId()
}

type dummySenderImpl struct {
	id string
}

func (dsi *dummySenderImpl) GetId() string {
	return fmt.Sprintf("dummySender %s", dsi.id)
}

func (dsi *dummySenderImpl) Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	shutdownFunc := func() {
		time.Sleep(5 * time.Second)
	}

	go func() {
		defer close(retCh)

		c := time.Tick(10 * time.Second)
		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					Logger.Info("inputCh closed", "id", dsi.id)
				} else {
					Logger.Info("Notify send from dummySender", "id", dsi.id, "message", n.Message)
				}

			case <-c:
				shutdownFunc()
				retCh <- fmt.Errorf("timeout")
				return

			case <-done:
				shutdownFunc()
				return
			}
		}
	}()

	return retCh
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
	listenAddress          string
	defaultSubscriber      string
	vapidPrivateKey        string
	vapidPublicKey         string
	subscriptionRepository SubscriptionRepository
}

func (wpsi *webPushSenderImpl) GetId() string {
	return fmt.Sprintf("webPushSender %s", wpsi.id)
}

func (wpsi *webPushSenderImpl) Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error {
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
			Logger.Error("Decoding posted subscription to JSON failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = wpsi.subscriptionRepository.Store(subscription)
		if err != nil {
			Logger.Error("Store subscription to repository failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		Logger.Info("Receive subscription", "id", wpsi.GetId())
		w.WriteHeader(http.StatusOK)
	})

	serveMux.HandleFunc("DELETE /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		var subscription webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			Logger.Error("Decoding passed subscription to JSON failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		err = wpsi.subscriptionRepository.Delete(subscription)
		if err != nil {
			Logger.Error("Delete subscription from repository failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		Logger.Info("Delete subscription", "id", wpsi.GetId())

		w.WriteHeader(http.StatusNoContent)
		return
	})

	serveMux.HandleFunc("GET /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		subscriptions, err := wpsi.subscriptionRepository.LoadAll()
		if err != nil {
			Logger.Error("LoadAll subscription from repository failed", "id", wpsi.GetId(), "err", err)
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
					Logger.Info("inputCh closed", "id", wpsi.id)
				} else {
					subscriptions, err := wpsi.subscriptionRepository.LoadAll()

					if err != nil {
						Logger.Error("LoadAll subscription from repository failed", "id", wpsi.GetId(), "err", err)
						errCh <- err
						return
					}

					for _, subscription := range subscriptions {
						res, err := webpush.SendNotification([]byte(n.Message), &subscription, &webpush.Options{
							Subscriber:      wpsi.defaultSubscriber,
							VAPIDPublicKey:  wpsi.vapidPublicKey,
							VAPIDPrivateKey: wpsi.vapidPrivateKey,
						})

						Logger.Info("Notify send to WebPush Endpoint from webPushSender", "id", wpsi.id, "response", res.Status)

						if err != nil {
							Logger.Error("SendNotification failed", "id", wpsi.GetId(), "err", err)
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
