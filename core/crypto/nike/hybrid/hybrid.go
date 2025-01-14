package hybrid

import (
	"encoding/base64"
	"io"

	"github.com/katzenpost/katzenpost/core/crypto/nike"
)

var _ nike.PrivateKey = (*privateKey)(nil)
var _ nike.PublicKey = (*publicKey)(nil)
var _ nike.Scheme = (*Scheme)(nil)

type publicKey struct {
	Scheme *Scheme
	first  nike.PublicKey
	second nike.PublicKey
}

type privateKey struct {
	Scheme *Scheme
	first  nike.PrivateKey
	second nike.PrivateKey
}

type Scheme struct {
	name   string
	first  nike.Scheme
	second nike.Scheme
}

func New(name string, first nike.Scheme, second nike.Scheme) *Scheme {
    return &Scheme{name: name, first: first, second: second}
}

func (s *Scheme) Name() string {
	return s.name
}

func (s *Scheme) PublicKeySize() int {
	return s.first.PublicKeySize() + s.second.PublicKeySize()
}

func (s *Scheme) PrivateKeySize() int {
	return s.first.PrivateKeySize() + s.second.PrivateKeySize()
}

func (s *Scheme) GeneratePrivateKey(rng io.Reader) nike.PrivateKey {
	return &privateKey{
		Scheme: s,
		first:  s.first.GeneratePrivateKey(rng),
		second: s.second.GeneratePrivateKey(rng),
	}
}

func (s *Scheme) GenerateKeyPairFromEntropy(rng io.Reader) (nike.PublicKey, nike.PrivateKey, error) {
	pubKey1, privKey1, err := s.first.GenerateKeyPairFromEntropy(rng)
	if err != nil {
		return nil, nil, err
	}
	pubKey2, privKey2, err := s.second.GenerateKeyPairFromEntropy(rng)
	if err != nil {
		return nil, nil, err
	}
	return &publicKey{
			Scheme: s,
			first:  pubKey1,
			second: pubKey2,
		}, &privateKey{
			Scheme: s,
			first:  privKey1,
			second: privKey2,
		}, nil
}

func (s *Scheme) GenerateKeyPair() (nike.PublicKey, nike.PrivateKey, error) {
	pubKey1, privKey1, err := s.first.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	pubKey2, privKey2, err := s.second.GenerateKeyPair()
	if err != nil {
		return nil, nil, err
	}
	return &publicKey{
			Scheme: s,
			first:  pubKey1,
			second: pubKey2,
		}, &privateKey{
			Scheme: s,
			first:  privKey1,
			second: privKey2,
		}, nil
}

func (s *Scheme) DeriveSecret(privKey nike.PrivateKey, pubKey nike.PublicKey) []byte {
	return append(privKey.(*privateKey).Scheme.first.DeriveSecret(privKey.(*privateKey).first, pubKey.(*publicKey).first),
		privKey.(*privateKey).Scheme.second.DeriveSecret(privKey.(*privateKey).second, pubKey.(*publicKey).second)...)
}

func (s *Scheme) DerivePublicKey(privKey nike.PrivateKey) nike.PublicKey {
	return &publicKey{
		Scheme: s,
		first:  privKey.(*privateKey).Scheme.first.DerivePublicKey(privKey.(*privateKey).first),
		second: privKey.(*privateKey).Scheme.second.DerivePublicKey(privKey.(*privateKey).second),
	}
}

func (s *Scheme) Blind(groupMember nike.PublicKey, blindingFactor nike.PrivateKey) nike.PublicKey {
	return &publicKey{
		Scheme: s,
		first:  s.first.Blind(groupMember.(*publicKey).first, blindingFactor.(*privateKey).first),
		second: s.second.Blind(groupMember.(*publicKey).second, blindingFactor.(*privateKey).second),
	}
}

func (s *Scheme) NewEmptyPublicKey() nike.PublicKey {
	return &publicKey{
		Scheme: s,
		first:  s.first.NewEmptyPublicKey(),
		second: s.second.NewEmptyPublicKey(),
	}
}

func (s *Scheme) NewEmptyPrivateKey() nike.PrivateKey {
	return &privateKey{
		Scheme: s,
		first:  s.first.NewEmptyPrivateKey(),
		second: s.second.NewEmptyPrivateKey(),
	}
}

func (s *Scheme) UnmarshalBinaryPublicKey(b []byte) (nike.PublicKey, error) {
	pubkey := s.NewEmptyPublicKey()
	err := pubkey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return pubkey, nil
}

func (s *Scheme) UnmarshalBinaryPrivateKey(b []byte) (nike.PrivateKey, error) {
	privkey := s.NewEmptyPrivateKey()
	err := privkey.FromBytes(b)
	if err != nil {
		return nil, err
	}
	return privkey, nil
}

func (p *privateKey) Public() nike.PublicKey {
	return &publicKey{
		Scheme: p.Scheme,
		first:  p.first.Public(),
		second: p.second.Public(),
	}
}

func (p *privateKey) Reset() {
	p.first.Reset()
	p.second.Reset()
}

func (p *privateKey) Bytes() []byte {
	return append(p.first.Bytes(), p.second.Bytes()...)
}

func (p *privateKey) FromBytes(b []byte) error {
	err := p.first.FromBytes(b[:p.Scheme.first.PrivateKeySize()])
	if err != nil {
		return err
	}
	return p.second.FromBytes(b[p.Scheme.first.PrivateKeySize():])
}

// MarshalBinary is an implementation of a method on the
// BinaryMarshaler interface defined in https://golang.org/pkg/encoding/
func (p *privateKey) MarshalBinary() ([]byte, error) {
	return p.Bytes(), nil
}

// UnmarshalBinary is an implementation of a method on the
// BinaryUnmarshaler interface defined in https://golang.org/pkg/encoding/
func (p *privateKey) UnmarshalBinary(data []byte) error {
	return p.FromBytes(data)
}

// MarshalText is an implementation of a method on the
// TextMarshaler interface defined in https://golang.org/pkg/encoding/
func (p *privateKey) MarshalText() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(p.Bytes())), nil
}

// UnmarshalText is an implementation of a method on the
// TextUnmarshaler interface defined in https://golang.org/pkg/encoding/
func (p *privateKey) UnmarshalText(data []byte) error {
	raw, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	return p.FromBytes(raw)
}

func (p *publicKey) Blind(blindingFactor nike.PrivateKey) error {
	err := p.first.Blind(blindingFactor.(*privateKey).first)
	if err != nil {
		p.Reset()
		return err
	}
	err = p.second.Blind(blindingFactor.(*privateKey).second)
	if err != nil {
		p.Reset()
		return err
	}
	return nil
}

func (p *publicKey) Reset() {
	p.first.Reset()
	p.second.Reset()
}

func (p *publicKey) Bytes() []byte {
	return append(p.first.Bytes(), p.second.Bytes()...)
}

func (p *publicKey) FromBytes(b []byte) error {
	err := p.first.FromBytes(b[:p.Scheme.first.PublicKeySize()])
	if err != nil {
		return err
	}
	return p.second.FromBytes(b[p.Scheme.first.PublicKeySize():])
}

// MarshalBinary is an implementation of a method on the
// BinaryMarshaler interface defined in https://golang.org/pkg/encoding/
func (p *publicKey) MarshalBinary() ([]byte, error) {
	return p.Bytes(), nil
}

// UnmarshalBinary is an implementation of a method on the
// BinaryUnmarshaler interface defined in https://golang.org/pkg/encoding/
func (p *publicKey) UnmarshalBinary(data []byte) error {
	return p.FromBytes(data)
}

// MarshalText is an implementation of a method on the
// TextMarshaler interface defined in https://golang.org/pkg/encoding/
func (p *publicKey) MarshalText() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(p.Bytes())), nil
}

// UnmarshalText is an implementation of a method on the
// TextUnmarshaler interface defined in https://golang.org/pkg/encoding/
func (p *publicKey) UnmarshalText(data []byte) error {
	raw, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	return p.FromBytes(raw)
}
