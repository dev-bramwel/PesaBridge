# Setting Up and Running Local M-Pesa STK Push with Ngrok

This guide explains how to configure, test, and recreate a local M-Pesa STK Push initiation and handle the incoming Daraja callback in the PesaBridge sandbox environment.

---

## 1. Prerequisites

1. **Go installed**: Go 1.27.1 or later installed (matching `go.mod`) (`go version`).
2. **Ngrok installed**: [ngrok CLI](https://ngrok.com/download) installed and authenticated (`ngrok config add-authtoken <token>`).
3. **Safaricom Daraja Developer Account**:
   - Register at [Safaricom Developer Portal](https://developer.safaricom.co.ke/).
   - Create a Sandbox App to obtain:
     - **Consumer Key**
     - **Consumer Secret**
     - **Business Short Code** (Default sandbox test shortcode is `174379`)
     - **PassKey** (Found on the Daraja Test Credentials page)

---

## 2. Environment Configuration

1. In the root of the project, copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Fill in your sandbox credentials:
   ```env
   MPESA_ENVIRONMENT=sandbox
   MPESA_CONSUMER_KEY=your_sandbox_consumer_key
   MPESA_CONSUMER_SECRET=your_sandbox_consumer_secret
   MPESA_BUSINESS_SHORT_CODE=174379
   MPESA_PASSKEY=your_sandbox_passkey

   # Callback URL will be populated in the next step
   MPESA_CALLBACK_URL=https://<your-subdomain>.ngrok-free.app/callback

   # Optional default test phone number (e.g., 07XXXXXXXX or 2547XXXXXXXX)
   MPESA_TEST_PHONE=07XXXXXXXX
   ```

> [!IMPORTANT]
> The `.env` file is excluded in `.gitignore` to prevent leaking sandbox credentials. Never commit credentials to version control.

---

## 3. Exposing the Local Callback Listener via Ngrok

Safaricom requires a publicly accessible HTTPS callback URL to deliver the asynchronous transaction status after a customer enters their PIN.

1. Open a dedicated terminal window and run:
   ```bash
   ngrok http 8080
   ```

2. Note the HTTPS forwarding URL from the ngrok output, for example:
   ```text
   Forwarding   https://abcdef-1234.ngrok-free.app -> http://localhost:8080
   ```

3. Update `MPESA_CALLBACK_URL` in `.env` to include the `/callback` path:
   ```env
   MPESA_CALLBACK_URL=https://abcdef-1234.ngrok-free.app/callback
   ```
   *(Ensure you include `/callback` at the end).*

---

## 4. Running the Demo Application

In another terminal, start the demo CLI runner:

```bash
go run ./cmd/stkdemo
```

You can also pass optional command line flags:
```bash
go run ./cmd/stkdemo --phone 0712345678 --amount 1 --ref INV-001 --desc "Test Payment"
```

### What Happens:
1. **Local Callback Server**: Starts listening on `http://0.0.0.0:8080/callback`.
2. **OAuth Authentication**: Calls Daraja's `/oauth/v1/generate` to obtain an access token.
3. **STK Push Initiation**: Encodes the password (`Base64(ShortCode + PassKey + Timestamp)`) and sends the request to `/mpesa/stkpush/v1/processrequest`.
4. **Phone Prompt**: You will receive an STK Push PIN prompt on your Safaricom test phone.
5. **Callback Receipt**: When you enter your PIN (or cancel), Safaricom sends the callback JSON through ngrok to your listener.

---

## 5. Verifying the Callback

### Expected Terminal Output on Success:

```json
==== [CALLBACK RECEIVED via 127.0.0.1:...] ====
{
  "Body": {
    "stkCallback": {
      "CallbackMetadata": {
        "Item": [
          {
            "Name": "Amount",
            "Value": 1
          },
          {
            "Name": "MpesaReceiptNumber",
            "Value": "TIR7XXXXXX"
          },
          {
            "Name": "TransactionDate",
            "Value": 20260926184530
          },
          {
            "Name": "PhoneNumber",
            "Value": 2547XXXXXXXX
          }
        ]
      },
      "CheckoutRequestID": "ws_CO_26092026184512345678",
      "MerchantRequestID": "29115-34620561-1",
      "ResultCode": 0,
      "ResultDesc": "The service request is processed successfully."
    }
  }
}
=======================================
```

- **`ResultCode: 0`**: Payment completed successfully.
- **`ResultCode: 1032`**: Cancelled by the user.
- **`ResultCode: 1037`**: Timeout (user took too long to enter PIN).

---

## 6. Troubleshooting

### 1. Terminal is stuck at `Waiting for Daraja callback from ngrok...`
- **Is ngrok active?** If ngrok was restarted, its URL changes (on free tier). Verify that `MPESA_CALLBACK_URL` in `.env` matches the currently active ngrok URL.
- **Test your tunnel locally**:
  ```bash
  curl -X POST https://<your-active-url>.ngrok-free.app/callback \
       -H "Content-Type: application/json" \
       -d '{"ping": "pong"}'
  ```
  If this does not immediately print in the demo terminal, check ngrok's web dashboard at [http://localhost:4040](http://localhost:4040).
- **Endpoint offline (`ERR_NGROK_3200`)**: Indicates ngrok is stopped or the subdomain in `.env` does not match the running ngrok session.

### 2. Invalid Phone Number Error
- Input numbers can be formatted as `07XXXXXXXX`, `01XXXXXXXX`, or `2547XXXXXXXX`. The library normalizes all numbers to the international `254...` format.
