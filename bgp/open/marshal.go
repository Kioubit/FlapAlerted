package open

import (
	"bytes"
	"encoding/binary"
	"net/netip"
)

func (id RouterID) String() string {
	var addr [4]byte
	binary.BigEndian.PutUint32(addr[:], uint32(id))
	return netip.AddrFrom4(addr).String()
}

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
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c FourByteASNCapability) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c MultiProtocolCapability) MarshalBinary() ([]byte, error) {
	var b bytes.Buffer
	if err := binary.Write(&b, binary.BigEndian, &c); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (c UnknownCapability) MarshalBinary() ([]byte, error) {
	return c.Value, nil
}
