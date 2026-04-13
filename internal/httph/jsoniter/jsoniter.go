package jsoniter

import (
	"io"

	"github.com/go-openapi/runtime"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// JSONConsumer создает нового потребителя JSON
func JSONConsumer() runtime.Consumer {
	return runtime.ConsumerFunc(func(reader io.Reader, data interface{}) error {
		dec := json.NewDecoder(reader)
		dec.UseNumber() // preserve number formats
		return dec.Decode(data)
	})
}

// JSONProducer создает нового производителя JSON
func JSONProducer() runtime.Producer {
	return runtime.ProducerFunc(func(writer io.Writer, data interface{}) error {
		enc := json.NewEncoder(writer)
		enc.SetEscapeHTML(false)
		return enc.Encode(data)
	})
}
