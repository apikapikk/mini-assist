package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"mini-assisted/internal/chatlog"
)

func main() {
	inPath := flag.String("in", "", "Path ke berkas teks ekspor WhatsApp")
	outPath := flag.String("out", "", "Path ke berkas keluaran JSONL (default: stdout)")
	sender := flag.String("sender", "", "Filter nama pengirim yang diambil (opsional)")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "Error: flag -in wajib diisi")
		flag.Usage()
		os.Exit(1)
	}

	inFile, err := os.Open(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membuka berkas input %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	defer inFile.Close()

	opts := chatlog.Options{
		SenderFilter: *sender,
	}

	messages, err := chatlog.Parse(inFile, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing chat: %v\n", err)
		os.Exit(1)
	}

	var outWriter io.Writer = os.Stdout
	if *outPath != "" {
		outFile, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error membuat berkas output %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		defer outFile.Close()
		outWriter = outFile
	}

	enc := json.NewEncoder(outWriter)
	for _, msg := range messages {
		if err := enc.Encode(msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	}

	filterInfo := "semua pengirim"
	if *sender != "" {
		filterInfo = fmt.Sprintf("pengirim '%s'", *sender)
	}
	fmt.Fprintf(os.Stderr, "Sukses: Berhasil mem-parse %d pesan (%s)\n", len(messages), filterInfo)
}
