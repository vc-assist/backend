// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: vcassist/services/keychain/v1/auth_flow.proto

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

type UsernamePasswordFlow struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *UsernamePasswordFlow) Reset() {
	*x = UsernamePasswordFlow{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UsernamePasswordFlow) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UsernamePasswordFlow) ProtoMessage() {}

func (x *UsernamePasswordFlow) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UsernamePasswordFlow.ProtoReflect.Descriptor instead.
func (*UsernamePasswordFlow) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP(), []int{0}
}

type OAuthFlow struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	BaseLoginUrl    string `protobuf:"bytes,1,opt,name=base_login_url,json=baseLoginUrl,proto3" json:"base_login_url,omitempty"`
	AccessType      string `protobuf:"bytes,2,opt,name=access_type,json=accessType,proto3" json:"access_type,omitempty"`
	Scope           string `protobuf:"bytes,3,opt,name=scope,proto3" json:"scope,omitempty"`
	RedirectUri     string `protobuf:"bytes,4,opt,name=redirect_uri,json=redirectUri,proto3" json:"redirect_uri,omitempty"`
	CodeVerifier    string `protobuf:"bytes,5,opt,name=code_verifier,json=codeVerifier,proto3" json:"code_verifier,omitempty"`
	ClientId        string `protobuf:"bytes,6,opt,name=client_id,json=clientId,proto3" json:"client_id,omitempty"`
	TokenRequestUrl string `protobuf:"bytes,7,opt,name=token_request_url,json=tokenRequestUrl,proto3" json:"token_request_url,omitempty"`
}

func (x *OAuthFlow) Reset() {
	*x = OAuthFlow{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OAuthFlow) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OAuthFlow) ProtoMessage() {}

func (x *OAuthFlow) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OAuthFlow.ProtoReflect.Descriptor instead.
func (*OAuthFlow) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP(), []int{1}
}

func (x *OAuthFlow) GetBaseLoginUrl() string {
	if x != nil {
		return x.BaseLoginUrl
	}
	return ""
}

func (x *OAuthFlow) GetAccessType() string {
	if x != nil {
		return x.AccessType
	}
	return ""
}

func (x *OAuthFlow) GetScope() string {
	if x != nil {
		return x.Scope
	}
	return ""
}

func (x *OAuthFlow) GetRedirectUri() string {
	if x != nil {
		return x.RedirectUri
	}
	return ""
}

func (x *OAuthFlow) GetCodeVerifier() string {
	if x != nil {
		return x.CodeVerifier
	}
	return ""
}

func (x *OAuthFlow) GetClientId() string {
	if x != nil {
		return x.ClientId
	}
	return ""
}

func (x *OAuthFlow) GetTokenRequestUrl() string {
	if x != nil {
		return x.TokenRequestUrl
	}
	return ""
}

type CredentialStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Picture  string `protobuf:"bytes,2,opt,name=picture,proto3" json:"picture,omitempty"`
	Provided bool   `protobuf:"varint,3,opt,name=provided,proto3" json:"provided,omitempty"`
	// Types that are assignable to LoginFlow:
	//
	//	*CredentialStatus_UsernamePassword
	//	*CredentialStatus_Oauth
	LoginFlow isCredentialStatus_LoginFlow `protobuf_oneof:"login_flow"`
}

func (x *CredentialStatus) Reset() {
	*x = CredentialStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CredentialStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CredentialStatus) ProtoMessage() {}

func (x *CredentialStatus) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CredentialStatus.ProtoReflect.Descriptor instead.
func (*CredentialStatus) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP(), []int{2}
}

func (x *CredentialStatus) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *CredentialStatus) GetPicture() string {
	if x != nil {
		return x.Picture
	}
	return ""
}

func (x *CredentialStatus) GetProvided() bool {
	if x != nil {
		return x.Provided
	}
	return false
}

func (m *CredentialStatus) GetLoginFlow() isCredentialStatus_LoginFlow {
	if m != nil {
		return m.LoginFlow
	}
	return nil
}

func (x *CredentialStatus) GetUsernamePassword() *UsernamePasswordFlow {
	if x, ok := x.GetLoginFlow().(*CredentialStatus_UsernamePassword); ok {
		return x.UsernamePassword
	}
	return nil
}

func (x *CredentialStatus) GetOauth() *OAuthFlow {
	if x, ok := x.GetLoginFlow().(*CredentialStatus_Oauth); ok {
		return x.Oauth
	}
	return nil
}

type isCredentialStatus_LoginFlow interface {
	isCredentialStatus_LoginFlow()
}

type CredentialStatus_UsernamePassword struct {
	UsernamePassword *UsernamePasswordFlow `protobuf:"bytes,4,opt,name=username_password,json=usernamePassword,proto3,oneof"`
}

type CredentialStatus_Oauth struct {
	Oauth *OAuthFlow `protobuf:"bytes,5,opt,name=oauth,proto3,oneof"`
}

func (*CredentialStatus_UsernamePassword) isCredentialStatus_LoginFlow() {}

func (*CredentialStatus_Oauth) isCredentialStatus_LoginFlow() {}

type UsernamePasswordProvision struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Username string `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password string `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
}

func (x *UsernamePasswordProvision) Reset() {
	*x = UsernamePasswordProvision{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UsernamePasswordProvision) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UsernamePasswordProvision) ProtoMessage() {}

func (x *UsernamePasswordProvision) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UsernamePasswordProvision.ProtoReflect.Descriptor instead.
func (*UsernamePasswordProvision) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP(), []int{3}
}

func (x *UsernamePasswordProvision) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *UsernamePasswordProvision) GetPassword() string {
	if x != nil {
		return x.Password
	}
	return ""
}

type OAuthTokenProvision struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Token string `protobuf:"bytes,1,opt,name=token,proto3" json:"token,omitempty"`
}

func (x *OAuthTokenProvision) Reset() {
	*x = OAuthTokenProvision{}
	if protoimpl.UnsafeEnabled {
		mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OAuthTokenProvision) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OAuthTokenProvision) ProtoMessage() {}

func (x *OAuthTokenProvision) ProtoReflect() protoreflect.Message {
	mi := &file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OAuthTokenProvision.ProtoReflect.Descriptor instead.
func (*OAuthTokenProvision) Descriptor() ([]byte, []int) {
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP(), []int{4}
}

func (x *OAuthTokenProvision) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

var File_vcassist_services_keychain_v1_auth_flow_proto protoreflect.FileDescriptor

var file_vcassist_services_keychain_v1_auth_flow_proto_rawDesc = []byte{
	0x0a, 0x2d, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x73, 0x2f, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2f, 0x76, 0x31, 0x2f,
	0x61, 0x75, 0x74, 0x68, 0x5f, 0x66, 0x6c, 0x6f, 0x77, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x1d, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x22, 0x16,
	0x0a, 0x14, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f,
	0x72, 0x64, 0x46, 0x6c, 0x6f, 0x77, 0x22, 0xf9, 0x01, 0x0a, 0x09, 0x4f, 0x41, 0x75, 0x74, 0x68,
	0x46, 0x6c, 0x6f, 0x77, 0x12, 0x24, 0x0a, 0x0e, 0x62, 0x61, 0x73, 0x65, 0x5f, 0x6c, 0x6f, 0x67,
	0x69, 0x6e, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x62, 0x61,
	0x73, 0x65, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x55, 0x72, 0x6c, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x63,
	0x63, 0x65, 0x73, 0x73, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0a, 0x61, 0x63, 0x63, 0x65, 0x73, 0x73, 0x54, 0x79, 0x70, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x73,
	0x63, 0x6f, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x73, 0x63, 0x6f, 0x70,
	0x65, 0x12, 0x21, 0x0a, 0x0c, 0x72, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63, 0x74, 0x5f, 0x75, 0x72,
	0x69, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x72, 0x65, 0x64, 0x69, 0x72, 0x65, 0x63,
	0x74, 0x55, 0x72, 0x69, 0x12, 0x23, 0x0a, 0x0d, 0x63, 0x6f, 0x64, 0x65, 0x5f, 0x76, 0x65, 0x72,
	0x69, 0x66, 0x69, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x63, 0x6f, 0x64,
	0x65, 0x56, 0x65, 0x72, 0x69, 0x66, 0x69, 0x65, 0x72, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x6c, 0x69,
	0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x6c,
	0x69, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x2a, 0x0a, 0x11, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x5f,
	0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x75, 0x72, 0x6c, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0f, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x55,
	0x72, 0x6c, 0x22, 0x90, 0x02, 0x0a, 0x10, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61,
	0x6c, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70,
	0x69, 0x63, 0x74, 0x75, 0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x70, 0x69,
	0x63, 0x74, 0x75, 0x72, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x64, 0x12, 0x62, 0x0a, 0x11, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x5f, 0x70, 0x61,
	0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x33, 0x2e, 0x76,
	0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x55, 0x73, 0x65,
	0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x46, 0x6c, 0x6f,
	0x77, 0x48, 0x00, 0x52, 0x10, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x50, 0x61, 0x73,
	0x73, 0x77, 0x6f, 0x72, 0x64, 0x12, 0x40, 0x0a, 0x05, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x2e, 0x76, 0x31, 0x2e, 0x4f, 0x41, 0x75, 0x74, 0x68, 0x46, 0x6c, 0x6f, 0x77, 0x48, 0x00,
	0x52, 0x05, 0x6f, 0x61, 0x75, 0x74, 0x68, 0x42, 0x0c, 0x0a, 0x0a, 0x6c, 0x6f, 0x67, 0x69, 0x6e,
	0x5f, 0x66, 0x6c, 0x6f, 0x77, 0x22, 0x53, 0x0a, 0x19, 0x55, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d,
	0x65, 0x50, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69,
	0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65, 0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a,
	0x0a, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x08, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6f, 0x72, 0x64, 0x22, 0x2b, 0x0a, 0x13, 0x4f, 0x41,
	0x75, 0x74, 0x68, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x73, 0x69, 0x6f,
	0x6e, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e, 0x42, 0x8a, 0x02, 0x0a, 0x21, 0x63, 0x6f, 0x6d, 0x2e,
	0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x73, 0x2e, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x2e, 0x76, 0x31, 0x42, 0x0d, 0x41,
	0x75, 0x74, 0x68, 0x46, 0x6c, 0x6f, 0x77, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x3f,
	0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2d, 0x62, 0x61, 0x63, 0x6b, 0x65, 0x6e, 0x64,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x76, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x2f,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69,
	0x6e, 0x2f, 0x76, 0x31, 0x3b, 0x6b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x76, 0x31, 0xa2,
	0x02, 0x03, 0x56, 0x53, 0x4b, 0xaa, 0x02, 0x1d, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74,
	0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x2e, 0x56, 0x31, 0xca, 0x02, 0x1d, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74,
	0x5c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x5c, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x5c, 0x56, 0x31, 0xe2, 0x02, 0x29, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74,
	0x5c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x5c, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61,
	0x69, 0x6e, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74,
	0x61, 0xea, 0x02, 0x20, 0x56, 0x63, 0x61, 0x73, 0x73, 0x69, 0x73, 0x74, 0x3a, 0x3a, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x3a, 0x3a, 0x4b, 0x65, 0x79, 0x63, 0x68, 0x61, 0x69, 0x6e,
	0x3a, 0x3a, 0x56, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_vcassist_services_keychain_v1_auth_flow_proto_rawDescOnce sync.Once
	file_vcassist_services_keychain_v1_auth_flow_proto_rawDescData = file_vcassist_services_keychain_v1_auth_flow_proto_rawDesc
)

func file_vcassist_services_keychain_v1_auth_flow_proto_rawDescGZIP() []byte {
	file_vcassist_services_keychain_v1_auth_flow_proto_rawDescOnce.Do(func() {
		file_vcassist_services_keychain_v1_auth_flow_proto_rawDescData = protoimpl.X.CompressGZIP(file_vcassist_services_keychain_v1_auth_flow_proto_rawDescData)
	})
	return file_vcassist_services_keychain_v1_auth_flow_proto_rawDescData
}

var file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_vcassist_services_keychain_v1_auth_flow_proto_goTypes = []any{
	(*UsernamePasswordFlow)(nil),      // 0: vcassist.services.keychain.v1.UsernamePasswordFlow
	(*OAuthFlow)(nil),                 // 1: vcassist.services.keychain.v1.OAuthFlow
	(*CredentialStatus)(nil),          // 2: vcassist.services.keychain.v1.CredentialStatus
	(*UsernamePasswordProvision)(nil), // 3: vcassist.services.keychain.v1.UsernamePasswordProvision
	(*OAuthTokenProvision)(nil),       // 4: vcassist.services.keychain.v1.OAuthTokenProvision
}
var file_vcassist_services_keychain_v1_auth_flow_proto_depIdxs = []int32{
	0, // 0: vcassist.services.keychain.v1.CredentialStatus.username_password:type_name -> vcassist.services.keychain.v1.UsernamePasswordFlow
	1, // 1: vcassist.services.keychain.v1.CredentialStatus.oauth:type_name -> vcassist.services.keychain.v1.OAuthFlow
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_vcassist_services_keychain_v1_auth_flow_proto_init() }
func file_vcassist_services_keychain_v1_auth_flow_proto_init() {
	if File_vcassist_services_keychain_v1_auth_flow_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*UsernamePasswordFlow); i {
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
		file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*OAuthFlow); i {
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
		file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*CredentialStatus); i {
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
		file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*UsernamePasswordProvision); i {
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
		file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*OAuthTokenProvision); i {
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
	file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes[2].OneofWrappers = []any{
		(*CredentialStatus_UsernamePassword)(nil),
		(*CredentialStatus_Oauth)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_vcassist_services_keychain_v1_auth_flow_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_vcassist_services_keychain_v1_auth_flow_proto_goTypes,
		DependencyIndexes: file_vcassist_services_keychain_v1_auth_flow_proto_depIdxs,
		MessageInfos:      file_vcassist_services_keychain_v1_auth_flow_proto_msgTypes,
	}.Build()
	File_vcassist_services_keychain_v1_auth_flow_proto = out.File
	file_vcassist_services_keychain_v1_auth_flow_proto_rawDesc = nil
	file_vcassist_services_keychain_v1_auth_flow_proto_goTypes = nil
	file_vcassist_services_keychain_v1_auth_flow_proto_depIdxs = nil
}
