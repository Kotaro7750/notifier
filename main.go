package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Notification struct {
	Message string `json:"message"`
}

type Configuration struct {
	ReceiverConfigurations []AbstractChannelComponentConfig `yaml:"receivers,flow"`
	SenderConfigurations   []AbstractChannelComponentConfig `yaml:"senders,flow"`
}

func validateConfiguration(config Configuration) error {
	if config.ReceiverConfigurations == nil {
		return fmt.Errorf("receivers is not defined")
	}

	if len(config.ReceiverConfigurations) == 0 {
		return fmt.Errorf("At least one receiver is required")
	}

	for _, receiverConfig := range config.ReceiverConfigurations {
		if err := receiverConfig.validate(); err != nil {
			return err
		}
	}

	if config.SenderConfigurations == nil {
		return fmt.Errorf("senders is not defined")
	}

	if len(config.SenderConfigurations) == 0 {
		return fmt.Errorf("At least one sender is required")
	}

	for _, senderConfig := range config.SenderConfigurations {
		if err := senderConfig.validate(); err != nil {
			return err
		}
	}

	return nil
}

type AbstractChannelComponent interface {
	GetId() string
	Start(ch chan Notification, done <-chan struct{}) <-chan error
}

type AutonomousChannelComponent struct {
	chanComponent  AbstractChannelComponent
	ch             chan Notification
	shutdownCh     chan struct{}
	isStarted      bool
	isShuttingDown bool
	lock           sync.Mutex
}

func NewAutonomousChannelComponent(chanComponent AbstractChannelComponent) *AutonomousChannelComponent {
	return &AutonomousChannelComponent{
		chanComponent:  chanComponent,
		ch:             make(chan Notification),
		shutdownCh:     make(chan struct{}),
		isStarted:      false,
		isShuttingDown: false,
		lock:           sync.Mutex{},
	}
}

func (acc *AutonomousChannelComponent) Start() <-chan struct{} {
	Logger.Info("Start invoked", "id", acc.chanComponent.GetId())
	acc.lock.Lock()
	defer acc.lock.Unlock()

	if acc.isStarted {
		Logger.Info("Already started", "id", acc.chanComponent.GetId())
		return nil
	}
	acc.isStarted = true
	acc.shutdownCh = make(chan struct{})

	completedCh := make(chan struct{})

	go func(stopCh <-chan struct{}) {
		defer close(completedCh)
		for {
			Logger.Info("Starting", "id", acc.chanComponent.GetId())
			select {
			case err := <-acc.chanComponent.Start(acc.ch, stopCh):
				if err != nil {
					Logger.Error("Error in channel component", "id", acc.chanComponent.GetId(), "error", err)
				}

				acc.lock.Lock()
				if acc.isShuttingDown {
					defer acc.lock.Unlock()
					defer close(acc.ch)
					Logger.Info("Shutting down", "id", acc.chanComponent.GetId())

					acc.isStarted = false
					acc.isShuttingDown = false
					return
				}
				acc.lock.Unlock()
			}

			Logger.Info("Restart after 1s", "id", acc.chanComponent.GetId())

			time.Sleep(1 * time.Second)
		}
	}(acc.shutdownCh)

	return completedCh
}

func (acc *AutonomousChannelComponent) GetChannel() chan Notification {
	return acc.ch
}

func (acc *AutonomousChannelComponent) Shutdown() {
	Logger.Info("Shutdown invoked", "id", acc.chanComponent.GetId())
	acc.lock.Lock()
	defer acc.lock.Unlock()

	if !acc.isStarted {
		Logger.Info("Not started", "id", acc.chanComponent.GetId())
		return
	}
	acc.isShuttingDown = true

	close(acc.shutdownCh)
	return
}

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

	config := Configuration{}
	err = yaml.Unmarshal(fileContent, &config)
	if err != nil {
		Logger.Error("Error parsing YAML file", "err", err)
		return
	}

	if err := validateConfiguration(config); err != nil {
		Logger.Error("Invalid configuration", "err", err)
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt, os.Kill)

	receivers, senders, err := Build(config.ReceiverConfigurations, config.SenderConfigurations)

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
	routerCh := make(chan Notification)

	for _, receiver := range receivers {
		go func(r *AutonomousChannelComponent) {
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
		Logger.Info("Shutting down receiver", "id", receiver.chanComponent.GetId())
		wg.Add(1)
		go func() {
			receiver.Shutdown()
			<-receiverChs[i]
			wg.Done()
			Logger.Info("Complete shut down receiver", "id", receiver.chanComponent.GetId())
		}()
	}

	wg.Wait()
	Logger.Info("All receivers are shut down")

	Logger.Info("Shutting down senders")

	wg = sync.WaitGroup{}
	for i, sender := range senders {
		Logger.Info("Shutting down sender", "id", sender.chanComponent.GetId())
		wg.Add(1)
		go func() {
			sender.Shutdown()
			<-senderChs[i]
			wg.Done()
			Logger.Info("Complete shut down sender", "id", sender.chanComponent.GetId())
		}()
	}

	wg.Wait()
	Logger.Info("All senders are shut down")

	close(routerCh)
}
