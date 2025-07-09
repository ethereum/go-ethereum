package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/scroll-tech/go-ethereum/common"
	coreTypes "github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
	"github.com/scroll-tech/go-ethereum/rollup/missing_header_fields"
)

// dedupCmd represents the dedup command
var dedupCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Deduplicate the headers file, print unique values and create a new file with the deduplicated headers",
	Long: `Deduplicate the headers file, print unique values and create a new file with the deduplicated headers.

The binary layout of the deduplicated file is as follows:
- 1 byte for the count of unique vanity
- 32 bytes for each unique vanity
- for each header:
  - 1 byte (bitmask, lsb first): 
	- bit 0-5: index of the vanity in the sorted vanities list
	- bit 6: 0 if difficulty is 2, 1 if difficulty is 1
	- bit 7: 0 if seal length is 65, 1 if seal length is 85
  - 65 or 85 bytes for the seal`,
	Run: func(cmd *cobra.Command, args []string) {
		inputFile, err := cmd.Flags().GetString("input")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}
		outputFile, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}
		verifyFile, err := cmd.Flags().GetString("verify")
		if err != nil {
			log.Fatalf("Error reading verify flag: %v", err)
		}

		// uncomment the following line to copy from the verify file to the input file. This is useful to generate a deduplicated header file for testing purposes.
		// copyFromVerifyFile(verifyFile, inputFile)

		if verifyFile != "" {
			verifyInputFile(verifyFile, inputFile)
		}

		_, seenVanity, _ := runAnalysis(inputFile)
		runDedup(inputFile, outputFile, seenVanity)

		if verifyFile != "" {
			verifyOutputFile(verifyFile, outputFile)
		}

		runSHA256(outputFile)
	},
}

func init() {
	rootCmd.AddCommand(dedupCmd)

	dedupCmd.Flags().String("input", "headers.bin", "headers file")
	dedupCmd.Flags().String("output", "headers-dedup.bin", "deduplicated, binary formatted file")
	dedupCmd.Flags().String("verify", "", "verify the input and output files with the given .csv file")
}

func runAnalysis(inputFile string) (seenDifficulty map[uint64]int, seenVanity map[[32]byte]bool, seenSealLen map[int]int) {
	reader := newHeaderReader(inputFile)
	defer reader.close()

	// track header fields we've seen
	seenDifficulty = make(map[uint64]int)
	seenVanity = make(map[[32]byte]bool)
	seenSealLen = make(map[int]int)

	reader.read(func(header *types.Header) {
		seenDifficulty[header.Difficulty]++
		seenVanity[header.Vanity()] = true
		seenSealLen[header.SealLen()]++
	})

	// Print distinct values and report
	fmt.Println("--------------------------------------------------")
	for diff, count := range seenDifficulty {
		fmt.Printf("Difficulty %d: %d\n", diff, count)
	}

	for vanity := range seenVanity {
		fmt.Printf("Vanity: %x\n", vanity)
	}

	for sealLen, count := range seenSealLen {
		fmt.Printf("SealLen %d bytes: %d\n", sealLen, count)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Printf("Unique values seen in the headers file (last seen block: %d):\n", reader.lastHeader.Number)
	fmt.Printf("Distinct count: Difficulty:%d, Vanity:%d, SealLen:%d\n", len(seenDifficulty), len(seenVanity), len(seenSealLen))
	fmt.Printf("--------------------------------------------------\n\n")

	return seenDifficulty, seenVanity, seenSealLen
}

func runDedup(inputFile, outputFile string, seenVanity map[[32]byte]bool) {
	reader := newHeaderReader(inputFile)
	defer reader.close()

	writer := newMissingHeaderFileWriter(outputFile, seenVanity)
	defer writer.close()

	writer.missingHeaderWriter.writeVanities()

	reader.read(func(header *types.Header) {
		writer.missingHeaderWriter.write(header)
	})
}

func runSHA256(outputFile string) {
	f, err := os.Open(outputFile)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		log.Fatalf("Error hashing file: %v", err)
	}

	fmt.Printf("Deduplicated headers written to %s with sha256 checksum: %x\n", outputFile, h.Sum(nil))
}

type headerReader struct {
	file       *os.File
	reader     *bufio.Reader
	lastHeader *types.Header
}

func newHeaderReader(inputFile string) *headerReader {
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening input file: %v", err)
	}

	h := &headerReader{
		file:   f,
		reader: bufio.NewReader(f),
	}

	return h
}

func (h *headerReader) read(callback func(header *types.Header)) {
	headerSizeBytes := make([]byte, types.HeaderSizeSerialized)

	for {
		_, err := io.ReadFull(h.reader, headerSizeBytes)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading headerSizeBytes: %v\n", err)
		}
		headerSize := binary.BigEndian.Uint16(headerSizeBytes)

		headerBytes := make([]byte, headerSize)
		_, err = io.ReadFull(h.reader, headerBytes)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error reading headerBytes: %v\n", err)
		}
		header := new(types.Header).FromBytes(headerBytes)

		// sanity check: make sure headers are in order
		if h.lastHeader != nil && header.Number != h.lastHeader.Number+1 {
			fmt.Println("lastHeader:", h.lastHeader.String())
			log.Fatalf("Missing block: %d, got %d instead", h.lastHeader.Number+1, header.Number)
		}
		h.lastHeader = header

		callback(header)
	}
}

func (h *headerReader) close() {
	h.file.Close()
}

type csvHeaderReader struct {
	file   *os.File
	reader *bufio.Reader
}

func newCSVHeaderReader(verifyFile string) *csvHeaderReader {
	f, err := os.Open(verifyFile)
	if err != nil {
		log.Fatalf("Error opening verify file: %v", err)
	}

	h := &csvHeaderReader{
		file:   f,
		reader: bufio.NewReader(f),
	}

	return h
}

func (h *csvHeaderReader) readNext() *types.Header {
	line, err := h.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil
		}
		log.Fatalf("Error reading line: %v", err)
	}

	s := strings.Split(strings.TrimSpace(line), ",")
	if len(s) != 6 {
		log.Fatalf("Malformed CSV line: %q", line)
	}

	num, err := strconv.ParseUint(s[0], 10, 64)
	if err != nil {
		log.Fatalf("Error parsing block number: %v", err)
	}
	difficulty, err := strconv.ParseUint(s[1], 10, 64)
	if err != nil {
		log.Fatalf("Error parsing difficulty: %v", err)
	}

	stateRoot := common.HexToHash(s[2])
	coinbase := common.HexToAddress(s[3])
	nonceBytes := common.Hex2Bytes(s[4])
	extra := common.FromHex(strings.Split(s[5], "\n")[0])

	header := types.NewHeader(num, difficulty, stateRoot, coinbase, coreTypes.BlockNonce(nonceBytes), extra)
	return header
}

func (h *csvHeaderReader) close() {
	h.file.Close()
}

func copyFromVerifyFile(verifyFile, inputFile string) {
	fmt.Println("Copying from", verifyFile, "to", inputFile)

	csvReader := newCSVHeaderReader(verifyFile)
	defer csvReader.close()

	writer := newFilesWriter(inputFile, "")
	defer writer.close()

	for header := csvReader.readNext(); header != nil; header = csvReader.readNext() {
		writer.write(header)
	}
}

func verifyInputFile(verifyFile, inputFile string) {
	csvReader := newCSVHeaderReader(verifyFile)
	defer csvReader.close()

	binaryReader := newHeaderReader(inputFile)
	defer binaryReader.close()

	binaryReader.read(func(header *types.Header) {
		csvHeader := csvReader.readNext()

		if !csvHeader.Equal(header) {
			log.Fatalf("Header mismatch: %v != %v", csvHeader, header)
		}
	})

	log.Printf("All headers match in %s and %s\n", verifyFile, inputFile)
}

func verifyOutputFile(verifyFile, outputFile string) {
	csvReader := newCSVHeaderReader(verifyFile)
	defer csvReader.close()

	dedupReader, err := missing_header_fields.NewReader(outputFile)
	if err != nil {
		log.Fatalf("Error opening dedup file: %v", err)
	}
	defer dedupReader.Close()

	for {
		header := csvReader.readNext()
		if header == nil {
			if err = dedupReader.ReadNext(); err == nil {
				log.Fatalf("Expected EOF, got more headers")
			}
			break
		}

		difficulty, stateRoot, coinbase, nonce, extraData, err := dedupReader.Read(header.Number)
		if err != nil {
			log.Fatalf("Error reading header: %v", err)
		}

		if header.Difficulty != difficulty {
			log.Fatalf("Difficulty mismatch: headerNum %d: %d != %d", header.Number, header.Difficulty, difficulty)
		}
		if header.StateRoot != stateRoot {
			log.Fatalf("StateRoot mismatch: headerNum %d: %s != %s", header.Number, header.StateRoot, stateRoot)
		}
		if header.Coinbase != coinbase {
			log.Fatalf("Coinbase mismatch: headerNum %d: %s != %s", header.Number, header.Coinbase.Hex(), coinbase.Hex())
		}
		if header.Nonce != nonce {
			log.Fatalf("Nonce mismatch: headerNum %d: %s != %s", header.Number, common.Bytes2Hex(header.Nonce[:]), common.Bytes2Hex(nonce[:]))
		}
		if !bytes.Equal(header.ExtraData, extraData) {
			log.Fatalf("ExtraData mismatch: headerNum %d: %x != %x", header.Number, header.ExtraData, extraData)
		}
	}

	log.Printf("All headers match in %s and %s\n", verifyFile, outputFile)
}
