package tmpl

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

const nanoidAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// idDocs documents the identifier-generating template functions, all
// sourced from the Engine's seeded entropy.
func (e *Engine) idDocs() []FuncDoc {
	return []FuncDoc{
		{Name: "uuid", Category: CategoryIdentifiers, Returns: "string",
			Description: "Random UUIDv4, seeded from the session's PRNG.", Fn: e.uuidv4},
		{Name: "uuidv7", Category: CategoryIdentifiers, Returns: "string",
			Description: "Time-ordered UUIDv7. NOT reproducible under a fixed --seed: its rand_a field derives from a package-level wall-clock counter, not the seeded entropy source.", Fn: e.uuidv7},
		{Name: "ulid", Category: CategoryIdentifiers, Returns: "string",
			Description: "Lexicographically-sortable ULID. NOT reproducible under a fixed --seed: its 48-bit timestamp component is real wall-clock time by design.", Fn: e.ulid},
		{Name: "nanoid", Category: CategoryIdentifiers, Args: "[size]", Returns: "string",
			Description: "URL-safe random ID, default length 21.", Fn: e.nanoid},
		{Name: "objectid", Category: CategoryIdentifiers, Returns: "string",
			Description: "24-hex-char MongoDB-style ObjectID (4-byte timestamp + 5 random + 3-byte counter).", Fn: e.objectid},
		{Name: "ksuid", Category: CategoryIdentifiers, Returns: "string",
			Description: "27-char base62 K-Sortable ID (4-byte timestamp + 16 random bytes).", Fn: e.ksuid},
		{Name: "messageID", Category: CategoryIdentifiers, Args: "[domain]", Returns: "string",
			Description: `RFC 5322 email Message-Id value, e.g. <hex@domain>. domain defaults to "maelsink.local".`, Fn: e.messageID},
	}
}

func (e *Engine) uuidv4() (string, error) {
	id, err := uuid.NewRandomFromReader(e.entropy)
	if err != nil {
		return "", fmt.Errorf("tmpl: uuidv4: %w", err)
	}
	return id.String(), nil
}

// uuidv7 generates a version 7 UUID via google/uuid's NewV7FromReader. Note
// that library derives the 12-bit "rand_a" sequence field from a
// package-level monotonic wall-clock counter rather than purely from the
// entropy reader, so uuidv7 output (unlike every other id/fake function in
// this package) is not byte-for-byte reproducible across two Engine
// instances constructed with the same seed.
func (e *Engine) uuidv7() (string, error) {
	id, err := uuid.NewV7FromReader(e.entropy)
	if err != nil {
		return "", fmt.Errorf("tmpl: uuidv7: %w", err)
	}
	return id.String(), nil
}

// ulid generates a ULID via oklog/ulid's ulid.New. Note that ULID's whole
// purpose is lexicographic sortability by real creation time, so its 48-bit
// timestamp component is deliberately derived from time.Now() rather than
// from the seeded entropy reader — unlike every other id/fake function in
// this package, ulid output (like uuidv7, see the comment above) is not
// byte-for-byte reproducible across two Engine instances constructed with
// the same seed, because two constructions microseconds apart can straddle
// a millisecond boundary. Deriving the timestamp from the seeded PRNG
// instead would make ULIDs deterministic but would defeat their actual
// real-world purpose, so this is accepted the same way uuidv7's
// non-determinism is.
func (e *Engine) ulid() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), e.entropy)
	if err != nil {
		return "", fmt.Errorf("tmpl: ulid: %w", err)
	}
	return id.String(), nil
}

// nanoid generates a URL-safe nanoid of the given size (default 21).
func (e *Engine) nanoid(size ...int) (string, error) {
	n := 21
	if len(size) > 0 && size[0] > 0 {
		n = size[0]
	}
	buf := make([]byte, n)
	if _, err := e.entropy.Read(buf); err != nil {
		return "", fmt.Errorf("tmpl: nanoid: %w", err)
	}
	for i, b := range buf {
		buf[i] = nanoidAlphabet[int(b)%len(nanoidAlphabet)]
	}
	return string(buf), nil
}

// objectid generates a 24-hex-char Mongo-style ObjectID: 4-byte timestamp,
// 5 random bytes, 3-byte counter (here also random, since this is a
// stateless template function rather than a real driver).
func (e *Engine) objectid() (string, error) {
	var buf [12]byte

	ts := uint32(time.Now().Unix())
	buf[0] = byte(ts >> 24)
	buf[1] = byte(ts >> 16)
	buf[2] = byte(ts >> 8)
	buf[3] = byte(ts)

	if _, err := e.entropy.Read(buf[4:9]); err != nil {
		return "", fmt.Errorf("tmpl: objectid: %w", err)
	}
	if _, err := e.entropy.Read(buf[9:12]); err != nil {
		return "", fmt.Errorf("tmpl: objectid: %w", err)
	}

	return hex.EncodeToString(buf[:]), nil
}

// ksuid generates a 27-char base62 KSUID-shaped identifier: 4-byte
// timestamp followed by 16 random bytes.
func (e *Engine) ksuid() (string, error) {
	var raw [20]byte

	ts := uint32(time.Now().Unix())
	raw[0] = byte(ts >> 24)
	raw[1] = byte(ts >> 16)
	raw[2] = byte(ts >> 8)
	raw[3] = byte(ts)

	if _, err := e.entropy.Read(raw[4:]); err != nil {
		return "", fmt.Errorf("tmpl: ksuid: %w", err)
	}

	return base62Encode(raw[:], 27), nil
}

// base62Encode encodes data as a base62 string, left-padded with '0' to at
// least minLen characters.
func base62Encode(data []byte, minLen int) string {
	// Treat data as a big-endian big integer and repeatedly divide by 62.
	num := make([]byte, len(data))
	copy(num, data)

	var out []byte
	allZero := func(b []byte) bool {
		for _, v := range b {
			if v != 0 {
				return false
			}
		}
		return true
	}

	for !allZero(num) {
		var rem int
		for i := 0; i < len(num); i++ {
			cur := rem*256 + int(num[i])
			num[i] = byte(cur / 62)
			rem = cur % 62
		}
		out = append(out, base62Alphabet[rem])
	}

	// Reverse.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	for len(out) < minLen {
		out = append([]byte{base62Alphabet[0]}, out...)
	}

	return string(out)
}

// messageID generates an RFC-5322-style message ID: <randomhex@domain>.
func (e *Engine) messageID(domain ...string) (string, error) {
	d := "maelsink.local"
	if len(domain) > 0 && domain[0] != "" {
		d = domain[0]
	}

	buf := make([]byte, 16)
	if _, err := e.entropy.Read(buf); err != nil {
		return "", fmt.Errorf("tmpl: messageID: %w", err)
	}

	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(buf), d), nil
}
