package grpc_server

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const redactedPlaceholder = "***"

func Redact(req interface{}) interface{} {
	msg, ok := req.(proto.Message)
	if !ok {
		return req
	}
	return redactMessage(msg.ProtoReflect())
}

func redactMessage(m protoreflect.Message) map[string]interface{} {
	out := make(map[string]interface{})
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		name := string(fd.Name())
		if isSensitive(fd) {
			out[name] = redactedPlaceholder
			return true
		}
		out[name] = redactValue(fd, v)
		return true
	})
	return out
}

func isSensitive(fd protoreflect.FieldDescriptor) bool {
	opts, ok := fd.Options().(*descriptorpb.FieldOptions)
	return ok && opts.GetDebugRedact()
}

func redactValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) interface{} {
	switch {
	case fd.IsList():
		list := v.List()
		out := make([]interface{}, 0, list.Len())
		for i := 0; i < list.Len(); i++ {
			out = append(out, redactSingular(fd, list.Get(i)))
		}
		return out
	case fd.IsMap():
		out := make(map[string]interface{})
		v.Map().Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
			out[mk.String()] = redactSingular(fd.MapValue(), mv)
			return true
		})
		return out
	default:
		return redactSingular(fd, v)
	}
}

func redactSingular(fd protoreflect.FieldDescriptor, v protoreflect.Value) interface{} {
	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return redactMessage(v.Message())
	default:
		return v.Interface()
	}
}
