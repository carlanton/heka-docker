package heka_docker

import (
	//	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	"github.com/mozilla-services/heka/pipeline"
	"log"
	"time"
)

var debugMode bool

func debug(v ...interface{}) {
	if debugMode {
		log.Println(v...)
	}
}

func assert(err error, context string) {
	if err != nil {
		log.Fatal(context+": ", err)
	}
}

type DockerInputConfig struct {
	Endpoint    string `toml:"endpoint"`
	DecoderName string `toml:"decoder"`
}

type DockerInput struct {
	client *docker.Client
	conf   *DockerInputConfig
	conn   redis.Conn
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

	if decoder != nil {
	}

	attacher := NewAttachManager(di.client)

	logstream := make(chan *Log)
	defer close(logstream)

	closer := make(chan bool)

	go attacher.Listen(nil, logstream, closer)

	for logline := range logstream {
		pack = <-packSupply
		pack.Message.SetType("docker_log")
		//pack.Message.SetLogger(n.Channel)
		pack.Message.SetPayload(logline.Data)
		pack.Message.SetTimestamp(time.Now().UnixNano())
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

	//Connect to the channel
	/*
		psc := redis.PubSubConn{Conn: di.conn}
		psc.PSubscribe(di.conf.Channel)

		for {
			switch n := psc.Receive().(type) {
			case redis.PMessage:
				// Grab an empty PipelinePack from the InputRunner
				pack = <-packSupply
				pack.Message.SetType("redis_pub_sub")
				pack.Message.SetLogger(n.Channel)
				pack.Message.SetPayload(string(n.Data))
				pack.Message.SetTimestamp(time.Now().UnixNano())
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
						ir.LogError(fmt.Errorf("Couldn't parse Redis message: %s", n.Data))
					}
					pack.Recycle()
				}
			case redis.Subscription:
				ir.LogMessage(fmt.Sprintf("Subscription: %s %s %d\n", n.Kind, n.Channel, n.Count))
				if n.Count == 0 {
					return errors.New("No channel to subscribe")
				}
			case error:
				fmt.Printf("error: %v\n", n)
				return n
			}
		}
	*/

	return nil
}

func (di *DockerInput) Stop() {
	di.conn.Close()
}

func init() {
	pipeline.RegisterPlugin("DockerInput", func() interface{} {
		return new(DockerInput)
	})
}
