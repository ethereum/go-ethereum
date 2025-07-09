package cmd

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/scroll-tech/go-ethereum/ethclient"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch missing block header fields from a running Scroll L2 node via RPC and store in a file",
	Long: `Fetch allows to retrieve the missing block header fields from a running Scroll L2 node via RPC.
It produces a binary file and optionally a human readable csv file with the missing fields.`,
	Run: func(cmd *cobra.Command, args []string) {
		rpcs, err := cmd.Flags().GetString("rpc")
		if err != nil {
			log.Fatalf("Error reading rpc flag: %v", err)
		}
		if rpcs == "" {
			log.Fatal("No RPC URLs provided, please use the --rpc flag to specify at least one RPC URL.")
		}
		rpcNodes := strings.Split(rpcs, ",")
		var clients []*ethclient.Client
		for _, rpc := range rpcNodes {
			client, err := ethclient.Dial(rpc)
			if err != nil {
				log.Fatalf("Error connecting to RPC: %v", err)
			}
			clients = append(clients, client)
		}
		startBlockNum, err := cmd.Flags().GetUint64("start")
		if err != nil {
			log.Fatalf("Error reading start flag: %v", err)
		}
		endBlockNum, err := cmd.Flags().GetUint64("end")
		if err != nil {
			log.Fatalf("Error reading end flag: %v", err)
		}
		batchSize, err := cmd.Flags().GetUint64("batch")
		if err != nil {
			log.Fatalf("Error reading batch flag: %v", err)
		}
		maxParallelGoroutines, err := cmd.Flags().GetInt("parallelism")
		if err != nil {
			log.Fatalf("Error reading parallelism flag: %v", err)
		}
		outputFile, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}
		humanReadableOutputFile, err := cmd.Flags().GetString("humanOutput")
		if err != nil {
			log.Fatalf("Error reading humanReadable flag: %v", err)
		}
		continueFile, err := cmd.Flags().GetString("continue")
		if err != nil {
			log.Fatalf("Error reading continue flag: %v", err)
		}
		dbDSN, err := cmd.Flags().GetString("db")
		if err != nil {
			log.Fatalf("Error reading db flag: %v", err)
		}

		var db *gorm.DB
		if dbDSN != "" {
			db, err = gorm.Open(postgres.Open(dbDSN), &gorm.Config{})
			if err != nil {
				log.Fatalf("Error connecting to database: %v", err)
			}
		}

		if continueFile != "" {
			fmt.Println("Continue fetching block header fields from", continueFile)

			reader := newHeaderReader(continueFile)
			defer reader.close()

			var lastSeenHeader uint64
			reader.read(func(header *types.Header) {
				lastSeenHeader = header.Number
			})
			fmt.Println("Last Seen Header:", lastSeenHeader)

			startBlockNum = lastSeenHeader + 1

			fmt.Println("Overriding start block number to:", startBlockNum)

			if startBlockNum > endBlockNum {
				log.Fatalf("Start block number %d exceeds end block number %d after continuing from file", startBlockNum, endBlockNum)
			}
		}

		runFetch(clients, db, startBlockNum, endBlockNum, batchSize, maxParallelGoroutines, outputFile, humanReadableOutputFile, continueFile)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().String("rpc", "http://localhost:8545,http://localhost:8546", "RPC URLs, separated by commas. Example: http://localhost:8545,http://localhost:8546")
	fetchCmd.Flags().Uint64("start", 0, "start block number")
	fetchCmd.Flags().Uint64("end", 1000, "end block number")
	fetchCmd.Flags().Uint64("batch", 100, "batch size")
	fetchCmd.Flags().Int("parallelism", 10, "max parallel goroutines each working on batch size blocks")
	fetchCmd.Flags().String("output", "headers.bin", "output file")
	fetchCmd.Flags().String("humanOutput", "", "additionally produce human readable csv file")
	fetchCmd.Flags().String("continue", "", "continue fetching block header fields from the last seen block number in the specified continue file")
	fetchCmd.Flags().String("db", "", "database to use instead of fetching from RPC")
}

func headerByNumberWithRetry(client *ethclient.Client, blockNum uint64, maxRetries int) (*types.Header, error) {
	var innerErr error
	for i := 0; i < maxRetries; i++ {
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(blockNum)))
		if err == nil {
			return types.NewHeader(
				header.Number.Uint64(),
				header.Difficulty.Uint64(),
				header.Root,
				header.Coinbase,
				header.Nonce,
				header.Extra,
			), nil
		}

		innerErr = err // save the last error to return it if all retries fail

		// Wait before retrying
		time.Sleep(time.Duration(i*200) * time.Millisecond)
		log.Printf("Retrying header fetch for block %d, retry %d, error %v", blockNum, i+1, err)
	}

	return nil, fmt.Errorf("error fetching header for block %d: %v", blockNum, innerErr)
}

func fetchHeadersFromDB(db *gorm.DB, start, end uint64, headersChan chan<- *types.Header) {
	blockORM := types.NewL2Block(db)
	blocks, err := blockORM.GetL2BlocksInRange(context.Background(), start, end)
	if err != nil {
		log.Fatalf("Error fetching blocks from database: %v", err)
	}

	for _, block := range blocks {
		headersChan <- types.NewHeader(
			block.Header.Number.Uint64(),
			block.Header.Difficulty.Uint64(),
			block.Header.Root,
			block.Header.Coinbase,
			block.Header.Nonce,
			block.Header.Extra,
		)
	}
}

func fetchHeadersFromRPC(clients []*ethclient.Client, start, end uint64, headersChan chan<- *types.Header) {
	// randomize client selection to distribute load
	r := uint64(rand.Int())

	// log time taken for fetching headers
	startTime := time.Now()
	var fetchTimeAvg, writeTimeAvg time.Duration

	for i := start; i <= end; i++ {
		startTimeBlockFetch := time.Now()
		client := clients[(r+i)%uint64(len(clients))] // round-robin client selection
		header, err := headerByNumberWithRetry(client, i, 15)
		if err != nil {
			log.Fatalf("Error fetching header %d: %v", i, err)
		}
		fetchTimeAvg += time.Since(startTimeBlockFetch)

		startTimeHeaderWrite := time.Now()
		headersChan <- header
		writeTimeAvg += time.Since(startTimeHeaderWrite)
	}
	totalDuration := time.Since(startTime)

	fetchTimeAvg = fetchTimeAvg / time.Duration(end-start+1)
	writeTimeAvg = writeTimeAvg / time.Duration(end-start+1)
	log.Printf("Fetched %d header in %s (avg=%s, wrote to channel in avg %s", end-start+1, totalDuration, fetchTimeAvg, writeTimeAvg)
}

func writeHeadersToFile(outputFile string, humanReadableOutputFile string, continueFile string, startBlockNum uint64, headersChan <-chan *types.Header) {
	writer := newFilesWriter(outputFile, humanReadableOutputFile)
	defer writer.close()

	if continueFile != "" {
		reader := newHeaderReader(continueFile)

		var lastSeenHeader uint64
		var totalHeaders uint64
		reader.read(func(header *types.Header) {
			writer.write(header)
			totalHeaders++
			lastSeenHeader = header.Number
		})

		fmt.Println("Copied ", totalHeaders, "headers from continue file, last seen block number:", lastSeenHeader)
		reader.close()
	}

	headerHeap := &types.HeaderHeap{}
	heap.Init(headerHeap)

	nextHeaderNum := startBlockNum

	// receive all headers and write them in order by using a sorted heap
	for header := range headersChan {
		heap.Push(headerHeap, header)

		// write all headers that are in order
		for headerHeap.Len() > 0 && (*headerHeap)[0].Number == nextHeaderNum {
			nextHeaderNum++
			sortedHeader := heap.Pop(headerHeap).(*types.Header)
			writer.write(sortedHeader)
		}
	}

	fmt.Println("Finished writing headers to file, last block number:", nextHeaderNum-1)
}

func runFetch(clients []*ethclient.Client, db *gorm.DB, startBlockNum uint64, endBlockNum uint64, batchSize uint64, maxGoroutines int, outputFile string, humanReadableOutputFile string, continueFile string) {
	headersChan := make(chan *types.Header, maxGoroutines*int(batchSize))
	tasks := make(chan task)

	var wgConsumer sync.WaitGroup
	// start consumer goroutine to sort and write headers to file
	wgConsumer.Add(1)
	go func() {
		writeHeadersToFile(outputFile, humanReadableOutputFile, continueFile, startBlockNum, headersChan)
		wgConsumer.Done()
	}()

	var wgProducers sync.WaitGroup
	// start producer goroutines to fetch headers
	for i := 0; i < maxGoroutines; i++ {
		wgProducers.Add(1)
		go func() {
			for {
				t, ok := <-tasks
				if !ok {
					break
				}
				log.Println("Received task", t.start, "to", t.end)

				// use DB if dbDSN is provided, otherwise fetch from RPC
				if db != nil {
					fetchHeadersFromDB(db, t.start, t.end, headersChan)
				} else {
					fetchHeadersFromRPC(clients, t.start, t.end, headersChan)
				}
			}
			wgProducers.Done()
		}()
	}

	// need to fetch block 0 from RPC
	if startBlockNum == 0 && db != nil {
		fmt.Println("Fetching headers from database... and header 0 from RPC")
		fetchHeadersFromRPC(clients, 0, 0, headersChan)
		startBlockNum = 1
	}

	// create tasks/work packages for producer goroutines
	for start := startBlockNum; start <= endBlockNum; start += batchSize {
		end := start + batchSize - 1
		if end > endBlockNum {
			end = endBlockNum
		}
		fmt.Println("Created task for blocks", start, "to", end)

		tasks <- task{start, end}
	}

	close(tasks)
	wgProducers.Wait()
	close(headersChan)
	wgConsumer.Wait()
}

type task struct {
	start uint64
	end   uint64
}

// filesWriter is a helper struct to write headers to binary and human-readable csv files at the same time.
type filesWriter struct {
	binaryFile   *os.File
	binaryWriter *bufio.Writer

	humanReadable bool
	csvFile       *os.File
	csvWriter     *bufio.Writer
}

func newFilesWriter(outputFile string, humanReadableOutputFile string) *filesWriter {
	binaryFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating binary file: %v", err)
	}

	f := &filesWriter{
		binaryFile:    binaryFile,
		binaryWriter:  bufio.NewWriter(binaryFile),
		humanReadable: humanReadableOutputFile != "",
	}

	if humanReadableOutputFile != "" {
		csvFile, err := os.Create(humanReadableOutputFile)
		if err != nil {
			log.Fatalf("Error creating human readable file: %v", err)
		}
		f.csvFile = csvFile
		f.csvWriter = bufio.NewWriter(csvFile)
	}

	return f
}

func (f *filesWriter) close() {
	if err := f.binaryWriter.Flush(); err != nil {
		log.Fatalf("Error flushing binary buffer: %v", err)
	}
	if f.humanReadable {
		if err := f.csvWriter.Flush(); err != nil {
			log.Fatalf("Error flushing csv buffer: %v", err)
		}
	}

	f.binaryFile.Close()
	if f.humanReadable {
		f.csvFile.Close()
	}
}
func (f *filesWriter) write(header *types.Header) {
	bytes, err := header.Bytes()
	if err != nil {
		log.Fatalf("Error converting header to bytes: %v", err)
	}

	if _, err = f.binaryWriter.Write(bytes); err != nil {
		log.Fatalf("Error writing to binary file: %v", err)
	}

	if f.humanReadable {
		if _, err = f.csvWriter.WriteString(header.String()); err != nil {
			log.Fatalf("Error writing to human readable file: %v", err)
		}
	}
}
