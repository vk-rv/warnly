// Package session provides session management functionality, including secure cookies,
// session storage, and session encoding/decoding using CBOR and secure cookies.
package session

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/vk-rv/warnly/internal/warnly"
)

type Session struct {
	store   Store
	Options *Options
	ID      string
	name    string
	Values  Values
	IsNew   bool
}

type Values struct {
	User warnly.User `cbor:"user"`
}

type Options struct {
	Path        string
	Domain      string
	MaxAge      int
	Secure      bool
	HTTPOnly    bool
	Partitioned bool
	SameSite    http.SameSite
}

type Store interface {
	Get(r *http.Request, name string) (*Session, error)
	New(r *http.Request, name string) (*Session, error)
	Save(r *http.Request, w http.ResponseWriter, s *Session) error
}

type Codec interface {
	Encode(name string, value any) (string, error)
	Decode(name, value string, dst any) error
}

func CodecsFromPairs(now func() time.Time, keyPairs ...[]byte) []Codec {
	codecs := make([]Codec, len(keyPairs)/2+len(keyPairs)%2)
	for i := 0; i < len(keyPairs); i += 2 {
		var blockKey []byte
		if i+1 < len(keyPairs) {
			blockKey = keyPairs[i+1]
		}
		codecs[i/2] = NewSecureCookie(keyPairs[i], blockKey, now)
	}

	return codecs
}

// CBOREncoder encodes and decodes values using CBOR.
type CBOREncoder struct{}

// Serialize encodes a value using CBOR.
func (e CBOREncoder) Serialize(src any) ([]byte, error) {
	var buf bytes.Buffer
	enc := cbor.NewEncoder(&buf)
	if err := enc.Encode(src); err != nil {
		return nil, fmt.Errorf("cbor: serialize problem %w", err)
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a value using CBOR.
func (e CBOREncoder) Deserialize(src []byte, dst any) error {
	dec := cbor.NewDecoder(bytes.NewReader(src))
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("cbor: deserialize problem %w", err)
	}
	return nil
}

// SecureCookie encodes and decodes authenticated and optionally encrypted
// cookie values.
type SecureCookie struct {
	block     cipher.Block
	err       error
	sz        Serializer
	hashFunc  func() hash.Hash
	timeFunc  func() int64
	now       func() time.Time
	hashKey   []byte
	blockKey  []byte
	maxLength int
	maxAge    int64
	minAge    int64
}

func NewSecureCookie(hashKey, blockKey []byte, now func() time.Time) *SecureCookie {
	s := &SecureCookie{
		hashKey:   hashKey,
		blockKey:  blockKey,
		hashFunc:  sha256.New,
		maxAge:    86400 * 30,
		maxLength: 4096,
		sz:        CBOREncoder{},
		now:       now,
	}
	if blockKey != nil {
		s.BlockFunc(aes.NewCipher)
	}
	return s
}

func (s *SecureCookie) BlockFunc(f func([]byte) (cipher.Block, error)) *SecureCookie {
	if block, err := f(s.blockKey); err == nil {
		s.block = block
	}

	return s
}

// Serializer provides an interface for providing custom serializers for cookie
// values.
type Serializer interface {
	Serialize(src any) ([]byte, error)
	Deserialize(src []byte, dst any) error
}

func (s *SecureCookie) Encode(name string, value any) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.hashKey == nil {
		return "", errors.New("securecookie: hashKey is not set")
	}
	var err error
	var b []byte
	// 1. Serialize.
	if b, err = s.sz.Serialize(value); err != nil {
		return "", fmt.Errorf("%s: %w", "serialization err", err)
	}
	// 2. Encrypt (optional).
	if s.block != nil {
		if b, err = encrypt(s.block, b); err != nil {
			return "", fmt.Errorf("%s: %w", "encryption err", err)
		}
	}
	b = encode(b)
	// 3. Create MAC for "name|date|value". Extra pipe to be used later.
	b = []byte(fmt.Sprintf("%s|%d|%s|", name, s.timestamp(), b))
	mac := createMac(hmac.New(s.hashFunc, s.hashKey), b[:len(b)-1])
	// Append mac, remove name.
	b = append(b, mac...)[len(name)+1:]
	// 4. Encode to base64.
	b = encode(b)
	// 5. Check length.
	if s.maxLength != 0 && len(b) > s.maxLength {
		return "", fmt.Errorf("%s: %d", "err encoded too long", len(b))
	}

	return string(b), nil
}

func (s *SecureCookie) Decode(name, value string, dst any) error {
	if s.err != nil {
		return s.err
	}
	if s.hashKey == nil {
		return errors.New("securecookie: hashKey is not set")
	}
	if s.maxLength != 0 && len(value) > s.maxLength {
		return fmt.Errorf("%s: %d", "err value too long", len(value))
	}

	b, err := decode([]byte(value))
	if err != nil {
		return err
	}

	parts, err := splitAndVerifyMac(b, name, s.hashFunc, s.hashKey)
	if err != nil {
		return err
	}

	t1, err := strconv.ParseInt(string(parts[0]), 10, 64)
	if err != nil {
		return fmt.Errorf("%s: %w", "err invalid timestamp", err)
	}
	if err = verifyTimestamp(t1, s.timestamp(), s.minAge, s.maxAge); err != nil {
		return err
	}

	b, err = decode(parts[1])
	if err != nil {
		return err
	}
	if s.block != nil {
		if b, err = decrypt(s.block, b); err != nil {
			return err
		}
	}

	return s.deserialize(b, dst)
}

func GenerateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

// CookieStore stores sessions using secure cookies.
type CookieStore struct {
	Now     func() time.Time
	Options *Options
	Codecs  []Codec
}

func NewCookieStore(now func() time.Time, keyPairs ...[]byte) *CookieStore {
	cs := &CookieStore{
		Now:    now,
		Codecs: CodecsFromPairs(now, keyPairs...),
		Options: &Options{
			Path:     "/",
			MaxAge:   86400 * 30,
			SameSite: http.SameSiteNoneMode,
			Secure:   true,
		},
	}

	cs.MaxAge(cs.Options.MaxAge)
	return cs
}

// New returns a session for the given name without adding it to the registry.
//
// The difference between New() and Get() is that calling New() twice will
// decode the session data twice, while Get() registers and reuses the same
// decoded session after the first call.
func (s *CookieStore) New(r *http.Request, name string) (*Session, error) {
	session := NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = DecodeMulti(name, c.Value, &session.Values, s.Codecs...)
		if err == nil {
			session.IsNew = false
		}
	}
	return session, err
}

func NewSession(store Store, name string) *Session {
	return &Session{
		Values:  Values{},
		store:   store,
		name:    name,
		Options: new(Options),
	}
}

// Name returns the name used to register the session.
func (s *Session) Name() string {
	return s.name
}

func (s *CookieStore) Get(r *http.Request, name string) (*Session, error) {
	reg, err := GetRegistry(r)
	if err != nil {
		return nil, err
	}
	return reg.Get(s, name)
}

// contextKey is the type used to store the registry in the context.
type contextKey int

// registryKey is the key used to store the registry in the context.
const registryKey contextKey = 0

type sessionInfo struct {
	s *Session
	e error
}

// Registry stores sessions used during a request.
type Registry struct {
	request  *http.Request
	sessions map[string]sessionInfo
}

func GetRegistry(r *http.Request) (*Registry, error) {
	ctx := r.Context()
	registry := ctx.Value(registryKey)
	if registry != nil {
		reg, ok := registry.(*Registry)
		if !ok {
			return nil, errors.New("sessions: invalid registry")
		}
		return reg, nil
	}
	newRegistry := &Registry{
		request:  r,
		sessions: make(map[string]sessionInfo),
	}
	*r = *r.WithContext(context.WithValue(ctx, registryKey, newRegistry))
	return newRegistry, nil
}

// Get registers and returns a session for the given name and session store.
//
// It returns a new session if there are no sessions registered for the name.
func (s *Registry) Get(store Store, name string) (session *Session, err error) {
	if !isCookieNameValid(name) {
		return nil, fmt.Errorf("sessions: invalid character in cookie name: %s", name)
	}
	if info, ok := s.sessions[name]; ok {
		session, err = info.s, info.e
	} else {
		session, err = store.New(s.request, name)
		session.name = name
		s.sessions[name] = sessionInfo{s: session, e: err}
	}
	session.store = store
	return session, err
}

var isTokenTable = [127]bool{
	'!':  true,
	'#':  true,
	'$':  true,
	'%':  true,
	'&':  true,
	'\'': true,
	'*':  true,
	'+':  true,
	'-':  true,
	'.':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'W':  true,
	'V':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'^':  true,
	'_':  true,
	'`':  true,
	'a':  true,
	'b':  true,
	'c':  true,
	'd':  true,
	'e':  true,
	'f':  true,
	'g':  true,
	'h':  true,
	'i':  true,
	'j':  true,
	'k':  true,
	'l':  true,
	'm':  true,
	'n':  true,
	'o':  true,
	'p':  true,
	'q':  true,
	'r':  true,
	's':  true,
	't':  true,
	'u':  true,
	'v':  true,
	'w':  true,
	'x':  true,
	'y':  true,
	'z':  true,
	'|':  true,
	'~':  true,
}

// Save adds a single session to the response.
func (s *CookieStore) Save(r *http.Request, w http.ResponseWriter, session *Session) error {
	encoded, err := EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, NewCookie(session.Name(), encoded, session.Options, s.Now))

	return nil
}

// NewCookie returns an http.Cookie with the options set. It also sets
// the Expires field calculated based on the MaxAge value, for Internet
// Explorer compatibility.
func NewCookie(name, value string, options *Options, now func() time.Time) *http.Cookie {
	cookie := newCookieFromOptions(name, value, options)
	if options.MaxAge > 0 {
		d := time.Duration(options.MaxAge) * time.Second
		cookie.Expires = now().Add(d)
	} else if options.MaxAge < 0 {
		// Set it to the past to expire now.
		cookie.Expires = time.Unix(1, 0)
	}
	return cookie
}

func EncodeMulti(name string, value any, codecs ...Codec) (string, error) {
	if len(codecs) == 0 {
		return "", errors.New("securecookie: no codecs provided")
	}
	errs := make([]error, 0, len(codecs))
	for _, codec := range codecs {
		encoded, err := codec.Encode(name, value)
		if err == nil {
			return encoded, nil
		}
		errs = append(errs, err)
	}
	return "", errs[0]
}

// MaxAge restricts the maximum age, in seconds, for the cookie value.
//
// Default is 86400 * 30. Set it to 0 for no restriction.
func (s *SecureCookie) MaxAge(value int) *SecureCookie {
	s.maxAge = int64(value)
	return s
}

func (s *CookieStore) MaxAge(age int) {
	s.Options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range s.Codecs {
		if sc, ok := codec.(*SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}

func DecodeMulti(name, value string, dst any, codecs ...Codec) error {
	if len(codecs) == 0 {
		return errors.New("securecookie: no codecs provided")
	}

	errs := make([]error, 0, len(codecs))
	for _, codec := range codecs {
		err := codec.Decode(name, value, dst)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errs[0]
}

// newCookieFromOptions returns an http.Cookie with the options set.
func newCookieFromOptions(name, value string, options *Options) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HTTPOnly,
		SameSite: options.SameSite,
	}
}

func (s *SecureCookie) deserialize(b []byte, dst any) error {
	if err := s.sz.Deserialize(b, dst); err != nil {
		return fmt.Errorf("error deserializing: %w", err)
	}
	return nil
}

func splitAndVerifyMac(b []byte, name string, hashFunc func() hash.Hash, hashKey []byte) ([][]byte, error) {
	parts := bytes.SplitN(b, []byte("|"), 3)
	if len(parts) != 3 {
		return nil, errors.New("err invalid hmac")
	}
	h := hmac.New(hashFunc, hashKey)
	b = append([]byte(name+"|"), b[:len(b)-len(parts[2])-1]...)
	if err := verifyMac(h, b, parts[2]); err != nil {
		return nil, err
	}
	return parts, nil
}

func verifyTimestamp(t1, t2, minAge, maxAge int64) error {
	if minAge != 0 && t1 > t2-minAge {
		return fmt.Errorf("%s: %d", "err timestamp too new", t1)
	}
	if maxAge != 0 && t1 < t2-maxAge {
		return fmt.Errorf("%s: %d", "err timestamp too old", t1)
	}
	return nil
}

// decode decodes a cookie using base64.
func decode(value []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(value)))
	b, err := base64.URLEncoding.Decode(decoded, value)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return decoded[:b], nil
}

// encode encodes a value using base64.
func encode(value []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(value)))
	base64.URLEncoding.Encode(encoded, value)
	return encoded
}

// createMac creates a message authentication code (MAC).
func createMac(h hash.Hash, value []byte) []byte {
	h.Write(value)
	return h.Sum(nil)
}

func (s *SecureCookie) timestamp() int64 {
	if s.timeFunc == nil {
		return s.now().UTC().Unix()
	}
	return s.timeFunc()
}

func encrypt(block cipher.Block, value []byte) ([]byte, error) {
	iv := GenerateRandomKey(block.BlockSize())
	if iv == nil {
		return nil, errors.New("securecookie: failed to generate random iv")
	}
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	return append(iv, value...), nil
}

func isCookieNameValid(raw string) bool {
	if raw == "" {
		return false
	}
	return strings.IndexFunc(raw, isNotToken) < 0
}

func isToken(r rune) bool {
	i := int(r)
	return i < len(isTokenTable) && isTokenTable[i]
}

func isNotToken(r rune) bool {
	return !isToken(r)
}

func verifyMac(h hash.Hash, value, mac []byte) error {
	mac2 := createMac(h, value)
	if len(mac) == len(mac2) && subtle.ConstantTimeCompare(mac, mac2) == 1 {
		return nil
	}
	return errors.New("mac is invalid")
}

func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}
	return nil, fmt.Errorf("securecookie: invalid value length: %d", len(value))
}
