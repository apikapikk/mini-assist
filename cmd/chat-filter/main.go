package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"mini-assisted/internal/chatfilter"
	"mini-assisted/internal/chatlog"
)

func main() {
	inPath := flag.String("in", "", "Path ke berkas JSONL hasil chat-parse")
	outPath := flag.String("out", "", "Path ke berkas JSONL hasil filter (default: stdout)")
	blocklistPath := flag.String("blocklist", "", "Path ke berkas blocklist nama/tempat (opsional)")
	sensitivePath := flag.String("sensitif", "config/sensitif.txt", "Path ke berkas kata kunci sensitif (default: config/sensitif.txt)")
	reviewPath := flag.String("review", "", "Path ke berkas tinjauan pesan meragukan (opsional)")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "Error: flag -in wajib diisi")
		flag.Usage()
		os.Exit(1)
	}

	sensitiveWords, err := chatfilter.LoadWordList(*sensitivePath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Peringatan gagal membaca sensitif list (%s): %v\n", *sensitivePath, err)
	}

	var blocklistWords []string
	if *blocklistPath != "" {
		bw, err := chatfilter.LoadWordList(*blocklistPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Peringatan gagal membaca blocklist (%s): %v\n", *blocklistPath, err)
		} else {
			blocklistWords = bw
		}
	}

	cfg := chatfilter.Config{
		SensitiveKeywords: sensitiveWords,
		Blocklist:         blocklistWords,
	}

	inFile, err := os.Open(*inPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membuka input %s: %v\n", *inPath, err)
		os.Exit(1)
	}
	defer inFile.Close()

	var messages []chatlog.Message
	scanner := bufio.NewScanner(inFile)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg chatlog.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca input JSONL: %v\n", err)
		os.Exit(1)
	}

	accepted, questionable, stats := chatfilter.ProcessMessages(messages, cfg)

	var outWriter io.Writer = os.Stdout
	if *outPath != "" {
		outFile, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error membuat output %s: %v\n", *outPath, err)
			os.Exit(1)
		}
		defer outFile.Close()
		outWriter = outFile
	}

	enc := json.NewEncoder(outWriter)
	for _, m := range accepted {
		if err := enc.Encode(m); err != nil {
			fmt.Fprintf(os.Stderr, "Error menulis output JSONL: %v\n", err)
			os.Exit(1)
		}
	}

	if *reviewPath != "" && len(questionable) > 0 {
		reviewFile, err := os.Create(*reviewPath)
		if err == nil {
			defer reviewFile.Close()
			revEnc := json.NewEncoder(reviewFile)
			for _, q := range questionable {
				_ = revEnc.Encode(q)
			}
		}
	}

	// Cetak laporan HANYA berupa angka sesuai ketentuan SCOPE.md
	fmt.Fprintf(os.Stderr, "Masuk: %d, Diterima: %d, Dibuang: %d, Diragukan: %d\n",
		stats.TotalInput, stats.Accepted, stats.Dropped, stats.Questionable)
}
