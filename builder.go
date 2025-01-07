package main

import "fmt"

type AbstractChannelComponentConfig struct {
	id   string
	kind string
}

type AbstractChannelComponentBuilder func(id string) AbstractChannelComponent

var senderBuilderMap map[string]AbstractChannelComponentBuilder = make(map[string]AbstractChannelComponentBuilder)
var receiverBuilderMap map[string]AbstractChannelComponentBuilder = make(map[string]AbstractChannelComponentBuilder)

func init() {
	receiverBuilderMap["dummy"] = func(id string) AbstractChannelComponent {
		return &Receiver{&dummyReceiverImpl{id: id}}
	}
	receiverBuilderMap["HTTP"] = func(id string) AbstractChannelComponent {
		return &Receiver{&HTTPReceiverImpl{id: id}}
	}

	senderBuilderMap["dummy"] = func(id string) AbstractChannelComponent {
		return &Sender{&dummySenderImpl{id: id}}
	}
}

func Build(receiverConfigs []AbstractChannelComponentConfig, senderConfigs []AbstractChannelComponentConfig) (receivers []*AutonomousChannelComponent, senders []*AutonomousChannelComponent, err error) {
	receivers = make([]*AutonomousChannelComponent, 0)

	for _, config := range receiverConfigs {
		builder, ok := receiverBuilderMap[config.kind]
		if !ok {
			err = fmt.Errorf("receiver kind: %s for %s is not found", config.kind, config.id)
			return nil, nil, err
		}
		receivers = append(receivers, NewAutonomousChannelComponent(builder(config.id)))
	}

	senders = make([]*AutonomousChannelComponent, 0)

	for _, config := range senderConfigs {
		builder, ok := senderBuilderMap[config.kind]
		if !ok {
			err = fmt.Errorf("sender kind: %s for %s is not found", config.kind, config.id)
			return nil, nil, err
		}
		senders = append(senders, NewAutonomousChannelComponent(builder(config.id)))
	}
	return
}
