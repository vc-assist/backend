// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: vcassist/services/keychain/v1/api.proto

package keychainv1

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

type UsernamePasswordKey struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *UsernamePasswordKey) Reset() {
	*x = UsernamePasswordKey{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UsernamePasswordKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UsernamePasswordKey) ProtoMessage() {}

func (x *UsernamePasswordKey) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UsernamePasswordKey.ProtoReflect.Descriptor instead.
func (*UsernamePasswordKey) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{0}
}

func (x *UsernamePasswordKey) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *UsernamePasswordKey) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type OAuthKey struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token      string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
	RefreshUrl string `protobuf:"bytes,2,opt,name=refresh_url,json=refreshUrl,proto3" json:"refresh_url,omitempty"`
	ClientId   string `protobuf:"bytes,3,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	ExpiresAt  int64  `protobuf:"varint,4,opt,name=expires_at,json=expiresAt,proto3" json:"expires_at,omitempty"`
}

func (x *OAuthKey) Reset() {
	*x = OAuthKey{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OAuthKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OAuthKey) ProtoMessage() {}

func (x *OAuthKey) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OAuthKey.ProtoReflect.Descriptor instead.
func (*OAuthKey) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{1}
}

func (x *OAuthKey) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *OAuthKey) GetRefreshUrl() string {
	if x != nil {
		return x.RefreshUrl
	}
	return ""
}

func (x *OAuthKey) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *OAuthKey) GetExpiresAt() int64 {
	if x != nil {
		return x.ExpiresAt
	}
	return 0
}

// SetOAuth
type SetOAuthRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string    `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Id        string    `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Key       *OAuthKey `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *SetOAuthRequest) Reset() {
	*x = SetOAuthRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetOAuthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetOAuthRequest) ProtoMessage() {}

func (x *SetOAuthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetOAuthRequest.ProtoReflect.Descriptor instead.
func (*SetOAuthRequest) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{2}
}

func (x *SetOAuthRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *SetOAuthRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SetOAuthRequest) GetKey() *OAuthKey {
	if x != nil {
		return x.Key
	}
	return nil
}

type SetOAuthResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SetOAuthResponse) Reset() {
	*x = SetOAuthResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetOAuthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetOAuthResponse) ProtoMessage() {}

func (x *SetOAuthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetOAuthResponse.ProtoReflect.Descriptor instead.
func (*SetOAuthResponse) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{3}
}

// SetUsernamePassword
type SetUsernamePasswordRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string               `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Id        string               `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Key       *UsernamePasswordKey `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *SetUsernamePasswordRequest) Reset() {
	*x = SetUsernamePasswordRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetUsernamePasswordRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetUsernamePasswordRequest) ProtoMessage() {}

func (x *SetUsernamePasswordRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetUsernamePasswordRequest.ProtoReflect.Descriptor instead.
func (*SetUsernamePasswordRequest) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{4}
}

func (x *SetUsernamePasswordRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *SetUsernamePasswordRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SetUsernamePasswordRequest) GetKey() *UsernamePasswordKey {
	if x != nil {
		return x.Key
	}
	return nil
}

type SetUsernamePasswordResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SetUsernamePasswordResponse) Reset() {
	*x = SetUsernamePasswordResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetUsernamePasswordResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetUsernamePasswordResponse) ProtoMessage() {}

func (x *SetUsernamePasswordResponse) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetUsernamePasswordResponse.ProtoReflect.Descriptor instead.
func (*SetUsernamePasswordResponse) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{5}
}

// GetOAuth
type GetOAuthRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Id        string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetOAuthRequest) Reset() {
	*x = GetOAuthRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetOAuthRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOAuthRequest) ProtoMessage() {}

func (x *GetOAuthRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOAuthRequest.ProtoReflect.Descriptor instead.
func (*GetOAuthRequest) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{6}
}

func (x *GetOAuthRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *GetOAuthRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type GetOAuthResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// this will be null if a key cannot be found or is expired
	Key *OAuthKey `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *GetOAuthResponse) Reset() {
	*x = GetOAuthResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetOAuthResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOAuthResponse) ProtoMessage() {}

func (x *GetOAuthResponse) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOAuthResponse.ProtoReflect.Descriptor instead.
func (*GetOAuthResponse) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{7}
}

func (x *GetOAuthResponse) GetKey() *OAuthKey {
	if x != nil {
		return x.Key
	}
	return nil
}

// GetUsernamePassword
type GetUsernamePasswordRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Id        string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *GetUsernamePasswordRequest) Reset() {
	*x = GetUsernamePasswordRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetUsernamePasswordRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUsernamePasswordRequest) ProtoMessage() {}

func (x *GetUsernamePasswordRequest) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUsernamePasswordRequest.ProtoReflect.Descriptor instead.
func (*GetUsernamePasswordRequest) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{8}
}

func (x *GetUsernamePasswordRequest) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *GetUsernamePasswordRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

type GetUsernamePasswordResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// this will be null if a key cannot be found or is expired
	Key *UsernamePasswordKey `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *GetUsernamePasswordResponse) Reset() {
	*x = GetUsernamePasswordResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetUsernamePasswordResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetUsernamePasswordResponse) ProtoMessage() {}

func (x *GetUsernamePasswordResponse) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_api_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetUsernamePasswordResponse.ProtoReflect.Descriptor instead.
func (*GetUsernamePasswordResponse) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_api_proto_rawDescGZIP(), []int{9}
}

func (x *GetUsernamePasswordResponse) GetKey() *UsernamePasswordKey {
	if x != nil {
		return x.Key
	}
	return nil
}

var File_vcassist_services_keychain_v1_api_proto protoreflect.FileDescriptor

var file_vcassist_services_keychain_v1_api_proto_rawDesc = []byte{
	0x0a, 0x27, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x2f, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x2f,
	0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1d, 0x76, 0x63, 0x61, 0x73, 0x73,
	0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79,
	0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x22, 0x4d, 0x0a, 0x13, 0x55, 0x73, 0x65, 0x72,
	0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x4b, 0x65, 0x79, 0x12,
	0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70,
	0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x70,
	0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0x7d, 0x0a, 0x08, 0x4f, 0x41, 0x75, 0x74, 0x68,
	0x4b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x12, 0x1f, 0x0a, 0x0b, 0x72, 0x65, 0x66,
	0x72, 0x65, 0x73, 0x68, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a,
	0x72, 0x65, 0x66, 0x72, 0x65, 0x73, 0x68, 0x55, 0x72, 0x6c, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63,
	0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72,
	0x65, 0x73, 0x5f, 0x61, 0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x65, 0x78, 0x70,
	0x69, 0x72, 0x65, 0x73, 0x41, 0x74, 0x22, 0x7a, 0x0a, 0x0f, 0x53, 0x65, 0x74, 0x4f, 0x41, 0x75,
	0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d,
	0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x12, 0x39, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x4b, 0x65, 0x79, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x22, 0x12, 0x0a, 0x10, 0x53, 0x65, 0x74, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x90, 0x01, 0x0a, 0x1a, 0x53, 0x65, 0x74, 0x55, 0x73,
	0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x02, 0x69, 0x64, 0x12, 0x44, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x32, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31,
	0x2e, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
	0x64, 0x4b, 0x65, 0x79, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x22, 0x1d, 0x0a, 0x1b, 0x53, 0x65, 0x74,
	0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x3f, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x4f,
	0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e,
	0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69, 0x64, 0x22, 0x4d, 0x0a, 0x10, 0x47, 0x65, 0x74,
	0x4f, 0x41, 0x75, 0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x39, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x76, 0x63, 0x61,
	0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b,
	0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x4f, 0x41, 0x75, 0x74, 0x68,
	0x4b, 0x65, 0x79, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x22, 0x4a, 0x0a, 0x1a, 0x47, 0x65, 0x74, 0x55,
	0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x02, 0x69, 0x64, 0x22, 0x63, 0x0a, 0x1b, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x6e,
	0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x44, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x32, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31,
	0x2e, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72,
	0x64, 0x4b, 0x65, 0x79, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x32, 0x89, 0x04, 0x0a, 0x0f, 0x4b, 0x65,
	0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x6b, 0x0a,
	0x08, 0x53, 0x65, 0x74, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x12, 0x2e, 0x2e, 0x76, 0x63, 0x61, 0x73,
	0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65,
	0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x4f, 0x41, 0x75,
	0x74, 0x68, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2f, 0x2e, 0x76, 0x63, 0x61, 0x73,
	0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65,
	0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x4f, 0x41, 0x75,
	0x74, 0x68, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x6b, 0x0a, 0x08, 0x47, 0x65,
	0x74, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x12, 0x2e, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73,
	0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2f, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73,
	0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68,
	0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x8c, 0x01, 0x0a, 0x13, 0x53, 0x65, 0x74, 0x55,
	0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12,
	0x39, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e,
	0x53, 0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77,
	0x6f, 0x72, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3a, 0x2e, 0x76, 0x63, 0x61,
	0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b,
	0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x55, 0x73,
	0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x8c, 0x01, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x55, 0x73,
	0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x39,
	0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x47,
	0x65, 0x74, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3a, 0x2e, 0x76, 0x63, 0x61, 0x73,
	0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65,
	0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x55, 0x73, 0x65,
	0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x85, 0x02, 0x0a, 0x21, 0x63, 0x6f, 0x6d, 0x2e, 0x76, 0x63,
	0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e,
	0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x42, 0x08, 0x41, 0x70, 0x69,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x3f, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73,
	0x74, 0x2d, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f,
	0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x73, 0x2f, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x3b, 0x6b, 0x65,
	0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x76, 0x31, 0xa2, 0x02, 0x03, 0x56, 0x53, 0x4b, 0xaa, 0x02,
	0x1d, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x73, 0x2e, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x56, 0x31, 0xca, 0x02,
	0x1d, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x5c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x73, 0x5c, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5c, 0x56, 0x31, 0xe2, 0x02,
	0x29, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x5c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x73, 0x5c, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x5c, 0x56, 0x31, 0x5c, 0x47,
	0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x20, 0x56, 0x63, 0x61,
	0x73, 0x73, 0x69, 0x73, 0x74, 0x3a, 0x3a, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x3a,
	0x3a, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_vcassist_services_keychain_v1_api_proto_rawDescOnce sync.Once
	file_vcassist_services_keychain_v1_api_proto_rawDescData = file_vcassist_services_keychain_v1_api_proto_rawDesc
)

func file_vcassist_services_keychain_v1_api_proto_rawDescGZIP() []byte {
	file_vcassist_services_keychain_v1_api_proto_rawDescOnce.Do(func() {
		file_vcassist_services_keychain_v1_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_vcassist_services_keychain_v1_api_proto_rawDescData)
	})
	return file_vcassist_services_keychain_v1_api_proto_rawDescData
}

var file_vcassist_services_keychain_v1_api_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_vcassist_services_keychain_v1_api_proto_goTypes = []any{
	(*UsernamePasswordKey)(nil),         // 0: vcassist.services.keychain.v1.UsernamePasswordKey
	(*OAuthKey)(nil),                    // 1: vcassist.services.keychain.v1.OAuthKey
	(*SetOAuthRequest)(nil),             // 2: vcassist.services.keychain.v1.SetOAuthRequest
	(*SetOAuthResponse)(nil),            // 3: vcassist.services.keychain.v1.SetOAuthResponse
	(*SetUsernamePasswordRequest)(nil),  // 4: vcassist.services.keychain.v1.SetUsernamePasswordRequest
	(*SetUsernamePasswordResponse)(nil), // 5: vcassist.services.keychain.v1.SetUsernamePasswordResponse
	(*GetOAuthRequest)(nil),             // 6: vcassist.services.keychain.v1.GetOAuthRequest
	(*GetOAuthResponse)(nil),            // 7: vcassist.services.keychain.v1.GetOAuthResponse
	(*GetUsernamePasswordRequest)(nil),  // 8: vcassist.services.keychain.v1.GetUsernamePasswordRequest
	(*GetUsernamePasswordResponse)(nil), // 9: vcassist.services.keychain.v1.GetUsernamePasswordResponse
}
var file_vcassist_services_keychain_v1_api_proto_depIdxs = []int32{
	1, // 0: vcassist.services.keychain.v1.SetOAuthRequest.key:type_name -> vcassist.services.keychain.v1.OAuthKey
	0, // 1: vcassist.services.keychain.v1.SetUsernamePasswordRequest.key:type_name -> vcassist.services.keychain.v1.UsernamePasswordKey
	1, // 2: vcassist.services.keychain.v1.GetOAuthResponse.key:type_name -> vcassist.services.keychain.v1.OAuthKey
	0, // 3: vcassist.services.keychain.v1.GetUsernamePasswordResponse.key:type_name -> vcassist.services.keychain.v1.UsernamePasswordKey
	2, // 4: vcassist.services.keychain.v1.KeychainService.SetOAuth:input_type -> vcassist.services.keychain.v1.SetOAuthRequest
	6, // 5: vcassist.services.keychain.v1.KeychainService.GetOAuth:input_type -> vcassist.services.keychain.v1.GetOAuthRequest
	4, // 6: vcassist.services.keychain.v1.KeychainService.SetUsernamePassword:input_type -> vcassist.services.keychain.v1.SetUsernamePasswordRequest
	8, // 7: vcassist.services.keychain.v1.KeychainService.GetUsernamePassword:input_type -> vcassist.services.keychain.v1.GetUsernamePasswordRequest
	3, // 8: vcassist.services.keychain.v1.KeychainService.SetOAuth:output_type -> vcassist.services.keychain.v1.SetOAuthResponse
	7, // 9: vcassist.services.keychain.v1.KeychainService.GetOAuth:output_type -> vcassist.services.keychain.v1.GetOAuthResponse
	5, // 10: vcassist.services.keychain.v1.KeychainService.SetUsernamePassword:output_type -> vcassist.services.keychain.v1.SetUsernamePasswordResponse
	9, // 11: vcassist.services.keychain.v1.KeychainService.GetUsernamePassword:output_type -> vcassist.services.keychain.v1.GetUsernamePasswordResponse
	8, // [8:12] is the sub-list for method output_type
	4, // [4:8] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_vcassist_services_keychain_v1_api_proto_init() }
func file_vcassist_services_keychain_v1_api_proto_init() {
	if File_vcassist_services_keychain_v1_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_vcassist_services_keychain_v1_api_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*UsernamePasswordKey); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*OAuthKey); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*SetOAuthRequest); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*SetOAuthResponse); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*SetUsernamePasswordRequest); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*SetUsernamePasswordResponse); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*GetOAuthRequest); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[7].Exporter = func(v any, i int) any {
			switch v := v.(*GetOAuthResponse); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[8].Exporter = func(v any, i int) any {
			switch v := v.(*GetUsernamePasswordRequest); i {
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
		file_vcassist_services_keychain_v1_api_proto_msgTypes[9].Exporter = func(v any, i int) any {
			switch v := v.(*GetUsernamePasswordResponse); i {
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
			RawDescriptor: file_vcassist_services_keychain_v1_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_vcassist_services_keychain_v1_api_proto_goTypes,
		DependencyIndexes: file_vcassist_services_keychain_v1_api_proto_depIdxs,
		MessageInfos:      file_vcassist_services_keychain_v1_api_proto_msgTypes,
	}.Build()
	File_vcassist_services_keychain_v1_api_proto = out.File
	file_vcassist_services_keychain_v1_api_proto_rawDesc = nil
	file_vcassist_services_keychain_v1_api_proto_goTypes = nil
	file_vcassist_services_keychain_v1_api_proto_depIdxs = nil
}
