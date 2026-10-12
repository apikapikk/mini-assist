// cmd/genhash adalah alat bantu untuk menghasilkan bcrypt hash dari password.
// Jalankan sekali saat setup, salin hasilnya ke OWNER_PASSWORD_HASH di .env
//
// Penggunaan:
//
//	go run ./cmd/genhash
//	go run ./cmd/genhash mysecretpassword
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var password string

	if len(os.Args) >= 2 {
		password = os.Args[1]
	} else {
		fmt.Print("Masukkan password: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			password = strings.TrimSpace(scanner.Text())
		}
	}

	if password == "" {
		fmt.Fprintln(os.Stderr, "Error: password tidak boleh kosong")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generate hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Hash bcrypt:")
	fmt.Println(string(hash))
	fmt.Println()
	fmt.Println("Salin baris ini ke .env:")
	fmt.Printf("OWNER_PASSWORD_HASH=%s\n", string(hash))
}
