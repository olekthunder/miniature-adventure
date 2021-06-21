package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/kylelemons/go-gypsy/yaml"
	"gopkg.in/urfave/cli.v1"
)

var client *api.Client

func writeConsulKV(key string, value []byte) {
	kv := client.KV()
	p := &api.KVPair{Key: strings.TrimLeft(key, "/"), Value: value}
	_, err := kv.Put(p, nil)

	if err != nil {
		log.Fatal(err)
	}
}

func nodeIterator(path string, node yaml.Node) {

	yamlMap, isYamlMap := node.(yaml.Map)
	if isYamlMap {
		for key, node := range yamlMap {
			nodeIterator(fmt.Sprint(path, "/", key), node)
		}
		return
	}

	yamlScalar, isYamlScalar := node.(yaml.Scalar)
	if isYamlScalar {
		writeConsulKV(path, []byte(yamlScalar))
		log.Printf("Key: \"%s\" Data: \"%s\"\n", strings.TrimLeft(path, "/"), yamlScalar)
		return
	}

	yamlList, isYamlList := node.(yaml.List)
	if isYamlList {
		buf := bytes.NewBuffer(nil)
		for _, fileNameNode := range yamlList {
			fileName, _ := fileNameNode.(yaml.Scalar)
			file, err := os.Open(string(fileName))
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(buf, file)
			file.Close()
		}

		writeConsulKV(path, buf.Bytes())
		log.Printf("Key: \"%s\" Data: \"File(%d Bytes)\"\n", strings.TrimLeft(path, "/"), buf.Len())
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "consul-kv-bootstrap"
	app.Usage = "Simple importer tool to load a yaml file into the key value database consul, existing consul values will be updated"
	app.UsageText = ""
	app.Version = "v0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file, f",
			Usage: "Your YAML file for populating the consul key value database",
		},
		cli.StringFlag{
			Name:  "prefix, p",
			Usage: "A consul prefix for your YAML config (/test/bootstrap/)",
		},
	}

	app.Action = func(c *cli.Context) error {

		if len(c.String("file")) == 0 {
			fmt.Printf("Missing required parameter -file (-file app.yaml).\n")
			os.Exit(1)
		}

		file, err := yaml.ReadFile(c.String("file"))
		if err != nil {
			fmt.Printf("Could not open file %s.\n", c.String("file"))
			os.Exit(1)
		}

		config := api.DefaultConfig()
		config.Address = os.Getenv("CONSUL_HTTP_ADDR")
		client, err = api.NewClient(config)
		if err != nil {
			log.Fatal(err)
		}

		nodeIterator(strings.TrimRight(c.String("prefix"), "/"), file.Root)

		return nil
	}

	app.Run(os.Args)
}
