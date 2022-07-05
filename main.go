package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	yaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime"

	_ "k8s.io/api/apps/v1"
	_ "k8s.io/api/batch/v1"
	_ "k8s.io/api/core/v1"
	_ "k8s.io/api/storage/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/stripe/skycfg"
	"github.com/stripe/skycfg/gogocompat"
)

var (
	filename = flag.String("dry_run", "foo.sky", "The file to ")
)

var k8sProtoMagic = []byte("k8s\x00")

// marshal wraps msg into runtime.Unknown object and prepends magic sequence
// to conform with Kubernetes protobuf content type.
func marshal(msg proto.Message) ([]byte, error) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	unknownBytes, err := proto.Marshal(&runtime.Unknown{Raw: msgBytes})
	if err != nil {
		return nil, err
	}
	return append(k8sProtoMagic, unknownBytes...), nil
}

func main() {
	ctx := context.Background()
	config, err := skycfg.Load(ctx, *filename, skycfg.WithProtoRegistry(gogocompat.ProtoRegistry()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading %q: %v\n", *filename, err)
		os.Exit(1)
	}

	var jsonMarshaler = &jsonpb.Marshaler{OrigName: true}
	protos, err := config.Main(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error evaluating %q: %v\n", config.Filename(), err)
		os.Exit(1)
	}
	for _, msg := range protos {
		marshaled, err := jsonMarshaler.MarshalToString(msg)
		sep := ""
		if err != nil {
			fmt.Fprintf(os.Stderr, "json.Marshal: %v\n", err)
			continue
		}
		var yamlMap yaml.MapSlice
		if err := yaml.Unmarshal([]byte(marshaled), &yamlMap); err != nil {
			panic(fmt.Sprintf("yaml.Unmarshal: %v", err))
		}
		yamlMarshaled, err := yaml.Marshal(yamlMap)
		if err != nil {
			panic(fmt.Sprintf("yaml.Marshal: %v", err))
		}
		marshaled = string(yamlMarshaled)
		sep = "---\n"
		fmt.Printf("%s%s\n", sep, marshaled)
	}
}
