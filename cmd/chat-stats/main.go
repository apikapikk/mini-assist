package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"mini-assisted/internal/chatlog"
	"mini-assisted/internal/chatstats"
)

func main() {
	inPath := flag.String("in", "", "Path ke berkas JSONL hasil chat-filter")
	outPath := flag.String("out", "", "Path ke berkas JSON hasil statistik (default: stdout)")
	minCount := flag.Int("min-count", 10, "Batas minimal kemunculan token/frasa (default: 10)")
	sender := flag.String("sender", "", "Filter nama pengirim tertentu jika berkas masih berisi beragam pengirim")
	flag.Parse()

	if *inPath == "" {
		fmt.Fprintln(os.Stderr, "Error: flag -in wajib diisi")
		flag.Usage()
		os.Exit(1)
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
		if *sender != "" && !strings.EqualFold(strings.TrimSpace(msg.Sender), strings.TrimSpace(*sender)) {
			continue
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca input JSONL: %v\n", err)
		os.Exit(1)
	}

	result := chatstats.Analyze(messages, chatstats.Options{
		MinTokenCount: *minCount,
	})

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
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON hasil statistik: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Statistik selesai dihitung dari %d pesan (min-count: %d)\n", result.TotalMessages, *minCount)
}
