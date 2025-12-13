package open

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net/netip"
)

func (m Msg) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, m.Version); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, m.ASN); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, m.HoldTime); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, m.RouterID); err != nil {
		return nil, err
	}

	optionalParameterBytes := make([]byte, 0)
	for _, parameter := range m.OptionalParameters {
		b, err := parameter.MarshalBinary()
		if err != nil {
			return nil, err
		}
		optionalParameterBytes = append(optionalParameterBytes, b...)
	}

	if err := binary.Write(&b, binary.BigEndian, uint8(len(optionalParameterBytes))); err != nil {
		return nil, err
	}
	_, err := b.Write(optionalParameterBytes)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), err
}

func (p OptionalParameter) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &p.ParameterType); err != nil {
		return nil, err
	}

	valueBytes, err := p.ParameterValue.MarshalBinary()
	if err != nil {
		return nil, err
	}

	p.ParameterLength = uint8(len(valueBytes))
	if err := binary.Write(&b, binary.BigEndian, &p.ParameterLength); err != nil {
		return nil, err
	}
	_, err = b.Write(valueBytes)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c CapabilityList) MarshalBinary() ([]byte, error) {
	var b = make([]byte, 0)
	for _, o := range c.List {
		oBytes, err := o.MarshalBinary()
		if err != nil {
			return nil, err
		}
		b = append(b, oBytes...)
	}
	return b, nil
}

func (c CapabilityOptionalParameter) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &c.CapabilityCode); err != nil {
		return nil, err
	}

	valueBytes, err := c.CapabilityValue.MarshalBinary()
	if err != nil {
		return nil, err
	}

	c.CapabilityLength = uint8(len(valueBytes))

	if err := binary.Write(&b, binary.BigEndian, &c.CapabilityLength); err != nil {
		return nil, err
	}

	_, err = b.Write(valueBytes)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (a AddPathCapabilityList) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0)
	for _, c := range a {
		m, err := c.MarshalBinary()
		if err != nil {
			return nil, err
		}
		b = append(b, m...)
	}
	return b, nil
}

func (c AddPathCapability) MarshalBinary() ([]byte, error) {
	return marshalStructValueCapability(c)
}

func (c FourByteASNCapability) MarshalBinary() ([]byte, error) {
	return marshalStructValueCapability(c)
}

func (c MultiProtocolCapability) MarshalBinary() ([]byte, error) {
	return marshalStructValueCapability(c)
}

func (c ExtendedMessageCapability) MarshalBinary() ([]byte, error) {
	return []byte{}, nil
}

func (c HostnameCapability) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	writeString := func(s string) error {
		if len(s) > 255 {
			return fmt.Errorf("hostname capability string too long: %d", len(s))
		}
		if err := binary.Write(&b, binary.BigEndian, uint8(len(s))); err != nil {
			return err
		}
		_, err := b.Write([]byte(s))
		return err
	}
	if err := writeString(c.Hostname); err != nil {
		return []byte{}, err
	}
	if err := writeString(c.DomainName); err != nil {
		return []byte{}, err
	}
	return b.Bytes(), nil
}

func (e ExtendedNextHopCapabilityList) MarshalBinary() ([]byte, error) {
	b := make([]byte, 0)
	for _, c := range e {
		m, err := c.MarshalBinary()
		if err != nil {
			return nil, err
		}
		b = append(b, m...)
	}
	return b, nil
}

func (c ExtendedNextHopCapability) MarshalBinary() ([]byte, error) {
	// The SAFI field in this capability is 2 octets in contrast to the regular SAFI field, thus
	// necessitating individual handling of the fields
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, uint16(c.AFI)); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, uint16(c.SAFI)); err != nil {
		return nil, err
	}
	if err := binary.Write(&b, binary.BigEndian, uint16(c.NextHopAFI)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c UnknownCapability) MarshalBinary() ([]byte, error) {
	return c.Value, nil
}

type structValueCapability interface {
	AddPathCapability | FourByteASNCapability | MultiProtocolCapability
}

func marshalStructValueCapability[T structValueCapability](c T) ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func routerIDFromNetAddr(routerID netip.Addr) RouterID {
	routerIDBytes := routerID.As4()
	routerIDUint32 := binary.BigEndian.Uint32(routerIDBytes[:])
	return RouterID(routerIDUint32)
}

func (id RouterID) String() string {
	return id.ToNetAddr().String()
}

func (id RouterID) ToNetAddr() netip.Addr {
	var addr [4]byte
	binary.BigEndian.PutUint32(addr[:], uint32(id))
	return netip.AddrFrom4(addr)
}
