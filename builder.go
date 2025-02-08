package main

import "fmt"

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

		return &Sender{&webPushSenderImpl{id: id, listenAddress: listenAddrStr, defaultSubscriber: defaultSubscriberStr}}, nil
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
