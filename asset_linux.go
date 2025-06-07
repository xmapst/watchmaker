package watchmaker

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"log"
)

const textSection = ".text"
const relocationSection = ".rela.text"

// LoadFakeImageFromEmbedFs builds FakeImage from the embed filesystem. It parses the ELF file and extract the variables from the relocation section, reserves the space for them at the end of content, then calculates and saves offsets as "manually relocation"
func LoadFakeImageFromEmbedFs(filename string, symbolName string) (*FakeImage, error) {
	path := "fakeclock/" + filename
	log.Printf("[LOAD DEBUG] %s: reading %s", symbolName, path)
	object, err := fakeclock.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%T read file from embedded fs %s", err, path)
	}

	elfFile, err := elf.NewFile(bytes.NewReader(object))
	if err != nil {
		return nil, fmt.Errorf("%T parse elf file %s", err, path)
	}

	syms, err := elfFile.Symbols()
	if err != nil {
		return nil, fmt.Errorf("%T get symbols %s", err, path)
	}

	var imageContent []byte
	imageOffset := make(map[string]int)

	for _, r := range elfFile.Sections {
		if r.Type == elf.SHT_PROGBITS && r.Name == textSection {
			log.Printf("[LOAD DEBUG] %s: importing textSection", symbolName)
			imageContent, err = r.Data()
			if err != nil {
				return nil, fmt.Errorf("%T read text section data %s", err, path)
			}
			break
		}
	}

	for _, r := range elfFile.Sections {
		if r.Type == elf.SHT_RELA && r.Name == relocationSection {
			log.Printf("[LOAD DEBUG] %s: importing relocationSection", symbolName)
			relaSection, err := r.Data()
			if err != nil {
				return nil, fmt.Errorf("%T read rela section data %s", err, path)
			}
			relaSectionReader := bytes.NewReader(relaSection)

			var rela elf.Rela64
			for relaSectionReader.Len() > 0 {
				err := binary.Read(relaSectionReader, elfFile.ByteOrder, &rela)
				if err != nil {
					return nil, fmt.Errorf("%T read rela section rela64 entry %s", err, path)
				}

				symNo := rela.Info >> 32
				if symNo == 0 || symNo > uint64(len(syms)) {
					continue
				}

				sym := syms[symNo-1]
				byteorder := elfFile.ByteOrder
				if elfFile.Machine == elf.EM_X86_64 || elfFile.Machine == elf.EM_AARCH64 {
					log.Printf("[LOAD DEBUG] %s: loading %s from %#x with %s", symbolName, sym.Name, rela.Off, byteorder)
					AssetLD(rela, imageOffset, &imageContent, sym, byteorder)
				} else {
					return nil, fmt.Errorf("unsupported architecture in %s: '%s'", path, elfFile.Machine)
				}
			}

			break
		}
	}
	return NewFakeImage(
		symbolName,
		imageContent,
		imageOffset,
	), nil
}
