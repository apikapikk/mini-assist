// cmd/genvapid menghasilkan pasangan kunci VAPID publik dan privat untuk Web Push.
// Jalankan sekali saat setup dan salin ke .env
package main

import (
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("Gagal generate VAPID keys: %v", err)
	}

	fmt.Println("=== Kunci VAPID Berhasil Dibuat ===")
	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", publicKey)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privateKey)
	fmt.Println()
	fmt.Println("Salin kedua baris di atas ke file .env kamu!")
}
