// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v1.0.0
// source: github.com/rancher/opni/pkg/apis/alerting/v2/condition.proto

package v2

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto protoreflect.FileDescriptor

var file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x6e,
	0x63, 0x68, 0x65, 0x72, 0x2f, 0x6f, 0x70, 0x6e, 0x69, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70,
	0x69, 0x73, 0x2f, 0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x32, 0x2f, 0x63,
	0x6f, 0x6e, 0x64, 0x69, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08,
	0x61, 0x6c, 0x65, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x61, 0x6e, 0x63, 0x68, 0x65, 0x72, 0x2f, 0x6f,
	0x70, 0x6e, 0x69, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70, 0x69, 0x73, 0x2f, 0x61, 0x6c, 0x65,
	0x72, 0x74, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x32, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_goTypes = []interface{}{}
var file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_init() }
func file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_init() {
	if File_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_goTypes,
		DependencyIndexes: file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_depIdxs,
	}.Build()
	File_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto = out.File
	file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_rawDesc = nil
	file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_goTypes = nil
	file_github_com_rancher_opni_pkg_apis_alerting_v2_condition_proto_depIdxs = nil
}
