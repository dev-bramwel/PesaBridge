package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dev-bramwel/pesabridge"
)

// loadDotEnv parses a simple key=value .env file if it exists.
func loadDotEnv(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

func main() {
	phoneFlag := flag.String("phone", "", "Customer phone number (e.g. 0712345678 or 254712345678)")
	amountFlag := flag.Int64("amount", 1, "Amount in KES to initiate")
	accountRefFlag := flag.String("ref", "TestOrder-001", "Account or invoice reference")
	descFlag := flag.String("desc", "Sandbox Test Payment", "Transaction description")
	portFlag := flag.Int("port", 8080, "Port for callback listener")
	listenFlag := flag.Bool("listen", true, "Run local HTTP callback listener server")
	flag.Parse()

	// Load .env if present
	loadDotEnv(".env")

	// Get ngrok / callback url
	callbackURL := os.Getenv("MPESA_CALLBACK_URL")
	if callbackURL == "" {
		log.Println("MPESA_CALLBACK_URL is not set.")
		log.Println("If using ngrok, start ngrok in another terminal: ngrok http 8080")
		log.Println("Then set MPESA_CALLBACK_URL=https://<your-subdomain>.ngrok-free.app/callback")
	}

	cfg, err := pesabridge.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Config error: %v\nMake sure to provide MPESA_CONSUMER_KEY, MPESA_CONSUMER_SECRET, MPESA_PASSKEY, and MPESA_CALLBACK_URL.", err)
	}

	phone := *phoneFlag
	if phone == "" {
		phone = os.Getenv("MPESA_TEST_PHONE")
	}
	if phone == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter phone number to receive STK prompt (e.g. 07XXXXXXXX): ")
		text, _ := reader.ReadString('\n')
		phone = strings.TrimSpace(text)
	}

	if phone == "" {
		log.Fatal("Phone number is required. Pass --phone or set MPESA_TEST_PHONE.")
	}

	// Optionally start the callback listener so the ngrok webhook can be received
	if *listenFlag {
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			log.Printf("\n==== [CALLBACK RECEIVED via %s] ====\n", r.RemoteAddr)

			var prettyJSON map[string]interface{}
			if err := json.Unmarshal(body, &prettyJSON); err == nil {
				formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
				log.Printf("%s\n", string(formatted))
			} else {
				log.Printf("%s\n", string(body))
			}
			log.Println("=======================================")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ResultCode":0,"ResultDesc":"Accepted"}`))
		})

		go func() {
			addr := ":" + strconv.Itoa(*portFlag)
			log.Printf("Callback listener active at http://0.0.0.0%s/callback", addr)
			if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
				log.Printf("Callback server error: %v", err)
			}
		}()
	}

	client, err := pesabridge.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize PesaBridge client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	log.Printf("Initiating STK Push to %s for KES %d (Ref: %s)...", phone, *amountFlag, *accountRefFlag)
	resp, err := client.InitiateSTK(ctx, pesabridge.PaymentRequest{
		Amount:           *amountFlag,
		PhoneNumber:      phone,
		AccountReference: *accountRefFlag,
		Description:      *descFlag,
	})

	if err != nil {
		log.Fatalf("STK Push failed: %v", err)
	}

	fmt.Println("\n==========================================")
	fmt.Println(" STK PUSH INITIATED SUCCESSFULLY! ")
	fmt.Println("==========================================")
	fmt.Printf("MerchantRequestID:   %s\n", resp.MerchantRequestID)
	fmt.Printf("CheckoutRequestID:   %s\n", resp.CheckoutRequestID)
	fmt.Printf("ResponseCode:        %s\n", resp.ResponseCode)
	fmt.Printf("ResponseDescription: %s\n", resp.ResponseDescription)
	fmt.Printf("CustomerMessage:     %s\n", resp.CustomerMessage)
	fmt.Println("==========================================")

	if *listenFlag {
		fmt.Println("Waiting for Daraja callback from ngrok (Press Ctrl+C to exit)...")
		select {}
	}
}
