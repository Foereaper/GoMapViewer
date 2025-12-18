package mpq

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

/* =========================
   Crypto
   ========================= */

var cryptTable [0x500]uint32

func init() {
	var seed uint32 = 0x00100001
	for i := 0; i < 0x100; i++ {
		for j := 0; j < 5; j++ {
			seed = (seed*125 + 3) % 0x2AAAAB
			temp1 := (seed & 0xFFFF) << 16
			seed = (seed*125 + 3) % 0x2AAAAB
			temp2 := seed & 0xFFFF
			cryptTable[i+j*0x100] = temp1 | temp2
		}
	}
}

const (
	MPQ_HASH_TABLE_OFFSET = 0
	MPQ_HASH_NAME_A       = 1
	MPQ_HASH_NAME_B       = 2
	MPQ_HASH_FILE_KEY     = 3
)

func mpqHashString(str string, hashType uint32) uint32 {
	var seed1 uint32 = 0x7FED7FED
	var seed2 uint32 = 0xEEEEEEEE

	str = strings.ToUpper(str)
	for i := 0; i < len(str); i++ {
		ch := str[i]
		value := cryptTable[(hashType<<8)+uint32(ch)]
		seed1 = value ^ (seed1 + seed2)
		seed2 = uint32(ch) + seed1 + seed2 + (seed2 << 5) + 3
	}
	return seed1
}

func mpqDecrypt(data []byte, key uint32) {
	var seed uint32 = 0xEEEEEEEE
	for i := 0; i+4 <= len(data); i += 4 {
		seed += cryptTable[0x400+(key&0xFF)]
		value := binary.LittleEndian.Uint32(data[i:])
		value ^= key + seed
		binary.LittleEndian.PutUint32(data[i:], value)
		key = ((^key << 21) + 0x11111111) | (key >> 11)
		seed = value + seed + (seed << 5) + 3
	}
}

/* =========================
   MPQ constants
   ========================= */

const (
	MPQ_FILE_COMPRESS_MASK = 0x0000FF00
	MPQ_FILE_ENCRYPTED     = 0x00010000
	MPQ_FILE_FIX_KEY       = 0x00020000
	MPQ_FILE_SINGLE_UNIT   = 0x01000000
	MPQ_FILE_SECTOR_CRC    = 0x04000000
)

const (
	MPQ_COMPRESSION_ZLIB = 0x02
)

/* =========================
   Structures
   ========================= */

type Header struct {
	ID                uint32
	HeaderSize        uint32
	ArchiveSize       uint32
	FormatVersion     uint16
	SectorSizeShift   uint16
	HashTableOffset   uint32
	BlockTableOffset  uint32
	HashTableEntries  uint32
	BlockTableEntries uint32
}

type HashEntry struct {
	NameA    uint32
	NameB    uint32
	Locale   uint16
	Platform uint16
	BlockIdx uint32
}

type BlockEntry struct {
	Offset            uint32
	CompressedSize    uint32
	UncompressedSize  uint32
	Flags             uint32
}

type MPQ struct {
	f          *os.File
	header     Header
	hashTable  []HashEntry
	blockTable []BlockEntry
	archivePos int64
}

/* =========================
   Open / Close
   ========================= */

func Open(path string) (*MPQ, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	mpq := &MPQ{f: f}

	if err := binary.Read(f, binary.LittleEndian, &mpq.header); err != nil {
		return nil, err
	}

	if mpq.header.ID != 0x1A51504D {
		return nil, errors.New("not an MPQ archive")
	}

	mpq.archivePos, _ = f.Seek(0, io.SeekCurrent)
	mpq.archivePos -= int64(mpq.header.HeaderSize)

	if err := mpq.readTables(); err != nil {
		return nil, err
	}

	return mpq, nil
}

func (m *MPQ) Close() error {
	return m.f.Close()
}

/* =========================
   Tables
   ========================= */

func (m *MPQ) readTables() error {
	m.hashTable = make([]HashEntry, m.header.HashTableEntries)
	m.blockTable = make([]BlockEntry, m.header.BlockTableEntries)

	hashKey := mpqHashString("(hash table)", MPQ_HASH_FILE_KEY)
	blockKey := mpqHashString("(block table)", MPQ_HASH_FILE_KEY)

	if err := m.readEncryptedTable(
		int64(m.header.HashTableOffset),
		m.hashTable,
		hashKey,
	); err != nil {
		return err
	}

	if err := m.readEncryptedTable(
		int64(m.header.BlockTableOffset),
		m.blockTable,
		blockKey,
	); err != nil {
		return err
	}

	return nil
}

func (m *MPQ) readEncryptedTable(offset int64, table interface{}, key uint32) error {
	size := binary.Size(table)
	buf := make([]byte, size)

	if _, err := m.f.ReadAt(buf, offset); err != nil {
		return err
	}

	mpqDecrypt(buf, key)
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, table)
}

/* =========================
   File lookup
   ========================= */

func (m *MPQ) findHashEntries(name string) []HashEntry {
	name = strings.ReplaceAll(name, "/", "\\")
	hashA := mpqHashString(name, MPQ_HASH_NAME_A)
	hashB := mpqHashString(name, MPQ_HASH_NAME_B)

	start := mpqHashString(name, MPQ_HASH_TABLE_OFFSET) % m.header.HashTableEntries
	var matches []HashEntry

	for i := uint32(0); i < m.header.HashTableEntries; i++ {
		h := m.hashTable[(start+i)%m.header.HashTableEntries]

		if h.BlockIdx == 0xFFFFFFFF {
			break
		}
		if h.BlockIdx == 0xFFFFFFFE {
			continue
		}
		if h.NameA == hashA && h.NameB == hashB {
			matches = append(matches, h)
		}
	}
	return matches
}

/* =========================
   ReadFile
   ========================= */

func (m *MPQ) ReadFile(name string) ([]byte, error) {
	candidates := m.findHashEntries(name)
	if len(candidates) == 0 {
		return nil, errors.New("file not found")
	}

	var h HashEntry
	found := false
	for _, e := range candidates {
		if e.Locale == 0 {
			h = e
			found = true
			break
		}
	}
	if !found {
		h = candidates[0]
	}

	block := m.blockTable[h.BlockIdx]
	fileOffset := int64(block.Offset)

	if block.Flags&MPQ_FILE_SINGLE_UNIT != 0 {
		raw := make([]byte, block.CompressedSize)
		if _, err := m.f.ReadAt(raw, fileOffset); err != nil {
			return nil, err
		}

		if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
			mpqDecrypt(raw, m.fileKey(name, block, fileOffset))
		}

		if block.Flags&MPQ_FILE_COMPRESS_MASK == 0 {
			return raw, nil
		}

		return decompressSingleUnit(raw, block.UncompressedSize)
	}

	if block.Flags&MPQ_FILE_COMPRESS_MASK != 0 {
		return m.readSectorCompressedFile(fileOffset, block, name)
	}

	// Plain file
	out := make([]byte, block.UncompressedSize)
	_, err := m.f.ReadAt(out, fileOffset)
	return out, err
}

/* =========================
   Sector-compressed reader
   ========================= */

func (m *MPQ) readSectorCompressedFile(offset int64, block BlockEntry, name string) ([]byte, error) {
	sectorSize := uint32(512) << m.header.SectorSizeShift
	sectorCount := (block.UncompressedSize + sectorSize - 1) / sectorSize

	tableDWORDs := sectorCount + 1
	if block.Flags&MPQ_FILE_SECTOR_CRC != 0 {
		tableDWORDs++
	}

	table := make([]byte, tableDWORDs*4)
	if _, err := m.f.ReadAt(table, offset); err != nil {
		return nil, err
	}

	key := m.fileKey(name, block, offset)
	if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
		mpqDecrypt(table, key-1)
	}

	offsets := make([]uint32, tableDWORDs)
	for i := range offsets {
		offsets[i] = binary.LittleEndian.Uint32(table[i*4:])
	}

	out := make([]byte, 0, block.UncompressedSize)

	for i := uint32(0); i < sectorCount; i++ {
		start, end := offsets[i], offsets[i+1]
		size := end - start

		sector := make([]byte, size)
		readPos := offset + int64(int32(start))
		if _, err := m.f.ReadAt(sector, readPos); err != nil {
			return nil, err
		}

		if block.Flags&MPQ_FILE_ENCRYPTED != 0 {
			mpqDecrypt(sector, key+i)
		}

		expected := sectorSize
		remain := block.UncompressedSize - i*sectorSize
		if remain < expected {
			expected = remain
		}

		if size == expected {
			out = append(out, sector...)
			continue
		}

		switch sector[0] {
		case MPQ_COMPRESSION_ZLIB:
			r, err := zlib.NewReader(bytes.NewReader(sector[1:]))
			if err != nil {
				return nil, err
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				return nil, err
			}
			out = append(out, data...)
		default:
			return nil, fmt.Errorf("unsupported compression: 0x%02X", sector[0])
		}
	}

	return out[:block.UncompressedSize], nil
}

/* =========================
   Helpers
   ========================= */

func decompressSingleUnit(data []byte, expected uint32) ([]byte, error) {
	switch data[0] {
	case MPQ_COMPRESSION_ZLIB:
		r, err := zlib.NewReader(bytes.NewReader(data[1:]))
		if err != nil {
			return nil, err
		}
		out, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			return nil, err
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported compression: 0x%02X", data[0])
	}
}

func baseNameForKey(name string) string {
	name = strings.ReplaceAll(name, "/", "\\")
	if i := strings.LastIndex(name, "\\"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func (m *MPQ) fileKey(name string, block BlockEntry, absOffset int64) uint32 {
	key := mpqHashString(baseNameForKey(name), MPQ_HASH_FILE_KEY)
	if block.Flags&MPQ_FILE_FIX_KEY != 0 {
		key = (key + uint32(absOffset-m.archivePos)) ^ block.UncompressedSize
	}
	return key
}
