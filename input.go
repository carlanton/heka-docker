package heka_docker

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"os"
	"time"
)

type DockerInputConfig struct {
	Endpoint    string `toml:"endpoint"`
	DecoderName string `toml:"decoder"`
}

type DockerInput struct {
	client   *docker.Client
	conf     *DockerInputConfig
	stopChan chan bool
}

func (di *DockerInput) ConfigStruct() interface{} {
	return &DockerInputConfig{"unix:///var/run/docker.sock", ""}
}

func (di *DockerInput) Init(config interface{}) error {
	di.conf = config.(*DockerInputConfig)

	var err error
	di.client, err = docker.NewClient(di.conf.Endpoint)

	if err != nil {
		return fmt.Errorf("connecting to - %s", err.Error())
	}

	return nil
}

func (di *DockerInput) Run(ir pipeline.InputRunner, h pipeline.PluginHelper) error {
	var (
		dRunner pipeline.DecoderRunner
		decoder pipeline.Decoder
		pack    *pipeline.PipelinePack
		e       error
		ok      bool
	)
	// Get the InputRunner's chan to receive empty PipelinePacks
	packSupply := ir.InChan()

	if di.conf.DecoderName != "" {
		if dRunner, ok = h.DecoderRunner(di.conf.DecoderName, fmt.Sprintf("%s-%s", ir.Name(), di.conf.DecoderName)); !ok {
			return fmt.Errorf("Decoder not found: %s", di.conf.DecoderName)
		}
		decoder = dRunner.Decoder()
	}

	di.stopChan = make(chan bool)
	logstream := make(chan *Log)
	defer close(logstream)

	closer := make(chan bool)
	go NewAttachManager(di.client).Listen(nil, logstream, closer)

	stopped := false

	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	for !stopped {
		select {
		case <-di.stopChan:
			stopped = true
		case logline := <-logstream:
			pack = <-packSupply

			pack.Message.SetType("docker-container")
			pack.Message.SetLogger(logline.Type) // stderr or stdout
			pack.Message.SetHostname(hostname)   // Use the host's hosntame
			pack.Message.SetPayload(logline.Data)
			pack.Message.SetTimestamp(time.Now().UnixNano())
			pack.Message.SetUuid(uuid.NewRandom())
			message.NewStringField(pack.Message, "ContainerID", logline.ID)
			message.NewStringField(pack.Message, "ContainerName", logline.Name)

			var packs []*pipeline.PipelinePack
			if decoder == nil {
				packs = []*pipeline.PipelinePack{pack}
			} else {
				packs, e = decoder.Decode(pack)
			}
			if packs != nil {
				for _, p := range packs {
					ir.Inject(p)
				}
			} else {
				if e != nil {
					ir.LogError(fmt.Errorf("Couldn't parse Docker message!"))
				}
				pack.Recycle()
			}
		}
	}

	closer <- true
	return nil
}

func (di *DockerInput) Stop() {
	close(di.stopChan)
}

func init() {
	pipeline.RegisterPlugin("DockerInput", func() interface{} {
		return new(DockerInput)
	})
}
