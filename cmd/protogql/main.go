package main

import (
	"flag"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fresh8gaming/go-proto-gql/pkg/protoparser"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/fresh8gaming/go-proto-gql/pkg/generator"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "str list"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	importPaths = arrayFlags{}
	fileNames   = arrayFlags{}
	svc         = flag.Bool("svc", false, "Use service annotations for nodes corresponding to a GRPC call")
	merge       = flag.Bool("merge", false, "Merge all the proto files found in one directory into one graphql file")
	extension   = flag.String("ext", generator.DefaultExtension, "Extension of the graphql file, Default: '.graphql'")
)

func main() {
	flag.Var(&importPaths, "I", "path")
	flag.Var(&fileNames, "f", "path")
	flag.Parse()
	descs, err := protoparser.Parse(importPaths, fileNames)
	fatal(err)
	p, err := protogen.Options{}.New(&pluginpb.CodeGeneratorRequest{
		FileToGenerate: fileNames,
		ProtoFile:      generator.ResolveProtoFilesRecursively(descs).AsFileDescriptorProto(),
	})
	fatal(err)
	gqlDesc, err := generator.NewSchemas(descs, *merge, *svc, p)
	fatal(err)
	for _, schema := range gqlDesc {
		if len(schema.FileDescriptors) < 1 {
			log.Fatalf("unexpected number of proto descriptors: %d for gql schema", len(schema.FileDescriptors))
		}
		if len(schema.FileDescriptors) > 1 {
			err := generateFile(schema, true)
			fatal(err)
			break
		}
		err := generateFile(schema, *merge)
		fatal(err)
	}
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func generateFile(schema *generator.SchemaDescriptor, merge bool) error {

	sc, err := os.Create(resolveGraphqlFilename(schema.FileDescriptors[0].GetName(), merge, *extension))
	if err != nil {
		return err
	}
	defer sc.Close()

	formatter.NewFormatter(sc).FormatSchema(schema.AsGraphql())
	return nil
}

func hasSchema(schema *ast.Schema) bool {
	if schema.Query != nil {
		return true
	}
	if schema.Mutation != nil {
		return true
	}
	if schema.Subscription != nil {
		return true
	}
	return false
}

func resolveGraphqlFilename(protoFileName string, merge bool, extension string) string {
	if merge {
		gqlFileName := "schema." + extension
		absProtoFileName, err := filepath.Abs(protoFileName)
		if err == nil {
			protoDirSlice := strings.Split(filepath.Dir(absProtoFileName), string(filepath.Separator))
			if len(protoDirSlice) > 0 {
				gqlFileName = protoDirSlice[len(protoDirSlice)-1] + "." + extension
			}
		}
		protoDir, _ := path.Split(protoFileName)
		return path.Join(protoDir, gqlFileName)
	}

	return strings.TrimSuffix(protoFileName, path.Ext(protoFileName)) + "." + extension
}
