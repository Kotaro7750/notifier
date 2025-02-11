package builder

import (
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/receiver"
	"github.com/Kotaro7750/notifier/sender"
)

var senderBuilderMap map[string]abstraction.AbstractChannelComponentBuilder = make(map[string]abstraction.AbstractChannelComponentBuilder)
var receiverBuilderMap map[string]abstraction.AbstractChannelComponentBuilder = make(map[string]abstraction.AbstractChannelComponentBuilder)

func init() {
	receiverBuilderMap["dummy"] = receiver.DummyReceiverBuilder

	receiverBuilderMap["HTTP"] = receiver.HTTPReceiverBuilder

	senderBuilderMap["dummy"] = sender.DummySenderBuilder

	senderBuilderMap["webPush"] = sender.WebPushSenderBuilder
}

func Build(
	baseLogger *slog.Logger,
	receiverConfigs []abstraction.AbstractChannelComponentConfig,
	senderConfigs []abstraction.AbstractChannelComponentConfig,
) (
	receivers []*abstraction.AutonomousChannelComponent,
	senders []*abstraction.AutonomousChannelComponent,
	err error,
) {
	receivers = make([]*abstraction.AutonomousChannelComponent, 0)

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
		component.SetLogger(baseLogger.With("type", "receiver", "kind", config.Kind, "id", component.GetId()))

		receivers = append(receivers, abstraction.NewAutonomousChannelComponent(component))
	}

	senders = make([]*abstraction.AutonomousChannelComponent, 0)

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
		component.SetLogger(baseLogger.With("type", "sender", "kind", config.Kind, "id", component.GetId()))

		senders = append(senders, abstraction.NewAutonomousChannelComponent(component))
	}
	return
}
