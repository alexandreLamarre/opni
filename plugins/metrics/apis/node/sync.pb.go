// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v1.0.0
// source: github.com/rancher/opni/plugins/metrics/apis/node/sync.proto

package node

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type SyncRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CurrentConfig *MetricsCapabilityStatus `protobuf:"bytes,1,opt,name=currentConfig,proto3" json:"currentConfig,omitempty"`
}

func (x *SyncRequest) Reset() {
	*x = SyncRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SyncRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncRequest) ProtoMessage() {}

func (x *SyncRequest) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncRequest.ProtoReflect.Descriptor instead.
func (*SyncRequest) Descriptor() ([]byte, []int) {
	return file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescGZIP(), []int{0}
}

func (x *SyncRequest) GetCurrentConfig() *MetricsCapabilityStatus {
	if x != nil {
		return x.CurrentConfig
	}
	return nil
}

type SyncResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConfigStatus  ConfigStatus             `protobuf:"varint,1,opt,name=configStatus,proto3,enum=node.metrics.config.ConfigStatus" json:"configStatus,omitempty"`
	UpdatedConfig *MetricsCapabilityStatus `protobuf:"bytes,2,opt,name=updatedConfig,proto3" json:"updatedConfig,omitempty"`
}

func (x *SyncResponse) Reset() {
	*x = SyncResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SyncResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SyncResponse) ProtoMessage() {}

func (x *SyncResponse) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SyncResponse.ProtoReflect.Descriptor instead.
func (*SyncResponse) Descriptor() ([]byte, []int) {
	return file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescGZIP(), []int{1}
}

func (x *SyncResponse) GetConfigStatus() ConfigStatus {
	if x != nil {
		return x.ConfigStatus
	}
	return ConfigStatus_Unknown
}

func (x *SyncResponse) GetUpdatedConfig() *MetricsCapabilityStatus {
	if x != nil {
		return x.UpdatedConfig
	}
	return nil
}

var File_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto protoreflect.FileDescriptor

var file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x6e,
	0x63, 0x68, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x6e, 0x69, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e,
	0x73, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x6e,
	0x6f, 0x64, 0x65, 0x2f, 0x73, 0x79, 0x6e, 0x63, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c,
	0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x1a, 0x3e, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x72,
	0x2f, 0x6f, 0x70, 0x6e, 0x69, 0x2f, 0x70, 0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x2f,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x61, 0x0a, 0x0b,
	0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x52, 0x0a, 0x0d, 0x63,
	0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63,
	0x73, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73,
	0x43, 0x61, 0x70, 0x61, 0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x52, 0x0d, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22,
	0xa9, 0x01, 0x0a, 0x0c, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x45, 0x0a, 0x0c, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x21, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x6d, 0x65,
	0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2e, 0x43, 0x6f, 0x6e,
	0x66, 0x69, 0x67, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x0c, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x52, 0x0a, 0x0d, 0x75, 0x70, 0x64, 0x61, 0x74,
	0x65, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c,
	0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x43, 0x61, 0x70, 0x61,
	0x62, 0x69, 0x6c, 0x69, 0x74, 0x79, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x0d, 0x75, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x32, 0x56, 0x0a, 0x15, 0x4e,
	0x6f, 0x64, 0x65, 0x4d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x43, 0x61, 0x70, 0x61, 0x62, 0x69,
	0x6c, 0x69, 0x74, 0x79, 0x12, 0x3d, 0x0a, 0x04, 0x53, 0x79, 0x6e, 0x63, 0x12, 0x19, 0x2e, 0x6e,
	0x6f, 0x64, 0x65, 0x2e, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x53, 0x79, 0x6e, 0x63,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1a, 0x2e, 0x6e, 0x6f, 0x64, 0x65, 0x2e, 0x6d,
	0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2e, 0x53, 0x79, 0x6e, 0x63, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x42, 0x33, 0x5a, 0x31, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x6e, 0x69, 0x2f, 0x70,
	0x6c, 0x75, 0x67, 0x69, 0x6e, 0x73, 0x2f, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x63, 0x73, 0x2f, 0x61,
	0x70, 0x69, 0x73, 0x2f, 0x6e, 0x6f, 0x64, 0x65, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescOnce sync.Once
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescData = file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDesc
)

func file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescGZIP() []byte {
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescOnce.Do(func() {
		file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescData)
	})
	return file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDescData
}

var file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_goTypes = []interface{}{
	(*SyncRequest)(nil),             // 0: node.metrics.SyncRequest
	(*SyncResponse)(nil),            // 1: node.metrics.SyncResponse
	(*MetricsCapabilityStatus)(nil), // 2: node.metrics.config.MetricsCapabilityStatus
	(ConfigStatus)(0),               // 3: node.metrics.config.ConfigStatus
}
var file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_depIdxs = []int32{
	2, // 0: node.metrics.SyncRequest.currentConfig:type_name -> node.metrics.config.MetricsCapabilityStatus
	3, // 1: node.metrics.SyncResponse.configStatus:type_name -> node.metrics.config.ConfigStatus
	2, // 2: node.metrics.SyncResponse.updatedConfig:type_name -> node.metrics.config.MetricsCapabilityStatus
	0, // 3: node.metrics.NodeMetricsCapability.Sync:input_type -> node.metrics.SyncRequest
	1, // 4: node.metrics.NodeMetricsCapability.Sync:output_type -> node.metrics.SyncResponse
	4, // [4:5] is the sub-list for method output_type
	3, // [3:4] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_init() }
func file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_init() {
	if File_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto != nil {
		return
	}
	file_github_com_rancher_opni_plugins_metrics_apis_node_config_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SyncRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SyncResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_goTypes,
		DependencyIndexes: file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_depIdxs,
		MessageInfos:      file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_msgTypes,
	}.Build()
	File_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto = out.File
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_rawDesc = nil
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_goTypes = nil
	file_github_com_rancher_opni_plugins_metrics_apis_node_sync_proto_depIdxs = nil
}
