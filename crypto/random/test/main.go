package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/safing/portbase/crypto/random"
)

func noise() {
	// do some aes ctr for noise

	key, _ := hex.DecodeString("6368616e676520746869732070617373")
	data := []byte("some plaintext x")

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCTR(block, iv)
	for {
		stream.XORKeyStream(data, data)
	}

}

func main() {
	// generates 1MB and writes to stdout

	runtime.GOMAXPROCS(1)

	if len(os.Args) < 2 {
		fmt.Printf("usage: ./%s {fortuna|tickfeeder}\n", os.Args[0])
		os.Exit(1)
	}

	os.Stderr.WriteString("writing 1MB to stdout, a \".\" will be printed at every 1024 bytes.\n")

	var bytesWritten int

	switch os.Args[1] {
	case "fortuna":

		err := random.Start()
		if err != nil {
			panic(err)
		}

		for {
			b, err := random.Bytes(64)
			if err != nil {
				panic(err)
			}
			os.Stdout.Write(b)

			bytesWritten += 64
			if bytesWritten%1024 == 0 {
				os.Stderr.WriteString(".")
			}
			if bytesWritten%65536 == 0 {
				fmt.Fprintf(os.Stderr, "\n%d bytes written\n", bytesWritten)
			}
			if bytesWritten >= 1000000 {
				os.Stderr.WriteString("\n")
				break
			}
		}

		os.Exit(0)
	case "tickfeeder":

		go noise()

		var value int64
		var pushes int

		for {
			time.Sleep(10 * time.Nanosecond)

			value = (value << 1) | (time.Now().UnixNano() % 2)
			pushes++

			if pushes >= 64 {
				b := make([]byte, 8)
				binary.LittleEndian.PutUint64(b, uint64(value))
				// fmt.Fprintf(os.Stderr, "write: %d\n", value)
				os.Stdout.Write(b)
				bytesWritten += 8
				if bytesWritten%1024 == 0 {
					os.Stderr.WriteString(".")
				}
				if bytesWritten%65536 == 0 {
					fmt.Fprintf(os.Stderr, "\n%d bytes written\n", bytesWritten)
				}
				pushes = 0
			}

			if bytesWritten >= 1000000 {
				os.Stderr.WriteString("\n")
				break
			}
		}

		os.Exit(0)
	default:
		fmt.Printf("usage: %s {fortuna|tickfeeder}\n", os.Args[0])
		os.Exit(1)
	}

}
