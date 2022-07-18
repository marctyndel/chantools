// Package bip39 is the Golang implementation of the BIP39 spec.
// This code was copied from https://github.com/tyler-smith/go-bip39 which is
// also MIT licensed.
//
// The official BIP39 spec can be found at
// https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki
package bip39

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var (
	// Some bitwise operands for working with big.Ints.
	shift11BitsMask = big.NewInt(2048)
	bigOne          = big.NewInt(1)

	// Used to isolate the checksum bits from the entropy+checksum byte
	// array.
	wordLengthChecksumMasksMapping = map[int]*big.Int{
		12: big.NewInt(15),
		15: big.NewInt(31),
		18: big.NewInt(63),
		21: big.NewInt(127),
		24: big.NewInt(255),
	}
	// Used to use only the desired x of 8 available checksum bits.
	// 256 bit (word length 24) requires all 8 bits of the checksum,
	// and thus no shifting is needed for it (we would get a divByZero crash
	// if we did).
	wordLengthChecksumShiftMapping = map[int]*big.Int{
		12: big.NewInt(16),
		15: big.NewInt(8),
		18: big.NewInt(4),
		21: big.NewInt(2),
	}
)

var (
	// ErrInvalidMnemonic is returned when trying to use a malformed
	// mnemonic.
	ErrInvalidMnemonic = errors.New("invalid mnenomic")

	// ErrChecksumIncorrect is returned when entropy has the incorrect
	// checksum.
	ErrChecksumIncorrect = errors.New("checksum incorrect")
)

// EntropyFromMnemonic takes a mnemonic generated by this library,
// and returns the input entropy used to generate the given mnemonic.
// An error is returned if the given mnemonic is invalid.
func EntropyFromMnemonic(mnemonic string) ([]byte, error) {
	mnemonicSlice, isValid := splitMnemonicWords(mnemonic)
	if !isValid {
		return nil, ErrInvalidMnemonic
	}

	wordMap := make(map[string]int)
	for i, v := range English {
		wordMap[v] = i
	}

	// Decode the words into a big.Int.
	b := big.NewInt(0)
	for _, v := range mnemonicSlice {
		index, found := wordMap[v]
		if !found {
			return nil, fmt.Errorf("word `%v` not found in "+
				"reverse map", v)
		}
		var wordBytes [2]byte
		binary.BigEndian.PutUint16(wordBytes[:], uint16(index))
		b = b.Mul(b, shift11BitsMask)
		b = b.Or(b, big.NewInt(0).SetBytes(wordBytes[:]))
	}

	// Build and add the checksum to the big.Int.
	checksum := big.NewInt(0)
	checksumMask := wordLengthChecksumMasksMapping[len(mnemonicSlice)]
	checksum = checksum.And(b, checksumMask)

	b.Div(b, big.NewInt(0).Add(checksumMask, bigOne))

	// The entropy is the underlying bytes of the big.Int. Any upper bytes
	// of all 0's are not returned so we pad the beginning of the slice with
	// empty bytes if necessary.
	entropy := b.Bytes()
	entropy = padByteSlice(entropy, len(mnemonicSlice)/3*4)

	// Generate the checksum and compare with the one we got from the
	// mneomnic.
	entropyChecksumBytes := computeChecksum(entropy)
	entropyChecksum := big.NewInt(int64(entropyChecksumBytes[0]))
	if l := len(mnemonicSlice); l != 24 {
		checksumShift := wordLengthChecksumShiftMapping[l]
		entropyChecksum.Div(entropyChecksum, checksumShift)
	}

	if checksum.Cmp(entropyChecksum) != 0 {
		return nil, ErrChecksumIncorrect
	}

	return entropy, nil
}

func computeChecksum(data []byte) []byte {
	hasher := sha256.New()
	_, _ = hasher.Write(data)
	return hasher.Sum(nil)
}

// padByteSlice returns a byte slice of the given size with contents of the
// given slice left padded and any empty spaces filled with 0's.
func padByteSlice(slice []byte, length int) []byte {
	offset := length - len(slice)
	if offset <= 0 {
		return slice
	}
	newSlice := make([]byte, length)
	copy(newSlice[offset:], slice)
	return newSlice
}

func splitMnemonicWords(mnemonic string) ([]string, bool) {
	// Create a list of all the words in the mnemonic sentence.
	words := strings.Fields(mnemonic)

	// Get num of words.
	numOfWords := len(words)

	// The number of words should be 12, 15, 18, 21 or 24.
	if numOfWords%3 != 0 || numOfWords < 12 || numOfWords > 24 {
		return nil, false
	}
	return words, true
}
