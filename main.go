package main

import (
	"bytes"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/builder"
	"github.com/Kotaro7750/notifier/config"
	"github.com/Kotaro7750/notifier/notification"

	"gopkg.in/yaml.v3"
)

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	if len(os.Args) < 2 {
		Logger.Error("Configuration file is required")
		return
	}

	configFileNAme := os.Args[1]
	fileContent, err := os.ReadFile(configFileNAme)
	if err != nil {
		Logger.Error("Error reading file", "err", err)
		return
	}

	cfg := config.Configuration{}
	decoder := yaml.NewDecoder(bytes.NewReader(fileContent))
	decoder.KnownFields(true)
	err = decoder.Decode(&cfg)
	if err != nil {
		Logger.Error("Error parsing YAML file", "err", err)
		return
	}

	if err := cfg.Validate(); err != nil {
		Logger.Error("Invalid configuration", "err", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt, os.Kill)

	receivers, senders, err := builder.Build(Logger, cfg.ReceiverConfigurations, cfg.SenderConfigurations)

	if err != nil {
		Logger.Error("Error in build", "error", err)
		return
	}

	senderChs := make([]<-chan struct{}, len(senders))

	receiverChs := make([]<-chan struct{}, len(receivers))

	for i, sender := range senders {
		senderChs[i] = sender.Start()
	}

	for i, receiver := range receivers {
		receiverChs[i] = receiver.Start()
	}

	router := Router{senders: senders}
	routerCh := make(chan notification.Notification)

	for _, receiver := range receivers {
		go func(r *abstraction.AutonomousChannelComponent) {
			for n := range r.GetChannel() {
				routerCh <- n
			}
		}(receiver)
	}

	go func() {
		for n := range routerCh {
			router.Route(n)
		}
	}()

	<-sigCh
	Logger.Info("Received signal")

	Logger.Info("Shutting down receivers")

	wg := sync.WaitGroup{}
	for i, receiver := range receivers {
		receiver.GetLogger().Info("Shutting down receiver")
		wg.Add(1)
		go func() {
			receiver.Shutdown()
			<-receiverChs[i]
			wg.Done()
			receiver.GetLogger().Info("Complete shut down receiver")
		}()
	}

	wg.Wait()
	Logger.Info("All receivers are shut down")

	Logger.Info("Shutting down senders")

	wg = sync.WaitGroup{}
	for i, sender := range senders {
		sender.GetLogger().Info("Shutting down sender")
		wg.Add(1)
		go func() {
			sender.Shutdown()
			<-senderChs[i]
			wg.Done()
			sender.GetLogger().Info("Complete shut down sender")
		}()
	}

	wg.Wait()
	Logger.Info("All senders are shut down")

	close(routerCh)
}
