package open

import "FlapAlerted/bgp/update"

type Msg struct {
	Version                 uint8
	ASN                     uint16
	HoldTime                HoldTime
	RouterID                RouterID
	OptionalParameterLength uint8
	OptionalParameters      []OptionalParameter
}

type RouterID uint32

const ASTrans uint16 = 23456

type HoldTime uint16

type OptionalParameter struct {
	ParameterType   OptionalParameterType
	ParameterLength uint8
	ParameterValue  OptionalParameterValue
}
type OptionalParameterType uint8

const (
	CapabilityParameter OptionalParameterType = 2
)

type OptionalParameterValue interface {
	MarshalBinary() (data []byte, err error)
}

type CapabilityList struct {
	List []CapabilityOptionalParameter
}

type CapabilityOptionalParameter struct {
	CapabilityCode   CapabilityCode
	CapabilityLength uint8
	CapabilityValue  CapabilityValue
}

type CapabilityCode uint8

const (
	CapabilityCodeMultiProtocol   CapabilityCode = 1
	CapabilityCodeFourByteASN     CapabilityCode = 65
	CapabilityCodeAddPath         CapabilityCode = 69
	CapabilityCodeExtendedMessage CapabilityCode = 6
	CapabilityCodeHostname        CapabilityCode = 73
)

type CapabilityValue interface {
	MarshalBinary() ([]byte, error)
}

type AddPathCapabilityList []AddPathCapability

type AddPathCapability struct {
	AFI  update.AFI
	SAFI update.SAFI
	TXRX AddPathTXRX
}

type AddPathTXRX uint8

const (
	ReceiveOnly    AddPathTXRX = 1
	SendOnly       AddPathTXRX = 2
	SendAndReceive AddPathTXRX = 3
)

type FourByteASNCapability struct {
	ASN uint32
}

type MultiProtocolCapability struct {
	AFI      update.AFI
	Reserved uint8
	SAFI     update.SAFI
}

type ExtendedMessageCapability struct{}

type HostnameCapability struct {
	Hostname   string
	DomainName string
}

type UnknownCapability struct {
	Value []byte
}
